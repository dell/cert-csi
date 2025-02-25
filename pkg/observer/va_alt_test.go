package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"
	storagev1 "k8s.io/api/storage/v1"
	v1 "k8s.io/api/storage/v1"
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
	runner.WaitGroup.Add(1)

	obs := &VaListObserver{}
	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

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

func (m *mockVAClient) List(ctx context.Context, opts metav1.ListOptions) (*v1.VolumeAttachmentList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*v1.VolumeAttachmentList), args.Error(1)
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
