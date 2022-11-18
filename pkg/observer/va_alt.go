package observer

import (
	"cert-csi/pkg/k8sclient"
	"cert-csi/pkg/store"
	"context"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

type VaListObserver struct {
	finished chan bool
}

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
			if len(attachedVAs) == len(deletedVAs) || runner.ShouldClean == false {
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
					Type:      store.PVC_ATTACH_STARTED,
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
					Type:      store.PVC_ATTACH_ENDED,
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
					Type:      store.PVC_UNATTACH_STARTED,
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
					Type:      store.PVC_UNATTACH_ENDED,
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

func (vao *VaListObserver) StopWatching() {
	vao.finished <- true
}

func (vao *VaListObserver) GetName() string {
	return "VolumeAttachmentObserver"
}

func (vao *VaListObserver) MakeChannel() {
	vao.finished = make(chan bool)
}
