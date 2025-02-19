package vgs

import (
	"context"
	"errors"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/volumegroupsnapshot"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore/suites/mockutils"
	vgsAlpha "github.com/dell/csi-volumegroup-snapshotter/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestVolumeGroupSnapSuite_Run
func TestVolumeGroupSnapSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new VolumeGroupSnapSuite instance
	vgs := &VolumeGroupSnapSuite{
		SnapClass:  "testSnap",
		AccessMode: "ReadWriteOnce",
	}

	// Create a fake storage class with VolumeBindingMode set to WaitForFirstConsumer
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}

	// Create a fake k8s clientset with the storage class
	clientset := fake.NewSimpleClientset(storageClass)
	clientset = mockutils.MockClientSetPodFunctions(clientset).(*fake.Clientset)

	// Create a fake k8sclient.KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientset,
		Config:    &rest.Config{},
	}

	// Create PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Create Pod client
	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	// Create PV client
	pvClient, _ := kubeClient.CreatePVClient()

	scheme := runtime.NewScheme()
	if err := vgsAlpha.AddToScheme(scheme); err != nil {
		panic(err)
	}
	restMapperMock := &mockutils.RESTMapping{}
	restMapperMock.On("RESTMapping", mock.Anything, mock.Anything).Return(restMapperMock, nil)
	k8sClient, _ := client.New(kubeClient.Config, client.Options{Scheme: scheme, Mapper: restMapperMock})
	vgsClient := &volumegroupsnapshot.Client{
		Interface: k8sClient,
	}

	// Update the k8sclient.Clients instance with the fake clients
	clients := &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		PersistentVolumeClient: pvClient,
		KubeClient:             kubeClient,
		VgsClient:              vgsClient,
	}

	_, err := vgs.Run(ctx, "test-storage-class", clients)

	expectedError := errors.New("connection refused")
	assert.Contains(t, err.Error(), expectedError.Error())
}

func TestVolumeGroupSnapSuite_GetObservers(t *testing.T) {
	vgs := &VolumeGroupSnapSuite{}
	obsType := observer.Type("someType")
	observers := vgs.GetObservers(obsType)
	if observers == nil {
		t.Errorf("Expected observers, got nil")
	}
	// Add more assertions based on expected behavior
}

func TestVolumeGroupSnapSuite_GetClients(t *testing.T) {
	client := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

	type args struct {
		namespace string
		client    *k8sclient.KubeClient
	}
	tests := []struct {
		name    string
		vgs     *VolumeGroupSnapSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			vgs:  &VolumeGroupSnapSuite{SnapClass: "test-volsnap-class"},
			args: args{
				namespace: "test-namespace",
				client:    &kubeClient,
			},
			want: &k8sclient.Clients{
				PVCClient:     pvcClient,
				MetricsClient: metricsClient,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.vgs.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			expectedError := errors.New("connection refused")
			assert.Contains(t, err.Error(), expectedError.Error())
		})
	}
}

func TestVolumeGroupSnapSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		vgs  *VolumeGroupSnapSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			vgs:  &VolumeGroupSnapSuite{},
			want: "vgs-volsnap-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vgs.GetNamespace(); got != tt.want {
				t.Errorf("VolumeGroupSnapSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeGroupSnapSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		vgs  *VolumeGroupSnapSuite
		want string
	}{
		{
			name: "Testing GetName",
			vgs:  &VolumeGroupSnapSuite{},
			want: "VolumeGroupSnapSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vgs.GetName(); got != tt.want {
				t.Errorf("VolumeGroupSnapSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeGroupSnapSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		vgs  *VolumeGroupSnapSuite
		want string
	}{
		{
			name: "Testing Parameters",
			vgs: &VolumeGroupSnapSuite{
				VolumeNumber: 5,
				VolumeSize:   "3Gi",
			},
			want: "{volumes: 5, volumeSize: 3Gi}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vgs.Parameters(); got != tt.want {
				t.Errorf("VolumeGroupSnapSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
