package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestPvcObserver_StartWatching(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

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
	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

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
