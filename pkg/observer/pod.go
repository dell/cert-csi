package observer

import (
	"cert-csi/pkg/k8sclient"
	kubepod "cert-csi/pkg/k8sclient/resources/pod"
	"cert-csi/pkg/store"
	"context"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"strings"
	"time"
)

type PodObserver struct {
	finished chan bool
}

func (po *PodObserver) StartWatching(ctx context.Context, runner *Runner) {
	defer runner.WaitGroup.Done()

	log.Debugf("%s started watching", po.GetName())
	client := runner.Clients.PodClient
	if client == nil {
		log.Errorf("Pod client can't be nil")
		return
	}
	timeout := WatchTimeout
	w, watchErr := client.Interface.Watch(context.Background(), metav1.ListOptions{
		TimeoutSeconds: &timeout,
	})
	if watchErr != nil {
		log.Errorf("Can't watch podClient; error = %v", watchErr)
		return
	}
	defer w.Stop()

	var events []*store.Event
	entities := make(map[string]*store.Entity)

	readyPods := make(map[string]bool)
	terminatingPods := make(map[string]bool)

	for {
		select {
		case <-po.finished:
			err := runner.Database.SaveEvents(events)
			if err != nil {
				log.Errorf("Error saving events; error=%v", err)
				return
			}
			log.Debugf("%s finished watching", po.GetName())
			return
		case data := <-w.ResultChan():
			if data.Object == nil {
				// ignore nil
				break
			}

			pod, ok := data.Object.(*v1.Pod)
			if !ok {
				log.Errorf("PodObserver: unexpected type in %v", data)
				break
			}

			switch data.Type {
			case watch.Added:
				entity := &store.Entity{
					Name:   pod.Name,
					K8sUid: string(pod.UID),
					TcID:   runner.TestCase.ID,
					Type:   store.POD,
				}
				err := runner.Database.SaveEntities([]*store.Entity{entity})
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
					Type:      store.POD_ADDED,
					Timestamp: time.Now(),
				})
				break
			case watch.Modified:
				if !readyPods[pod.Name] && kubepod.IsPodReady(pod) {
					// Pod is READY, adding event
					readyPods[pod.Name] = true
					events = append(events, &store.Event{
						Name:      "event-pod-modified-" + k8sclient.RandomSuffix(),
						TcID:      runner.TestCase.ID,
						EntityID:  entities[pod.Name].ID,
						Type:      store.POD_READY,
						Timestamp: time.Now(),
					})
					break
				}
				if pod.DeletionTimestamp != nil && !terminatingPods[pod.Name] {
					// Pod started deletion
					terminatingPods[pod.Name] = true
					events = append(events, &store.Event{
						Name:      "event-pod-modified-" + k8sclient.RandomSuffix(),
						TcID:      runner.TestCase.ID,
						EntityID:  entities[pod.Name].ID,
						Type:      store.POD_TERMINATING,
						Timestamp: time.Now(),
					})
					break
				}
				break
			case watch.Deleted:
				events = append(events, &store.Event{
					Name:      "event-pod-deleted-" + k8sclient.RandomSuffix(),
					TcID:      runner.TestCase.ID,
					EntityID:  entities[pod.Name].ID,
					Type:      store.POD_DELETED,
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

func (po *PodObserver) StopWatching() {
	po.finished <- true
}

func (po *PodObserver) GetName() string {
	return "Pod Observer"
}

func (po *PodObserver) MakeChannel() {
	po.finished = make(chan bool)
}
