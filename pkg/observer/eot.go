package observer

import (
	"cert-csi/pkg/k8sclient/resources/pod"
	"cert-csi/pkg/k8sclient/resources/pvc"
	"cert-csi/pkg/store"
	"context"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync"
	"time"
)

const (
	// EntityNumberPoll is a poll interval for StatefulSet tests
	EntityNumberPoll = 5 * time.Second

	// EntityNumberTimeout entity timeout
	EntityNumberTimeout = 1800 * time.Second
)

type EntityNumberObserver struct {
	finished chan bool

	interrupted bool
	mutex       sync.Mutex
}

func (eno *EntityNumberObserver) Interrupt() {
	eno.mutex.Lock()
	defer eno.mutex.Unlock()
	eno.interrupted = true
}

func (eno *EntityNumberObserver) Interrupted() bool {
	eno.mutex.Lock()
	defer eno.mutex.Unlock()
	return eno.interrupted
}

func (eno *EntityNumberObserver) StartWatching(ctx context.Context, runner *Runner) {
	defer runner.WaitGroup.Done()

	var nEntities []*store.NumberEntities
	log.Debugf("%s started watching", eno.GetName())
	pvcClient := runner.Clients.PVCClient
	podClient := runner.Clients.PodClient
	timeout := EntityNumberTimeout
	if pvcClient != nil {
		clientTimeout := pvcClient.Timeout
		if clientTimeout != 0 {
			timeout = time.Duration(clientTimeout) * time.Second
		}
	}

	pollErr := wait.PollImmediate(EntityNumberPoll, timeout, func() (bool, error) {
		select {
		case <-eno.finished:
			log.Debugf("%s finished watching", eno.GetName())
			return true, nil
		default:
			break
		}

		info := &store.NumberEntities{TcID: runner.TestCase.ID}
		b, e := eno.checkPods(podClient, info)
		if e != nil {
			return b, e
		}

		i, e := eno.checkPvcs(pvcClient, info)
		if e != nil {
			return i, e
		}
		info.Timestamp = time.Now()
		nEntities = append(nEntities, info)

		return false, nil
	})

	if pollErr != nil {
		log.Errorf("Error with polling; error=%v", pollErr)
		eno.Interrupt()
	}

	err := runner.Database.SaveNumberEntities(nEntities)
	if err != nil {
		log.Errorf("Can't save number of entities; error=%v", pollErr)
	}

}

func (eno *EntityNumberObserver) checkPvcs(
	pvcClient *pvc.Client,
	info *store.NumberEntities) (bool, error) {
	if pvcClient == nil {
		return false, nil
	}

	pvcList, pvcListErr := pvcClient.Interface.List(context.Background(), metav1.ListOptions{})
	if pvcListErr != nil {
		return false, pvcListErr
	}
	for _, pvc := range pvcList.Items {
		if pvc.Status.Phase == v1.ClaimPending && pvc.DeletionTimestamp == nil {
			info.PvcCreating++
			continue
		}
		if pvc.DeletionTimestamp != nil {
			info.PvcTerminating++
			continue
		}
		if pvc.Status.Phase == v1.ClaimBound {
			info.PvcBound++
			continue
		}
	}
	return false, nil
}

func (eno *EntityNumberObserver) checkPods(
	podClient *pod.Client,
	info *store.NumberEntities) (bool, error) {
	if podClient == nil {
		return false, nil
	}
	podList, podListErr := podClient.Interface.List(context.Background(), metav1.ListOptions{})
	if podListErr != nil {
		return false, podListErr
	}
	for i, p := range podList.Items {
		if p.Status.Phase == v1.PodPending && p.DeletionTimestamp == nil {
			info.PodsCreating++
			continue
		}
		if p.DeletionTimestamp != nil {
			info.PodsTerminating++
			continue
		}
		if p.Status.Phase == v1.PodRunning && pod.IsPodReady(&podList.Items[i]) {
			info.PodsReady++
			continue
		}
	}
	return false, nil
}

func (eno *EntityNumberObserver) StopWatching() {
	if !eno.Interrupted() {
		eno.finished <- true
	}
}

func (eno *EntityNumberObserver) GetName() string {
	return "ContainerMetricsObserver"
}

func (eno *EntityNumberObserver) MakeChannel() {
	eno.finished = make(chan bool)
}
