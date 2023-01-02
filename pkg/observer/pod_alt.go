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
	"cert-csi/pkg/k8sclient"
	kubepod "cert-csi/pkg/k8sclient/resources/pod"
	"cert-csi/pkg/store"
	"context"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// PodListObserver is used to manage Pod list observer
type PodListObserver struct {
	finished chan bool
}

// StartWatching watches pods
func (po *PodListObserver) StartWatching(ctx context.Context, runner *Runner) {
	defer runner.WaitGroup.Done()

	log.Debugf("%s started watching", po.GetName())
	client := runner.Clients.PodClient
	if client == nil {
		log.Errorf("Pod client can't be nil")
		return
	}
	timeout := WatchTimeout
	var events []*store.Event
	entities := make(map[string]*store.Entity)

	addedPods := make(map[string]bool)
	readyPods := make(map[string]bool)
	terminatingPods := make(map[string]bool)
	previousState := make(map[string]bool)

	pollErr := wait.PollImmediate(1*time.Second, time.Duration(timeout)*time.Second, func() (bool, error) {
		select {
		case <-po.finished:
			log.Debugf("%s finished watching", po.GetName())
			saveErr := runner.Database.SaveEvents(events)
			if saveErr != nil {
				log.Errorf("Error saving events; error=%v", saveErr)
				return false, saveErr
			}
			return true, nil
		default:
			break
		}

		currentState := make(map[string]bool)

		podList, err := client.Interface.List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		for i, pod := range podList.Items {
			// case watch.Added event
			currentState[pod.Name] = true

			if !addedPods[pod.Name] {
				entity := &store.Entity{
					Name:   pod.Name,
					K8sUID: string(pod.UID),
					TcID:   runner.TestCase.ID,
					Type:   store.Pod,
				}
				err = runner.Database.SaveEntities([]*store.Entity{entity})
				if err != nil {
					msg := err.Error()
					if !strings.Contains(msg, "UNIQUE constraint failed") {
						log.Errorf("Can't save entity; error=%v", err)
					}
				}

				entities[pod.Name] = entity
				events = append(events, &store.Event{
					Name:      "event-pod-added-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entity.ID,
					Type:      store.PodAdded,
					Timestamp: time.Now(),
				})
				addedPods[pod.Name] = true
				continue
			}

			// case watch.Modified event
			if !readyPods[pod.Name] && kubepod.IsPodReady(&podList.Items[i]) {
				// Pod is READY, adding event
				readyPods[pod.Name] = true
				events = append(events, &store.Event{
					Name:      "event-pod-modified-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entities[pod.Name].ID,
					Type:      store.PodReady,
					Timestamp: time.Now(),
				})
				continue
			}

			if pod.DeletionTimestamp != nil && !terminatingPods[pod.Name] {
				// Pod started deletion
				terminatingPods[pod.Name] = true
				events = append(events, &store.Event{
					Name:      "event-pod-modified-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entities[pod.Name].ID,
					Type:      store.PodTerminating,
					Timestamp: time.Now(),
				})
				continue
			}
		}

		for name := range previousState {
			if !currentState[name] {
				// case watch.Deleted event
				events = append(events, &store.Event{
					Name:      "event-pod-deleted-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entities[name].ID,
					Type:      store.PodDeleted,
					Timestamp: time.Now(),
				})
				delete(previousState, name)
			}
		}
		// Copy state
		for name, value := range currentState {
			previousState[name] = value
		}

		return false, nil
	})

	if pollErr != nil {
		log.Errorf("Can't poll podClient; error = %v", pollErr)
		return
	}
}

// StopWatching stops watching pods
func (po *PodListObserver) StopWatching() {
	po.finished <- true
}

// GetName returns name of Pod list observer
func (po *PodListObserver) GetName() string {
	return "Pod Observer"
}

// MakeChannel creates a new channel
func (po *PodListObserver) MakeChannel() {
	po.finished = make(chan bool)
}
