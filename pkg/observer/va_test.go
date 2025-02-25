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
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
)

func TestVaObserver_StartWatching(t *testing.T) {
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

	obs := &VaObserver{}
	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StopWatching(t *testing.T) {
	// Test case: Stopping watching volume attachments
	obs := &VaObserver{}

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

func TestVaObserver_GetName(t *testing.T) {
	// Test case: Getting name of VA observer
	obs := &VaObserver{}

	name := obs.GetName()

	assert.Equal(t, "VolumeAttachmentObserver", name)
}

func TestVaObserver_MakeChannel(t *testing.T) {
	// Test case: Creating a new channel
	obs := &VaObserver{}

	obs.MakeChannel()

	assert.NotNil(t, obs.finished)
}

func (m *mockVAClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(watch.Interface), args.Error(1)
}
