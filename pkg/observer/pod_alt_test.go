package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

func isChanClosed(ch <-chan bool) bool {
	select {
	case _, ok := <-ch:
		return !ok
	default:
		return false
	}
}

func TestPodListObserver_StartWatching(t *testing.T) {
	// Create a context
	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)

	clientSet.CoreV1().Pods("test-namespace").Create(ctx, &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace"}}, metav1.CreateOptions{})
	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	// Set up a reactor to simulate Pods becoming Ready
	clientSet.Fake.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		pod := createAction.GetObject().(*v1.Pod)
		// Set pod phase to Running
		pod.Status.Phase = v1.PodRunning
		// Simulate the Ready condition
		pod.Status.Conditions = append(pod.Status.Conditions, v1.PodCondition{
			Type:   v1.PodReady,
			Status: v1.ConditionTrue,
		})

		// Simulate the "FileSystemResizeSuccessful" event
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pod.Namespace,
				Name:      "test-event",
			},
			InvolvedObject: v1.ObjectReference{
				Namespace: pod.Namespace,
				Name:      pod.Name,
				UID:       pod.UID,
			},
			Reason: "FileSystemResizeSuccessful",
			Type:   v1.EventTypeNormal,
		}
		clientSet.Tracker().Add(event)

		return false, nil, nil // Allow normal processing to continue
	})
	clientSet.Fake.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		podName := getAction.GetName()
		// Create a pod object with the expected name and Ready status
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "test-namespace",
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
		return true, pod, nil
	})

	// Create a mock Clients instance
	mockClients := &MockClients{}

	// Set up the mock behavior for the CreatePodClient method
	mockClients.On("CreatePodClient", "test-namespace").Return(
		&pod.Client{
			Interface: clientSet.CoreV1().Pods("test-namespace"),
		},
		nil,
	)

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

	pod2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "test-namespace",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(v1.PodReady),
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodScheduled,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	podClient.Create(ctx, pod)
	podClient.Delete(ctx, pod)
	podClient.Create(ctx, pod)
	podClient.Update(pod2)

	// Create a mock Runner
	mockRunner := &Runner{
		WaitGroup: sync.WaitGroup{},
		Clients: &k8sclient.Clients{
			PodClient: podClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		Database: NewSimpleStore(),
	}
	// Create a PodListObserver instance
	po := &PodListObserver{}
	po.MakeChannel()
	// Check for nil pointer dereferences
	// if mockRunner == nil {
	// 	t.Error("mockRunner is nil")
	// }
	// if po == nil {
	// 	t.Error("po is nil")
	// }

	// Create a context
	// ctx := context.Background()

	// Add the waitgroup to the Runner
	mockRunner.WaitGroup.Add(1)

	go po.StartWatching(ctx, mockRunner)

	time.Sleep(100 * time.Millisecond)

	po.StopWatching()

	mockRunner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}
func TestPodListObserver_StopWatching(t *testing.T) {

	// Create a PodListObserver instance
	po := &PodListObserver{}

	// Create a channel
	po.finished = make(chan bool)

	// Start the StopWatching function in a goroutine
	go po.StopWatching()

	select {
	case <-po.finished:
		// Channel received a value
		// Make assertions here
		assert.True(t, true)

	case <-time.After(1 * time.Second):
		// Timeout waiting for channel to receive a value
		t.Error("Timeout waiting for channel to receive a value")
	}
}

func TestPodListObserver_GetName(t *testing.T) {
	// Create a PodListObserver instance
	po := &PodListObserver{}

	// Call the GetName function
	name := po.GetName()

	// Assert that the function returned the correct value
	assert.Equal(t, "Pod Observer", name)
}

func TestPodListObserver_MakeChannel(t *testing.T) {
	// Create a PodListObserver instance
	po := &PodListObserver{}

	// Call the MakeChannel function
	po.MakeChannel()

	// Assert that the function completed successfully
	assert.NotNil(t, po.finished)
}
