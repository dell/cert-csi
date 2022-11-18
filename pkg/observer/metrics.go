package observer

import (
	"cert-csi/pkg/store"
	"context"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync"
	"time"
)

const (
	// MetricsPoll is a poll interval for StatefulSet tests
	MetricsPoll = 5 * time.Second

	// MetricsTimeout
	MetricsTimeout = 1800 * time.Second
)

type ContainerMetricsObserver struct {
	finished chan bool

	interrupted bool
	mutex       sync.Mutex
}

func (cmo *ContainerMetricsObserver) Interrupt() {
	cmo.mutex.Lock()
	defer cmo.mutex.Unlock()
	cmo.interrupted = true
}

func (cmo *ContainerMetricsObserver) Interrupted() bool {
	cmo.mutex.Lock()
	defer cmo.mutex.Unlock()
	return cmo.interrupted
}

func (cmo *ContainerMetricsObserver) StartWatching(ctx context.Context, runner *Runner) {
	defer runner.WaitGroup.Done()
	if runner.DriverNamespace == "" {
		cmo.Interrupt()
		return
	}

	var resUsage []*store.ResourceUsage
	log.Debugf("%s started watching", cmo.GetName())
	mc := runner.Clients.MetricsClient
	timeout := MetricsTimeout
	if mc != nil {
		clientTimeout := mc.Timeout
		if clientTimeout != 0 {
			timeout = time.Duration(clientTimeout) * time.Second
		}
	}

	pollErr := wait.PollImmediate(MetricsPoll, timeout, func() (bool, error) {
		select {
		case <-cmo.finished:
			log.Debugf("%s finished watching", cmo.GetName())
			return true, nil
		default:
			break
		}

		metricList, err := mc.Interface.MetricsV1beta1().PodMetricses(runner.DriverNamespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("Can't watch metricsClient. error = %v", err)
			log.Warnf("Please use instruction in README to install metrics-server")
			log.Warnf("Or use instruction at https://github.com/kubernetes-incubator/metrics-server")
			return true, err
		}

		for _, pod := range metricList.Items {
			for _, container := range pod.Containers {
				info := &store.ResourceUsage{
					TcID:          runner.TestCase.ID,
					Timestamp:     time.Now(),
					PodName:       pod.Name,
					ContainerName: container.Name,
					Cpu:           container.Usage.Cpu().MilliValue(),
					Mem:           container.Usage.Memory().Value() / (1024 * 1024),
				}
				resUsage = append(resUsage, info)
			}
		}

		return false, nil
	})

	if pollErr != nil {
		log.Errorf("Error with polling; error=%v", pollErr)
		cmo.Interrupt()
	}
	err := runner.Database.SaveResourceUsage(resUsage)
	if err != nil {
		log.Errorf("Can't save resource usage; error=%v", pollErr)
	}
}

func (cmo *ContainerMetricsObserver) StopWatching() {
	if !cmo.Interrupted() {
		cmo.finished <- true
	}
}

func (cmo *ContainerMetricsObserver) GetName() string {
	return "ContainerMetricsObserver"
}

func (cmo *ContainerMetricsObserver) MakeChannel() {
	cmo.finished = make(chan bool)
}
