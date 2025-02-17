package migration

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"reflect"
	"testing"
)

// TODO TestGetSnapshotClient
func TestVolumeMigrateSuite_Run(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Mock storageClass
	storageClass := "test-storage-class"

	// Create a VolumeMigrateSuite instance
	vms := &VolumeMigrateSuite{
		TargetSC:     storageClass,
		Description:  "test-desc",
		VolumeNumber: 1,
		PodNumber:    3,
		Flag:         true,
	}

	// Create a fake KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: fake.NewSimpleClientset(),
		Config:    &rest.Config{},
	}

	namespace := vms.GetNamespace()

	// Create the necessary clients
	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
	podClient, _ := kubeClient.CreatePodClient(namespace)
	scClient, _ := kubeClient.CreateSCClient()
	pvClient, _ := kubeClient.CreatePVClient()
	// snapGA, snapBeta, snErr := GetSnapshotClient(namespace, client)

	clients := &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		StatefulSetClient:      nil,
		SCClient:               scClient,
		PersistentVolumeClient: pvClient,
		// SnapClientGA:      snapGA,
		// SnapClientBeta:    snapBeta,
	}

	// Run the suite with error
	delFunc, err := vms.Run(ctx, storageClass, clients)
	assert.Error(t, err)
	assert.Nil(t, delFunc)
}

func TestVolumeMigrateSuite_GetObservers(t *testing.T) {
	vms := &VolumeMigrateSuite{}
	obsType := observer.Type("someType")
	observers := vms.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

func TestVolumeMigrateSuite_GetClients(t *testing.T) {
	client := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	// Simulate the existence of the storage class
	client.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "target-sc",
		},
	}, metav1.CreateOptions{})

	pvClient, _ := kubeClient.CreatePVClient()
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	scClient, _ := kubeClient.CreateSCClient()
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	stsClient, _ := kubeClient.CreateStatefulSetClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

	type args struct {
		namespace string
		client    *k8sclient.KubeClient
	}
	tests := []struct {
		name    string
		vms     *VolumeMigrateSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			vms:  &VolumeMigrateSuite{TargetSC: "target-sc"},
			args: args{
				namespace: "test-namespace",
				client:    &kubeClient,
			},
			want: &k8sclient.Clients{
				PersistentVolumeClient: pvClient,
				PVCClient:              pvcClient,
				PodClient:              podClient,
				SCClient:               scClient,
				StatefulSetClient:      stsClient,
				VaClient:               vaClient,
				MetricsClient:          metricsClient,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.vms.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("VolumeMigrateSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("VolumeMigrateSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeMigrateSuite_GetNamespace(t *testing.T) {
	vms := &VolumeMigrateSuite{}
	namespace := vms.GetNamespace()
	expectedNamespace := "migration-test"
	if namespace != expectedNamespace {
		t.Errorf("VolumeMigrateSuite.GetNamespace() = %v, want %v", namespace, expectedNamespace)
	}
}

func TestVolumeMigrateSuite_GetName(t *testing.T) {
	vms := &VolumeMigrateSuite{Description: "Test Suite"}
	name := vms.GetName()
	expectedName := "Test Suite"
	if name != expectedName {
		t.Errorf("VolumeMigrateSuite.GetName() = %v, want %v", name, expectedName)
	}

	vms.Description = ""
	name = vms.GetName()
	expectedName = "VolumeMigrationSuite"
	if name != expectedName {
		t.Errorf("VolumeMigrateSuite.GetName() = %v, want %v", name, expectedName)
	}
}

func TestVolumeMigrateSuite_Parameters(t *testing.T) {
	vms := &VolumeMigrateSuite{
		TargetSC:     "fast-storage",
		VolumeNumber: 5,
		PodNumber:    3,
	}
	params := vms.Parameters()
	expectedParams := "{Target storageclass: fast-storage, volumes: 5, pods: 3}"
	if params != expectedParams {
		t.Errorf("VolumeMigrateSuite.Parameters() = %v, want %v", params, expectedParams)
	}
}
