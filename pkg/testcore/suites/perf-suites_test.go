package suites

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

// TestVolumeCreationSuite_Run
func TestVolumeCreationSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new VolumeCreationSuite instance
	vcs := &VolumeCreationSuite{
		VolumeNumber: 1,
		VolumeSize:   "1Gi",
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
	clientset := fake.NewSimpleClientset(storageClass)

	// Create a fake k8sclient.KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientset,
		// Other fields can be left as zero values for simplicity
	}

	// Create a PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Update the k8sclient.Clients instance with the fake PVC client
	clients := &k8sclient.Clients{
		PVCClient: pvcClient,
	}

	// Call the Run method
	_, err := vcs.Run(ctx, "test-storage-class", clients)
	// Check if there was an error
	if err != nil {
		t.Errorf("Error running VolumeCreationSuite.Run(): %v", err)
	}
}

// Testshouldwaitforfirstconsumer
func TestShouldWaitForFirstConsumer(t *testing.T) {
	// Create a fake storage class with VolumeBindingWaitForFirstConsumer mode.
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"}, // Use metav1.ObjectMeta
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}

	client := fake.NewSimpleClientset(storageClass)
	pvcClient := &pvc.Client{ClientSet: client} // Assuming pvc.Client is defined

	// Call the function under test
	result, err := shouldWaitForFirstConsumer(context.Background(), storageClass.Name, pvcClient)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check the expected result
	expected := true
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestVolumeCreationSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		vcs  *VolumeCreationSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			vcs: &VolumeCreationSuite{
				Description: "test-description",
			},
			want: "test-description",
		},
		{
			name: "Testing GetName when Description is empty",
			vcs:  &VolumeCreationSuite{},
			want: "VolumeCreationSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vcs.GetName(); got != tt.want {
				t.Errorf("VolumeCreationSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeCreationSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		v    *VolumeCreationSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			v:    &VolumeCreationSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PvcObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			v:    &VolumeCreationSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{
				&observer.PvcListObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VolumeCreationSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCustomName(t *testing.T) {
	tests := []struct {
		name       string
		customName string
		volumes    int
		want       bool
	}{
		{
			name:       "Single volume with custom name",
			customName: "custom-pvc",
			volumes:    1,
			want:       true,
		},
		{
			name:       "Single volume without custom name",
			customName: "",
			volumes:    1,
			want:       false,
		},
		{
			name:       "Multiple volumes with custom name",
			customName: "custom-pvc",
			volumes:    2,
			want:       false,
		},
		{
			name:       "Multiple volumes without custom name",
			customName: "",
			volumes:    2,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateCustomName(tt.customName, tt.volumes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVolumeCreationSuite_GetClients(t *testing.T) {
	client := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

	type args struct {
		namespace string
		client    *k8sclient.KubeClient
	}
	tests := []struct {
		name    string
		vcs     *VolumeCreationSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			vcs:  &VolumeCreationSuite{},
			args: args{
				namespace: "test-namespace",
				client:    &kubeClient,
			},
			want: &k8sclient.Clients{
				PVCClient:     pvcClient,
				MetricsClient: metricsClient,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.vcs.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("VolumeCreationSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("VolumeCreationSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeCreationSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		vcs  *VolumeCreationSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			vcs:  &VolumeCreationSuite{},
			want: "vcs-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vcs.GetNamespace(); got != tt.want {
				t.Errorf("VolumeCreationSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeCreationSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		vcs  *VolumeCreationSuite
		want string
	}{
		{
			name: "Testing Parameters",
			vcs: &VolumeCreationSuite{
				VolumeNumber: 10,
				VolumeSize:   "3Gi",
				RawBlock:     true,
			},
			want: "{number: 10, size: 3Gi, raw-block: true}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vcs.Parameters(); got != tt.want {
				t.Errorf("VolumeCreationSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestProvisioningSuite_Run
func TestProvisioningSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new ProvisioningSuite instance
	ps := &ProvisioningSuite{
		VolumeNumber: 1,
		PodNumber:    1,
		VolumeSize:   "1Gi",
		Image:        "quay.io/centos/centos:latest",
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
		return false, nil, nil // Allow normal processing to continue
	})

	// Also, when getting pods, return the pod with Running status and Ready condition
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

	// Create a fake k8sclient.KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientset,
	}

	// Create PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Create Pod client
	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	// Update the k8sclient.Clients instance with the fake clients
	clients := &k8sclient.Clients{
		PVCClient: pvcClient,
		PodClient: podClient,
	}

	// Call the Run method
	_, err := ps.Run(ctx, "test-storage-class", clients)
	// Check if there was an error
	if err != nil {
		t.Errorf("Error running ProvisioningSuite.Run(): %v", err)
	}
}

func TestProvisioningSuite_GetObservers(t *testing.T) {
	ps := &ProvisioningSuite{}
	obsType := observer.Type("someType")
	observers := ps.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

func TestProvisioningSuite_GetClients(t *testing.T) {
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
		ps      *ProvisioningSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			ps:   &ProvisioningSuite{},
			args: args{
				namespace: "test-namespace",
				client:    &kubeClient,
			},
			want: &k8sclient.Clients{
				PVCClient:     pvcClient,
				PodClient:     podClient,
				VaClient:      vaClient,
				MetricsClient: metricsClient,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ps.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProvisioningSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("ProvisioningSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvisioningSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		ps   *ProvisioningSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			ps:   &ProvisioningSuite{},
			want: "prov-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.GetNamespace(); got != tt.want {
				t.Errorf("ProvisioningSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvisioningSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		ps   *ProvisioningSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			ps: &ProvisioningSuite{
				Description: "test-description",
			},
			want: "test-description",
		},
		{
			name: "Testing GetName when Description is empty",
			ps:   &ProvisioningSuite{},
			want: "ProvisioningSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.GetName(); got != tt.want {
				t.Errorf("ProvisioningSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvisioningSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		ps   *ProvisioningSuite
		want string
	}{
		{
			name: "Testing Parameters",
			ps: &ProvisioningSuite{
				PodNumber:    1,
				VolumeNumber: 5,
				VolumeSize:   "3Gi",
			},
			want: "{pods: 1, volumes: 5, volumeSize: 3Gi}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.Parameters(); got != tt.want {
				t.Errorf("ProvisioningSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvisioningSuite_validateCustomPodName(t *testing.T) {
	tests := []struct {
		name string
		ps   *ProvisioningSuite
		want string
	}{
		{
			name: "Testing validateCustomPodName with single pod and custom name",
			ps: &ProvisioningSuite{
				PodNumber:     1,
				PodCustomName: "custom-pod",
			},
			want: "custom-pod",
		},
		{
			name: "Testing validateCustomPodName with multiple pods",
			ps: &ProvisioningSuite{
				PodNumber:     2,
				PodCustomName: "custom-pod",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ps.validateCustomPodName()
			if got := tt.ps.PodCustomName; got != tt.want {
				t.Errorf("ProvisioningSuite.validateCustomPodName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO TestRemoteReplicationProvisioningSuite_Run

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

// TODO TestScalingSuite_Run

func TestScalingSuite_GetObservers(t *testing.T) {
	ss := &ScalingSuite{}
	obsType := observer.Type("someType")
	observers := ss.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

// TestScalingSuite_GetClients
func TestScalingSuite_GetClients(t *testing.T) {
	client := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	stsClient, _ := kubeClient.CreateStatefulSetClient("test-namespace")
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

	type args struct {
		namespace string
		client    *k8sclient.KubeClient
	}
	tests := []struct {
		name    string
		ss      *ScalingSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			ss:   &ScalingSuite{},
			args: args{
				namespace: "test-namespace",
				client:    &kubeClient,
			},
			want: &k8sclient.Clients{
				PVCClient:         pvcClient,
				PodClient:         podClient,
				VaClient:          vaClient,
				StatefulSetClient: stsClient,
				MetricsClient:     metricsClient,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ss.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("ScalingSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("ScalingSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScalingSuite_GetNamespace(t *testing.T) {
	ss := &ScalingSuite{}
	namespace := ss.GetNamespace()
	assert.Equal(t, "scale-test", namespace)
}

func TestScalingSuite_GetName(t *testing.T) {
	ss := &ScalingSuite{}
	name := ss.GetName()
	assert.Equal(t, "ScalingSuite", name)
}

func TestScalingSuite_Parameters(t *testing.T) {
	ss := &ScalingSuite{
		ReplicaNumber: 5,
		VolumeNumber:  10,
		VolumeSize:    "3Gi",
	}
	params := ss.Parameters()
	expected := "{replicas: 5, volumes: 10, volumeSize: 3Gi}"
	assert.Equal(t, expected, params)
}

// TODO TestVolumeIoSuite_Run

func TestVolumeIoSuite_GetObservers(t *testing.T) {
	vis := &VolumeIoSuite{}
	obsType := observer.Type("someType")
	observers := vis.GetObservers(obsType)
	if observers == nil {
		t.Errorf("Expected observers, got nil")
	}
	// Add more assertions based on expected behavior
}

func TestVolumeIoSuite_GetClients(t *testing.T) {
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
		vis     *VolumeIoSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			vis:  &VolumeIoSuite{},
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
			got, err := tt.vis.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			if (err != nil) != tt.wantErr {
				t.Errorf("VolumeIoSuite.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(reflect.TypeOf(got), reflect.TypeOf(tt.want))
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("VolumeIoSuite.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeIoSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		vis  *VolumeIoSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			vis:  &VolumeIoSuite{},
			want: "volumeio-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vis.GetNamespace(); got != tt.want {
				t.Errorf("VolumeIoSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeIoSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		vis  *VolumeIoSuite
		want string
	}{
		{
			name: "Testing GetName",
			vis:  &VolumeIoSuite{},
			want: "VolumeIoSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vis.GetName(); got != tt.want {
				t.Errorf("VolumeIoSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeIoSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		vis  *VolumeIoSuite
		want string
	}{
		{
			name: "Testing Parameters",
			vis: &VolumeIoSuite{
				VolumeNumber: 5,
				VolumeSize:   "3Gi",
				ChainNumber:  2,
				ChainLength:  10,
			},
			want: "{volumes: 5, volumeSize: 3Gi chains: 2-10}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vis.Parameters(); got != tt.want {
				t.Errorf("VolumeIoSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO TestVolumeGroupSnapSuite_Run

func TestVolumeGroupSnapSuite_GetObservers(t *testing.T) {
	vgs := &VolumeGroupSnapSuite{}
	obsType := observer.Type("someType")
	observers := vgs.GetObservers(obsType)
	if observers == nil {
		t.Errorf("Expected observers, got nil")
	}
	// Add more assertions based on expected behavior
}

// TODO TestVolumeGroupSnapSuite_GetClients
func TestVolumeGroupSnapSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		vgs  *VolumeGroupSnapSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			vgs:  &VolumeGroupSnapSuite{},
			want: "vgs-snap-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vgs.GetNamespace(); got != tt.want {
				t.Errorf("VolumeGroupSnapSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeGroupSnapSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		vgs  *VolumeGroupSnapSuite
		want string
	}{
		{
			name: "Testing GetName",
			vgs:  &VolumeGroupSnapSuite{},
			want: "VolumeGroupSnapSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vgs.GetName(); got != tt.want {
				t.Errorf("VolumeGroupSnapSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeGroupSnapSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		vgs  *VolumeGroupSnapSuite
		want string
	}{
		{
			name: "Testing Parameters",
			vgs: &VolumeGroupSnapSuite{
				VolumeNumber: 5,
				VolumeSize:   "3Gi",
			},
			want: "{volumes: 5, volumeSize: 3Gi}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vgs.Parameters(); got != tt.want {
				t.Errorf("VolumeGroupSnapSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO TestSnapSuite_Run
func TestSnapSuite_GetObservers(t *testing.T) {
	ss := &SnapSuite{}
	obsType := observer.Type("someType")
	observers := ss.GetObservers(obsType)
	if observers == nil {
		t.Errorf("Expected observers, got nil")
	}
	// Add more assertions based on expected behavior
}

// TODO TestSnapSuite_GetClients

func TestSnapSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		ss   *SnapSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			ss:   &SnapSuite{},
			want: "snap-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ss.GetNamespace(); got != tt.want {
				t.Errorf("SnapSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		ss   *SnapSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			ss: &SnapSuite{
				Description: "test-description",
			},
			want: "test-description",
		},
		{
			name: "Testing GetName when Description is empty",
			ss:   &SnapSuite{},
			want: "SnapSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ss.GetName(); got != tt.want {
				t.Errorf("SnapSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		ss   *SnapSuite
		want string
	}{
		{
			name: "Testing Parameters",
			ss: &SnapSuite{
				SnapAmount: 5,
				VolumeSize: "3Gi",
			},
			want: "{snapshots: 5, volumeSize; 3Gi}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ss.Parameters(); got != tt.want {
				t.Errorf("SnapSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAllObservers(t *testing.T) {
	tests := []struct {
		name    string
		obsType observer.Type
		want    []observer.Interface
	}{
		{
			name:    "Testing EVENT observer type",
			obsType: observer.EVENT,
			want: []observer.Interface{
				&observer.PvcObserver{},
				&observer.VaObserver{},
				&observer.PodObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name:    "Testing LIST observer type",
			obsType: observer.LIST,
			want: []observer.Interface{
				&observer.PvcListObserver{},
				&observer.VaListObserver{},
				&observer.PodListObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name:    "Testing unknown observer type",
			obsType: observer.Type("UNKNOWN"),
			want:    []observer.Interface{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAllObservers(tt.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO TestReplicationSuite_Run
func TestReplicationSuite_GetObservers(t *testing.T) {
	rs := &ReplicationSuite{}
	obsType := observer.Type("someType")
	observers := rs.GetObservers(obsType)
	if observers == nil {
		t.Errorf("Expected observers, got nil")
	}
	// Add more assertions based on expected behavior
}

// TODO TestReplicationSuite_GetClients

func TestReplicationSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		rs   *ReplicationSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			rs:   &ReplicationSuite{},
			want: "replication-suite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.GetNamespace(); got != tt.want {
				t.Errorf("ReplicationSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplicationSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		rs   *ReplicationSuite
		want string
	}{
		{
			name: "Testing GetName",
			rs:   &ReplicationSuite{},
			want: "ReplicationSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.GetName(); got != tt.want {
				t.Errorf("ReplicationSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplicationSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		rs   *ReplicationSuite
		want string
	}{
		{
			name: "Testing Parameters",
			rs: &ReplicationSuite{
				PodNumber:    3,
				VolumeNumber: 5,
				VolumeSize:   "3Gi",
			},
			want: "{pods: 3, volumes: 5, volumeSize: 3Gi}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.Parameters(); got != tt.want {
				t.Errorf("ReplicationSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO TestVolumeExpansionSuite_Run
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

// TODO TestVolumeHealthMetricsSuite_Run
func TestVolumeHealthMetricsSuite_GetObservers(t *testing.T) {
	vh := &VolumeHealthMetricsSuite{}
	obsType := observer.Type("someType")
	observers := vh.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

// TestVolumeHealthMetricsSuite_GetClients
func TestVolumeHealthMetricsSuite_GetClients(t *testing.T) {
	client := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
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

// TODO TestCloneVolumeSuite_Run
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

// TODO TestMultiAttachSuite_Run
// func TestMultiAttachSuite_Run(t *testing.T) {
// 	// Create a context
// 	ctx := context.Background()

// 	// Create a MultiAttachSuite instance
// 	mas := &MultiAttachSuite{
// 		PodNumber:  2,
// 		RawBlock:   false,
// 		AccessMode: "ReadWriteMany",
// 		VolumeSize: "1Gi",
// 		Image:      "quay.io/centos/centos:latest",
// 	}

// 	// Mock storageClass
// 	storageClass := "test-storage-class"

// 	// Create a fake KubeClient
// 	kubeClient := &k8sclient.KubeClient{
// 		ClientSet: fake.NewSimpleClientset(),
// 		Config:    &rest.Config{},
// 	}

// 	// Create the necessary clients
// 	clients, err := mas.GetClients(mas.GetNamespace(), kubeClient)
// 	assert.NoError(t, err)

// 	// Mock PVCClient and PodClient methods if needed (e.g., using a mocking library)

// 	// Run the suite
// 	delFunc, err := mas.Run(ctx, storageClass, clients)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, delFunc)

// 	// Optionally, invoke delFunc to test deletion logic
// 	err = delFunc()
// 	assert.NoError(t, err)
// }

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

func TestBlockSnapSuite_Run(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a MultiAttachSuite instance
	bss := &BlockSnapSuite{
		SnapClass:   "testSnap",
		Description: "testDesc",
		AccessMode:  "test",
	}

	// Mock storageClass
	storageClass := "test-storage-class"

	// Create a fake KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: fake.NewSimpleClientset(),
		Config:    &rest.Config{},
	}

	namespace := bss.GetNamespace()

	// Create the necessary clients
	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
	podClient, _ := kubeClient.CreatePodClient(namespace)
	vaClient, _ := kubeClient.CreateVaClient(namespace)
	metricsClient, _ := kubeClient.CreateMetricsClient(namespace)
	// snapGA, snapBeta, snErr := GetSnapshotClient(namespace, client)

	clients := &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		// SnapClientGA:      snapGA,
		// SnapClientBeta:    snapBeta,
	}

	// Run the suite with invalid storage class error
	delFunc, err := bss.Run(ctx, storageClass, clients)
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
	expectedNamespace := "block-snap-test"
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

// TODO TestGetSnapshotClient
// TODO TestVolumeMigrateSuite_Run

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
