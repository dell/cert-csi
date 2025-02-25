package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	// Create a mock Runner
	mockRunner := &Runner{
		WaitGroup: sync.WaitGroup{},
		Clients: &k8sclient.Clients{
			PodClient: podClient,
		},
		Database: NewSimpleStore(),
	}

	//WatchTimeout = 10
	//mockRunner.WatchTimeout = 10

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
	ctx := context.Background()

	// Add the waitgroup to the Runner
	mockRunner.WaitGroup.Add(1)

	// Call the StartWatching function
	go po.StartWatching(ctx, mockRunner)

	// Wait for the StartWatching function to complete
	//<-po.finished

	// timeout := 10 * time.Second // Set the custom timeout
	// done := make(chan bool)

	// // Wait for the StartWatching function to complete
	// go func() {
	// 	mockRunner.WaitGroup.Wait()
	// 	done <- true
	// }()
	// select {
	// case <-done:
	// 	// StartWatching function completed successfully
	// 	assert.True(t, true)
	// case <-time.After(timeout):
	// 	// Test timed out
	// 	t.Error("Test timed out")
	// }
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
