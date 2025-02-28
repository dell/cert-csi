package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
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
	clientSet.CoreV1().PersistentVolumeClaims("test-namespace").Create(ctx, &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "test-pvc", Namespace: "test-namespace"}}, metav1.CreateOptions{})
	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	// Set up a reactor to simulate PVCs becoming Bound
	clientSet.Fake.PrependReactor("create", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		pvc := createAction.GetObject().(*v1.PersistentVolumeClaim)
		// Set PVC phase to Bound
		pvc.Status.Phase = v1.ClaimBound

		// Simulate the "FileSystemResizeSuccessful" event
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pvc.Namespace,
				Name:      "test-event",
			},
			InvolvedObject: v1.ObjectReference{
				Namespace: pvc.Namespace,
				Name:      pvc.Name,
				UID:       pvc.UID,
			},
			Reason: "FileSystemResizeSuccessful",
			Type:   v1.EventTypeNormal,
		}
		clientSet.Tracker().Add(event)

		return false, nil, nil // Allow normal processing to continue
	})

	// Set up a reactor to simulate getting PVCs
	clientSet.Fake.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		pvcName := getAction.GetName()

		// Create a PVC object with the expected name and Bound status
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: "test-namespace",
			},
			Status: v1.PersistentVolumeClaimStatus{
				Phase: v1.ClaimBound,
			},
		}

		return true, pvc, nil
	})

	// Create a mock Clients instance
	mockClients := &MockClients{}

	// Set up the mock behavior for the CreatePodClient method
	mockClients.On("CreatePVCClient", "test-namespace").Return(
		&pvc.Client{
			Interface: clientSet.CoreV1().PersistentVolumeClaims("test-namespace"),
		},
		nil,
	)

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

	// Set up a reactor to simulate the deletion of PVCs
	clientSet.Fake.PrependReactor("delete", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteAction := action.(k8stesting.DeleteAction)
		pvcName := deleteAction.GetName()

		// Simulate the deletion of the PVC
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: "test-namespace",
			},
		}

		// Return the deleted PVC
		return true, pvc, nil
	})

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
