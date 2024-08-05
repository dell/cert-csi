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
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// VaListObserver is used to manage volume attachment observer
type VaListObserver struct {
	finished chan bool
}

// StartWatching starts watching a volume attachment and related events
func (vao *VaListObserver) StartWatching(ctx context.Context, runner *Runner) {
	defer runner.WaitGroup.Done()

	log.Debugf("%s started watching", vao.GetName())
	client := runner.Clients.VaClient
	if client == nil {
		log.Errorf("VolumeAttachment client can't be nil")
		return
	}

	timeout := WatchTimeout

	var events []*store.Event
	addedVAs := make(map[string]bool)
	attachedVAs := make(map[string]bool)
	deletingVAs := make(map[string]bool)
	deletedVAs := make(map[string]bool)
	previousState := make(map[string]bool)
	var shouldExit bool

	pollErr := wait.PollImmediate(500*time.Millisecond, time.Duration(timeout)*time.Second, func() (bool, error) {
		select {
		case <-vao.finished:
			if len(attachedVAs) == len(deletedVAs) || !runner.ShouldClean {
				log.Debugf("%s finished watching", vao.GetName())
				saveErr := runner.Database.SaveEvents(events)
				if saveErr != nil {
					log.Errorf("Error saving events; error=%v", saveErr)
					return false, saveErr
				}
				return true, nil
			}
			log.Info("Waiting for volumeattachments to be deleted")
			shouldExit = true
			break
		default:
			break
		}

		currentState := make(map[string]bool)

		vaList, err := client.Interface.List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		for _, va := range vaList.Items {
			loaded, ok := runner.PvcShare.Load(*va.Spec.Source.PersistentVolumeName)
			if !ok {
				continue
			}

			entity := loaded.(*store.Entity)

			// case watch.Added event
			currentState[*va.Spec.Source.PersistentVolumeName] = true

			if !addedVAs[va.Name] {
				events = append(events, &store.Event{
					Name:      "event-va-added-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entity.ID,
					Type:      store.PvcAttachStarted,
					Timestamp: time.Now(),
				})
				addedVAs[va.Name] = true
				continue
			}

			// case watch.Modified event
			if va.Status.Attached && !attachedVAs[va.Name] {
				attachedVAs[va.Name] = true
				events = append(events, &store.Event{
					Name:      "event-va-modified-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entity.ID,
					Type:      store.PvcAttachEnded,
					Timestamp: time.Now(),
				})
				continue
			}

			if va.DeletionTimestamp != nil && !deletingVAs[va.Name] {
				deletingVAs[va.Name] = true
				events = append(events, &store.Event{
					Name:      "event-va-modified-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entity.ID,
					Type:      store.PvcUnattachStarted,
					Timestamp: time.Now(),
				})
				continue
			}
		}

		for name := range previousState {
			if !currentState[name] {
				loaded, ok := runner.PvcShare.Load(name)
				if !ok {
					continue
				}

				entity := loaded.(*store.Entity)
				// case watch.Deleted event
				events = append(events, &store.Event{
					Name:      "event-va-deleted-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entity.ID,
					Type:      store.PvcUnattachEnded,
					Timestamp: time.Now(),
				})
				deletedVAs[name] = true
				delete(previousState, name)
			}
		}
		// Copy state
		for name, value := range currentState {
			previousState[name] = value
		}

		if shouldExit && len(attachedVAs) == len(deletedVAs) {
			err = runner.Database.SaveEvents(events)
			if err != nil {
				log.Errorf("Error saving events; error=%v", err)
				return false, err
			}
			log.Debugf("%s finished watching", vao.GetName())
			return true, nil
		}

		return false, nil
	})

	if pollErr != nil {
		log.Errorf("Can't poll podClient; error = %v", pollErr)
		return
	}
}

// StopWatching stops watching a volume attachment
func (vao *VaListObserver) StopWatching() {
	vao.finished <- true
}

// GetName returns name of VA observer
func (vao *VaListObserver) GetName() string {
	return "VolumeAttachmentObserver"
}

// MakeChannel creates a new channel
func (vao *VaListObserver) MakeChannel() {
	vao.finished = make(chan bool)
}
