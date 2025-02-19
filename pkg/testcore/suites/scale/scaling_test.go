package scale

import (
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"reflect"
	"testing"
)

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
