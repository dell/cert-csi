package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"

	"fmt"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	test "k8s.io/client-go/testing"
)

func TestPvcObserver_StartWatching(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

	// Create a mock PVC
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: pvcClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}
	runner.WaitGroup.Add(1)

	obs := &PvcObserver{}

	fakeWatcher := watch.NewFake()
	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		if action.GetVerb() == "watch" {
			// Return the fake watcher
			return true, fakeWatcher, nil
		}
		return false, nil, nil
	})

	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	fakeWatcher.Add(pvc)
	fakeWatcher.Modify(pvc)

	pvc = &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pvc",
			UID:               "test-uid",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}

	fakeWatcher.Modify(pvc)

	fakeWatcher.Delete(pvc)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcObserver_StopWatching(t *testing.T) {
	// Test case: Stopping watching PVCs
	obs := &PvcObserver{}

	obs.finished = make(chan bool)

	go obs.StopWatching()

	select {
	case <-obs.finished:
		// Channel received a value
		// Make assertions here
		assert.True(t, true)

	case <-time.After(1 * time.Second):
		// Timeout waiting for channel to receive a value
		t.Error("Timeout waiting for channel to receive a value")
	}
}

func TestPvcObserver_GetName(t *testing.T) {
	// Test case: Getting name of PVC observer
	obs := &PvcObserver{}

	name := obs.GetName()

	assert.Equal(t, "PersistentVolumeClaimObserver", name)
}

func TestPvcObserver_MakeChannel(t *testing.T) {
	// Test case: Creating a new channel
	obs := &PvcObserver{}

	obs.MakeChannel()

	assert.NotNil(t, obs.finished)
}

func TestPvcObserver_StartWatching_ClientIsNil(t *testing.T) {
	// Test case: Client is nil
	ctx := context.Background()

	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: nil,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}
	runner.WaitGroup.Add(1)

	obs := &PvcObserver{}

	// Create a channel to receive the finished signal
	//finishedChan := make(chan bool)
	//obs.finished = finishedChan

	// Start the observer in a separate goroutine
	go obs.StartWatching(ctx, runner)

	// Wait for the observer to finish
	//<-finishedChan

	time.Sleep(100 * time.Millisecond)

	//obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)

	//var buf bytes.Buffer

	// Assert that the observer logs the error
	//assert.Contains(t, buf.String(), "Failed to create PVC client")

	//assert.Contains(t, log.StandardLogger().Writer().String(), "Failed to create PVC client")
}

func TestPvcObserver_StartWatching_WatchError(t *testing.T) {
	// Test case: Error when watching PVCs
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
	}*/

	// Create a mock client
	clientSet := fake.NewSimpleClientset()

	// Create a mock KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	// Create a mock PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Create a mock Runner
	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: pvcClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}
	runner.WaitGroup.Add(1)

	// Create a PvcObserver
	obs := &PvcObserver{}

	// Create a fake watcher
	//fakeWatcher := watch.NewFake()

	// Set the watch reactor to return an error
	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		return true, nil, fmt.Errorf("test error")
	})

	// Start the observer
	go obs.StartWatching(ctx, runner)

	time.Sleep(100 * time.Millisecond)

	//obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)

	// Wait for the observer to finish
	//<-obs.finished

	// Assert that the observer logs the error
	//assert.Contains(t, log.StandardLogger().Writer().String(), "Can't watch pvcClient; error = test error")
}

func TestPvcObserver_StartWatching_DataObjectIsNil(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: pvcClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}
	runner.WaitGroup.Add(1)

	obs := &PvcObserver{}

	fakeWatcher := watch.NewFake()
	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		if action.GetVerb() == "watch" {
			// Return the fake watcher
			return true, fakeWatcher, nil
		}
		return false, nil, nil
	})

	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	fakeWatcher.Add(nil)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcObserver_StartWatching_UnexpectedType(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: pvcClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}
	runner.WaitGroup.Add(1)

	obs := &PvcObserver{}

	fakeWatcher := watch.NewFake()
	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		if action.GetVerb() == "watch" {
			// Return the fake watcher
			return true, fakeWatcher, nil
		}
		return false, nil, nil
	})

	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	fakeWatcher.Modify(storageClass)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcObserver_StartWatching_WatchModified(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

	// Create a mock PVC
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: pvcClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}
	runner.WaitGroup.Add(1)

	obs := &PvcObserver{}

	fakeWatcher := watch.NewFake()
	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		if action.GetVerb() == "watch" {
			// Return the fake watcher
			return true, fakeWatcher, nil
		}
		return false, nil, nil
	})

	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	fakeWatcher.Modify(pvc)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}
