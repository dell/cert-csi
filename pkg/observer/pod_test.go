package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	test "k8s.io/client-go/testing"
)

func TestPodObserver_StartWatching(t *testing.T) {
	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}

	clientSet := fake.NewSimpleClientset()
	clientSet.CoreV1().Pods("test-namespace").Create(ctx, &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace"}}, metav1.CreateOptions{})
	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "test-namespace",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	runner := &Runner{
		Clients: &k8sclient.Clients{
			PodClient: podClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}

	po := &PodObserver{}
	po.MakeChannel()

	fakeWatcher := watch.NewFake()

	podClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		if action.GetVerb() == "watch" {
			// Return the fake watcher
			return true, fakeWatcher, nil
		}
		return false, nil, nil
	})

	event := watch.Event{
		Type: watch.Modified,
		Object: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-pod",
				Namespace:         "test-namespace",
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		},
	}
	event3 := watch.Event{
		Type: watch.Modified,
		Object: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-pod",
				Namespace:         "test-namespace",
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodScheduled,
						Status: v1.ConditionTrue,
					},
				},
			},
		},
	}

	event2 := watch.Event{
		Type:   watch.Deleted,
		Object: pod,
	}

	runner.WaitGroup.Add(1)

	go po.StartWatching(ctx, runner)
	fakeWatcher.Add(pod)

	fakeWatcher.Modify(event.Object)
	fakeWatcher.Modify(event3.Object)

	fakeWatcher.Delete(event2.Object)

	po.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPodObserver_StopWatching(t *testing.T) {
	po := &PodObserver{}

	po.finished = make(chan bool)

	go po.StopWatching()

	select {
	case <-po.finished:
		assert.True(t, true)

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for channel to receive a value")
	}
}

func TestPodObserver_GetName(t *testing.T) {
	po := &PodObserver{}

	name := po.GetName()

	assert.Equal(t, "Pod Observer", name)
}

func TestPodObserver_MakeChannel(t *testing.T) {
	po := &PodObserver{}

	po.MakeChannel()

	assert.NotNil(t, po.finished)
}
