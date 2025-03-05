package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/va"
	"github.com/dell/cert-csi/pkg/store"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVaListObserver_StartWatching(t *testing.T) {
	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)
	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvName := "test-pv"
	deletionVA := &storagev1.VolumeAttachment{
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := pvName; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{
			Attached: false,
		},
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Name:              "test-volume-attachment",
		},
	}
	attachedVA := &storagev1.VolumeAttachment{
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pv-2"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{
			Attached: true,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-volume-attachment-2",
		},
	}

	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	vaClient.Interface.Create(ctx, deletionVA, metav1.CreateOptions{})
	vaClient.Interface.Create(ctx, attachedVA, metav1.CreateOptions{})

	var pvcShare sync.Map
	pvcShare.Store(pvName, &store.Entity{})
	pvcShare.Store("test-pv-2", &store.Entity{})

	tests := []struct {
		name                                   string
		runner                                 *Runner
		shouldAddedVABeTrue                    *bool
		shouldHaveUnequalAttachedAndDeletedVAs bool
	}{
		{
			name: "Test case: nil vaClient",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					VaClient: nil,
				},
				Database: NewSimpleStore(),
				PvcShare: pvcShare,
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup: sync.WaitGroup{},
			},
			shouldAddedVABeTrue: nil,
		},
		{
			name: "Test case: vaClient with original addedVA",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					VaClient: vaClient,
				},
				Database:    NewSimpleStore(),
				PvcShare:    pvcShare,
				ShouldClean: false,
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup: sync.WaitGroup{},
			},
			shouldAddedVABeTrue: nil,
		},
		{
			name: "Test case: vaClient with mocked addedVA and attached VA and deletion VA",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					VaClient: vaClient,
				},
				Database:    NewSimpleStore(),
				PvcShare:    pvcShare,
				ShouldClean: false,
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup: sync.WaitGroup{},
			},
			shouldAddedVABeTrue: func() *bool { b := true; return &b }(),
		},
		{
			name: "Test case: vaClient with mocked addedVA and attached VA and deletion VA and should clean",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					VaClient: vaClient,
				},
				Database:    NewSimpleStore(),
				PvcShare:    pvcShare,
				ShouldClean: true,
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup: sync.WaitGroup{},
			},
			shouldAddedVABeTrue:                    func() *bool { b := true; return &b }(),
			shouldHaveUnequalAttachedAndDeletedVAs: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var vaoFinishedWg sync.WaitGroup
			test.runner.WaitGroup.Add(1)

			vao := &VaListObserver{}
			vao.MakeChannel()
			// We want to run through pollImmediate at least twice
			pollRunCount := 0

			originalGetBoolValueFromMapWithKey := getBoolValueFromMapWithKey
			getBoolValueFromMapWithKey = func(m map[string]bool, key string) bool {
				pollRunCount++
				// After a couple runs, we test new state by deleting a new VA
				if pollRunCount == 6 {
					vaClient.Interface.Delete(ctx, attachedVA.Name, metav1.DeleteOptions{})
					if test.shouldHaveUnequalAttachedAndDeletedVAs {
						vaoFinishedWg.Done()
					}
				}
				// After a couple more runs, we mark as complete
				if pollRunCount == 10 && !test.shouldHaveUnequalAttachedAndDeletedVAs {
					vaoFinishedWg.Done()
				}
				if test.shouldAddedVABeTrue != nil {
					return *test.shouldAddedVABeTrue
				} else {
					return originalGetBoolValueFromMapWithKey(m, key)
				}
			}
			defer func() {
				vaClient.Interface.Create(ctx, attachedVA, metav1.CreateOptions{})
				getBoolValueFromMapWithKey = originalGetBoolValueFromMapWithKey
			}()

			go vao.StartWatching(ctx, test.runner)
			if test.runner.Clients.VaClient != nil {
				vaoFinishedWg.Add(1)
				vaoFinishedWg.Wait()
				vao.finished <- true
			}
			test.runner.WaitGroup.Wait()
		})
	}
}

func TestVaListObserver_StopWatching(t *testing.T) {
	// Test case: Stopping watching volume attachments
	obs := &VaListObserver{}

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

func TestVaListObserver_GetName(t *testing.T) {
	// Test case: Getting name of VA observer
	obs := &VaListObserver{}

	name := obs.GetName()

	assert.Equal(t, "VolumeAttachmentObserver", name)
}

func TestVaListObserver_MakeChannel(t *testing.T) {
	// Test case: Creating a new channel
	obs := &VaListObserver{}

	obs.MakeChannel()

	assert.NotNil(t, obs.finished)
}

// Mock implementation of VaClient
type mockVAClient struct {
	va.Client
	mock.Mock
}

func (m *mockVAClient) List(ctx context.Context, opts metav1.ListOptions) (*storagev1.VolumeAttachmentList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*storagev1.VolumeAttachmentList), args.Error(1)
}

// Mock implementation of Database
type mockDatabase struct {
	mock.Mock
}

// Mock implementation of PvcShare
type mockPvcShare struct {
	mock.Mock
}

func (m *mockPvcShare) Load(key interface{}) (value interface{}, ok bool) {
	args := m.Called(key)
	return args.Get(0), args.Bool(1)
}
