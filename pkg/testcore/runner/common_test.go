package runner

import (
	"errors"
	"testing"

	"github.com/dell/cert-csi/pkg/testcore/runner/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetSuiteRunner(t *testing.T) {
	tests := []struct {
		name            string
		configPath      string
		driverNs        string
		observerType    string
		timeout         int
		noCleanup       bool
		noCleanupOnFail bool
		noreport        bool
		k8s             func() K8sClientInterface
	}{
		{
			name:         "Valid parameters",
			configPath:   "config.yaml",
			driverNs:     "driver-namespace",
			observerType: "EVENT",
			timeout:      30,
			noreport:     false,
			k8s: func() K8sClientInterface {
				mock := mocks.NewMockK8sClientInterface(gomock.NewController(t))
				mock.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(nil, errors.New("new error"))
				mock.EXPECT().NewKubeClient(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.New("new error"))
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := getSuiteRunner(
				tt.configPath,
				tt.driverNs,
				tt.observerType,
				tt.timeout,
				tt.noCleanup,
				tt.noCleanupOnFail,
				tt.noreport,
				tt.k8s(),
			)
			assert.NotNil(t, client)

		})
	}
}
