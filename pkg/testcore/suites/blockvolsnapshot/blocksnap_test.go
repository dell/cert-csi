package blockvolsnapshot

import (
	"context"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore/suites/mockutils"
	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"testing"
)

func TestBlockSnapSuite_Run(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a BlockSnapSuite instance
	bss := &BlockSnapSuite{
		SnapClass:   "testSnap",
		Description: "testDesc",
		AccessMode:  "test",
	}

	namespace := bss.GetNamespace()

	// Mock storageClass
	// Create a fake storage class with VolumeBindingMode set to WaitForFirstConsumer
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}

	clientSet := mockutils.NewFakeClientsetWithRestClient(storageClass)
	clientSet = mockutils.MockClientSetPodFunctions(clientSet).(*mockutils.FakeExtendedClientset)

	// Create a fake KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	// Create the necessary clients
	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
	podClient, _ := kubeClient.CreatePodClient(namespace)
	podClient.RemoteExecutor = &mockutils.FakeRemoteExecutor{}
	vaClient, _ := kubeClient.CreateVaClient(namespace)
	metricsClient, _ := kubeClient.CreateMetricsClient(namespace)
	// snapGA, snapBeta, _ := GetSnapshotClient(namespace, kubeClient)
	snapGA, _ := kubeClient.CreateSnapshotGAClient(namespace)
	snapGA.Interface = &mockutils.FakeVolumeSnapshotInterface{}
	snapBeta, _ := kubeClient.CreateSnapshotBetaClient(namespace)

	clients := &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      snapGA,
		SnapClientBeta:    snapBeta,
	}

	// Run the suite with connection refused error
	delFunc, err := bss.Run(ctx, "test-storage-class", clients)
	assert.Error(t, err)
	assert.Nil(t, delFunc)
}

func TestBlockSnapSuite_GetObservers(t *testing.T) {
	bss := &BlockSnapSuite{}
	obsType := observer.Type("someType")
	observers := bss.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

// TODO TestBlockSnapSuite_GetClients

func TestBlockSnapSuite_GetNamespace(t *testing.T) {
	bss := &BlockSnapSuite{}
	namespace := bss.GetNamespace()
	expectedNamespace := "block-volsnap-test"
	if namespace != expectedNamespace {
		t.Errorf("BlockSnapSuite.GetNamespace() = %v, want %v", namespace, expectedNamespace)
	}
}

func TestBlockSnapSuite_GetName(t *testing.T) {
	bss := &BlockSnapSuite{Description: "Test Suite"}
	name := bss.GetName()
	expectedName := "Test Suite"
	if name != expectedName {
		t.Errorf("BlockSnapSuite.GetName() = %v, want %v", name, expectedName)
	}

	bss.Description = ""
	name = bss.GetName()
	expectedName = "BlockSnapSuite"
	if name != expectedName {
		t.Errorf("BlockSnapSuite.GetName() = %v, want %v", name, expectedName)
	}
}

func TestBlockSnapSuite_Parameters(t *testing.T) {
	bss := &BlockSnapSuite{
		VolumeSize: "10Gi",
		AccessMode: "ReadWriteOnce",
	}
	params := bss.Parameters()
	expectedParams := "{size: 10Gi, accMode: ReadWriteOnce}"
	if params != expectedParams {
		t.Errorf("BlockSnapSuite.Parameters() = %v, want %v", params, expectedParams)
	}
}

// func TestFindDriverLogs(t *testing.T) {
// 	command := []string{"echo", "Hello, World!"}
// 	expectedOutput := "Hello, World!\n"

// 	got, err := FindDriverLogs(command)
// 	if err != nil {
// 		t.Errorf("FindDriverLogs() returned an error: %v", err)
// 	}
// 	if got != expectedOutput {
// 		t.Errorf("FindDriverLogs() = %v, want %v", got, expectedOutput)
// 	}
// }

// func TestBlockSnapSuite_GetClients(t *testing.T) {
// 	client := fake.NewSimpleClientset()

// 	kubeClient := k8sclient.KubeClient{
// 		ClientSet:   client,
// 		Config:      &rest.Config{},
// 		VersionInfo: nil,
// 		Minor:       19,
// 	}

// 	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
// 	podClient, _ := kubeClient.CreatePodClient("test-namespace")
// 	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
// 	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")
// 	snapGA, snapBeta, _ := GetSnapshotClient("test-namespace", &kubeClient)

// 	type args struct {
// 		namespace string
// 		client    *k8sclient.KubeClient
// 	}
// 	tests := []struct {
// 		name    string
// 		bss     *BlockSnapSuite
// 		args    args
// 		want    *k8sclient.Clients
// 		wantErr bool
// 	}{
// 		{
// 			name: "Testing GetClients",
// 			bss:  &BlockSnapSuite{SnapClass: "test-volsnap-class"},
// 			args: args{
// 				namespace: "test-namespace",
// 				client:    &kubeClient,
// 			},
// 			want: &k8sclient.Clients{
// 				PVCClient:         pvcClient,
// 				PodClient:         podClient,
// 				VaClient:          vaClient,
// 				StatefulSetClient: nil,
// 				MetricsClient:     metricsClient,
// 				SnapClientGA:      snapGA,
// 				SnapClientBeta:    snapBeta,
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.bss.GetClients(tt.args.namespace, tt.args.client)
// 			fmt.Println(got, err)

// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("BlockSnapSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
// 			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
// 				t.Errorf("BlockSnapSuite.GetClients() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

/*func TestSnapSuite_Run(t *testing.T) {

	// Mock storageClass
	// Create a fake storage class with VolumeBindingMode set to WaitForFirstConsumer
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}

	clientset := NewFakeClientsetWithRestClient(storageClass)

	// Create a pod to delete
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
	}

	// Add the pod to the fake client
	_, err := clientset.CoreV1().Pods("test-namespace").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Error creating pod: %v\n", err)
		return
	}

	clientset.Fake.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
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

	// Create a fake k8sclient.KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientset,
		Config:    &rest.Config{},
	}

	// Create PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Create Pod client
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	podClient.RemoteExecutor = &FakeRemoteExecutor{}

	// Create PV client
	pvClient, _ := kubeClient.CreatePVClient()
	snapGA, _ := kubeClient.CreateSnapshotGAClient("test-namespace")
	snapGA.Interface = &FakeVolumeSnapshotInterface{}
	snapBeta, _ := kubeClient.CreateSnapshotBetaClient("test-namespace")

	// Update the k8sclient.Clients instance with the fake clients
	clients := &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		PersistentVolumeClient: pvClient,
		KubeClient:             kubeClient,
		SnapClientGA:           snapGA,
		SnapClientBeta:         snapBeta,
	}

	// Delete the pod
	err = clientset.CoreV1().Pods("test-namespace").Delete(context.TODO(), "test-pod", metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("Error deleting pod: %v\n", err)
		return
	}

	tests := []struct {
		name           string
		snapSuite      *SnapSuite
		storageClass   string
		clients        *k8sclient.Clients
		wantError      bool
		wantDeleteFunc bool
	}{
		/*{
			name: "Testing Run with default parameters",
			snapSuite: &SnapSuite{
				SnapAmount:         0,
				SnapClass:          "",
				VolumeSize:         "",
				Description:        "",
				CustomSnapName:     "",
				AccessModeOriginal: "",
				AccessModeRestored: "",
				Image:              "",
			},
			storageClass:   "test-storage-class",
			clients:        clients,
			wantError:      false,
			wantDeleteFunc: false,
		},
		{
			name: "Testing Run with custom parameters",
			snapSuite: &SnapSuite{
				SnapAmount:         5,
				SnapClass:          "test-volsnap-class",
				VolumeSize:         "10Gi",
				Description:        "test-description",
				CustomSnapName:     "test-volsnap-name",
				AccessModeOriginal: "ReadWriteOnce",
				AccessModeRestored: "ReadWriteMany",
				Image:              "quay.io/centos/centos:latest",
			},
			storageClass:   "test-storage-class",
			clients:        clients,
			wantError:      false,
			wantDeleteFunc: false,
		},
		{
			name: "Testing Run with error",
			snapSuite: &SnapSuite{
				SnapAmount:         5,
				SnapClass:          "test-volsnap-class",
				VolumeSize:         "10Gi",
				Description:        "test-description",
				CustomSnapName:     "test-volsnap-name",
				AccessModeOriginal: "ReadWriteOnce",
				AccessModeRestored: "ReadWriteMany",
				Image:              "quay.io/centos/centos:latest",
			},
			storageClass:   "test-storage-class",
			clients:        &k8sclient.Clients{},
			wantError:      true,
			wantDeleteFunc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delFunc, err := tt.snapSuite.Run(context.Background(), tt.storageClass, tt.clients)
			if tt.wantError {
				if err == nil {
					t.Errorf("SnapSuite.Run() expected error, but got nil")
				}
			} else if err != nil {
				t.Errorf("SnapSuite.Run() returned error: %v", err)
			}

			if tt.wantDeleteFunc {
				if delFunc == nil {
					t.Errorf("SnapSuite.Run() expected delete function, but got nil")
				}
			} else if delFunc != nil {
				t.Errorf("SnapSuite.Run() returned unexpected delete function")
			}
		})
	}
}*/
