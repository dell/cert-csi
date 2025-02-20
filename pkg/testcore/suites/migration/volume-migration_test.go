package migration

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/testcore/suites/mockutils"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/statefulset"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestVolumeMigrateSuite_Run(t *testing.T) {
	// Create the necessary clients
	namespace := "fake"
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

	// Set up a reactor to simulate Pods becoming Ready
	clientSet.Fake.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		podItemObj := createAction.GetObject().(*v1.Pod)
		// Set podObj phase to Running
		podItemObj.Status.Phase = v1.PodRunning
		// Simulate the Ready condition
		podItemObj.Status.Conditions = append(podItemObj.Status.Conditions, v1.PodCondition{
			Type:   v1.PodReady,
			Status: v1.ConditionTrue,
		})
		podItemObj.Labels = map[string]string{"app": "unified-test"}
		return false, nil, nil // Allow normal processing to continue
	})

	clientSet.Fake.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteAction := action.(k8stesting.DeleteAction)
		podName := deleteAction.GetName()
		logrus.Infof("Deleting pod %s", podName)
		return true, nil, nil // Allow normal processing to continue
	})

	clientSet.Fake.PrependReactor("create", "statefulsets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		statefulSet := createAction.GetObject().(*appsv1.StatefulSet)
		statefulSet.Status.Replicas = *statefulSet.Spec.Replicas
		statefulSet.Status.ReadyReplicas = 3
		return true, statefulSet, nil
	})
	clientSet.Fake.PrependReactor("delete", "statefulsets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		deleteAction := action.(k8stesting.DeleteAction)
		name := deleteAction.GetName()
		logrus.Infof("Deleting statefulset %s", name)
		return true, nil, nil
	})
	// Create a mock Clients instance
	mockClients := &mockutils.MockClients{}

	// Set up the mock behavior for the CreatePodClient method
	mockClients.On("CreatePodClient", namespace).Return(
		&pod.Client{
			Interface: clientSet.CoreV1().Pods(namespace),
		},
		nil,
	)

	// Set up the mock behavior for the CreatePodClient method
	mockClients.On("CreateStatefulSetClient", namespace).Return(
		&statefulset.Client{
			Interface: clientSet.AppsV1().StatefulSets(namespace),
		},
		nil,
	)

	kubeClient := k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, err := kubeClient.CreatePVCClient(namespace)
	podClient, err := kubeClient.CreatePodClient(namespace)
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
	}

	storageClass2 := "source-storage-class"
	// Simulate the existence of the storage class
	_, err = clientSet.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: storageClass2,
		},
	}, metav1.CreateOptions{})

	assert.NoError(t, err)

	_, err = clientSet.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "target-storage-class",
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	// create a pod in fake namespace
	podObj := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "nginx2",
					Image: "nginx:latest",
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 9090,
						},
					},
				},
			},
		},
	}
	_, err = clientSet.CoreV1().Pods(namespace).Create(context.Background(), podObj, metav1.CreateOptions{})
	assert.NoError(t, err)

	var replicas int32 = 1
	// Create a fake statefulset
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			//Name:         "sts-volume-migrate-test",
			Namespace: namespace,
			//GenerateName: "sts-volume-migrate-test",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pvc",
					},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes: []v1.PersistentVolumeAccessMode{
							v1.ReadWriteOnce,
						},
					},
				},
			},
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "test-container",
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "test-volume",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: "test-pvc",
								},
							},
						},
					},
				},
			},
		},
	}

	_ = stsClient.Create(context.TODO(), sts)

	suite := &VolumeMigrateSuite{
		TargetSC:     "target-storage-class",
		Description:  "Test Volume Migration",
		VolumeNumber: 2,
		PodNumber:    1,
		Image:        "quay.io/centos/centos:latest",
	}
	t.Run("Test with Custom Values", func(t *testing.T) {
		delFunc, err := suite.Run(context.TODO(), "source-storage-class", clients)
		assert.Error(t, err)
		assert.NotNil(t, delFunc)
	})

	t.Run("Test with Default Values", func(t *testing.T) {
		suite.VolumeNumber = 0
		suite.PodNumber = 0
		suite.Image = ""
		delFunc, err := suite.Run(context.TODO(), "invalid-storage-class", clients)
		assert.Error(t, err)
		assert.Nil(t, delFunc)
	})

	t.Run("Test with Invalid source StorageClass", func(t *testing.T) {
		delFunc, err := suite.Run(ctx, "invalid-storage-class", clients)
		assert.Error(t, err)
		assert.Nil(t, delFunc)
	})
	t.Run("Test with Invalid target StorageClass", func(t *testing.T) {
		kubeClient := k8sclient.KubeClient{
			ClientSet: fake.NewSimpleClientset(),
			Config:    &rest.Config{},
		}
		scClient2, _ := kubeClient.CreateSCClient()
		clients2 := &k8sclient.Clients{
			PVCClient:              pvcClient,
			PodClient:              podClient,
			StatefulSetClient:      stsClient,
			SCClient:               scClient2,
			PersistentVolumeClient: pvClient,
		}
		// Create a fake storage class
		sc2 := &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "src",
			},
		}
		err = clients2.SCClient.Create(context.TODO(), sc2)
		assert.NoError(t, err)
		delFunc, err := suite.Run(ctx, "src", clients2)
		assert.Error(t, err)
		assert.Nil(t, delFunc)
	})
}

func TestVolumeMigrateSuite_GetClients(t *testing.T) {
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
	namespace := &v1.Namespace{
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

	t.Run("Test with Invalid namespace", func(t *testing.T) {
		clients, err := suite.GetClients("invalid-namespace", &kubeClient)
		assert.NoError(t, err)
		assert.NotNil(t, clients)
	})
	t.Run("Test for invalid sc", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		kubeClient := k8sclient.KubeClient{
			ClientSet: client,
			Config:    &rest.Config{},
		}
		clients, err := suite.GetClients("invalid-namespace", &kubeClient)
		assert.Error(t, err)
		assert.Nil(t, clients)
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

	clientSet.Fake.PrependReactor("create", "statefulsets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		statefulSet := createAction.GetObject().(*appsv1.StatefulSet)
		statefulSet.Status.Replicas = *statefulSet.Spec.Replicas

		return true, statefulSet, nil
	})
	// Create a mock Clients instance
	mockClients := &mockutils.MockClients{}

	// Set up the mock behavior for the CreatePodClient method
	mockClients.On("CreatePodClient", "test-namespace").Return(
		&pod.Client{
			Interface: clientSet.CoreV1().Pods("test-namespace"),
		},
		nil,
	)

	kubeClient := k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	// Create a new VolumeMigrateSuite instance
	vms := &VolumeMigrateSuite{
		TargetSC: "target-sc",
		Flag:     false,
		Image:    "quay.io/centos/centos:latest",
	}

	// Create a context.Context
	ctx := context.Background()

	// Create mock objects
	logger := utils.GetLoggerFromContext(ctx)
	pvcClient, err := kubeClient.CreatePVCClient("test-namespace")
	assert.NoError(t, err)

	stsConf := &statefulset.Config{}
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	podClient.RemoteExecutor = &mockutils.FakeRemoteExecutor{}
	podClient.RemoteExecutor = &mockutils.FakeHashRemoteExecutor{}
	pvClient, _ := kubeClient.CreatePVClient()

	// Create a fake v1.Pod object
	podObj := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-podObj",
			Namespace: "test-namespace",
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "test-volume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						},
					},
				},
			},
		},
	}
	// Create a PVC object
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "test-namespace",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName: "fake",
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimBound,
		},
	}

	// Create the namespace
	_ = pvcClient.Create(context.TODO(), pvc)
	_ = podClient.Create(context.TODO(), &podObj)

	// Create a fake PV object
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"migration.storage.dell.com/migrate-to": "target-storage-class",
			},
			Name: "fake",
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("1Gi"),
			},
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
		},
	}

	// Create the fake PV
	_, err = clientSet.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	assert.NoError(t, err)

	pv.Name = "fake-to-target-sc"
	// Create target fake PV
	_, err = clientSet.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Call the validateSTS function
	err = vms.validateSTS(logger, pvcClient, ctx, podObj.Spec.Volumes[0], nil, &[]string{}, pvClient, stsConf, podClient, podObj)
	// Assert the expected error
	assert.NoError(t, err)

	//when flag is set to true
	vms.Flag = true
	// Call the validateSTS function
	err = vms.validateSTS(logger, pvcClient, ctx, podObj.Spec.Volumes[0], nil, &[]string{}, pvClient, stsConf, podClient, podObj)
	// Assert the expected error
	assert.NoError(t, err)
}

func TestDeleteFunction(t *testing.T) {
	client := mockutils.NewFakeClientsetWithRestClient()
	kubeClient := k8sclient.KubeClient{
		ClientSet: client,
		Config:    &rest.Config{},
	}
	pvClient, _ := kubeClient.CreatePVClient()
	ctx := context.Background()
	log := logrus.Entry{Logger: logrus.New()}

	// Create some fake PVs
	pv1 := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake",
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("1Gi"),
			},
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
		},
	}

	// Create the fake PV
	_, err := client.CoreV1().PersistentVolumes().Create(context.TODO(), pv1, metav1.CreateOptions{})
	assert.NoError(t, err)

	t.Run("Test delete function with valid PV names", func(t *testing.T) {
		// Call the delete function with a list of PV names
		err = deleteFunction(&log, pvClient, ctx, []string{"fake"})
		assert.NoError(t, err)
	})
	t.Run("Test delete function with invalid PVs", func(t *testing.T) {
		// Call the delete function with a list of PV names
		err = deleteFunction(&log, pvClient, ctx, []string{})
		assert.NoError(t, err)
	})
}
