package runner

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/testmocks"
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

func TestGenerateTestRunDetails(t *testing.T) {
	tests := []struct {
		name                 string
		scDB                 *store.StorageClassDB
		host                 string
		expectedTestRunName  string
		expectedStartTime    time.Time
		expectedStorageClass string
		expectedClusterAddr  string
	}{
		{
			name:                 "valid inputs",
			scDB:                 &store.StorageClassDB{StorageClass: "test-sc"},
			host:                 "test-host",
			expectedTestRunName:  "test-run-",
			expectedStartTime:    time.Now(),
			expectedStorageClass: "test-sc",
			expectedClusterAddr:  "test-host",
		},
		{
			name:                 "empty host",
			scDB:                 &store.StorageClassDB{StorageClass: "test-sc"},
			host:                 "",
			expectedTestRunName:  "test-run-",
			expectedStartTime:    time.Now(),
			expectedStorageClass: "test-sc",
			expectedClusterAddr:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generateTestRunDetails(tt.scDB, nil, tt.host)
			if tt.scDB == nil {
				if tt.scDB != nil {
					t.Error("Expected storage class DB to be nil, got non-nil value")
				}
			} else {
				if tt.scDB.TestRun.Name == "" {
					t.Error("Expected test run name to be non-empty, got empty string")
				}
				if !strings.HasPrefix(tt.scDB.TestRun.Name, tt.expectedTestRunName) {
					t.Errorf("Expected test run name to start with %s, got %s", tt.expectedTestRunName, tt.scDB.TestRun.Name)
				}
				if tt.scDB.TestRun.StartTimestamp.IsZero() {
					t.Error("Expected test run start timestamp to be set, got zero value")
				}
				if tt.scDB.TestRun.StorageClass != tt.expectedStorageClass {
					t.Errorf("Expected test run storage class to be %s, got %s", tt.expectedStorageClass, tt.scDB.TestRun.StorageClass)
				}
				if tt.scDB.TestRun.ClusterAddress != tt.expectedClusterAddr {
					t.Errorf("Expected test run cluster address to be %s, got %s", tt.expectedClusterAddr, tt.scDB.TestRun.ClusterAddress)
				}
			}
		})
	}
}

func TestShouldClean(t *testing.T) {
	tests := []struct {
		name                string
		NoCleanupOnFail     bool
		suiteRes            TestResult
		noCleaning          bool
		expectedShouldClean bool
	}{
		{
			name:                "no cleanup on fail and suite result is failure",
			NoCleanupOnFail:     true,
			suiteRes:            FAILURE,
			noCleaning:          false,
			expectedShouldClean: false,
		},
		{
			name:                "no cleanup on fail and suite result is success",
			NoCleanupOnFail:     true,
			suiteRes:            SUCCESS,
			noCleaning:          false,
			expectedShouldClean: true,
		},
		{
			name:                "no cleanup on fail is false and suite result is failure",
			NoCleanupOnFail:     false,
			suiteRes:            FAILURE,
			noCleaning:          false,
			expectedShouldClean: true,
		},
		{
			name:                "no cleanup on fail is false and suite result is success",
			NoCleanupOnFail:     false,
			suiteRes:            SUCCESS,
			noCleaning:          false,
			expectedShouldClean: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := shouldClean(tt.NoCleanupOnFail, tt.suiteRes, tt.noCleaning)
			if res != tt.expectedShouldClean {
				t.Errorf("expected shouldClean to be %v, but got %v", tt.expectedShouldClean, res)
			}
		})
	}
}
