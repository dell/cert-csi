package runner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/mocks"
	"github.com/dell/cert-csi/pkg/store"
	"k8s.io/client-go/rest"

	runnermocks "github.com/dell/cert-csi/pkg/testcore/runner/mocks"
	"github.com/dell/cert-csi/pkg/testcore/suites"
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

func TestNewSuiteRunner(t *testing.T) {
	mock_kube := mocks.NewMockKubeClientInterface(gomock.NewController(t))
	mock_kube.EXPECT().StorageClassExists(gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)
	mock_kube.EXPECT().NamespaceExists(gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)
	mock := runnermocks.NewMockK8sClientInterface(gomock.NewController(t))
	mock.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(&rest.Config{
		Host: "localhost",
	}, nil)
	mock.EXPECT().NewKubeClient(gomock.Any(), gomock.Any()).AnyTimes().Return(mock_kube, nil)

	// Test case: Successful creation
	configPath := "config.yaml"
	driverNs := "driver-namespace"
	startHook := "start-hook"
	readyHook := "ready-hook"
	finishHook := "finish-hook"
	observerType := "EVENT"
	longevity := "1h"
	driverNSHealthMetrics := "driver-namespace-health-metrics"
	timeout := 30
	cooldown := 10
	sequentialExecution := true
	noCleanup := true
	noCleanupOnFail := true
	noMetrics := true
	noReport := true
	scDBs := []*store.StorageClassDB{{StorageClass: "sc1"}, {StorageClass: "sc2"}}
	runner := NewSuiteRunner(configPath, driverNs, startHook, readyHook, finishHook, observerType, longevity, driverNSHealthMetrics,
		timeout, cooldown, sequentialExecution, noCleanup, noCleanupOnFail, noMetrics, noReport, scDBs, mock)
	if runner.CoolDownPeriod != cooldown {
		t.Errorf("Expected CoolDownPeriod to be %d, got %d", cooldown, runner.CoolDownPeriod)
	}
	if runner.StartHookPath != startHook {
		t.Errorf("Expected StartHookPath to be %s, got %s", startHook, runner.StartHookPath)
	}
	if runner.ReadyHookPath != readyHook {
		t.Errorf("Expected ReadyHookPath to be %s, got %s", readyHook, runner.ReadyHookPath)
	}
	if runner.FinishHookPath != finishHook {
		t.Errorf("Expected FinishHookPath to be %s, got %s", finishHook, runner.FinishHookPath)
	}
	if runner.DriverNSHealthMetrics != driverNSHealthMetrics {
		t.Errorf("Expected DriverNSHealthMetrics to be %s, got %s", driverNSHealthMetrics, runner.DriverNSHealthMetrics)
	}
	if runner.sequentialExecution != sequentialExecution {
		t.Errorf("Expected sequentialExecution to be %t, got %t", sequentialExecution, runner.sequentialExecution)
	}
	if runner.NoMetrics != noMetrics {
		t.Errorf("Expected NoMetrics to be %t, got %t", noMetrics, runner.NoMetrics)
	}
	if runner.NoReport != noReport {
		t.Errorf("Expected NoReport to be %t, got %t", noReport, runner.NoReport)
	}
	if runner.IterationNum != -1 {
		t.Errorf("Expected IterationNum to be %d, got %d", -1, runner.IterationNum)
	}
	if runner.Duration != time.Hour {
		t.Errorf("Expected Duration to be %s, got %s", time.Hour, runner.Duration)
	}
	if len(runner.ScDBs) != len(scDBs) {
		t.Errorf("Expected ScDBs to have length %d, got %d", len(scDBs), len(runner.ScDBs))
	}
	// Test case: Error in storage class existence check
	configPath = "config.yaml"
	driverNs = "driver-namespace"
	observerType = "EVENT"
	longevity = "1h"
	driverNSHealthMetrics = "driver-namespace-health-metrics"
	timeout = 30
	cooldown = 10
	sequentialExecution = true
	noCleanup = true
	noCleanupOnFail = true
	noMetrics = true
	noReport = true
	scDBs = []*store.StorageClassDB{{StorageClass: "sc1"}}
	runner = NewSuiteRunner(configPath, driverNs, startHook, readyHook, finishHook, observerType, longevity, driverNSHealthMetrics,
		timeout, cooldown, sequentialExecution, noCleanup, noCleanupOnFail, noMetrics, noReport, scDBs, mock)
	if runner.ScDBs[0].StorageClass != "sc1" {
		t.Errorf("Expected StorageClass to be %s, got %s", "sc1", runner.ScDBs[0].StorageClass)
	}
	// Test case: Error in storage class check
	configPath = "config.yaml"
	driverNs = "driver-namespace"
	observerType = "EVENT"
	longevity = "1h"
	driverNSHealthMetrics = "driver-namespace-health-metrics"
	timeout = 30
	cooldown = 10
	sequentialExecution = true
	noCleanup = true
	noCleanupOnFail = true
	noMetrics = true
	noReport = true
	scDBs = []*store.StorageClassDB{{StorageClass: "sc1"}, {StorageClass: "sc2"}}
	runner = NewSuiteRunner(configPath, driverNs, startHook, readyHook, finishHook, observerType, longevity, driverNSHealthMetrics,
		timeout, cooldown, sequentialExecution, noCleanup, noCleanupOnFail, noMetrics, noReport, scDBs, mock)
	if len(runner.ScDBs) != len(scDBs) {
		t.Errorf("Expected ScDBs to have length %d, got %d", len(scDBs)-1, len(runner.ScDBs))
	}
	// Test case: Error in namespace check
	configPath = "config.yaml"
	driverNs = "driver-namespace"
	observerType = "EVENT"
	longevity = "1h"
	driverNSHealthMetrics = "driver-namespace-health-metrics"
	timeout = 30
	cooldown = 10
	sequentialExecution = true
	noCleanup = true
	noCleanupOnFail = true
	noMetrics = true
	noReport = true
	scDBs = []*store.StorageClassDB{{StorageClass: "sc1"}, {StorageClass: "sc2"}}
	runner = NewSuiteRunner(configPath, driverNs, startHook, readyHook, finishHook, observerType, longevity, driverNSHealthMetrics,
		timeout, cooldown, sequentialExecution, noCleanup, noCleanupOnFail, noMetrics, noReport, scDBs, mock)
	if len(runner.ScDBs) != len(scDBs) {
		t.Errorf("Expected ScDBs to have length %d, got %d", len(scDBs), len(runner.ScDBs))
	}
}
func TestExecuteSuite(t *testing.T) {
	tests := []struct {
		name        string
		iterCtx     context.Context
		num         int
		suites      func() map[string][]suites.Interface
		suite       func() suites.Interface
		sr          *SuiteRunner
		scDB        *store.StorageClassDB
		c           chan os.Signal
		wantSuccess bool
	}{
		{
			name:    "Successful suite run",
			iterCtx: context.Background(),
			num:     1,
			suites: func() map[string][]suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				// suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
				return map[string][]suites.Interface{
					"testSuite": {
						suite,
					},
				}
			},
			suite: func() suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				// suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				suite.EXPECT().GetName().AnyTimes().Return("test-suite-name")
				suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				return suite
			},
			sr:          &SuiteRunner{},
			scDB:        &store.StorageClassDB{},
			c:           make(chan os.Signal),
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the necessary dependencies for the test
			// For example, you can use testify/mock to create mock objects
			// and replace the actual dependencies with the mocks.

			// Call the ExecuteSuite function
			ExecuteSuite(tt.iterCtx, tt.num, tt.suites(), tt.suite(), tt.sr, tt.scDB, tt.c)

			// Assert the expected outcome
			// For example, you can use testify/assert to make assertions
			// or write custom assertions based on the expected behavior.
		})
	}
}

func TestRunHook(t *testing.T) {
	script, err := os.CreateTemp("", "script.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(script.Name())
	// Write the desired script content to the file
	scriptContent := `#!/bin/bash
echo "Hello, World!"
`
	_, err = script.Write([]byte(scriptContent))
	if err != nil {
		t.Fatal(err)
	}

	// Close the file to flush the content
	err = script.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Test case: Run the mock bash script
	startHook := script.Name()
	hookName := "Mock Bash Script"
	_ = os.Chmod(startHook, 0755)
	err = runHook(startHook, hookName)
	if err != nil {
		t.Errorf("Expected no error, got %s", err.Error())
	}
	tests := []struct {
		startHook string
		hookName  string
		wantErr   bool
	}{
		{
			startHook: "/path/to/nonexistent.sh",
			hookName:  "Non-Existent Script",
			wantErr:   true,
		},
		{
			startHook: "/path/to/nonexistent.py",
			hookName:  "Non-bash Script",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		err := runHook(tt.startHook, tt.hookName)
		if (err != nil) != tt.wantErr {
			t.Errorf("runHook(%s, %s) error = %v, wantErr %v", tt.startHook, tt.hookName, err, tt.wantErr)
		}
	}
}
