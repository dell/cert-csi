package volumeio

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kfake "k8s.io/client-go/kubernetes/fake"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	"testing"
)

// TODO put this repeated types and funcs in a common test file
type FakeHashRemoteExecutor struct{}

func (FakeHashRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout,
	stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	Output := "OK"
	_, err := fmt.Fprint(stdout, Output)
	if err != nil {
		return err
	}
	return nil
}

type FakeRemoteExecutor struct{}

type FakeRemoteExecutor_VolExpansion struct {
	callCount int
}

func (f *FakeRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout,
	stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	return nil
}

type FakeExtendedCoreV1 struct {
	typedcorev1.CoreV1Interface
	restClient rest.Interface
}

func (c *FakeExtendedCoreV1) RESTClient() rest.Interface {
	if c.restClient == nil {
		c.restClient = &restfake.RESTClient{}
	}
	return c.restClient
}

type FakeExtendedClientset struct {
	*kfake.Clientset
}

func (f *FakeExtendedClientset) CoreV1() typedcorev1.CoreV1Interface {
	return &FakeExtendedCoreV1{f.Clientset.CoreV1(), nil}
}

func NewFakeClientsetWithRestClient(objs ...runtime.Object) *FakeExtendedClientset {
	return &FakeExtendedClientset{kfake.NewSimpleClientset(objs...)}
}

type MockClients struct {
	mock.Mock
}

// CreatePodClient is a mock implementation of the CreatePodClient method
func (c *MockClients) CreatePodClient(namespace string) (*pod.Client, error) {
	args := c.Called(namespace)
	podClient, err := args.Get(0).(*pod.Client), args.Error(1)
	return podClient, err
}

// TODO TestVolumeIoSuite_Run
func TestVolumeIoSuite_Run(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a VolumeIoSuite instance
	vis := &VolumeIoSuite{
		VolumeNumber: 1,
		VolumeSize:   "1Gi",
		ChainNumber:  1,
		ChainLength:  2,
		Image:        "quay.io/centos/centos:latest",
	}

	vis2 := &VolumeIoSuite{
		VolumeNumber: 0,
		VolumeSize:   "1Gi",
		ChainNumber:  0,
		ChainLength:  0,
		Image:        "",
	}

	// Mock storageClass
	// Create a fake storage class with VolumeBindingMode set to WaitForFirstConsumer
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}

	// clientSet := fake.NewSimpleClientset(storageClass)
	clientSet := NewFakeClientsetWithRestClient(storageClass)

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
		return false, nil, nil // Allow normal processing to continue
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

	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})

	// Create a fake k8sclient.KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	podClient.RemoteExecutor = &FakeRemoteExecutor{}
	podClient.RemoteExecutor = &FakeHashRemoteExecutor{}
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")

	// Create a fake clients instance
	clients1 := &k8sclient.Clients{
		PVCClient: pvcClient,
		PodClient: podClient,
		VaClient:  vaClient,
	}

	tests := []struct {
		name         string
		vis          *VolumeIoSuite
		storageClass string
		clients      *k8sclient.Clients
		wantDelFunc  func() error
		wantErr      bool
	}{
		{
			name:         "Testing Run with valid inputs",
			vis:          vis,
			storageClass: "test-storage-class",
			clients:      clients1,
			wantErr:      false,
		},
		{
			name:         "Testing Run with valid inputs3",
			vis:          vis2,
			storageClass: "test-storage-class",
			clients:      clients1,
			wantDelFunc: func() error {
				return fmt.Errorf("persistentvolumeclaims \"\" already exists")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.vis.Run(context.TODO(), tt.storageClass, tt.clients)
			if (err != nil) != tt.wantErr {
				assert.NotNil(t, tt.wantDelFunc())
				assert.Equal(t, tt.wantDelFunc().Error(), err.Error())
			}
		})
	}
}

func TestVolumeIoSuite_GetObservers(t *testing.T) {
	vis := &VolumeIoSuite{}
	obsType := observer.Type("someType")
	observers := vis.GetObservers(obsType)
	if observers == nil {
		t.Errorf("Expected observers, got nil")
	}
	// Add more assertions based on expected behavior
}

func TestVolumeIoSuite_GetClients(t *testing.T) {
	client := kfake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet: client,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

	type args struct {
		namespace string
		client    *k8sclient.KubeClient
	}
	tests := []struct {
		name    string
		vis     *VolumeIoSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients with empty namespace",
			vis:  &VolumeIoSuite{},
			args: args{
				client: &kubeClient,
			},
			want: &k8sclient.Clients{
				PVCClient:         pvcClient,
				PodClient:         podClient,
				VaClient:          vaClient,
				StatefulSetClient: nil,
				MetricsClient:     metricsClient,
			},
			wantErr: false,
		},
		{
			name: "Testing GetClients with valid namespace",
			vis:  &VolumeIoSuite{},
			args: args{
				namespace: "test-namespace",
				client:    &kubeClient,
			},
			want: &k8sclient.Clients{
				PVCClient:         pvcClient,
				PodClient:         podClient,
				VaClient:          vaClient,
				StatefulSetClient: nil,
				MetricsClient:     metricsClient,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.vis.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestVolumeIoSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		vis  *VolumeIoSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			vis:  &VolumeIoSuite{},
			want: "volumeio-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vis.GetNamespace(); got != tt.want {
				t.Errorf("VolumeIoSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeIoSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		vis  *VolumeIoSuite
		want string
	}{
		{
			name: "Testing GetName",
			vis:  &VolumeIoSuite{},
			want: "VolumeIoSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vis.GetName(); got != tt.want {
				t.Errorf("VolumeIoSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeIoSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		vis  *VolumeIoSuite
		want string
	}{
		{
			name: "Testing Parameters",
			vis: &VolumeIoSuite{
				VolumeNumber: 5,
				VolumeSize:   "3Gi",
				ChainNumber:  2,
				ChainLength:  10,
			},
			want: "{volumes: 5, volumeSize: 3Gi chains: 2-10}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vis.Parameters(); got != tt.want {
				t.Errorf("VolumeIoSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
