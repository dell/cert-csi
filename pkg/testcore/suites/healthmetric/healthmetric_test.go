package healthmetric

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

// TestVolumeHealthMetricsSuite_Run
func TestVolumeHealthMetricsSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new VolumeHealthMetricsSuite instance
	vh := &VolumeHealthMetricsSuite{
		VolumeNumber: 1,
		Namespace:    "test-namespace",
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

	FindDriverLogs = func(_ []string) (string, error) {
		return "", nil
	}

	// Call the Run method
	_, err := vh.Run(ctx, "test-storage-class", clients)
	// Check if there was an error
	if err != nil {
		t.Errorf("Error running VolumeHealthMetricsSuite.Run(): %v", err)
	}
}

func TestVolumeHealthMetricsSuite_GetObservers(t *testing.T) {
	vh := &VolumeHealthMetricsSuite{}
	obsType := observer.Type("someType")
	observers := vh.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

// TestVolumeHealthMetricsSuite_GetClients
func TestVolumeHealthMetricsSuite_GetClients(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet: clientset,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	pvClient, _ := kubeClient.CreatePVClient()
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

	type args struct {
		namespace string
		client    *k8sclient.KubeClient
	}
	tests := []struct {
		name    string
		vhms    *VolumeHealthMetricsSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			vhms: &VolumeHealthMetricsSuite{},
			args: args{
				namespace: "test-namespace",
				client:    &kubeClient,
			},
			want: &k8sclient.Clients{
				PVCClient:              pvcClient,
				PodClient:              podClient,
				VaClient:               vaClient,
				StatefulSetClient:      nil,
				MetricsClient:          metricsClient,
				KubeClient:             &kubeClient,
				PersistentVolumeClient: pvClient,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.vhms.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("VolumeHealthMetricsSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("VolumeHealthMetricsSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeHealthMetricsSuite_GetNamespace(t *testing.T) {
	vh := &VolumeHealthMetricsSuite{}
	namespace := vh.GetNamespace()
	expectedNamespace := "volume-health-metrics"
	if namespace != expectedNamespace {
		t.Errorf("VolumeHealthMetricsSuite.GetNamespace() = %v, want %v", namespace, expectedNamespace)
	}
}

func TestVolumeHealthMetricsSuite_GetName(t *testing.T) {
	vh := &VolumeHealthMetricsSuite{Description: "Test Suite"}
	name := vh.GetName()
	expectedName := "Test Suite"
	if name != expectedName {
		t.Errorf("VolumeHealthMetricsSuite.GetName() = %v, want %v", name, expectedName)
	}

	vh.Description = ""
	name = vh.GetName()
	expectedName = "VolumeHealthMetricSuite"
	if name != expectedName {
		t.Errorf("VolumeHealthMetricsSuite.GetName() = %v, want %v", name, expectedName)
	}
}

func TestVolumeHealthMetricsSuite_Parameters(t *testing.T) {
	vh := &VolumeHealthMetricsSuite{
		PodNumber:    3,
		VolumeNumber: 5,
		VolumeSize:   "10Gi",
	}
	params := vh.Parameters()
	expectedParams := "{pods: 3, volumes: 5, size: 10Gi}"
	if params != expectedParams {
		t.Errorf("VolumeHealthMetricsSuite.Parameters() = %v, want %v", params, expectedParams)
	}
}
