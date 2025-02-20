package multivolattach

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore/suites/mockutils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"reflect"
	"testing"
)

// TODO TestMultiAttachSuite_Run
func TestMultiAttachSuite_Run(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a MultiAttachSuite instance
	mas := &MultiAttachSuite{
		PodNumber:  2,
		RawBlock:   false,
		AccessMode: "ReadWriteMany",
		VolumeSize: "1Gi",
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
	namespace := mas.GetNamespace()
	clientSet := fake.NewSimpleClientset(storageClass)
	clientSet = mockutils.MockClientSetPodFunctions(clientSet).(*fake.Clientset)

	// Create a fake KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	// Create the necessary clients
	clients, err := mas.GetClients(namespace, kubeClient)
	assert.NoError(t, err)

	// Mock PVCClient and PodClient methods if needed (e.g., using a mocking library)

	// Run the suite
	delFunc, err := mas.Run(ctx, "test-storage-class", clients)
	assert.Error(t, err)
	assert.Nil(t, delFunc)
}

func TestMultiAttachSuite_GenerateTopologySpreadConstraints(t *testing.T) {
	mas := &MultiAttachSuite{PodNumber: 5}
	nodeCount := 3
	labels := map[string]string{"app": "test"}
	constraints := mas.GenerateTopologySpreadConstraints(nodeCount, labels)
	assert.NotNil(t, constraints)
	assert.Equal(t, 1, len(constraints))
	assert.Equal(t, int32(2), constraints[0].MaxSkew)
	assert.Equal(t, "kubernetes.io/hostname", constraints[0].TopologyKey)
	assert.Equal(t, v1.ScheduleAnyway, constraints[0].WhenUnsatisfiable)
	assert.Equal(t, labels, constraints[0].LabelSelector.MatchLabels)
}

func TestMultiAttachSuite_GetObservers(t *testing.T) {
	mas := &MultiAttachSuite{}
	obsType := observer.Type("someType")
	observers := mas.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

func TestMultiAttachSuite_GetClients(t *testing.T) {
	client := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
		Minor:       19, // Simulate Kubernetes version 1.19 or higher
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")
	nodeClient, _ := kubeClient.CreateNodeClient()

	type args struct {
		namespace string
		client    *k8sclient.KubeClient
	}
	tests := []struct {
		name    string
		mas     *MultiAttachSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			mas:  &MultiAttachSuite{},
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
				NodeClient:        nodeClient,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.mas.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("MultiAttachSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("MultiAttachSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiAttachSuite_GetNamespace(t *testing.T) {
	mas := &MultiAttachSuite{}
	namespace := mas.GetNamespace()
	expectedNamespace := "mas-test"
	if namespace != expectedNamespace {
		t.Errorf("MultiAttachSuite.GetNamespace() = %v, want %v", namespace, expectedNamespace)
	}
}

func TestMultiAttachSuite_GetName(t *testing.T) {
	mas := &MultiAttachSuite{Description: "Test Suite"}
	name := mas.GetName()
	expectedName := "Test Suite"
	if name != expectedName {
		t.Errorf("MultiAttachSuite.GetName() = %v, want %v", name, expectedName)
	}

	mas.Description = ""
	name = mas.GetName()
	expectedName = "MultiAttachSuite"
	if name != expectedName {
		t.Errorf("MultiAttachSuite.GetName() = %v, want %v", name, expectedName)
	}
}

func TestMultiAttachSuite_Parameters(t *testing.T) {
	mas := &MultiAttachSuite{
		PodNumber:  3,
		RawBlock:   true,
		VolumeSize: "10Gi",
		AccessMode: "ReadWriteOnce",
	}
	params := mas.Parameters()
	expectedParams := "{pods: 3, rawBlock: true, size: 10Gi, accMode: ReadWriteOnce}"
	if params != expectedParams {
		t.Errorf("MultiAttachSuite.Parameters() = %v, want %v", params, expectedParams)
	}
}
