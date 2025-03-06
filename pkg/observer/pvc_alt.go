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
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// PvcListObserver is used to manage PVC list observer
type PvcListObserver struct {
	finished chan bool
}

var getPvcsList = func(ctx context.Context, client *pvc.Client, opts metav1.ListOptions) (*v1.PersistentVolumeClaimList, error) {
	return client.Interface.List(ctx, metav1.ListOptions{})
}

// StartWatching starts watching a list of PVCs
func (obs *PvcListObserver) StartWatching(ctx context.Context, runner *Runner) {
	defer runner.WaitGroup.Done()

	log.Debugf("%s started watching", obs.GetName())
	client := runner.Clients.PVCClient
	if client == nil {
		log.Errorf("PVCClient can't be nil")
		return
	}
	timeout := WatchTimeout

	var events []*store.Event
	entities := make(map[string]*store.Entity)

	addedPVCs := make(map[string]bool)
	boundPVCs := make(map[string]bool)
	deletingPVCs := make(map[string]bool)
	previousState := make(map[string]bool)

	pollErr := wait.PollImmediate(1*time.Second, time.Duration(timeout)*time.Second, func() (bool, error) {
		select {
		case <-obs.finished:
			log.Debugf("%s finished watching", obs.GetName())
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

		pvcList, err := getPvcsList(ctx, client, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		for _, pvc := range pvcList.Items {
			// case watch.Added event

			if !getBoolValueFromMapWithKey(addedPVCs, pvc.Name) {
				entity := &store.Entity{
					Name:   pvc.Name,
					K8sUID: string(pvc.UID),
					TcID:   runner.TestCase.ID,
					Type:   store.Pvc,
				}
				err = runner.Database.SaveEntities([]*store.Entity{entity})
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
				addedPVCs[pvc.Name] = true
				continue
			}

			// case watch.Modified event
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
				continue
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
				continue
			}
		}

		for name := range previousState {
			if !currentState[name] {
				// case watch.Deleted event
				events = append(events, &store.Event{
					Name:      "event-pvc-deleted-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entities[name].ID,
					Type:      store.PvcDeletingEnded,
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

// StopWatching stops watching a list of PVCs
func (obs *PvcListObserver) StopWatching() {
	obs.finished <- true
}

// GetName returns PVC list observer name
func (*PvcListObserver) GetName() string {
	return "PersistentVolumeClaimObserver"
}

// MakeChannel creates a new channel
func (obs *PvcListObserver) MakeChannel() {
	obs.finished = make(chan bool)
}
