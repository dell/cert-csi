package observer

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/metrics"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pv"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/replicationgroup"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

var (
	remotePVCObject  v1.PersistentVolumeClaim
	remotePVClient   *pv.Client
	remoteRGClient   *replicationgroup.Client
	remoteKubeClient *k8sclient.KubeClient
)

func (c *FakeExtendedCoreV1) RESTClient() rest.Interface {
	if c.restClient == nil {
		c.restClient = &restfake.RESTClient{}
	}
	return c.restClient
}

type FakeHashRemoteExecutor struct{}

func (FakeHashRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	Output := "OK"
	_, err := fmt.Fprint(stdout, Output)
	if err != nil {
		return err
	}
	return nil
}

type MockClients struct {
	mock.Mock
}
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

//	func NewMockMetricsClient(*gomock.Controller) *MockMetricsClient {
//		return &MockMetricsClient{}
//	}

func (m *mockDatabase) SaveResourceUsage(resUsage []*store.ResourceUsage) error {
	args := m.Called(resUsage)
	return args.Error(0)
}

// type mockMetricsClient struct {
// 	Interface MetricsV1beta1Interface
// }

type fakeMetricsServer struct {
	podMetrics []v1beta1.PodMetrics
}

// func (f *fakeMetricsServer) PodMetricses(namespace string) v1beta1.PodMetricsInterface {
// 	return &fakePodMetricsInterface{
// 		fakeMetricsServer: f,
// 		namespace:         namespace,
// 	}
// }

type fakePodMetricsInterface struct {
	fakeMetricsServer *fakeMetricsServer
	namespace         string
}

func (f *fakePodMetricsInterface) List(ctx context.Context, opts metav1.ListOptions) (*v1beta1.PodMetricsList, error) {
	var items []v1beta1.PodMetrics
	for _, pm := range f.fakeMetricsServer.podMetrics {
		if pm.Namespace == f.namespace {
			items = append(items, pm)
		}
	}
	return &v1beta1.PodMetricsList{
		Items: items,
	}, nil
}

type MockPodMetricses struct {
	mock.Mock
}

func (m *MockPodMetricses) List(ctx context.Context, opts metav1.ListOptions) (*v1beta1.PodMetricsList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*v1beta1.PodMetricsList), args.Error(1)
}

func TestContainerMetricsObserver_StartWatching(t *testing.T) {

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
	kubeClient.SetTimeout(3000)

	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	podClient.RemoteExecutor = &FakeHashRemoteExecutor{}
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

	// Create a PodMetrics object
	podMetrics := &v1beta1.PodMetrics{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMetrics",
			APIVersion: "metrics.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-metrics-example",
			Namespace: "default",
		},
		Timestamp: metav1.Time{Time: time.Now()},
		Window:    metav1.Duration{Duration: time.Minute},
		Containers: []v1beta1.ContainerMetrics{
			{
				Name:  "container-1",
				Usage: v1.ResourceList{},
			},
		},
	}

	// Create a PodMetricsList object
	podMetricsList := &v1beta1.PodMetricsList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMetricsList",
			APIVersion: "metrics.k8s.io/v1beta1",
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
		Items: []v1beta1.PodMetrics{
			*podMetrics,
		},
	}

	tests := []struct {
		name        string
		runner      *Runner
		metricsList *v1beta1.PodMetricsList
	}{
		{
			name: "Test case: Watching container metrics without driver namespace",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					PodClient:     podClient,
					MetricsClient: metricsClient,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup: sync.WaitGroup{},
				Database:  NewSimpleStore(),
			},
			metricsList: nil,
		},
		{
			name: "Test case: Watching container metrics with driver namespace and error getMetricsList",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					PodClient:     podClient,
					MetricsClient: metricsClient,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup:       sync.WaitGroup{},
				DriverNamespace: "test-namespace",
				Database:        NewSimpleStore(),
			},
			metricsList: nil,
		},
		{
			name: "Test case: Watching container metrics with driver namespace and metricList",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					PodClient:     podClient,
					MetricsClient: metricsClient,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup:       sync.WaitGroup{},
				DriverNamespace: "test-namespace",
				Database:        NewSimpleStore(),
			},
			metricsList: podMetricsList,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var metricListWg sync.WaitGroup

			ctx := context.Background()

			test.runner.WaitGroup.Add(1)

			cmo := &ContainerMetricsObserver{}
			defer func() {
				cmo.Interrupt()
				cmo.StopWatching()
			}()
			cmo.MakeChannel()

			if test.metricsList != nil {
				metricListWg.Add(1)
				originalGetMetricsList := getMetricsList
				getMetricsList = func(mc *metrics.Client, driverNamespace string, ctx context.Context, opts metav1.ListOptions) (*v1beta1.PodMetricsList, error) {
					metricListWg.Done()

					return test.metricsList, nil
				}
				defer func() {
					getMetricsList = originalGetMetricsList
				}()
			}

			go cmo.StartWatching(ctx, test.runner)
			if test.metricsList != nil {
				metricListWg.Wait()
				go func() {
					cmo.mutex.Lock()
					cmo.finished <- true
					cmo.mutex.Unlock()
				}()
			}
			test.runner.WaitGroup.Wait()

			// Assert that the function completed successfully
			assert.True(t, true)
		})
	}
}

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
