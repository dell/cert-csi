package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// type MockRunner struct {
// 	WaitGroup       sync.WaitGroup
// 	Observers       []Interface
// 	Clients         *k8sclient.Clients
// 	Database        store.Store
// 	TestCase        *store.TestCase
// 	PvcShare        sync.Map
// 	DriverNamespace string
// 	ShouldClean     bool
// }

// func (runner *MockRunner) Start(ctx context.Context) error {
// 	return nil
// }

// func (runner *MockRunner) Stop() error {
// 	return nil
// }

// func (runner *MockRunner) GetName() string {
// 	return "MockRunner"
// }

// func (runner *MockRunner) MakeChannel() {
// 	// do nothing
// }

// func (runner *MockRunner) waitTimeout(timeout time.Duration) bool {
// 	return false
// }

// func NewMockRunner() *MockRunner {
// 	return &MockRunner{}
// }

type MockRunner struct {
	WaitGroup       sync.WaitGroup
	Observers       []Interface
	Clients         *k8sclient.Clients
	Database        store.Store
	TestCase        *store.TestCase
	PvcShare        sync.Map
	DriverNamespace string
	ShouldClean     bool
}

// Start is a mock implementation of the Start method of the Runner interface
func (m *MockRunner) Start(ctx context.Context) error {
	return nil
}

// Stop is a mock implementation of the Stop method of the Runner interface
func (m *MockRunner) Stop() error {
	return nil
}

// GetName is a mock implementation of the GetName method of the Runner interface
func (m *MockRunner) GetName() string {
	return "MockRunner"
}

// MakeChannel is a mock implementation of the MakeChannel method of the Runner interface
func (m *MockRunner) MakeChannel() {
	// do nothing
}

// waitTimeout is a mock implementation of the waitTimeout method of the Runner interface
func (m *MockRunner) waitTimeout(timeout time.Duration) bool {
	return false
}

func NewMockRunner(*gomock.Controller) *MockRunner {
	return &MockRunner{
		Observers:       []Interface{&ContainerMetricsObserver{}},
		Clients:         &k8sclient.Clients{},
		Database:        NewSimpleStore(),
		TestCase:        &store.TestCase{},
		PvcShare:        sync.Map{},
		DriverNamespace: "driver-namespace",
		ShouldClean:     true,
	}
}

// func NewMockMetricsClient(*gomock.Controller) *MockMetricsClient {
// 	return &MockMetricsClient{}
// }

// func TestContainerMetricsObserver_StartWatching(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockRunner := NewMockRunner(ctrl)
// 	mockMetricsClient := NewMockMetricsClient(ctrl)

// 	storageClass := &storagev1.StorageClass{
// 		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
// 		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
// 			mode := storagev1.VolumeBindingWaitForFirstConsumer
// 			return &mode
// 		}(),
// 	}
// 	clientSet := NewFakeClientsetWithRestClient(storageClass)

// 	kubeClient := &k8sclient.KubeClient{
// 		ClientSet: clientSet,
// 		Config:    &rest.Config{},
// 	}

// 	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

// 	// Set the fields of the MockRunner
// 	mockRunner.Observers = []Interface{&ContainerMetricsObserver{}}
// 	mockRunner.Clients = &k8sclient.Clients{}
// 	mockRunner.Database = NewSimpleStore()
// 	mockRunner.TestCase = &store.TestCase{}
// 	mockRunner.PvcShare = sync.Map{}
// 	mockRunner.DriverNamespace = "driver-namespace"
// 	mockRunner.ShouldClean = true

// 	runner := &Runner{
// 		WaitGroup:       mockRunner.WaitGroup,
// 		Observers:       mockRunner.Observers,
// 		Clients:         mockRunner.Clients,
// 		Database:        mockRunner.Database,
// 		TestCase:        mockRunner.TestCase,
// 		PvcShare:        mockRunner.PvcShare,
// 		DriverNamespace: mockRunner.DriverNamespace,
// 		ShouldClean:     mockRunner.ShouldClean,
// 	}

// 	// Set the return values for the mockRunner functions
// 	mockRunner.On("GetDatabase").Return(nil).AnyTimes()
// 	mockRunner.On("GetTestCase").Return(nil).AnyTimes()
// 	// mockRunner.EXPECT().GetDatabase().Return(nil).AnyTimes()
// 	// mockRunner.EXPECT().GetTestCase().Return(nil).AnyTimes()

// 	// Test case when driver namespace is empty
// 	mockRunner.EXPECT().GetDriverNamespace().Return("").Times(1)

// 	cmo := &ContainerMetricsObserver{}
// 	cmo.StartWatching(context.Background(), runner)

// 	// Test case when driver namespace is not empty
// 	mockRunner.EXPECT().GetDriverNamespace().Return("driver-namespace").Times(1)
// 	mockRunner.EXPECT().GetMetricsClient().Return(metricsClient).Times(1)
// 	mockMetricsClient.EXPECT().Timeout.Return(0).Times(1)

// 	var wg sync.WaitGroup
// 	wg.Add(1)
// 	mockRunner.EXPECT().GetWaitGroup().Return(&wg).Times(1)

// 	mockMetricsClient.EXPECT().Interface.MetricsV1beta1().PodMetricses("driver-namespace").List(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

// 	cmo.StartWatching(context.Background(), runner)

// 	// Test case when there is an error in polling
// 	mockRunner.EXPECT().GetDriverNamespace().Return("driver-namespace").Times(1)
// 	mockRunner.EXPECT().GetMetricsClient().Return(mockMetricsClient).Times(1)
// 	metricsClient.EXPECT().Timeout.Return(0).Times(1)

// 	mockMetricsClient.EXPECT().Interface.MetricsV1beta1().PodMetricses("driver-namespace").List(gomock.Any(), gomock.Any()).Return(nil, assert.AnError).Times(1)

// 	cmo.StartWatching(context.Background(), runner)
// }

func TestContainerMetricsObserver_StopWatching(t *testing.T) {
	cmo := &ContainerMetricsObserver{}

	// Test case when Interrupted is false
	cmo.finished = make(chan bool)
	go func() {
		cmo.StopWatching()
	}()
	time.Sleep(1 * time.Second)

	// Test case when Interrupted is true
	cmo.Interrupt()
	cmo.StopWatching()
}

func TestContainerMetricsObserver_GetName(t *testing.T) {
	cmo := &ContainerMetricsObserver{}
	assert.Equal(t, "ContainerMetricsObserver", cmo.GetName())
}

func TestContainerMetricsObserver_MakeChannel(t *testing.T) {
	cmo := &ContainerMetricsObserver{}
	cmo.MakeChannel()
	assert.NotNil(t, cmo.finished)
}
