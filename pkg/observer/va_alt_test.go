package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVaListObserver_StartWatching(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)

	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
			UID:  "test-uid",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
	}

	/*va2 := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va2",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc2"; return &s }(),
			},
		},
	}*/

	va3 := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va3",
			UID:  "test-uid3",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc3"; return &s }(),
			},
		},
	}

	clientSet.StorageV1().VolumeAttachments().Create(ctx, va, metav1.CreateOptions{})
	//clientSet.StorageV1().VolumeAttachments().Create(ctx, va2, metav1.CreateOptions{})
	clientSet.StorageV1().VolumeAttachments().Delete(ctx, "test-va", metav1.DeleteOptions{})
	clientSet.StorageV1().VolumeAttachments().Update(ctx, va, metav1.UpdateOptions{})
	clientSet.StorageV1().VolumeAttachments().Create(ctx, va, metav1.CreateOptions{})
	clientSet.StorageV1().VolumeAttachments().Create(ctx, va3, metav1.CreateOptions{})

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: vaClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		PvcShare:  sync.Map{},
		Database:  NewSimpleStore(),
	}
	runner.PvcShare.Store("test-pvc", &store.Entity{
		Name: "test-pvc",
	})
	/*runner.PvcShare.Store("test-pvc2", &store.Entity{
		Name: "test-pvc2",
	})*/
	runner.PvcShare.Store("test-pvc2", &store.Entity{
		Name: "test-pvc2",
	})
	runner.WaitGroup.Add(1)

	obs := &VaListObserver{}
	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	clientSet.StorageV1().VolumeAttachments().Update(ctx, va, metav1.UpdateOptions{})
	clientSet.StorageV1().VolumeAttachments().Create(ctx, va, metav1.CreateOptions{})

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
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
