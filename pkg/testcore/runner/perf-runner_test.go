package runner

import (
	"testing"

	"github.com/dell/cert-csi/pkg/testcore/runner/mocks"
	"github.com/golang/mock/gomock"
)

func TestCheckValidNamespace(t *testing.T) {
	tests := []struct {
		name     string
		driverNs string
		wantErr  bool
	}{
		{
			name:     "valid namespace",
			driverNs: "test-namespace",
			wantErr:  false,
		},
		{
			name:     "invalid namespace",
			driverNs: "non-existing-namespace",
			wantErr:  true,
		},
		{
			name:     "empty namespace",
			driverNs: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		mockKubeclient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
		runner := &Runner{
			KubeClient: mockKubeclient,
		}
		if runner == nil {
			t.Errorf("Runner is nil")
			return
		}
		checkValidNamespace(tt.driverNs, runner)

	}
}
