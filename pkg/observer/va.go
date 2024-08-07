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
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// VaObserver is used to manage volume attachment observer
type VaObserver struct {
	finished chan bool
}

// StartWatching starts watching a volume attachment and related events
func (vao *VaObserver) StartWatching(_ context.Context, runner *Runner) {
	defer runner.WaitGroup.Done()

	log.Debugf("%s started watching", vao.GetName())
	client := runner.Clients.VaClient
	if client == nil {
		log.Errorf("VolumeAttachment client can't be nil")
		return
	}

	timeout := WatchTimeout
	w, watchErr := client.Interface.Watch(context.Background(), metav1.ListOptions{
		TimeoutSeconds: &timeout,
	})
	if watchErr != nil {
		log.Errorf("Can't watch VolumeAttachment client; error = %v", watchErr)
		return
	}
	defer w.Stop()

	var events []*store.Event
	attachedVAs := make(map[string]bool)
	deletingVAs := make(map[string]bool)
	deletedVAs := make(map[string]bool)

	var shouldExit bool

	for {
		select {
		case <-vao.finished:
			// We can't finish if we haven't received all deletion events
			if len(attachedVAs) == len(deletedVAs) || !runner.ShouldClean {
				err := runner.Database.SaveEvents(events)
				if err != nil {
					log.Errorf("Error saving events; error=%v", err)
					return
				}
				log.Debugf("%s finished watching", vao.GetName())
				return
			}
			// Wait until all deleted
			log.Info("Waiting for volumeattachments to be deleted")
			shouldExit = true

		case data := <-w.ResultChan():
			if data.Object == nil {
				// ignore nil
				break
			}

			va, ok := data.Object.(*storagev1.VolumeAttachment)
			if !ok {
				log.Errorf("VaObserver: unexpected type in %v", data)
				break
			}
			loaded, ok := runner.PvcShare.Load(*va.Spec.Source.PersistentVolumeName)
			if !ok {
				break
			}
			entity := loaded.(*store.Entity)

			switch data.Type {
			case watch.Added:
				events = append(events, &store.Event{
					Name:      "event-va-added-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entity.ID,
					Type:      store.PvcAttachStarted,
					Timestamp: time.Now(),
				})
				break
			case watch.Modified:
				if va.Status.Attached && !attachedVAs[va.Name] {
					attachedVAs[va.Name] = true
					events = append(events, &store.Event{
						Name:      "event-va-modified-" + k8sclient.RandomSuffix(),
						TcID:      runner.TestCase.ID,
						EntityID:  entity.ID,
						Type:      store.PvcAttachEnded,
						Timestamp: time.Now(),
					})
					break
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
					break
				}
				break
			case watch.Deleted:
				deletedVAs[va.Name] = true
				events = append(events, &store.Event{
					Name:      "event-va-deleted-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entity.ID,
					Type:      store.PvcUnattachEnded,
					Timestamp: time.Now(),
				})

				if shouldExit && len(attachedVAs) == len(deletedVAs) {
					err := runner.Database.SaveEvents(events)
					if err != nil {
						log.Errorf("Error saving events; error=%v", err)
						return
					}
					log.Debugf("%s finished watching", vao.GetName())
					return
				}
				break
			default:
				log.Errorf("Unexpected event %v", data)
				break
			}
		}
	}
}

// StopWatching stops watching a volume attachment
func (vao *VaObserver) StopWatching() {
	vao.finished <- true
}

// GetName returns name of VA observer
func (vao *VaObserver) GetName() string {
	return "VolumeAttachmentObserver"
}

// MakeChannel creates a new channel
func (vao *VaObserver) MakeChannel() {
	vao.finished = make(chan bool)
}
