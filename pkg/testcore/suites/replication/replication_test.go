package replication

import (
	"github.com/dell/cert-csi/pkg/observer"
	"testing"
)

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
