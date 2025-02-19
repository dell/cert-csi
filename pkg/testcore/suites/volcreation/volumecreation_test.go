package volcreation

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"reflect"
	"testing"
)

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

	simpleClientset := fake.NewSimpleClientset(storageClass)
	pvcClient := &pvc.Client{ClientSet: simpleClientset} // Assuming pvc.Client is defined

	// Call the function under test
	result, err := common.ShouldWaitForFirstConsumer(context.Background(), storageClass.Name, pvcClient)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check the expected result
	if result != true {
		t.Errorf("Expected %v, got %v", true, result)
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
			got := common.ValidateCustomName(tt.customName, tt.volumes)
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
