package volsnap

import (
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/stretchr/testify/assert"
	"testing"
)

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
			want: "volsnap-test",
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
