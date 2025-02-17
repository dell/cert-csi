package migration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pv"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/statefulset"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// TODO TestGetSnapshotClient
//func TestVolumeMigrateSuite_Run(t *testing.T) {
//	// Create a context
//	ctx := context.Background()
//
//	// Mock storageClass
//	storageClass := "test-storage-class"
//
//	client := fake.NewSimpleClientset()
//	kubeClient := k8sclient.KubeClient{
//		ClientSet:   client,
//		Config:      &rest.Config{},
//		VersionInfo: nil,
//	}
//
//	// Simulate the existence of the storage class
//	_, err := client.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
//		ObjectMeta: metav1.ObjectMeta{
//			Name: "test-storage-class",
//		},
//	}, metav1.CreateOptions{})
//
//	assert.NoError(t, err)
//
//	// Create a VolumeMigrateSuite instance
//	vms := &VolumeMigrateSuite{
//		TargetSC:    storageClass,
//		Description: "test-desc",
//		Flag:        true,
//	}
//
//	namespace := vms.GetNamespace()
//
//	// Create the necessary clients
//	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
//	podClient, _ := kubeClient.CreatePodClient(namespace)
//	scClient, _ := kubeClient.CreateSCClient()
//	pvClient, _ := kubeClient.CreatePVClient()
//	stsClient, _ := kubeClient.CreateStatefulSetClient(namespace)
//	// snapGA, snapBeta, snErr := GetSnapshotClient(namespace, client)
//
//	clients := &k8sclient.Clients{
//		PVCClient:              pvcClient,
//		PodClient:              podClient,
//		StatefulSetClient:      stsClient,
//		SCClient:               scClient,
//		PersistentVolumeClient: pvClient,
//		// SnapClientGA:      snapGA,
//		// SnapClientBeta:    snapBeta,
//	}
//
//	// Run the suite with error
//	delFunc, err := vms.Run(ctx, storageClass, clients)
//	assert.Error(t, err)
//	assert.Nil(t, delFunc)
//}
//
//func TestVolumeMigrateSuite_GetObservers(t *testing.T) {
//	vms := &VolumeMigrateSuite{}
//	obsType := observer.Type("someType")
//	observers := vms.GetObservers(obsType)
//	assert.NotNil(t, observers)
//}
//
//func TestVolumeMigrateSuite_GetClients(t *testing.T) {
//	client := fake.NewSimpleClientset()
//
//	kubeClient := k8sclient.KubeClient{
//		ClientSet: client,
//		Config:    &rest.Config{},
//	}
//
//	// Simulate the existence of the storage class
//	_, err := client.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
//		ObjectMeta: metav1.ObjectMeta{
//			Name: "target-sc",
//		},
//	}, metav1.CreateOptions{})
//
//	assert.NoError(t, err)
//
//	pvClient, _ := kubeClient.CreatePVClient()
//	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
//	scClient, _ := kubeClient.CreateSCClient()
//	podClient, _ := kubeClient.CreatePodClient("test-namespace")
//	stsClient, _ := kubeClient.CreateStatefulSetClient("test-namespace")
//	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
//	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")
//
//	type args struct {
//		namespace string
//		client    *k8sclient.KubeClient
//	}
//	tests := []struct {
//		name    string
//		vms     *VolumeMigrateSuite
//		args    args
//		want    *k8sclient.Clients
//		wantErr bool
//	}{
//		{
//			name: "Testing GetClients",
//			vms:  &VolumeMigrateSuite{TargetSC: "target-sc"},
//			args: args{
//				namespace: "test-namespace",
//				client:    &kubeClient,
//			},
//			want: &k8sclient.Clients{
//				PersistentVolumeClient: pvClient,
//				PVCClient:              pvcClient,
//				PodClient:              podClient,
//				SCClient:               scClient,
//				StatefulSetClient:      stsClient,
//				VaClient:               vaClient,
//				MetricsClient:          metricsClient,
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := tt.vms.GetClients(tt.args.namespace, tt.args.client)
//			fmt.Println(got, err)
//
//			if (err != nil) != tt.wantErr {
//				t.Errorf("VolumeMigrateSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
//			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
//				t.Errorf("VolumeMigrateSuite.GetClients() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestVolumeMigrateSuite_GetNamespace(t *testing.T) {
//	vms := &VolumeMigrateSuite{}
//	namespace := vms.GetNamespace()
//	expectedNamespace := "migration-test"
//	if namespace != expectedNamespace {
//		t.Errorf("VolumeMigrateSuite.GetNamespace() = %v, want %v", namespace, expectedNamespace)
//	}
//}
//
//func TestVolumeMigrateSuite_GetName(t *testing.T) {
//	vms := &VolumeMigrateSuite{Description: "Test Suite"}
//	name := vms.GetName()
//	expectedName := "Test Suite"
//	if name != expectedName {
//		t.Errorf("VolumeMigrateSuite.GetName() = %v, want %v", name, expectedName)
//	}
//
//	vms.Description = ""
//	name = vms.GetName()
//	expectedName = "VolumeMigrationSuite"
//	if name != expectedName {
//		t.Errorf("VolumeMigrateSuite.GetName() = %v, want %v", name, expectedName)
//	}
//}
//
//func TestVolumeMigrateSuite_Parameters(t *testing.T) {
//	vms := &VolumeMigrateSuite{
//		TargetSC:     "fast-storage",
//		VolumeNumber: 5,
//		PodNumber:    3,
//	}
//	params := vms.Parameters()
//	expectedParams := "{Target storageclass: fast-storage, volumes: 5, pods: 3}"
//	if params != expectedParams {
//		t.Errorf("VolumeMigrateSuite.Parameters() = %v, want %v", params, expectedParams)
//	}
//}

//import (
//"context"
//"fmt"
//"testing"
//
//"github.com/dell/cert-csi/pkg/k8sclient"
//"github.com/dell/cert-csi/pkg/testcore/suites/migration"
//"github.com/stretchr/testify/assert"
//)

func TestVolumeMigrateSuite_Run(t *testing.T) {
	// Create the necessary clients
	namespace := "fake"
	client := fake.NewSimpleClientset()
	kubeClient := k8sclient.KubeClient{
		ClientSet: client,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
	podClient, _ := kubeClient.CreatePodClient(namespace)
	scClient, _ := kubeClient.CreateSCClient()
	pvClient, _ := kubeClient.CreatePVClient()
	stsClient, _ := kubeClient.CreateStatefulSetClient(namespace)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	clients := &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		StatefulSetClient:      stsClient,
		SCClient:               scClient,
		PersistentVolumeClient: pvClient,
		// SnapClientGA:      snapGA,
		// SnapClientBeta:    snapBeta,
	}

	storageClass := "source-storage-class"
	// Simulate the existence of the storage class
	_, err := client.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: storageClass,
		},
	}, metav1.CreateOptions{})

	assert.NoError(t, err)

	_, err = client.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "target-storage-class",
		},
	}, metav1.CreateOptions{})

	assert.NoError(t, err)

	suite := &VolumeMigrateSuite{
		TargetSC:     "target-storage-class",
		Description:  "Test Volume Migration",
		VolumeNumber: 2,
		PodNumber:    3,
		Image:        "quay.io/centos/centos:latest",
	}

	t.Run("Test with Default Values", func(t *testing.T) {
		suite.VolumeNumber = 0
		suite.PodNumber = 0
		suite.Image = ""
		delFunc, err := suite.Run(ctx, "source-storage-class", clients)
		assert.Error(t, err)
		assert.Nil(t, delFunc)
	})

	//t.Run("Test with Custom Values", func(t *testing.T) {
	//	suite.VolumeNumber = 2
	//	suite.PodNumber = 3
	//	suite.Image = "custom-image"
	//	delFunc, err := suite.Run(ctx, "source-storage-class", clients)
	//	assert.NoError(t, err)
	//	assert.NotNil(t, delFunc)
	//})

	t.Run("Test with Invalid StorageClass", func(t *testing.T) {
		delFunc, err := suite.Run(ctx, "invalid-storage-class", clients)
		assert.Error(t, err)
		assert.Nil(t, delFunc)
	})

	//t.Run("Test StatefulSet Creation Failure", func(t *testing.T) {
	//	// Simulate StatefulSet creation failure
	//	delFunc, err := suite.Run(ctx, "source-storage-class", clients)
	//	assert.Error(t, err)
	//	assert.Nil(t, delFunc)
	//})

	//t.Run("Test PVC and PV Retrieval", func(t *testing.T) {
	//	delFunc, err := suite.Run(ctx, "source-storage-class", clients)
	//	assert.NoError(t, err)
	//	assert.NotNil(t, delFunc)
	//})

	//t.Run("Test Data Writing and Hash Calculation", func(t *testing.T) {
	//	delFunc, err := suite.Run(ctx, "source-storage-class", clients)
	//	assert.NoError(t, err)
	//	assert.NotNil(t, delFunc)
	//})

	//t.Run("Test PV Migration Annotation", func(t *testing.T) {
	//	delFunc, err := suite.Run(ctx, "source-storage-class", clients)
	//	assert.NoError(t, err)
	//	assert.NotNil(t, delFunc)
	//})
	//
	//t.Run("Test PV Migration Completion", func(t *testing.T) {
	//	delFunc, err := suite.Run(ctx, "source-storage-class", clients)
	//	assert.NoError(t, err)
	//	assert.NotNil(t, delFunc)
	//})
	//
	//t.Run("Test StatefulSet Deletion and Recreation", func(t *testing.T) {
	//	delFunc, err := suite.Run(ctx, "source-storage-class", clients)
	//	assert.NoError(t, err)
	//	assert.NotNil(t, delFunc)
	//})
	//
	//t.Run("Test Hash Verification", func(t *testing.T) {
	//	delFunc, err := suite.Run(ctx, "source-storage-class", clients)
	//	assert.NoError(t, err)
	//	assert.NotNil(t, delFunc)
	//})
}

func TestVolumeMigrateSuite_GetClients(t *testing.T) {
	//namespace := "fake"
	client := fake.NewSimpleClientset()
	kubeClient := k8sclient.KubeClient{
		ClientSet: client,
		Config:    &rest.Config{},
	}
	suite := &VolumeMigrateSuite{
		TargetSC: "target-storage-class",
	}

	validNs := "valid-namespace"
	// Define the namespace object
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: validNs,
		},
	}

	// Create the namespace
	_, err := client.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})

	// Simulate the existence of the namespace class
	_, err = client.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: suite.TargetSC,
		},
	}, metav1.CreateOptions{})

	assert.NoError(t, err)

	t.Run("Test with Valid Namespace", func(t *testing.T) {
		clients, err := suite.GetClients(validNs, &kubeClient)
		assert.NoError(t, err)
		assert.NotNil(t, clients)
	})

	t.Run("Test with Invalid StorageClass", func(t *testing.T) {
		clients, err := suite.GetClients("invalid-namespace", &kubeClient)
		assert.NoError(t, err)
		assert.NotNil(t, clients)
	})
}

func TestVolumeMigrateSuite_GetObservers(t *testing.T) {
	suite := &VolumeMigrateSuite{}

	t.Run("Test with Valid Observer Type", func(t *testing.T) {
		observers := suite.GetObservers("valid-observer-type")
		assert.NotNil(t, observers)
	})
}

func TestVolumeMigrateSuite_GetNamespace(t *testing.T) {
	suite := &VolumeMigrateSuite{}

	t.Run("Test Namespace Retrieval", func(t *testing.T) {
		namespace := suite.GetNamespace()
		assert.Equal(t, "migration-test", namespace)
	})
}

func TestVolumeMigrateSuite_GetName(t *testing.T) {
	suite := &VolumeMigrateSuite{
		Description: "Test Suite",
	}

	t.Run("Test Name Retrieval with Description", func(t *testing.T) {
		name := suite.GetName()
		assert.Equal(t, "Test Suite", name)
	})

	t.Run("Test Name Retrieval without Description", func(t *testing.T) {
		suite.Description = ""
		name := suite.GetName()
		assert.Equal(t, "VolumeMigrationSuite", name)
	})
}

func TestVolumeMigrateSuite_Parameters(t *testing.T) {
	suite := &VolumeMigrateSuite{
		TargetSC:     "target-storage-class",
		VolumeNumber: 2,
		PodNumber:    3,
	}

	t.Run("Test Parameters Formatting", func(t *testing.T) {
		params := suite.Parameters()
		expected := fmt.Sprintf("{Target storageclass: %s, volumes: %d, pods: %d}", suite.TargetSC, suite.VolumeNumber, suite.PodNumber)
		assert.Equal(t, expected, params)
	})
}

func TestVolumeMigrateSuite_validateSTS(t *testing.T) {
	// Create a new VolumeMigrateSuite instance
	vms := &VolumeMigrateSuite{
		TargetSC: "target-sc",
		Flag:     true,
		Image:    "quay.io/centos/centos:latest",
	}

	// Create a context.Context
	ctx := context.Background()

	// Create mock objects
	logger := utils.GetLoggerFromContext(ctx)
	pvcClient := &pvc.Client{}
	stsConf := &statefulset.Config{}
	podClient := &pod.Client{}
	pvClient := &pv.Client{}

	// Create a mock error
	// err := errors.New("mock error")

	// Create a mock v1.Pod
	// Create a fake v1.Pod object
	podObj := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-podObj",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}

	// Create a fake volume
	volume := corev1.Volume{
		Name:         "test-volume",
		VolumeSource: corev1.VolumeSource{
			// Add your volume source configuration here
		},
	}

	// Call the validateSTS function
	err := vms.validateSTS(logger, pvcClient, ctx, volume, nil, []string{"pv1"}, pvClient, stsConf, podClient, podObj)

	// Assert the expected error
	assert.NoError(t, err)

	// TODO: Add more test cases with different inputs and expected outputs
}
