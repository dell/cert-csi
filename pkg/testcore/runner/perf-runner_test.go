package runner

import (
	"context"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/mocks"
	"go.uber.org/mock/gomock"
)

func TestCheckValidNamespace(t *testing.T) {
	tests := []struct {
		name     string
		driverNs string
		k8s      k8sclient.KubeClientInterface
		wantErr  bool
	}{
		{
			name:     "valid namespace",
			driverNs: "test-namespace",
			k8s: func() k8sclient.KubeClientInterface {
				mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
				mockKubeClient.EXPECT().NamespaceExists(context.Background(), "test-namespace").Times(1).Return(true, nil)
				return mockKubeClient
			}(),
			wantErr: false,
		},
		{
			name:     "namespace doesn't exist",
			driverNs: "non-existing-namespace",
			k8s: func() k8sclient.KubeClientInterface {
				mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
				mockKubeClient.EXPECT().NamespaceExists(context.Background(), "non-existing-namespace").Times(1).Return(false, nil)
				return mockKubeClient
			}(),
			wantErr: true,
		},
		{
			name:     "empty namespace",
			driverNs: "",
			k8s: func() k8sclient.KubeClientInterface {
				return nil // no client needed since we aren't invoking the function due to an empty namespace name
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		runner := &Runner{
			KubeClient: tt.k8s,
		}
		// TODO return an error from checkValidNamespace to validate if it was found or not?
		checkValidNamespace(tt.driverNs, runner)
	}
}
