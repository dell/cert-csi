/*
 *
 * Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package observer

import (
	"context"
	"sync"
	"time"

	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// MetricsPoll is a poll interval for StatefulSet tests
	MetricsPoll = 5 * time.Second

	// MetricsTimeout is metrics timeout
	MetricsTimeout = 1800 * time.Second
)

// ContainerMetricsObserver is used to manage container metrics observer
type ContainerMetricsObserver struct {
	finished chan bool

	interrupted bool
	mutex       sync.Mutex
}

// Interrupt interrupts a container metrics observer
func (cmo *ContainerMetricsObserver) Interrupt() {
	cmo.mutex.Lock()
	defer cmo.mutex.Unlock()
	cmo.interrupted = true
}

// Interrupted checks whether container metrics observer is interrupted
func (cmo *ContainerMetricsObserver) Interrupted() bool {
	cmo.mutex.Lock()
	defer cmo.mutex.Unlock()
	return cmo.interrupted
}

// StartWatching starts watching container metrics
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
					CPU:           container.Usage.Cpu().MilliValue(),
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

// StopWatching stops watching container metrics
func (cmo *ContainerMetricsObserver) StopWatching() {
	if !cmo.Interrupted() {
		cmo.finished <- true
	}
}

// GetName returns name of container metrics observer
func (cmo *ContainerMetricsObserver) GetName() string {
	return "ContainerMetricsObserver"
}

// MakeChannel makes a new channel
func (cmo *ContainerMetricsObserver) MakeChannel() {
	cmo.finished = make(chan bool)
}
