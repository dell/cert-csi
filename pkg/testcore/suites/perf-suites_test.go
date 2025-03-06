package suites

import (
	"bytes"
	"context"
	"errors"
	"os"

	//"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pv"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/replicationgroup"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/volumegroupsnapshot"
	"github.com/dell/cert-csi/pkg/observer"
	vgsAlpha "github.com/dell/csi-volumegroup-snapshotter/api/v1"
	snapshotAPIV1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned/typed/volumesnapshot/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	kfake "k8s.io/client-go/kubernetes/fake"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	remotePVCObject  v1.PersistentVolumeClaim
	remotePVClient   *pv.Client
	remoteRGClient   *replicationgroup.Client
	remoteKubeClient *k8sclient.KubeClient
)

type FakeExtendedCoreV1 struct {
	typedcorev1.CoreV1Interface
	restClient rest.Interface
}

func (c *FakeExtendedCoreV1) RESTClient() rest.Interface {
	if c.restClient == nil {
		c.restClient = &restfake.RESTClient{}
	}
	return c.restClient
}

type FakeExtendedClientset struct {
	*kfake.Clientset
}

func (f *FakeExtendedClientset) CoreV1() typedcorev1.CoreV1Interface {
	return &FakeExtendedCoreV1{f.Clientset.CoreV1(), nil}
}

func NewFakeClientsetWithRestClient(objs ...runtime.Object) *FakeExtendedClientset {
	return &FakeExtendedClientset{kfake.NewSimpleClientset(objs...)}
}

type FakeVolumeSnapshotInterface struct {
	snapshotv1.VolumeSnapshotInterface
}

func (f *FakeVolumeSnapshotInterface) Create(ctx context.Context, snapshot *snapshotAPIV1.VolumeSnapshot, opts metav1.CreateOptions) (*snapshotAPIV1.VolumeSnapshot, error) {
	readyToUse := true
	return &snapshotAPIV1.VolumeSnapshot{
		Status: &snapshotAPIV1.VolumeSnapshotStatus{
			ReadyToUse: &readyToUse,
		},
	}, nil
}

func (f *FakeVolumeSnapshotInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*snapshotAPIV1.VolumeSnapshot, error) {
	readyToUse := true
	return &snapshotAPIV1.VolumeSnapshot{
		Status: &snapshotAPIV1.VolumeSnapshotStatus{
			ReadyToUse: &readyToUse,
		},
	}, nil
}

type RESTMapping struct {
	mock.Mock
}

func (m *RESTMapping) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	args := m.Called(resource)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (m *RESTMapping) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	args := m.Called(resource)
	return args.Get(0).([]schema.GroupVersionKind), args.Error(1)
}

func (m *RESTMapping) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	args := m.Called(input)
	return args.Get(0).(schema.GroupVersionResource), args.Error(1)
}

func (m *RESTMapping) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	args := m.Called(input)
	return args.Get(0).([]schema.GroupVersionResource), args.Error(1)
}

func createRESTMapping() *meta.RESTMapping {
	// Create a GroupVersionResource
	gvr := schema.GroupVersionResource{
		Group:    "group",
		Version:  "version",
		Resource: "resource",
	}

	// Create a RESTMapping
	restMapping := &meta.RESTMapping{
		Resource:         gvr,
		GroupVersionKind: schema.GroupVersionKind{},
		Scope:            meta.RESTScopeNamespace,
	}

	return restMapping
}

func (m *RESTMapping) RESTMapping(_ schema.GroupKind, _ ...string) (*meta.RESTMapping, error) {
	return createRESTMapping(), nil
}

func (m *RESTMapping) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	args := m.Called(gk, versions)
	return args.Get(0).([]*meta.RESTMapping), args.Error(1)
}

func (m *RESTMapping) ResourceSingularizer(resource string) (string, error) {
	args := m.Called(resource)
	return args.String(0), args.Error(1)
}

func mockClientSetPodFunctions(clientset interface{}) interface{} {
	// Set up a reactor to simulate Pods becoming Ready
	createPodFunc := func(action k8stesting.Action) (bool, runtime.Object, error) {
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
	}

	// Also, when getting pods, return the pod with Running status and Ready condition
	getPodFunc := func(action k8stesting.Action) (bool, runtime.Object, error) {
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
	}

	switch v := clientset.(type) {
	case *kfake.Clientset:
		v.Fake.PrependReactor("create", "pods", createPodFunc)
		v.Fake.PrependReactor("get", "pods", getPodFunc)
		return clientset
	case *FakeExtendedClientset:
		v.Fake.PrependReactor("create", "pods", createPodFunc)
		v.Fake.PrependReactor("get", "pods", getPodFunc)
		return clientset
	default:
		panic(fmt.Sprintf("unexpected type %T", clientset))
	}
}

// TestVolumeCreationSuite_Run
func TestVolumeCreationSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()
	// Create a new VolumeCreationSuite instance
	vcs := &VolumeCreationSuite{
		VolumeNumber: -1,
		VolumeSize:   "1Gi",
		AccessMode:   "ReadWriteOnce",
		RawBlock:     true,
		CustomName:   "test-custom-pvc-name",
	}

	vcs_new := &VolumeCreationSuite{
		VolumeNumber: 1,
		VolumeSize:   "",
		AccessMode:   "ReadWriteOnce",
		RawBlock:     true,
	}
	// Create a fake storage class with VolumeBindingMode set to Immediate
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
		ClientSet:   clientset,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}
	kubeClient.SetTimeout(2)
	namespace := "test-namespace"
	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
	// Create a fake PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
		},
	}
	clientset.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	// Create a fake PV
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
	}
	clientset.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
	// Create a fake Pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
	}
	clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	// Create a fake VolumeAttachment
	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
	}
	clientset.StorageV1().VolumeAttachments().Create(context.Background(), va, metav1.CreateOptions{})
	// Create a fake k8sclient.Clients instance
	k8Clients := &k8sclient.Clients{
		KubeClient: kubeClient,
		PVCClient:  pvcClient,
	}
	// Call the Run method
	_, err := vcs.Run(ctx, "test-storage-class", k8Clients)
	// Check if there was an error
	if err != nil {
		t.Errorf("Error running VolumeCreationSuite.Run(): %v", err)
	}
	// Call the Run method
	_, err1 := vcs_new.Run(ctx, "test-storage-class", k8Clients)
	// Check if there was an error
	if err1 != nil {
		t.Errorf("Error running VolumeCreationSuite.Run(): %v", err1)
	}
}

func TestValidateCustomSnapName(t *testing.T) {
	tests := []struct {
		name           string
		snapshotAmount int
		expected       bool
	}{
		{"customName", 1, true},
		{"", 1, false},
		{"customName", 2, false},
		{"", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCustomSnapName(tt.name, tt.snapshotAmount)
			assert.Equal(t, tt.expected, result)
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
	clientset = mockClientSetPodFunctions(clientset).(*kfake.Clientset)

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

func TestRemoteReplicationProvisioningSuite_Run(t *testing.T) {
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
	clientset := NewFakeClientsetWithRestClient(storageClass)

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
		t.Errorf("Error creating remoteKubeClient: %v", err)
	}

	rgClient, _ := remoteKubeClient.CreateRGClient()

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

// MockClients is a mock implementation of the Clients interface
type MockClients struct {
	mock.Mock
}

// CreatePodClient is a mock implementation of the CreatePodClient method
func (c *MockClients) CreatePodClient(namespace string) (*pod.Client, error) {
	args := c.Called(namespace)
	podClient, err := args.Get(0).(*pod.Client), args.Error(1)
	return podClient, err
}

type FakeHashRemoteExecutor struct{}

func (FakeHashRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	Output := "OK"
	_, err := fmt.Fprint(stdout, Output)
	if err != nil {
		return err
	}
	return nil
}

type FakeHashRemoteExecutor2 struct{}

func (FakeHashRemoteExecutor2) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	Output := "Test"
	_, err := fmt.Fprint(stdout, Output)
	if err != nil {
		return err
	}
	return nil
}

// FakeRemoteExecutor is a mock implementation of the RemoteExecutor interface
// FakeRemoteExecutor3 is a mock implementation of the RemoteExecutor interface
type FakeRemoteExecutor3 struct {
	ExecFunc func(method string, url *url.URL, config *rest.Config, headers http.Header, body io.Reader, contentType string, params url.Values, acceptHeaders []string, responseHeaders map[string]string, responseBody io.Writer, errorResponseBody io.Writer, quiet bool) error
}

// Exec is a mock implementation of the Exec method
func (f *FakeRemoteExecutor3) Exec(method string, url *url.URL, config *rest.Config, headers http.Header, body io.Reader, contentType string, params url.Values, acceptHeaders []string, responseHeaders map[string]string, responseBody io.Writer, errorResponseBody io.Writer, quiet bool) error {
	if f.ExecFunc != nil {
		return f.ExecFunc(method, url, config, headers, body, contentType, params, acceptHeaders, responseHeaders, responseBody, errorResponseBody, quiet)
	}
	return nil
}

// Execute is a mock implementation of the Execute method
func (f *FakeRemoteExecutor3) Execute(method string, url *url.URL, config *rest.Config, headers http.Header, body io.Reader, contentType string, params url.Values, acceptHeaders []string, responseHeaders map[string]string, responseBody io.Writer, errorResponseBody io.Writer, quiet bool) error {
	if f.ExecFunc != nil {
		return f.ExecFunc(method, url, config, headers, body, contentType, params, acceptHeaders, responseHeaders, responseBody, errorResponseBody, quiet)
	}
	return nil
}

// TODO TestVolumeIoSuite_Run
func TestVolumeIoSuite_Run(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a VolumeIoSuite instance
	vis := &VolumeIoSuite{
		VolumeNumber: 1,
		VolumeSize:   "1Gi",
		ChainNumber:  1,
		ChainLength:  2,
		Image:        "quay.io/centos/centos:latest",
	}
	vis4 := &VolumeIoSuite{
		VolumeNumber: 1,
		VolumeSize:   "1Gi",
		ChainNumber:  1,
		ChainLength:  2,
		Image:        "quay.io/centos/centos:latest",
	}

	vis2 := &VolumeIoSuite{
		VolumeNumber: 0,
		VolumeSize:   "1Gi",
		ChainNumber:  0,
		ChainLength:  0,
		Image:        "",
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

	// clientSet := fake.NewSimpleClientset(storageClass)
	clientSet := NewFakeClientsetWithRestClient(storageClass)

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

	// Create a mock Clients instance
	mockClients := &MockClients{}

	// Set up the mock behavior for the CreatePodClient method
	mockClients.On("CreatePodClient", "test-namespace").Return(
		&pod.Client{
			Interface: clientSet.CoreV1().Pods("test-namespace"),
		},
		nil,
	)

	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})
	// Create a fake k8sclient.KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	pvcClient2, _ := kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	podClient.RemoteExecutor = &FakeRemoteExecutor{}
	podClient.RemoteExecutor = &FakeHashRemoteExecutor{}

	podClient2, _ := kubeClient.CreatePodClient("test-namespace")
	podClient2.RemoteExecutor = &FakeRemoteExecutor{}

	// Create a fake PodClient
	pC3, _ := kubeClient.CreatePodClient("test-namespace")
	// podClient3.Exec(ctx, pod, []string, nil, nil, false) = func () error {
	// 	return fmt.Errorf("Exec is not working")
	// }
	// Create a fake PodClient
	podClient3 := &pod.Client{
		Interface: nil,
		RemoteExecutor: &FakeRemoteExecutor3{
			ExecFunc: func(method string, url *url.URL, config *rest.Config, headers http.Header, body io.Reader, contentType string, params url.Values, acceptHeaders []string, responseHeaders map[string]string, responseBody io.Writer, errorResponseBody io.Writer, quiet bool) error {
				return fmt.Errorf("Exec is not working")
			},
		},
	}
	//&pod.Client{
	// RemoteExecutor: &pod.Client.FakeRemoteExecutor3{
	// 	ExecFunc: func(method string, url *url.URL, config *rest.Config, headers http.Header, body io.Reader, contentType string, params url.Values, acceptHeaders []string, responseHeaders map[string]string, responseBody io.Writer, errorResponseBody io.Writer, quiet bool) error {
	//return fmt.Errorf("Exec is not working")
	// 	},
	// },
	// }

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Create a fake stdout and stderr
	writer := &bytes.Buffer{}

	// Call the Exec method
	err := podClient.Exec(ctx, pod, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false)

	// podClient3. = func (ctx, v1*v1.Pod, command []string, stdout, stderr io.Writer, quiet bool) error {
	// 	return fmt.Errorf("Exec is not working")
	// }
	//podClient3, _ := kubeClient.CreatePodClient("test-namespace")
	//podClient2.RemoteExecutor = &FakeHashRemoteExecutor2{}
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")

	//clientSet.StorageV1().StorageClasses().Get(ctx, storageClass, metav1.CreateOptions{})

	// Call the shouldWaitForFirstConsumer function
	firstConsumer, err := shouldWaitForFirstConsumer(ctx, "test", pvcClient)

	// Assert the error
	assert.Error(t, err)

	// // Assert the expected error message
	// assert.EqualError(t, err, "storageclass.storage.k8s.io \"test-storage-class\" not found")

	// // Assert the firstConsumer value
	// assert.True(t, firstConsumer)

	// Assert the error
	//assert.NoError(t, err)

	// Assert the firstConsumer value
	assert.False(t, firstConsumer)

	// Create a fake PVC
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "test-namespace",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName: "test-pv",
		},
	}
	// Set the PVC status to not bound
	pvc.Status.Phase = v1.ClaimPending

	_, err5 := pvcClient2.Interface.Get(ctx, pvc.ObjectMeta.Name, metav1.GetOptions{})
	// Assert the error
	//assert.NoError(t, err5)
	assert.EqualError(t, err5, "persistentvolumeclaims \"test-pvc\" not found")
	// Assert the gotPvc value
	//assert.Nil(t, gotPvc)

	// Create the PVC in the fake clientset
	_, err = clientSet.CoreV1().PersistentVolumeClaims("test-namespace").Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create PVC: %v", err)
	}

	// Call the WaitForAllToBeBound function
	// err6 := pvcClient.WaitForAllToBeBound(ctx)

	// // Assert the error
	// assert.Error(t, err6)

	// Assert the expected error message
	//assert.EqualError(t, err, "timed out waiting for PVCs to be bound")

	// Create a fake Pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
	}

	// Create the Pod in the fake clientset
	_, err = clientSet.CoreV1().Pods("test-namespace").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create Pod: %v", err)
	}

	// Set up a reactor to simulate the Delete operation returning an error
	clientSet.Fake.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteAction := action.(k8stesting.DeleteAction)
		if deleteAction.GetName() == pod.Name {
			return true, nil, fmt.Errorf("simulated delete error")
		}
		return false, nil, nil
	})

	// Call the Delete method
	// err7 := podClient3.Delete(ctx, pod).Sync(ctx).GetError()

	// // Assert the error
	// assert.Error(t, err7)

	// Create a fake PV
	// pv := &v1.PersistentVolume{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      "test-pv",
	// 		Namespace: "test-namespace",
	// 	},
	// }

	// Create the PV in the fake clientset
	// _, err = clientSet.CoreV1().PersistentVolumes().Create(ctx, pv, metav1.CreateOptions{})
	// if err != nil {
	// 	t.Fatalf("Failed to create PV: %v", err)
	// }

	// // Set up a reactor to simulate the Delete operation returning an error
	// clientSet.Fake.PrependReactor("delete", "persistentvolumes", func(action k8stesting.Action) (bool, runtime.Object, error) {
	// 	deleteAction := action.(k8stesting.DeleteAction)
	// 	if deleteAction.GetName() == pv.Name {
	// 		return true, nil, fmt.Errorf("simulated delete error")
	// 	}
	// 	return false, nil, nil
	// })

	// // Call the WaitUntilVaGone method
	// err = vaClient.WaitUntilVaGone(ctx, pv.Name)

	// // Assert the error
	// assert.Error(t, err)

	// Create a fake clients instance
	clients1 := &k8sclient.Clients{
		PVCClient: pvcClient,
		PodClient: podClient,
		VaClient:  vaClient,
	}
	clients2 := &k8sclient.Clients{
		PVCClient: pvcClient,
		PodClient: podClient2,
		VaClient:  vaClient,
	}
	// clients3 := &k8sclient.Clients{
	// 	PVCClient: pvcClient2,
	// 	PodClient: podClient,
	// 	VaClient:  vaClient,
	// }

	tests := []struct {
		name         string
		vis          *VolumeIoSuite
		storageClass string
		clients      *k8sclient.Clients
		wantDelFunc  func() error
		wantErr      bool
	}{
		// {
		// 	name:         "Testing Run with valid inputs",
		// 	vis:          vis,
		// 	storageClass: "test-storage-class",
		// 	clients:      clients1,
		// 	wantErr:      false,
		// },
		{
			name:         "Testing Run with valid inputs",
			vis:          vis,
			storageClass: "test-storage-class",
			clients:      clients1,
			// wantDelFunc: func() error {
			// 	return fmt.Errorf("hashes don't match")
			// },
			wantErr: false,
		},
		{
			name:         "Testing Run with valid inputs4",
			vis:          vis4,
			storageClass: "test-storage-class",
			clients:      clients2,
			wantDelFunc: func() error {
				return fmt.Errorf("hashes don't match")
			},
			wantErr: true,
		},
		// {
		// 	name:         "Testing Run with valid inputs4",
		// 	vis:          vis4,
		// 	storageClass: "test-storage-class",
		// 	clients:      clients3,
		// 	wantDelFunc: func() error {
		// 		return fmt.Errorf("persistentvolumeclaims \"test-pvc\" not found")
		// 	},
		// 	wantErr: true,
		// },
		{
			name:         "Testing Run with valid inputs3",
			vis:          vis2,
			storageClass: "test-storage-class",
			clients:      clients1,
			wantDelFunc: func() error {
				return fmt.Errorf("persistentvolumeclaims \"\" already exists")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.vis.Run(context.TODO(), tt.storageClass, tt.clients)
			// if (err != nil) != tt.wantErr {
			// 	assert.NotNil(t, tt.wantDelFunc())
			// 	assert.Equal(t, tt.wantDelFunc().Error(), err.Error())
			// }
			if tt.wantErr {
				if err == nil {
					t.Errorf("VolumeIO.Run() expected error, but got nil")
				}
			} else if err != nil {
				t.Errorf("VolumeIO.Run() returned error: %v", err)
			}

			if tt.wantDelFunc != nil {
				if (err != nil) != tt.wantErr {
					assert.NotNil(t, tt.wantDelFunc())
					assert.Equal(t, tt.wantDelFunc().Error(), err.Error())
				}
				// 	if delFunc == nil {
				// 		t.Errorf("VolumeIO.Run() expected delete function, but got nil")
				// 	}
				// } else if delFunc != nil {
				// 	t.Errorf("VolumeIO.Run() returned unexpected delete function")
			}

			err = clientSet.CoreV1().PersistentVolumeClaims("test-namespace").Delete(context.TODO(), "", metav1.DeleteOptions{})
			if err != nil {
				fmt.Printf("Error deleting pvc: %v\n", err)
				return
			}
			// err = clientSet.CoreV1().PersistentVolumeClaims("test-namespace").Delete(context.TODO(), "-restore", metav1.DeleteOptions{})
			// if err != nil {
			// 	fmt.Printf("Error deleting pvc: %v\n", err)
			// 	return
			// }

			// err = clientSet.CoreV1().Pods("test-namespace").Delete(context.TODO(), "-restore-pvc", metav1.DeleteOptions{})
			// if err != nil {
			// 	fmt.Printf("Error deleting pvc: %v\n", err)
			// 	return
			// }

		})
	}
}

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
			name: "Testing GetClients with empty namespace",
			vis:  &VolumeIoSuite{},
			args: args{
				client: &kubeClient,
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
		{
			name: "Testing GetClients with valid namespace",
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
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
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

// TestVolumeGroupSnapSuite_Run
func TestVolumeGroupSnapSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new VolumeGroupSnapSuite instance
	vgs := &VolumeGroupSnapSuite{
		SnapClass:  "testSnap",
		AccessMode: "ReadWriteOnce",
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
	clientset = mockClientSetPodFunctions(clientset).(*kfake.Clientset)

	// Create a fake k8sclient.KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientset,
		Config:    &rest.Config{},
	}

	// Create PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Create Pod client
	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	// Create PV client
	pvClient, _ := kubeClient.CreatePVClient()

	scheme := runtime.NewScheme()
	if err := vgsAlpha.AddToScheme(scheme); err != nil {
		panic(err)
	}
	restMapperMock := &RESTMapping{}
	restMapperMock.On("RESTMapping", mock.Anything, mock.Anything).Return(restMapperMock, nil)
	k8sClient, _ := client.New(kubeClient.Config, client.Options{Scheme: scheme, Mapper: restMapperMock})
	vgsClient := &volumegroupsnapshot.Client{
		Interface: k8sClient,
	}

	// Update the k8sclient.Clients instance with the fake clients
	clients := &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		PersistentVolumeClient: pvClient,
		KubeClient:             kubeClient,
		VgsClient:              vgsClient,
	}

	_, err := vgs.Run(ctx, "test-storage-class", clients)

	expectedError := errors.New("connection refused")
	assert.Contains(t, err.Error(), expectedError.Error())
}

func TestVolumeGroupSnapSuite_GetObservers(t *testing.T) {
	vgs := &VolumeGroupSnapSuite{}
	obsType := observer.Type("someType")
	observers := vgs.GetObservers(obsType)
	if observers == nil {
		t.Errorf("Expected observers, got nil")
	}
	// Add more assertions based on expected behavior
}

func TestVolumeGroupSnapSuite_GetClients(t *testing.T) {
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
		vgs     *VolumeGroupSnapSuite
		args    args
		want    *k8sclient.Clients
		wantErr bool
	}{
		{
			name: "Testing GetClients",
			vgs:  &VolumeGroupSnapSuite{SnapClass: "test-snap-class"},
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
			got, err := tt.vgs.GetClients(tt.args.namespace, tt.args.client)
			fmt.Println(got, err)

			expectedError := errors.New("connection refused")
			assert.Contains(t, err.Error(), expectedError.Error())
		})
	}
}

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
	clientset := NewFakeClientsetWithRestClient(storageClass)

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
		clientset.Tracker().Add(event)
		clientset.Tracker().Add(volume)

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
	podClient.RemoteExecutor = &FakeRemoteExecutor_VolExpansion{callCount: 0}
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
	clientset = mockClientSetPodFunctions(clientset).(*kfake.Clientset)

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

// TestCloneVolumeSuite_Run
func TestCloneVolumeSuite_Run(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Create a new CloneVolumeSuite instance
	cs := &CloneVolumeSuite{
		VolumeNumber:  1,
		CustomPvcName: "pvc-test",
		CustomPodName: "pod-test",
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
	clientset = mockClientSetPodFunctions(clientset).(*kfake.Clientset)

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

	// Call the Run method
	_, err := cs.Run(ctx, "test-storage-class", clients)
	// Check if there was an error
	if err != nil {
		t.Errorf("Error running CloneVolumeSuite.Run(): %v", err)
	}
}

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
	clientSet = mockClientSetPodFunctions(clientSet).(*kfake.Clientset)

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

type FakeRemoteExecutor struct{}

type FakeRemoteExecutor_VolExpansion struct {
	callCount int
}

func (f *FakeRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	return nil
}

func (f *FakeRemoteExecutor_VolExpansion) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
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

	clientSet := NewFakeClientsetWithRestClient(storageClass)
	clientSet = mockClientSetPodFunctions(clientSet).(*FakeExtendedClientset)

	// Create a fake KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	// Create the necessary clients
	pvcClient, _ := kubeClient.CreatePVCClient(namespace)
	podClient, _ := kubeClient.CreatePodClient(namespace)
	podClient.RemoteExecutor = &FakeRemoteExecutor{}
	vaClient, _ := kubeClient.CreateVaClient(namespace)
	metricsClient, _ := kubeClient.CreateMetricsClient(namespace)
	// snapGA, snapBeta, _ := GetSnapshotClient(namespace, kubeClient)
	snapGA, _ := kubeClient.CreateSnapshotGAClient(namespace)
	snapGA.Interface = &FakeVolumeSnapshotInterface{}
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
// 			bss:  &BlockSnapSuite{SnapClass: "test-snap-class"},
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
				SnapClass:          "test-snap-class",
				VolumeSize:         "10Gi",
				Description:        "test-description",
				CustomSnapName:     "test-snap-name",
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
				SnapClass:          "test-snap-class",
				VolumeSize:         "10Gi",
				Description:        "test-description",
				CustomSnapName:     "test-snap-name",
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
