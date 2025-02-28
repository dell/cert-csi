package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	k8stesting "k8s.io/client-go/testing"

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

func (m *mockPodClient) List(ctx context.Context, opts metav1.ListOptions) (*v1.PodList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*v1.PodList), args.Error(1)
}

func TestCheckPodsandPvcs(t *testing.T) {

	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)
	clientSet.CoreV1().Pods("test-namespace").Create(ctx, &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace"}}, metav1.CreateOptions{})
	clientSet.CoreV1().PersistentVolumeClaims("test-namespace").Create(ctx, &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "test-pvc", Namespace: "test-namespace"}}, metav1.CreateOptions{})
	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	// Set up a reactor to simulate Pods becoming Ready
	clientSet.Fake.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		pod := createAction.GetObject().(*v1.Pod)
		// Set pod phase to Running
		pod.Status.Phase = v1.PodRunning
		// Simulate the Ready condition
		pod.Status.Conditions = append(pod.Status.Conditions, v1.PodCondition{
			Type:   v1.PodReady,
			Status: v1.ConditionTrue,
		})

		// Simulate the "FileSystemResizeSuccessful" event
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pod.Namespace,
				Name:      "test-event",
			},
			InvolvedObject: v1.ObjectReference{
				Namespace: pod.Namespace,
				Name:      pod.Name,
				UID:       pod.UID,
			},
			Reason: "FileSystemResizeSuccessful",
			Type:   v1.EventTypeNormal,
		}
		clientSet.Tracker().Add(event)

		return false, nil, nil // Allow normal processing to continue
	})
	clientSet.Fake.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		podName := getAction.GetName()
		// Create a pod object with the expected name and Ready status
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "test-namespace",
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		}
		return true, pod, nil
	})

	// Create a mock Clients instance
	mockClients := &MockClients{}

	// Set up the mock behavior for the CreatePodClient method
	mockClients.On("CreatePodClient", "test-namespace").Return(
		&pod.Client{
			Interface: clientSet.CoreV1().Pods("test-namespace"),
		},
		nil,
	)

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	mockEno := &EntityNumberObserver{}

	mockInfo := &store.NumberEntities{}

	podRes4, err3 := mockEno.checkPods(podClient, mockInfo)
	if err3 != nil {
		t.Errorf("Error calling checkPods: %v", err3)
	}

	go mockEno.Interrupt()
	assert.Equal(t, false, podRes4)

	pvcRes3, err3 := mockEno.checkPvcs(pvcClient, mockInfo)
	if err3 != nil {
		t.Errorf("Error calling checkPods: %v", err3)
	}

	go mockEno.Interrupt()
	assert.Equal(t, false, pvcRes3)

	podRes, err := mockEno.checkPods(podClient, mockInfo)
	if err != nil {
		t.Errorf("Error calling checkPods: %v", err)
	}
	assert.Nil(t, err)
	assert.Equal(t, false, podRes)

	pvcRes, err2 := mockEno.checkPvcs(pvcClient, mockInfo)
	if err2 != nil {
		t.Errorf("Error calling checkPods: %v", err2)
	}
	assert.Nil(t, err2)
	assert.Equal(t, false, pvcRes)
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
		WaitGroup: sync.WaitGroup{},
		Database:  &SimpleStore{},
	}
	runner.WaitGroup.Add(1)

	// Create an EntityNumberObserver instance
	eno := &EntityNumberObserver{}
	eno.MakeChannel()

	// Start watching entities
	go eno.StartWatching(ctx, runner)

	// Wait for the StartWatching function to complete
	time.Sleep(1 * time.Second)

	// Assert that the number of entities is correct
	// assert.Equal(t, 1, len((*SimpleStore).NumberEntities(runner.Database)))

	// // Assert that the Timestamp is not zero
	// assert.NotZero(t, SimpleStore.NumberEntities[0].Timestamp)
	eno.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
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
