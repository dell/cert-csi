package observer

import (
	"context"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"

	"github.com/dell/cert-csi/pkg/store"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kfake "k8s.io/client-go/kubernetes/fake"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type FakeExtendedClientset struct {
	*kfake.Clientset
}

type FakeExtendedCoreV1 struct {
	typedcorev1.CoreV1Interface
	restClient rest.Interface
}

func (f *FakeExtendedClientset) CoreV1() typedcorev1.CoreV1Interface {
	return &FakeExtendedCoreV1{f.Clientset.CoreV1(), nil}
}

func NewFakeClientsetWithRestClient(objs ...runtime.Object) *FakeExtendedClientset {
	return &FakeExtendedClientset{kfake.NewSimpleClientset(objs...)}
}

type (
	// EntityTypeEnum specifies type of entity
	EntityTypeEnum string
	// EventTypeEnum specifies type of event
	EventTypeEnum string
)
type Event struct {
	ID        int64
	Name      string
	TcID      int64
	EntityID  int64
	Type      EventTypeEnum
	Timestamp time.Time
}

type SimpleStore struct {
	store.Store
	entities []*store.NumberEntities
	events   []*store.Event
}

func (s *SimpleStore) SaveEntities(entity []*store.Entity) error {
	return nil
}

func (s *SimpleStore) SaveNumberEntities(nEntities []*store.NumberEntities) error {
	s.entities = nEntities
	return nil
}

func (s *SimpleStore) SaveEntities(entity []*store.Entity) error {
	return nil
}
func (s *SimpleStore) SaveEvents(events []*store.Event) error {
	s.events = events
	return nil
}

func (s *SimpleStore) NumberEntities() []*store.NumberEntities {
	return s.entities
}

func (s *SimpleStore) SaveTestRun(tr *store.TestRun) error {
	// Implement the method here
	return nil
}

func (s *SimpleStore) GetTestRuns(whereConditions store.Conditions, orderBy string, limit int) ([]store.TestRun, error) {
	// Implement the method here
	return nil, nil
}

// Implement other methods as needed

func NewSimpleStore() *SimpleStore {
	return &SimpleStore{}
}

func TestEntityNumberObserver_StartWatching(t *testing.T) {
	// Create a context
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
	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	// Create a Runner instance
	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: pvcClient,
			PodClient: podClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		Database: &SimpleStore{},
	}

	// Create an EntityNumberObserver instance
	eno := &EntityNumberObserver{}

	// Start watching entities
	go eno.StartWatching(ctx, runner)

	// Wait for the StartWatching function to complete
	time.Sleep(1 * time.Second)

	// Assert that the number of entities is correct
	// assert.Equal(t, 1, len((*SimpleStore).NumberEntities(runner.Database)))

	// // Assert that the Timestamp is not zero
	// assert.NotZero(t, SimpleStore.NumberEntities[0].Timestamp)
}

func TestEntityNumberObserver_checkPvcs(t *testing.T) {

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

	// Create a PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Create an EntityNumberObserver instance
	eno := &EntityNumberObserver{}

	// Create a NumberEntities instance
	info := &store.NumberEntities{}

	// Call the checkPvcs function
	b, e := eno.checkPvcs(pvcClient, info)

	// Assert that the function returned false and no error
	assert.False(t, b)
	assert.NoError(t, e)

	// Assert that the PvcCreating, PvcTerminating, and PvcBound fields are correct
	assert.Equal(t, 0, info.PvcCreating)
	assert.Equal(t, 0, info.PvcTerminating)
	assert.Equal(t, 0, info.PvcBound)
}

func TestEntityNumberObserver_checkPods(t *testing.T) {

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

	// Create a Pod client
	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	// Create an EntityNumberObserver instance
	eno := &EntityNumberObserver{}

	// Create a NumberEntities instance
	info := &store.NumberEntities{}

	// Call the checkPods function
	b, e := eno.checkPods(podClient, info)

	// Assert that the function returned false and no error
	assert.False(t, b)
	assert.NoError(t, e)

	// Assert that the PodsCreating, PodsTerminating, and PodsReady fields are correct
	assert.Equal(t, 0, info.PodsCreating)
	assert.Equal(t, 0, info.PodsTerminating)
	assert.Equal(t, 0, info.PodsReady)
}

// FakeDatabase is a mock implementation of the Database interface
type FakeDatabase struct {
	NumberEntities []*store.NumberEntities
}

// SaveNumberEntities is a mock implementation of the SaveNumberEntities method
func (f *FakeDatabase) SaveNumberEntities(nEntities []*store.NumberEntities) error {
	f.NumberEntities = nEntities
	return nil
}

// Close is a mock implementation of the Close method
func (f *FakeDatabase) Close() error {
	return nil
}

// FakePVC is a mock implementation of the PVCClient interface
type FakePVC struct{}

// List is a mock implementation of the List method
func (f *FakePVC) List(ctx context.Context, opts metav1.ListOptions) (*v1.PersistentVolumeClaimList, error) {
	return &v1.PersistentVolumeClaimList{}, nil
}

// FakePod is a mock implementation of the PodClient interface
type FakePod struct{}

// List is a mock implementation of the List method
func (f *FakePod) List(ctx context.Context, opts metav1.ListOptions) (*v1.PodList, error) {
	return &v1.PodList{}, nil
}
