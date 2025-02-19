package expansion

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore/suites/mockutils"
	"github.com/stretchr/testify/assert"
	"io"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	"reflect"
	"testing"
)

type FakeRemoteExecutor struct{}

type FakeremoteexecutorVolexpansion struct {
	callCount int
}

func (f *FakeRemoteExecutor) Execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	return nil
}

func (f *FakeremoteexecutorVolexpansion) Execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	// Reset the call count if the output for df has been iterated through twice
	if f.callCount >= 2 {
		f.callCount = 0
	}
	// Increment the call count
	f.callCount++

	// Write the appropriate output based on the call count
	if f.callCount == 1 {
		stdout.Write([]byte("Filesystem     1K-blocks    Used Available Use% Mounted on\n"))
		stdout.Write([]byte("/dev/sda1      1048576   0        1048576    100% /"))
		stdout.Write([]byte("/data0      1048576   0        1048576    100% /abc"))
	} else if f.callCount == 2 {
		stdout.Write([]byte("Filesystem     1K-blocks    Used Available Use% Mounted on\n"))
		stdout.Write([]byte("/dev/sda1      2097152   0        2097152    100% /"))
		stdout.Write([]byte("/data0      2097152   0        2097152    100% /abc"))
	}

	return nil
}

func TestVolumeExpansionSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new VolumeExpansionSuite instance
	ves := &VolumeExpansionSuite{
		VolumeNumber: 1,
		//PodNumber:    1,
		IsBlock:      true,
		InitialSize:  "1Gi",
		ExpandedSize: "5Gi",
		Description:  "test-description",
		AccessMode:   "ReadWriteOnce",
		//Image:        "quay.io/centos/centos:latest",
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

	// Create a fake k8sclient.KubeClient
	kubeClient := k8sclient.KubeClient{
		ClientSet:   clientset,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	// Set up a reactor to simulate Pods becoming Ready
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

		// Simulate the "FileSystemResizeSuccessful" event
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pod.Namespace,
				Name:      "test-event",
			},
			InvolvedObject: v1.ObjectReference{
				Namespace: pod.Namespace,
				Name:      pod.Name,
				UID:       pod.UID,
			},
			Reason: "FileSystemResizeSuccessful",
			Type:   v1.EventTypeNormal,
		}
		clientset.Tracker().Add(event)

		return false, nil, nil // Allow normal processing to continue
	})

	// When getting pods, return the pod with Running status and Ready condition
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

	namespace := ves.GetNamespace()

	// Create the necessary clients
	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
	podClient, _ := kubeClient.CreatePodClient(namespace)
	scClient, _ := kubeClient.CreateSCClient()
	pvClient, _ := kubeClient.CreatePVClient()
	k8Clients := &k8sclient.Clients{
		KubeClient:             &kubeClient,
		PodClient:              podClient,
		PVCClient:              pvcClient,
		SCClient:               scClient,
		PersistentVolumeClient: pvClient,
	}

	// Run the suite
	_, err := ves.Run(ctx, "test-storage-class", k8Clients)

	// Check if there was an error
	if err != nil {
		t.Errorf("Error running VolumeExpansionSuite.Run(): %v", err)
	}
}

func TestVolumeExpansionSuite_Run_NonBlock(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new VolumeExpansionSuite instance
	ves := &VolumeExpansionSuite{
		VolumeNumber: 1,
		IsBlock:      false,
		InitialSize:  "1Gi",
		ExpandedSize: "2Gi",
		Description:  "test-description",
		AccessMode:   "ReadWriteOnce",
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
	clientset := mockutils.NewFakeClientsetWithRestClient(storageClass)

	// Create a fake k8sclient.KubeClient
	kubeClient := k8sclient.KubeClient{
		ClientSet:   clientset,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	// Set up a reactor to simulate Pods becoming Ready
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

		// Simulate the "FileSystemResizeSuccessful" event
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pod.Namespace,
				Name:      "test-event",
			},
			InvolvedObject: v1.ObjectReference{
				Namespace: pod.Namespace,
				Name:      pod.Name,
				UID:       pod.UID,
			},
			Message: "Filesystem resize successful",
			Reason:  "FileSystemResizeSuccessful",
			Type:    v1.EventTypeNormal,
		}

		volume := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pvc",
				Namespace: pod.Namespace,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				Resources: v1.VolumeResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse(ves.InitialSize),
					},
				},
				AccessModes: []v1.PersistentVolumeAccessMode{
					v1.PersistentVolumeAccessMode(ves.AccessMode),
				},
			},
			Status: v1.PersistentVolumeClaimStatus{
				Phase: v1.ClaimBound,
				Capacity: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse(ves.InitialSize),
				},
			},
		}

		// Add the event and volume to the fake clientset
		err := clientset.Tracker().Add(event)
		assert.NoError(t, err)
		err = clientset.Tracker().Add(volume)

		assert.NoError(t, err)

		return false, nil, nil // Allow normal processing to continue
	})

	// When getting pods, return the pod with Running status and Ready condition
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

	namespace := ves.GetNamespace()

	// Create the necessary clients
	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
	podClient, _ := kubeClient.CreatePodClient(namespace)
	podClient.RemoteExecutor = &FakeremoteexecutorVolexpansion{callCount: 0}
	scClient, _ := kubeClient.CreateSCClient()
	pvClient, _ := kubeClient.CreatePVClient()
	k8Clients := &k8sclient.Clients{
		KubeClient:             &kubeClient,
		PodClient:              podClient,
		PVCClient:              pvcClient,
		SCClient:               scClient,
		PersistentVolumeClient: pvClient,
	}

	// Run the suite
	_, err := ves.Run(ctx, "test-storage-class", k8Clients)
	// Check if there was an error
	if err != nil {
		t.Errorf("Error running NonBlock iteration of VolumeExpansionSuite.Run(): %v", err)
	}
}

// TODO TestCheckSize
// TODO TestConvertSpecSize
func TestVolumeExpansionSuite_GetObservers(t *testing.T) {
	ves := &VolumeExpansionSuite{}
	obsType := observer.Type("someType")
	observers := ves.GetObservers(obsType)
	if observers == nil {
		t.Errorf("Expected observers, got nil")
	}
	// Add more assertions based on expected behavior
}

// TestVolumeExpansionSuite_GetClients
func TestVolumeExpansionSuite_GetClients(t *testing.T) {
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
		ves     *VolumeExpansionSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			ves:  &VolumeExpansionSuite{},
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
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ves.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("VolumeExpansionSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("VolumeExpansionSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeExpansionSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		ves  *VolumeExpansionSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			ves:  &VolumeExpansionSuite{},
			want: "volume-expansion-suite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ves.GetNamespace(); got != tt.want {
				t.Errorf("VolumeExpansionSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeExpansionSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		ves  *VolumeExpansionSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			ves: &VolumeExpansionSuite{
				Description: "test-description",
			},
			want: "test-description",
		},
		{
			name: "Testing GetName when Description is empty",
			ves:  &VolumeExpansionSuite{},
			want: "VolumeExpansionSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ves.GetName(); got != tt.want {
				t.Errorf("VolumeExpansionSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeExpansionSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		ves  *VolumeExpansionSuite
		want string
	}{
		{
			name: "Testing Parameters",
			ves: &VolumeExpansionSuite{
				PodNumber:    3,
				VolumeNumber: 5,
				InitialSize:  "3Gi",
				ExpandedSize: "5Gi",
				IsBlock:      true,
			},
			want: "{pods: 3, volumes: 5, size: 3Gi, expSize: 5Gi, block: true}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ves.Parameters(); got != tt.want {
				t.Errorf("VolumeExpansionSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestConvertSpecSize(t *testing.T) {
	tests := []struct {
		specSize     string
		expectedSize int
		expectError  bool
	}{
		{"1Gi", 1048576, false},    // 1 GiB = 1048576 KiB
		{"512Mi", 524288, false},   // 512 MiB = 524288 KiB
		{"1Ti", 1073741824, false}, // 1 TiB = 1073741824 KiB
	}

	for _, tt := range tests {
		t.Run(tt.specSize, func(t *testing.T) {
			size, err := convertSpecSize(tt.specSize)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSize, size)
			}
		})
	}
}
