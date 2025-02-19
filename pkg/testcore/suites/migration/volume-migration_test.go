package migration

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/clientcmd/api"
	"net/http"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/statefulset"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestVolumeMigrateSuite_Run(t *testing.T) {
	// Create the necessary clients
	namespace := "fake"
	client := fake.NewSimpleClientset()
	kubeClient := k8sclient.KubeClient{
		ClientSet: client,
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

	storageClass := "source-storage-class"
	// Simulate the existence of the storage class
	_, err = client.StorageV1().StorageClasses().Create(context.TODO(), &storagev1.StorageClass{
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

	t.Run("Test with Invalid StorageClass", func(t *testing.T) {
		delFunc, err := suite.Run(ctx, "invalid-storage-class", clients)
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

// FakeRoundTripper is a custom RoundTripper that returns predefined HTTP responses.
type FakeRoundTripper struct {
	Response *http.Response
	Err      error
}

// RoundTrip executes a single HTTP transaction and returns a predefined response.
func (frt *FakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return frt.Response, frt.Err
}
func TestVolumeMigrateSuite_validateSTS(t *testing.T) {
	client := fake.NewSimpleClientset()
	kubeClient := k8sclient.KubeClient{
		ClientSet: client,
		Config: &rest.Config{
			Host:                "",
			APIPath:             "",
			ContentConfig:       rest.ContentConfig{},
			Username:            "",
			Password:            "",
			BearerToken:         "",
			BearerTokenFile:     "",
			Impersonate:         rest.ImpersonationConfig{},
			AuthProvider:        nil,
			AuthConfigPersister: nil,
			ExecProvider: &api.ExecConfig{
				Command:                 "",
				Args:                    nil,
				Env:                     nil,
				APIVersion:              "",
				InstallHint:             "",
				ProvideClusterInfo:      false,
				InteractiveMode:         "",
				StdinUnavailable:        false,
				StdinUnavailableMessage: "",
			},
			TLSClientConfig:    rest.TLSClientConfig{},
			UserAgent:          "",
			DisableCompression: false,
			Transport:          &FakeRoundTripper{},
			WrapTransport:      nil,
			QPS:                0,
			Burst:              0,
			RateLimiter:        nil,
			WarningHandler:     nil,
			Timeout:            0,
			Dial:               nil,
			Proxy:              nil,
		},
	}
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
	pvcClient, err := kubeClient.CreatePVCClient("test-namespace")
	assert.NoError(t, err)

	stsConf := &statefulset.Config{}
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	podClient.Config = &rest.Config{}
	podClient.RemoteExecutor = &pod.DefaultRemoteExecutor{}
	pvClient, _ := kubeClient.CreatePVClient()

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
			Volumes: []corev1.Volume{
				{
					Name: "test-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						},
					},
				},
			},
		},
	}
	// Create a PVC object
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "test-namespace",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "fake",
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
		},
	}

	// Create the namespace
	_ = pvcClient.Create(context.TODO(), pvc)
	_ = podClient.Create(context.TODO(), &podObj)

	// Create a fake PV object
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"migration.storage.dell.com/migrate-to": "target-storage-class",
			},
			Name: "fake",
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
		},
	}

	// Create the fake PV
	_, err = client.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	assert.NoError(t, err)

	pv.Name = "fake-to-target-sc"
	// Create target fake PV
	_, err = client.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Call the validateSTS function
	err = vms.validateSTS(logger, pvcClient, ctx, podObj.Spec.Volumes[0], nil, &[]string{}, pvClient, stsConf, podClient, podObj)

	// Assert the expected error
	assert.NoError(t, err)

}
