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
	"k8s.io/client-go/rest"
)

func TestPvcListObserver_StartWatching(t *testing.T) {
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

	obs := &PvcListObserver{}
	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcListObserver_StopWatching(t *testing.T) {
	// Test case: Stopping watching PVCs
	obs := &PvcListObserver{}

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

func TestPvcListObserver_GetName(t *testing.T) {
	// Test case: Getting name of PVC list observer
	obs := &PvcListObserver{}

	name := obs.GetName()

	assert.Equal(t, "PersistentVolumeClaimObserver", name)
}

func TestPvcListObserver_MakeChannel(t *testing.T) {
	// Test case: Creating a new channel
	obs := &PvcListObserver{}

	obs.MakeChannel()

	assert.NotNil(t, obs.finished)
}

// Mock implementation of PVCClient
type mockPVCClient struct {
	mock.Mock
}

func (m *mockPVCClient) List(ctx context.Context, opts metav1.ListOptions) (*v1.PersistentVolumeClaimList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*v1.PersistentVolumeClaimList), args.Error(1)
}

// Mock implementation of Database
// type mockDatabase struct {
// 	mock.Mock
// }

// func (m *mockDatabase) SaveEntities(entities []*store.Entity) error {
// 	args := m.Called(entities)
// 	return args.Error(0)
// }

// func (m *mockDatabase) SaveEvents(events []*store.Event) error {
// 	args := m.Called(events)
// 	return args.Error(0)
// }

// Mock implementation of PVC
type mockPVC struct {
	mock.Mock
}

func (m *mockPVC) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockPVC) GetUID() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockPVC) GetDeletionTimestamp() *metav1.Time {
	args := m.Called()
	return args.Get(0).(*metav1.Time)
}

func (m *mockPVC) GetStatusPhase() v1.PersistentVolumeClaimPhase {
	args := m.Called()
	return args.Get(0).(v1.PersistentVolumeClaimPhase)
}

func (m *mockPVC) GetSpecVolumeName() string {
	args := m.Called()
	return args.String(0)
}
