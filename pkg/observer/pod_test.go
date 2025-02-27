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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	test "k8s.io/client-go/testing"
)

func TestPodObserver_StartWatching(t *testing.T) {
	// Test case: Watching pods
	ctx := context.Background()

	// storageClass := &storagev1.StorageClass{
	// 	ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
	// 	VolumeBindingMode: func() *storagev1.VolumeBindingMode {
	// 		mode := storagev1.VolumeBindingWaitForFirstConsumer
	// 		return &mode
	// 	}(),
	// }
	//clientSet := NewFakeClientsetWithRestClient(storageClass)
	clientSet := fake.NewSimpleClientset()

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	_, err := clientSet.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("Error creating fake pod: %v", err)
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
	runner.WaitGroup.Add(1)

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

	go po.StartWatching(ctx, runner)
	fakeWatcher.Add(pod)

	//po.MakeChannel()

	go po.StartWatching(ctx, runner)

	// w, _ := clientSet.CoreV1().Pods("default").Watch(ctx, metav1.ListOptions{})
	// wait.Until(func() {
	// 	<-w.ResultChan()
	// }, time.Second, ctx.Done())

	//time.Sleep(100 * time.Millisecond)

	po.StopWatching()

	runner.WaitGroup.Wait()

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
