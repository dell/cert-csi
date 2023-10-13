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
	"strings"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// PvcObserver is used to manage PVC Observer
type PvcObserver struct {
	finished chan bool
}

// StartWatching starts watching a PVC
func (obs *PvcObserver) StartWatching(ctx context.Context, runner *Runner) {
	defer runner.WaitGroup.Done()

	log.Debugf("%s started watching", obs.GetName())
	client := runner.Clients.PVCClient
	if client == nil {
		log.Errorf("PVCClient can't be nil")
		return
	}
	timeout := WatchTimeout
	w, watchErr := client.Interface.Watch(context.Background(), metav1.ListOptions{
		TimeoutSeconds: &timeout,
	})
	if watchErr != nil {
		log.Errorf("Can't watch pvcClient; error = %v", watchErr)
		return
	}
	defer w.Stop()

	var events []*store.Event
	entities := make(map[string]*store.Entity)

	boundPVCs := make(map[string]bool)
	deletingPVCs := make(map[string]bool)

	for {
		select {
		case <-obs.finished:
			err := runner.Database.SaveEvents(events)
			if err != nil {
				log.Errorf("Error saving events; error=%v", err)
				return
			}
			log.Debugf("%s finished watching", obs.GetName())
			return
		case data := <-w.ResultChan():
			if data.Object == nil {
				// ignore nil
				break
			}

			pvc, ok := data.Object.(*v1.PersistentVolumeClaim)
			if !ok {
				log.Errorf("PvcObserver: unexpected type in %v", data)
				break
			}

			switch data.Type {
			case watch.Added:
				entity := &store.Entity{
					Name:   pvc.Name,
					K8sUID: string(pvc.UID),
					TcID:   runner.TestCase.ID,
					Type:   store.Pvc,
				}
				err := runner.Database.SaveEntities([]*store.Entity{entity})
				if err != nil {
					msg := err.Error()
					if !strings.Contains(msg, "UNIQUE constraint failed") {
						log.Errorf("Can't save entity; error=%v", err)
					}
				}

				entities[pvc.Name] = entity
				events = append(events, &store.Event{
					Name:      "event-pvc-added-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entity.ID,
					Type:      store.PvcAdded,
					Timestamp: time.Now(),
				})
				break
			case watch.Modified:
				if pvc.Status.Phase == v1.ClaimBound && !boundPVCs[pvc.Name] {
					// PVC BOUNDED, adding event
					boundPVCs[pvc.Name] = true
					events = append(events, &store.Event{
						Name:      "event-pvc-modified-" + k8sclient.RandomSuffix(),
						TcID:      runner.TestCase.ID,
						EntityID:  entities[pvc.Name].ID,
						Type:      store.PvcBound,
						Timestamp: time.Now(),
					})

					// Share pvc with volumeattachment observer
					runner.PvcShare.Store(pvc.Spec.VolumeName, entities[pvc.Name])
					break
				}
				if pvc.DeletionTimestamp != nil && !deletingPVCs[pvc.Name] {
					// PVC started deletion
					deletingPVCs[pvc.Name] = true
					events = append(events, &store.Event{
						Name:      "event-pvc-modified-" + k8sclient.RandomSuffix(),
						TcID:      runner.TestCase.ID,
						EntityID:  entities[pvc.Name].ID,
						Type:      store.PvcDeletingStarted,
						Timestamp: time.Now(),
					})
					break
				}
				break
			case watch.Deleted:
				events = append(events, &store.Event{
					Name:      "event-pvc-deleted-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entities[pvc.Name].ID,
					Type:      store.PvcDeletingEnded,
					Timestamp: time.Now(),
				})
				break
			default:
				log.Errorf("Unexpected event %v", data)
				break
			}
		}
	}
}

// StopWatching stops watching a PVC
func (obs *PvcObserver) StopWatching() {
	obs.finished <- true
}

// GetName returns name of PVC observer
func (*PvcObserver) GetName() string {
	return "PersistentVolumeClaimObserver"
}

// MakeChannel creates a new channel
func (obs *PvcObserver) MakeChannel() {
	obs.finished = make(chan bool)
}
