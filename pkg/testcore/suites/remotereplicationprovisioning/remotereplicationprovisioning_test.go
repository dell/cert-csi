package remotereplicationprovisioning

import (
	"context"
	"io"
	"net/url"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/remotecommand"
)

type FakeRemoteExecutor struct{}

func (f *FakeRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	return nil
}

func TestRemoteReplicationProvisioningSuite_GetObservers(t *testing.T) {
	rrps := &RemoteReplicationProvisioningSuite{}
	obsType := observer.Type("someType")
	observers := rrps.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

// TestRemoteReplicationProvisioningSuite_GetClients
func TestRemoteReplicationProvisioningSuite_GetClients(t *testing.T) {
	// Create a fake clientset
	client := fake.NewSimpleClientset()

	// Create a fake KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	type args struct {
		namespace string
		client    *k8sclient.KubeClient
	}
	tests := []struct {
		name    string
		rrps    *RemoteReplicationProvisioningSuite
		args    args
		wantErr bool
	}{
		{
			name: "Testing GetClients expecting error",
			rrps: &RemoteReplicationProvisioningSuite{},
			args: args{
				namespace: "test-namespace",
				client:    kubeClient,
			},
			wantErr: true, // We expect an error due to RG client creation failure
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.rrps.GetClients(tt.args.namespace, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoteReplicationProvisioningSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoteReplicationProvisioningSuite_GetNamespace(t *testing.T) {
	rrps := &RemoteReplicationProvisioningSuite{}
	namespace := rrps.GetNamespace()
	assert.Equal(t, "repl-prov-test", namespace)
}

func TestRemoteReplicationProvisioningSuite_GetName(t *testing.T) {
	rrps := &RemoteReplicationProvisioningSuite{}
	name := rrps.GetName()
	assert.Equal(t, "RemoteReplicationProvisioningSuite", name)

	rrps.Description = "CustomName"
	name = rrps.GetName()
	assert.Equal(t, "CustomName", name)
}

func TestRemoteReplicationProvisioningSuite_Parameters(t *testing.T) {
	rrps := &RemoteReplicationProvisioningSuite{
		VolumeNumber:     5,
		VolumeSize:       "10Gi",
		RemoteConfigPath: "/path/to/config",
	}
	params := rrps.Parameters()
	expected := "{volumes: 5, volumeSize: 10Gi, remoteConfig: /path/to/config}"
	assert.Equal(t, expected, params)
}

func TestRemoteReplicationProvisioningSuite_Run_Empty(t *testing.T) {
	ctx := context.Background()

	rrps := &RemoteReplicationProvisioningSuite{}

	// Create a fake storage class with VolumeBindingMode set to WaitForFirstConsumer
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
		Parameters: map[string]string{
			"replication.storage.dell.com/isReplicationEnabled": "true",
		},
	}

	// clientset := fake.NewSimpleClientset(storageClass)
	clientset := common.NewFakeClientsetWithRestClient(storageClass)

	// Intercept the creation of pod & assign conditions
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
		return false, nil, nil
	})

	// Create a fake k8s clientset with the storage class
	clientset.Fake.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		podName := getAction.GetName()
		// Create a pod object with the expected name and Ready status
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "test-namespace",
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		}
		return true, pod, nil
	})

	kubeClient := &k8sclient.KubeClient{
		ClientSet:   clientset,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	pvcClient, err := kubeClient.CreatePVCClient("test-namespace")
	if err != nil {
		t.Fatalf("Failed to get PVC Client: %v", err)
	}

	// Create the PVC status & set to Bound
	clientset.Fake.PrependReactor("create", "persistentvolumeclaims", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		createdPVC := createAction.GetObject().(*v1.PersistentVolumeClaim)
		createdPVC.Status.Phase = v1.ClaimBound
		return true, createdPVC, nil
	})

	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	scClient, _ := kubeClient.CreateSCClient()
	pvClient, _ := kubeClient.CreatePVClient()
	remoteKubeClient, err := k8sclient.NewRemoteKubeClient(kubeClient.Config, 10)
	if err != nil {
		t.Errorf("Error creating remote kube client: %v", err)
	}

	rgClient, err := remoteKubeClient.CreateRGClient()
	if err != nil {
		t.Errorf("Error creating replication group client: %v", err)
	}

	// Update the k8sclient.Clients instance with the fake clients
	k8sClients := &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		SCClient:               scClient,
		PersistentVolumeClient: pvClient,
		KubeClient:             kubeClient,
		RgClient:               rgClient,
	}

	// Run the RemoteReplicationProvisioningSuite
	gotRunFunc, err := rrps.Run(ctx, "test-storage-class", k8sClients)

	// Check if there was an error
	if gotRunFunc != nil {
		if err != nil {
			t.Errorf("Error running RemoteReplicationProvisioningSuite.Run(): %v", err)
		}
	}
}

func TestRemoteReplicationProvisioningSuite_Run(t *testing.T) {
	ctx := context.Background()

	rrps := &RemoteReplicationProvisioningSuite{
		VolumeNumber:     1,
		VolumeSize:       "10Gi",
		RemoteConfigPath: "./test-config/mock-kube-config.yaml",
		Description:      "RemoteReplicationProvisioningSuite",
		VolAccessMode:    "ReadWriteMany",
		NoFailover:       false,
		Image:            "quay.io/centos/centos:latest",
	}

	// Create a fake storage class with VolumeBindingMode set to WaitForFirstConsumer
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
		Parameters: map[string]string{
			"replication.storage.dell.com/isReplicationEnabled": "true",
		},
	}

	// clientset := fake.NewSimpleClientset(storageClass)
	clientset := common.NewFakeClientsetWithRestClient(storageClass)

	// Intercept the creation of pod & assign conditions
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
		return false, nil, nil
	})

	// Create a fake k8s clientset with the storage class
	clientset.Fake.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		podName := getAction.GetName()
		// Create a pod object with the expected name and Ready status
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "test-namespace",
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		}
		return true, pod, nil
	})

	kubeClient := &k8sclient.KubeClient{
		ClientSet:   clientset,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	pvcClient, err := kubeClient.CreatePVCClient("test-namespace")
	if err != nil {
		t.Fatalf("Failed to get PVC Client: %v", err)
	}

	// Create the PVC status & set to Bound
	clientset.Fake.PrependReactor("create", "persistentvolumeclaims", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		createdPVC := createAction.GetObject().(*v1.PersistentVolumeClaim)
		createdPVC.Status.Phase = v1.ClaimBound
		return true, createdPVC, nil
	})

	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	podClient.RemoteExecutor = &FakeRemoteExecutor{}
	scClient, _ := kubeClient.CreateSCClient()
	pvClient, _ := kubeClient.CreatePVClient()
	remoteKubeClient, err := k8sclient.NewRemoteKubeClient(kubeClient.Config, 10)
	if err != nil {
		t.Errorf("Error creating remote kube client: %v", err)
	}

	rgClient, err := remoteKubeClient.CreateRGClient()
	if err != nil {
		t.Errorf("Error creating replication group client: %v", err)
	}

	// Update the k8sclient.Clients instance with the fake clients
	k8sClients := &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		SCClient:               scClient,
		PersistentVolumeClient: pvClient,
		KubeClient:             kubeClient,
		RgClient:               rgClient,
	}

	/* 	// Run the RemoteReplicationProvisioningSuite
	   	delFunc, err := rrps.Run(ctx, "test-storage-class", k8sClients)

	   	// Check if there was an error
	   	if delFunc != nil {
	   		if err != nil {
	   			t.Errorf("Error running RemoteReplicationProvisioningSuite.Run(): %v", err)
	   		}
	   	} */

	delFunc, err := rrps.Run(ctx, "test-storage-class", k8sClients)
	assert.NoError(t, err)
	assert.Nil(t, delFunc)
}
