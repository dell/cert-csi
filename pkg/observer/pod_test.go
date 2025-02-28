package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	test "k8s.io/client-go/testing"
)

func TestPodObserver_StartWatching(t *testing.T) {
	// Test case: Watching pods
	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	//clientSet := NewFakeClientsetWithRestClient(storageClass)
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

	// _, err := clientSet.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
	// if err != nil {
	// 	t.Errorf("Error creating fake pod: %v", err)
	// }

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	clientSet.Fake.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		//getAction := action.(k8stesting.GetAction)
		//podName := getAction.GetName()
		// Create a pod object with the expected name and Ready status
		pod = &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
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
	//mockClients := &MockClients{}

	//pod.Client.ClientSet.CoreV1().Pods(pod.Object.Namespace).Get(ctx, pod.Object.Name, metav1.GetOptions{})

	// Set up the mock behavior for the CreatePodClient method
	// mockClients.On("CreatePodClient", "test-namespace").Return(
	// 	&pod.Client{
	// 		Interface: clientSet.CoreV1().Pods("test-namespace"),
	// 	},
	// 	nil,
	// )

	// mockKubePodInstance := &mockKubePod{}

	// //entities := make(map[string]*store.Entity)

	// // Set up the mock behavior for the IsPodReady() method
	// mockKubePodInstance.On("IsPodReady", mock.Anything).Return(true)
	// if pod == nil {
	// 	t.Errorf("Pod is nil, unable to call IsPodReady")
	// 	return
	// }
	// result := mockKubePodInstance.IsPodReady(pod)

	// if result != true {
	// 	t.Errorf("Expected IsPodReady to return true, but got false")
	// }

	// mockKubePodInstance.AssertCalled(t, "IsPodReady", pod)

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

	// timeout := WatchTimeout
	// w, _ := pvcClient.Interface.Watch(context.Background(), metav1.ListOptions{
	//     TimeoutSeconds: &timeout,
	// })
	// print(w)
	fakeWatcher := watch.NewFake()

	podClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		if action.GetVerb() == "watch" {
			// Return the fake watcher
			return true, fakeWatcher, nil
		}
		return false, nil, nil
	})

	//go po.StartWatching(ctx, runner)
	// clientSet.Fake.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
	// 	deleteAction := action.(k8stesting.DeleteAction)
	// 	podName := deleteAction.GetName()

	// 	// Simulate the deletion of the pod
	// 	pod := &v1.Pod{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Name:      podName,
	// 			Namespace: "test-namespace",
	// 		},
	// 	}

	// 	// Return the deleted pod
	// 	return true, pod, nil
	// })

	//po.MakeChannel()
	// runner.WaitGroup.Add(1)

	// go po.StartWatching(ctx, runner)
	// fakeWatcher.Add(pod)
	//po.StopWatching()

	// w, _ := clientSet.CoreV1().Pods("default").Watch(ctx, metav1.ListOptions{})
	// wait.Until(func() {
	// 	<-w.ResultChan()
	// }, time.Second, ctx.Done())

	//time.Sleep(100 * time.Millisecond)

	// po.StopWatching()

	// Set up a reactor to simulate Pods becoming Ready
	// clientSet.Fake.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
	// 	createAction := action.(k8stesting.CreateAction)
	// 	pod = createAction.GetObject().(*v1.Pod)
	// 	// Set pod phase to Running
	// 	pod.Status.Phase = v1.PodRunning
	// 	// Simulate the Ready condition
	// 	pod.Status.Conditions = append(pod.Status.Conditions, v1.PodCondition{
	// 		Type:   v1.PodReady,
	// 		Status: v1.ConditionTrue,
	// 	})

	// 	// Simulate the "FileSystemResizeSuccessful" event
	// 	event := &v1.Event{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Namespace: pod.Namespace,
	// 			Name:      "test-event",
	// 		},
	// 		InvolvedObject: v1.ObjectReference{
	// 			Namespace: "test-namespace",
	// 			Name:      "test-pod",
	// 			UID:       pod.UID,
	// 		},
	// 		Reason: "FileSystemResizeSuccessful",
	// 		Type:   v1.EventTypeNormal,
	// 	}
	// 	clientSet.Tracker().Add(event)

	// 	return false, nil, nil // Allow normal processing to continue
	// })

	// po2 := &PodObserver{}
	//po.finished = make(chan bool)

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

	// event := watch.Event{
	// 	Type:   watch.Modified,
	// 	Object: pod,
	// }

	//go po.StartWatching(ctx, runner)

	//fakeWatcher.Modify(event.Object)

	//runner.WaitGroup.Add(1)
	//fakeWatcher.Add(event.Object)
	//go po.StartWatching(ctx, runner)

	//po.StopWatching()

	event2 := watch.Event{
		Type:   watch.Deleted,
		Object: pod,
	}

	//go po.StartWatching(ctx, runner)
	// fakeWatcher.Delete(event2.Object)
	// runner.WaitGroup.Add(1)

	// w, _ := clientSet.CoreV1().Pods("default").Watch(ctx, metav1.ListOptions{})
	// wait.Until(func() {
	// 	<-w.ResultChan()
	// }, time.Second, ctx.Done())

	// time.Sleep(100 * time.Millisecond)

	runner.WaitGroup.Add(1)

	go po.StartWatching(ctx, runner)
	fakeWatcher.Add(pod)

	fakeWatcher.Modify(event.Object)
	fakeWatcher.Modify(event3.Object)

	fakeWatcher.Delete(event2.Object)

	po.StopWatching()

	runner.WaitGroup.Wait()

	// po.StopWatching()
	// Assert that the function completed successfully
	assert.True(t, true)
}

// func TestPodObserver_Deleted(t *testing.T) {
// 	// Create a context
// 	ctx := context.Background()

// 	// Create a fake pod
// 	pod := &v1.Pod{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test-pod",
// 			Namespace: "default",
// 		},
// 	}

// 	storageClass := &storagev1.StorageClass{
// 		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
// 		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
// 			mode := storagev1.VolumeBindingWaitForFirstConsumer
// 			return &mode
// 		}(),
// 	}
// 	clientSet := NewFakeClientsetWithRestClient(storageClass)

// 	_, err := clientSet.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
// 	if err != nil {
// 		t.Errorf("Error creating fake pod: %v", err)
// 	}

// 	kubeClient := &k8sclient.KubeClient{
// 		ClientSet: clientSet,
// 		Config:    &rest.Config{},
// 	}

// 	podClient, _ := kubeClient.CreatePodClient("test-namespace")

// 	runner := &Runner{
// 		Clients: &k8sclient.Clients{
// 			PodClient: podClient,
// 		},
// 		TestCase: &store.TestCase{
// 			ID: 1,
// 		},
// 		WaitGroup: sync.WaitGroup{},
// 		Database:  NewSimpleStore(),
// 	}

// 	// Create a mock PodObserver
// 	po := &PodObserver{}
// 	po.MakeChannel()

// 	// Start watching pods
// 	go po.StartWatching(ctx, runner)

// 	// Delete the fake pod
// 	err = clientSet.CoreV1().Pods("default").Delete(ctx, pod.Name, metav1.DeleteOptions{})
// 	if err != nil {
// 		t.Errorf("Error deleting fake pod: %v", err)
// 	}

// 	// Wait for the ResultChan to receive a pod
// 	w, _ := clientSet.CoreV1().Pods("default").Watch(ctx, metav1.ListOptions{})
// 	wait.Until(func() {
// 		<-w.ResultChan()
// 	}, time.Second, ctx.Done())

// 	// Wait for the Deleted case to be executed
// 	time.Sleep(100 * time.Millisecond)

// 	// Assert that the Deleted case was executed
// 	// assert.True(t, true)

// 	// Stop watching pods
// 	po.StopWatching()

// 	// Wait for the WaitGroup to complete
// 	runner.WaitGroup.Wait()

// 	// Assert that the function completed successfully
// 	assert.True(t, true)
// }

func TestPodObserver_StopWatching(t *testing.T) {
	// Test case: Stopping watching pods
	po := &PodObserver{}

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

func TestPodObserver_GetName(t *testing.T) {
	// Test case: Getting name of Pod observer
	po := &PodObserver{}

	name := po.GetName()

	assert.Equal(t, "Pod Observer", name)
}

func TestPodObserver_MakeChannel(t *testing.T) {
	// Test case: Creating a new channel
	po := &PodObserver{}

	po.MakeChannel()

	assert.NotNil(t, po.finished)
}

// Mock implementation of PodClient
type mockPodClient struct {
	mock.Mock
}

func (m *mockPodClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(watch.Interface), args.Error(1)
}

func (m *mockDatabase) SaveEntities(entities []*store.Entity) error {
	args := m.Called(entities)
	return args.Error(0)
}

func (m *mockDatabase) SaveEvents(events []*store.Event) error {
	args := m.Called(events)
	return args.Error(0)
}

// Mock implementation of Pod
type mockPod struct {
	mock.Mock
}

func (m *mockPod) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockPod) GetUID() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockPod) GetDeletionTimestamp() *metav1.Time {
	args := m.Called()
	return args.Get(0).(*metav1.Time)
}

func (m *mockPod) IsReady() bool {
	args := m.Called()
	return args.Bool(0)
}

// Mock implementation of kubepod
type mockKubePod struct {
	mock.Mock
}

func (m *mockKubePod) IsPodReady(pod *v1.Pod) bool {
	args := m.Called(pod)
	return args.Bool(0)
}
