package suites

import (
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/observer"
	"github.com/stretchr/testify/assert"
)

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

// TODO Test_shouldWaitForFirstConsumer

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

// TODO TestProvisioningSuite_GetObservers

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

// TODO Test_shouldWaitForFirstConsumer

func TestRemoteReplicationProvisioningSuite_GetObservers(t *testing.T) {
	rrps := &RemoteReplicationProvisioningSuite{}
	obsType := observer.Type("someType")
	observers := rrps.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

// TODO TestRemoteReplicationProvisioningSuite_GetClients

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

func TestScalingSuite_GetObservers(t *testing.T) {
	ss := &ScalingSuite{}
	obsType := observer.Type("someType")
	observers := ss.GetObservers(obsType)
	assert.NotNil(t, observers)
	// Add more assertions based on expected behavior
}

// TODO TestScalingSuite_GetClients

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

// TODO TestScalingSuite_Run
