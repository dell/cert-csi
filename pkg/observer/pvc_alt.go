package observer

import (
	"cert-csi/pkg/k8sclient"
	"cert-csi/pkg/store"
	"context"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"
	"time"
)

type PvcListObserver struct {
	finished chan bool
}

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

		pvcList, err := client.Interface.List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		for _, pvc := range pvcList.Items {
			// case watch.Added event
			currentState[pvc.Name] = true

			if !addedPVCs[pvc.Name] {
				entity := &store.Entity{
					Name:   pvc.Name,
					K8sUid: string(pvc.UID),
					TcID:   runner.TestCase.ID,
					Type:   store.PVC,
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
					Type:      store.PVC_ADDED,
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
					Type:      store.PVC_BOUND,
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
					Type:      store.PVC_DELETING_STARTED,
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
					Type:      store.PVC_DELETING_ENDED,
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

func (obs *PvcListObserver) StopWatching() {
	obs.finished <- true
}

func (*PvcListObserver) GetName() string {
	return "PersistentVolumeClaimObserver"
}

func (obs *PvcListObserver) MakeChannel() {
	obs.finished = make(chan bool)
}
