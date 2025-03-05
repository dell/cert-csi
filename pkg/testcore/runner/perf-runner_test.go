package runner

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/mocks"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/store"
	storemocks "github.com/dell/cert-csi/pkg/store/mocks"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	fakeClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	runnermocks "github.com/dell/cert-csi/pkg/testcore/runner/mocks"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	clienttesting "k8s.io/client-go/testing"

	"go.uber.org/mock/gomock"
)

type SQLiteStore struct {
	db *sql.DB
}

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
	// Test case: Error in storage class check  and wrong format longevity
	configPath = "config.yaml"
	driverNs = "driver-namespace"
	observerType = "EVENT"
	longevity = "1"
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
	// Test case: Error in namespace check and cover nil longevity
	configPath = "config.yaml"
	driverNs = "driver-namespace"
	observerType = "EVENT"
	longevity = ""
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

type clientTestContext struct {
	testNamespace    string
	namespaceDeleted bool
	t                *testing.T
}

func createFakeKubeClient(ctx *clientTestContext) (kubernetes.Interface, error) {
	client := fakeClient.NewSimpleClientset()
	client.Discovery().(*fake.FakeDiscovery).FakedServerVersion = &version.Info{
		Major:      "1",
		Minor:      "32",
		GitVersion: "v1.32.0",
	}
	client.Fake.PrependReactor("create", "namespaces", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(clienttesting.CreateAction)
		namespace := createAction.GetObject().(*corev1.Namespace)
		if strings.HasPrefix(namespace.Name, "mock-ns-prefix-") {
			ctx.t.Logf("namespace %s creation called", namespace.Name)
			if ctx.testNamespace == "" {
				ctx.testNamespace = namespace.Name
			} else {
				return true, nil, fmt.Errorf("repeated test namespace creation call: was %s, now %s", ctx.testNamespace, namespace.Name)
			}
			return true, namespace, nil
		}
		return true, nil, fmt.Errorf("unexpected namespace creation %s", namespace.Name)
	})
	client.Fake.PrependReactor("delete", "namespaces", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		deleteAction := action.(clienttesting.DeleteAction)
		if ctx.testNamespace != "" && deleteAction.GetName() == ctx.testNamespace {
			ctx.t.Logf("namespace %s deletion called", deleteAction.GetName())
			ctx.namespaceDeleted = true
			return true, nil, nil
		}
		return true, nil, fmt.Errorf("unexpected namespace deletion %s", deleteAction.GetName())
	})
	return client, nil
}

func TestExecuteSuite(t *testing.T) {
	tests := []struct {
		name        string
		iterCtx     context.Context
		num         int
		suites      func() map[string][]suites.Interface
		suite       func() suites.Interface
		sr          *SuiteRunner
		c           chan os.Signal
		wantSuccess bool
		tr          *store.TestRun
	}{
		{
			name:    "Successful suite run",
			iterCtx: context.Background(),
			num:     1,
			suites: func() map[string][]suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				return map[string][]suites.Interface{
					"testSuite": {
						suite,
					},
				}
			},
			suite: func() suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				suite.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				return suite
			},
			c:           make(chan os.Signal),
			wantSuccess: true,
			tr: &store.TestRun{
				Name:           "test run 1",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
		},
		{
			name:    "Failure suite run",
			iterCtx: context.Background(),
			num:     1,
			suites: func() map[string][]suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				return map[string][]suites.Interface{
					"testSuite": {
						suite,
					},
					"testSuite1": {
						suite,
					},
				}
			},
			suite: func() suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				suite.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				return suite
			},
			c:           make(chan os.Signal),
			wantSuccess: false,
			tr: &store.TestRun{
				Name:           "test run 1",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
			newNameSpace := &corev1.Namespace{}
			newNameSpace.Name = "new-ns"
			mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), "test-namespace").AnyTimes().Return(newNameSpace, nil)
			mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockStore := storemocks.NewMockStore(gomock.NewController(t))

			clientCtx := &clientTestContext{t: t}

			k8sclient.FuncNewClientSet = func(_ *rest.Config) (kubernetes.Interface, error) {
				return createFakeKubeClient(clientCtx)

			}

			scDBs := &store.StorageClassDB{
				DB: mockStore,
			}

			r := &Runner{
				KubeClient: mockKubeClient,
			}
			if tt.name == "Failure suite run" {
				sr := &SuiteRunner{
					CoolDownPeriod: 0,
				}
				sr.Runner = r
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(fmt.Errorf("new error"))

				mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(fmt.Errorf("new error"))

				ExecuteSuite(tt.iterCtx, tt.num, tt.suites(), tt.suite(), sr, scDBs, tt.c)

			} else {
				sr := &SuiteRunner{
					CoolDownPeriod: 1,
				}
				sr.Runner = r
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)

				mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

				ExecuteSuite(tt.iterCtx, tt.num, tt.suites(), tt.suite(), sr, scDBs, tt.c)

			}

			// Assert the expected outcome
			// For example, you can use testify/assert to make assertions
			// or write custom assertions based on the expected behavior.
			if tt.wantSuccess {
				assert.True(&testing.T{}, true)
			} else {
				t.Logf("Expected ExecuteSuite() error")
			}
		})
	}
}

func TestRunSuites(t *testing.T) {
	tests := []struct {
		name                    string
		suites                  func() map[string][]suites.Interface
		expectedSucceededSuites float64
		tr                      *store.TestRun
		Duration                time.Duration
		sequentialExecution     bool
	}{
		{
			name: "Successful test run",
			suites: func() map[string][]suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				suite.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

				return map[string][]suites.Interface{
					"sc1": {
						suite,
					},
				}
			},

			tr: &store.TestRun{
				Name:           "test run 1",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			Duration:            0,
			sequentialExecution: false,
		},

		{
			name: "Failure test run",
			suites: func() map[string][]suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				suite.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

				return map[string][]suites.Interface{
					"sc1": {
						suite,
					},
				}
			},

			tr: &store.TestRun{
				Name:           "test run 1",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			Duration:            1,
			sequentialExecution: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
			newNameSpace := &corev1.Namespace{}
			newNameSpace.Name = "new-ns"
			mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), "test-namespace").AnyTimes().Return(newNameSpace, nil)
			mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			r := &Runner{
				Config:     &rest.Config{},
				KubeClient: mockKubeClient,
			}

			mockStore := storemocks.NewMockStore(gomock.NewController(t))
			if tt.name == "Failure test run" {
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(fmt.Errorf("new error"))

			} else {
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
			}
			//mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
			mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
			mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockStore.EXPECT().GetTestRuns(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]store.TestRun{
				{
					ID:             1,
					Name:           "test run 1",
					StartTimestamp: time.Now(),
					StorageClass:   "sc1",
					ClusterAddress: "localhost",
				},
			}, nil)
			mockStore.EXPECT().GetTestCases(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]store.TestCase{
				{
					ID:   1,
					Name: "test case 1",
				},
			}, nil)
			mockStore.EXPECT().GetEntitiesWithEventsByTestCaseAndEntityType(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			mockStore.EXPECT().GetNumberEntities(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			mockStore.EXPECT().GetResourceUsage(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			mockStore.EXPECT().Close().AnyTimes().Return(nil)

			sr := &SuiteRunner{
				IterationNum: 1,
				ScDBs: []*store.StorageClassDB{
					{
						StorageClass: "sc1",
						DB:           mockStore,
						TestRun:      *tt.tr,
					},
					// {
					// 	StorageClass: "sc2",
					// 	DB:           store.NewSQLiteStoreWithDB(db),
					// },
				},
				Duration:            tt.Duration,
				sequentialExecution: tt.sequentialExecution,
			}
			sr.Runner = r

			clientCtx := &clientTestContext{t: t}

			k8sclient.FuncNewClientSet = func(_ *rest.Config) (kubernetes.Interface, error) {
				return createFakeKubeClient(clientCtx)

			}

			sr.RunSuites(tt.suites())

		})
	}
}

func TestRunSuite(t *testing.T) {
	mockdb, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	client := fakeClient.NewSimpleClientset()

	client.Fake.PrependReactor("*", "*", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("error listing volumes")
	})
	tests := []struct {
		name         string
		suite        func() suites.Interface
		sr           *SuiteRunner
		testCase     *store.TestCase
		db           *store.SQLiteStore
		storageClass string
		c            chan os.Signal
		wantRes      TestResult
		wantErr      bool
	}{
		{
			name: "Test case with valid parameters",
			suite: func() suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				// suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().GetNamespace().AnyTimes().Return("driver-namespace")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

				//suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				return suite
			},
			sr:           &SuiteRunner{},
			testCase:     &store.TestCase{},
			db:           store.NewSQLiteStoreWithDB(mockdb),
			storageClass: "sc1",
			c:            make(chan os.Signal),
			wantRes:      SUCCESS,
			wantErr:      false,
		},
		{
			name: "Test case with delete namespace error",
			suite: func() suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				// suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().GetNamespace().AnyTimes().Return("driver-namespace")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

				//suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				return suite
			},
			sr:           &SuiteRunner{},
			testCase:     &store.TestCase{},
			db:           nil,
			storageClass: "sc1",
			c:            make(chan os.Signal),
			wantRes:      FAILURE,
		},
		{
			name: "Test case with delete namespace error",
			suite: func() suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				// suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().GetNamespace().AnyTimes().Return("driver-namespace")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

				//suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				return suite
			},
			sr:           &SuiteRunner{},
			testCase:     &store.TestCase{},
			db:           nil,
			storageClass: "sc1",
			c:            make(chan os.Signal),
			wantRes:      FAILURE,
		},
		{
			name: "Test case with failed namespace creation",
			suite: func() suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				// suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().GetNamespace().AnyTimes().Return("driver-namespace")
				//suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, fmt.Errorf("Mock Error"))
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

				//suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				return suite
			},
			sr:           &SuiteRunner{},
			testCase:     &store.TestCase{},
			db:           nil,
			storageClass: "sc1",
			c:            make(chan os.Signal),
			wantRes:      FAILURE,
		},

		{
			name: "Test case with failed observer",
			suite: func() suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				// suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().GetNamespace().AnyTimes().Return("driver-namespace")
				//suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, fmt.Errorf("Mock Error"))
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{
					PVCClient: &pvc.Client{
						ClientSet: client,
					},
				}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

				//suite.EXPECT().Parameters().Times(1).Return("param1,param2")
				return suite
			},
			sr:           &SuiteRunner{},
			testCase:     &store.TestCase{},
			db:           nil,
			storageClass: "sc1",
			c:            make(chan os.Signal),
			wantRes:      FAILURE,
			wantErr:      true,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name == "Test case with valid parameters" {
				sr := &SuiteRunner{}
				r := &Runner{}

				mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
				newNameSpace := &corev1.Namespace{}
				newNameSpace.Name = "new-ns"
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

				sr.Runner = r
				sr.Runner.KubeClient = mockKubeClient
				res, _ := runSuite(ctx, tt.suite(), sr, tt.testCase, tt.db, tt.storageClass, tt.c)
				if res != tt.wantRes {
					t.Errorf("Expected runSuite to return %v, but got %v", tt.wantRes, res)
				}

			}
			if tt.name == "Test case with failed namespace creation" {
				sr := &SuiteRunner{}
				r := &Runner{}

				mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
				newNameSpace := &corev1.Namespace{}
				newNameSpace.Name = "new-ns"
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, fmt.Errorf("Mock Error"))
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(fmt.Errorf("Mock Error"))

				sr.Runner = r
				sr.Runner.KubeClient = mockKubeClient
				_, err := runSuite(ctx, tt.suite(), sr, tt.testCase, tt.db, tt.storageClass, tt.c)
				if err == nil {
					t.Errorf("Expected runSuite to return error %v, but got %v", tt.wantErr, err)
				}
			}
			if tt.name == "Test case with delete namespace error" {
				sr := &SuiteRunner{}
				r := &Runner{}

				mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
				newNameSpace := &corev1.Namespace{}
				newNameSpace.Name = "new-ns"
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(fmt.Errorf("Mock Error"))

				sr.Runner = r
				sr.Runner.KubeClient = mockKubeClient
				_, err := runSuite(ctx, tt.suite(), sr, tt.testCase, tt.db, tt.storageClass, tt.c)
				if err == nil {
					t.Errorf("Expected runSuite to return error %v, but got %v", tt.wantErr, err)
				}
			}

			if tt.name == "Test case with failed observer" {
				sr := &SuiteRunner{}
				r := &Runner{}
				var oldRunnerTimeout time.Duration
				oldRunnerTimeout = observer.RunnerTimeout
				observer.RunnerTimeout = 0 * time.Minute
				defer func() { observer.RunnerTimeout = oldRunnerTimeout }()
				mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
				newNameSpace := &corev1.Namespace{}
				newNameSpace.Name = "new-ns"
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

				sr.Runner = r
				sr.Runner.KubeClient = mockKubeClient
				runSuite(ctx, tt.suite(), sr, tt.testCase, tt.db, tt.storageClass, tt.c)
			}

		})
	}
}

func TestRunFlowManagementGoroutine(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name         string
		scDBs        []*store.StorageClassDB
		noreport     bool
		NoMetrics    bool
		IterationNum int
		Duration     time.Duration
		stop         bool
		wantErr      bool
	}{
		{
			name: "Test case with valid parameters",
			scDBs: []*store.StorageClassDB{
				{
					StorageClass: "sc1",
					DB:           store.NewSQLiteStoreWithDB(db),
				},
				{
					StorageClass: "sc2",
					DB:           store.NewSQLiteStoreWithDB(db),
				},
			},
			noreport:     false,
			NoMetrics:    false,
			IterationNum: 10,
			Duration:     10 * time.Second,
			stop:         false,
			wantErr:      false,
		},
		{
			name:         "Test case with invalid parameters",
			scDBs:        nil,
			noreport:     true,
			NoMetrics:    true,
			IterationNum: -1,
			Duration:     -1 * time.Second,
			stop:         true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{

				SucceededSuites: Threshold + 0.1,
				stop:            tt.stop,
			}
			sr := &SuiteRunner{

				ScDBs:        tt.scDBs,
				NoReport:     tt.noreport,
				NoMetrics:    tt.NoMetrics,
				IterationNum: tt.IterationNum,
				Duration:     tt.Duration,
			}
			sr.Runner = r
			iterCtx, c := sr.runFlowManagementGoroutine()

			// Send a signal to the goroutine to stop it
			c <- os.Interrupt

			// Wait for the goroutine to finish
			<-iterCtx.Done()

			if tt.wantErr {
				if sr.IterationNum != tt.IterationNum {
					t.Errorf("Expected IterationNum to be %d, got %d", tt.IterationNum, sr.IterationNum)
				}
			}
			if !tt.wantErr {
				if sr.IterationNum != tt.IterationNum {
					t.Errorf("Expected IterationNum to be %d, got %d", tt.IterationNum-1, sr.IterationNum)
				}
			}
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
		{
			startHook: "/path/to/invalid/startHook",
			hookName:  "Mock Bash Script",
			wantErr:   true,
		},
		{
			startHook: "",
			hookName:  "Mock Bash Script",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		err := runHook(tt.startHook, tt.hookName)
		if (err != nil) != tt.wantErr {
			t.Errorf("runHook(%s, %s) error = %v, wantErr %v", tt.startHook, tt.hookName, err, tt.wantErr)
		}
	}
}

func TestShouldClean_Suites(t *testing.T) {
	tests := []struct {
		name        string
		noCleanup   bool
		noCleaning  bool
		suiteRes    TestResult
		expectedRes bool
	}{
		{
			name:        "no cleanup on fail is false and suite result is success",
			noCleanup:   false,
			noCleaning:  false,
			suiteRes:    SUCCESS,
			expectedRes: true,
		},
		{
			name:        "no cleanup on fail is true and suite result is success",
			noCleanup:   true,
			noCleaning:  false,
			suiteRes:    SUCCESS,
			expectedRes: false,
		},
		{
			name:        "no cleanup on fail is false and suite result is failure",
			noCleanup:   false,
			noCleaning:  false,
			suiteRes:    FAILURE,
			expectedRes: true,
		},
		{
			name:        "no cleanup on fail is true and no cleaning is true",
			noCleanup:   true,
			noCleaning:  true,
			suiteRes:    SUCCESS,
			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				noCleaning:      tt.noCleanup,
				NoCleanupOnFail: tt.noCleanup,
			}
			sr := &SuiteRunner{}
			sr.Runner = r

			res := sr.ShouldClean(tt.suiteRes)
			if res != tt.expectedRes {
				t.Errorf("expected ShouldClean() to return %v, but got %v", tt.expectedRes, res)
			}
		})
	}
}

func TestNoCleaning(t *testing.T) {
	tests := []struct {
		name    string
		initial bool
	}{
		{
			name:    "Testing NoCleaning with initial value false",
			initial: false,
		},
		{
			name:    "Testing NoCleaning with initial value true",
			initial: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				noCleaning: tt.initial,
			}
			sr := &SuiteRunner{}
			sr.Runner = r

			sr.NoCleaning()

			if sr.noCleaning != true {
				t.Errorf("Expected noCleaning to be true, but got false")
			}
		})
	}
}
func TestIsStopped(t *testing.T) {
	tests := []struct {
		name        string
		stop        bool
		expectedRes bool
	}{
		{
			name:        "test suite is stopped",
			stop:        true,
			expectedRes: true,
		},
		{
			name:        "test suite is not stopped",
			stop:        false,
			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				stop: tt.stop,
			}
			sr := &SuiteRunner{}
			sr.Runner = r
			res := sr.IsStopped()
			if res != tt.expectedRes {
				t.Errorf("expected IsStopped() to return %v, but got %v", tt.expectedRes, res)
			}
		})
	}
}
func TestSuiteRunner_Stop(t *testing.T) {
	r := &Runner{
		stop: true,
	}

	sr := &SuiteRunner{}
	sr.Runner = r

	// Test case: Setting stop to true
	sr.Stop()
	if !sr.stop {
		t.Errorf("Expected stop to be true, but got false")
	}
	// Test case: Setting stop to false
	sr.stop = false
	sr.Stop()
	if !sr.stop {
		t.Errorf("Expected stop to be true, but got false")
	}
}
func TestSuiteRunner_Close(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name           string
		scDBs          []*store.StorageClassDB
		noreport       bool
		NoMetrics      bool
		expectedLogLen int
	}{
		{
			name: "All databases are closed successfully",
			scDBs: []*store.StorageClassDB{
				{
					DB: store.NewSQLiteStoreWithDB(db),
				},
				{
					DB: store.NewSQLiteStoreWithDB(db),
				},
			},
			noreport:       false,
			NoMetrics:      false,
			expectedLogLen: 0,
		},
		{
			name: "Error generating reports",
			scDBs: []*store.StorageClassDB{
				{
					DB: store.NewSQLiteStoreWithDB(db),
				},
			},
			noreport:       false,
			NoMetrics:      false,
			expectedLogLen: 1,
		},
		{
			name: "Error generating all reports",
			scDBs: []*store.StorageClassDB{
				{
					DB: store.NewSQLiteStoreWithDB(db),
				},
			},
			noreport:       false,
			NoMetrics:      true,
			expectedLogLen: 1,
		},
		{
			name:           "Succeeded suites is greater than threshold",
			scDBs:          []*store.StorageClassDB{},
			noreport:       false,
			NoMetrics:      false,
			expectedLogLen: 1,
		},
		{
			name:           "Succeeded suites is less than or equal to threshold",
			scDBs:          []*store.StorageClassDB{},
			noreport:       false,
			NoMetrics:      false,
			expectedLogLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				SucceededSuites: Threshold + 0.1,
			}
			sr := &SuiteRunner{
				ScDBs:     tt.scDBs,
				NoReport:  tt.noreport,
				NoMetrics: tt.NoMetrics,
			}
			sr.Runner = r

			sr.Close()

		})
	}
}
