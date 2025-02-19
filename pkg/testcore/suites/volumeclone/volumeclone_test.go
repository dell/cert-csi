package volumeclone

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore/suites/mockutils"
	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"reflect"
	"testing"
)

// TestCloneVolumeSuite_Run
func TestCloneVolumeSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new CloneVolumeSuite instance
	cs := &CloneVolumeSuite{
		VolumeNumber:  1,
		CustomPvcName: "pvc-test",
		CustomPodName: "pod-test",
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
	}

	// Create PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Create Pod client
	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	// Create PV client
	pvClient, _ := kubeClient.CreatePVClient()

	// Update the k8sclient.Clients instance with the fake clients
	clients := &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		PersistentVolumeClient: pvClient,
		KubeClient:             kubeClient,
	}

	// Call the Run method
	_, err := cs.Run(ctx, "test-storage-class", clients)
	// Check if there was an error
	if err != nil {
		t.Errorf("Error running CloneVolumeSuite.Run(): %v", err)
	}
}

func TestCloneVolumeSuite_GetObservers(t *testing.T) {
	cs := &CloneVolumeSuite{}
	obsType := observer.Type("someType")
	observers := cs.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

// TestCloneVolumeSuite_GetClients
func TestCloneVolumeSuite_GetClients(t *testing.T) {
	client := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
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
		cs      *CloneVolumeSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			cs:   &CloneVolumeSuite{},
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
				SnapClientGA:      nil,
				SnapClientBeta:    nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cs.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("CloneVolumeSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("CloneVolumeSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloneVolumeSuite_GetNamespace(t *testing.T) {
	cs := &CloneVolumeSuite{}
	namespace := cs.GetNamespace()
	expectedNamespace := "clonevolume-suite"
	if namespace != expectedNamespace {
		t.Errorf("CloneVolumeSuite.GetNamespace() = %v, want %v", namespace, expectedNamespace)
	}
}

func TestCloneVolumeSuite_GetName(t *testing.T) {
	cs := &CloneVolumeSuite{Description: "Test Suite"}
	name := cs.GetName()
	expectedName := "Test Suite"
	if name != expectedName {
		t.Errorf("CloneVolumeSuite.GetName() = %v, want %v", name, expectedName)
	}

	cs.Description = ""
	name = cs.GetName()
	expectedName = "CloneVolumeSuite"
	if name != expectedName {
		t.Errorf("CloneVolumeSuite.GetName() = %v, want %v", name, expectedName)
	}
}

func TestCloneVolumeSuite_Parameters(t *testing.T) {
	cs := &CloneVolumeSuite{
		PodNumber:    3,
		VolumeNumber: 5,
		VolumeSize:   "10Gi",
	}
	params := cs.Parameters()
	expectedParams := "{pods: 3, volumes: 5, volumeSize: 10Gi}"
	if params != expectedParams {
		t.Errorf("CloneVolumeSuite.Parameters() = %v, want %v", params, expectedParams)
	}
}
