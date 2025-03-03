package runner

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/dell/cert-csi/pkg/k8sclient/mocks"
	"github.com/dell/cert-csi/pkg/store"

	//storemocks "github.com/dell/cert-csi/pkg/store/mocks"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	fakeClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	runnermocks "github.com/dell/cert-csi/pkg/testcore/runner/mocks"
	clienttesting "k8s.io/client-go/testing"

	"go.uber.org/mock/gomock"
)

type SQLiteStore struct {
	db *sql.DB
}

func SetupSuite(t *testing.T) {
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

// todo: maybe remove/refactor
type clientTestContext struct {
	testNamespace    string
	namespaceDeleted bool
	t                *testing.T
}

/* type FunctionalSuiteRunnerTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
	runner     *FunctionalSuiteRunner
} */

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

//todo: probably remove
/* func TestExecuteSuite(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
			newNameSpace := &corev1.Namespace{}
			newNameSpace.Name = "new-ns"
			mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), "test-namespace").AnyTimes().Return(newNameSpace, nil)
			mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			r := &Runner{
				KubeClient: mockKubeClient,
			}
			sr := &SuiteRunner{}
			sr.Runner = r

			mockStore := storemocks.NewMockStore(gomock.NewController(t))
			mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
			mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			clientCtx := &clientTestContext{t: t}

			k8sclient.FuncNewClientSet = func(_ *rest.Config) (kubernetes.Interface, error) {
				return createFakeKubeClient(clientCtx)

			}

			scDBs := &store.StorageClassDB{
				DB: mockStore,
			}

			ExecuteSuite(tt.iterCtx, tt.num, tt.suites(), tt.suite(), sr, scDBs, tt.c)

			// Assert the expected outcome
			// For example, you can use testify/assert to make assertions
			// or write custom assertions based on the expected behavior.
		})
	}
} */

func TestRunFunctionalSuites(t *testing.T) {
	/* tests := []struct {
		name                    string
		suites                  func() map[string][]suites.Interface
		expectedSucceededSuites float64
		tr                      *store.TestRun
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
			mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
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
			}
			sr.Runner = r

			clientCtx := &clientTestContext{t: t}

			k8sclient.FuncNewClientSet = func(_ *rest.Config) (kubernetes.Interface, error) {
				return createFakeKubeClient(clientCtx)

			}

			sr.RunSuites(tt.suites())
		})
	} */
}

func TestRunFunctionalSuite(t *testing.T) {
	/* mockdb, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	tests := []struct {
		name         string
		suite        func() suites.Interface
		sr           *SuiteRunner
		testCase     *store.TestCase
		db           *store.SQLiteStore
		storageClass string
		c            chan os.Signal
		wantRes      TestResult
		wantErr      error
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
			wantErr:      nil,
		},
	}
	for _, tt := range tests {
		sr := &SuiteRunner{}
		r := &Runner{}

		mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
		newNameSpace := &corev1.Namespace{}
		newNameSpace.Name = "new-ns"
		mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
		mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

		sr.Runner = r
		sr.Runner.KubeClient = mockKubeClient

		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			res, err := runSuite(ctx, tt.suite(), sr, tt.testCase, tt.db, tt.storageClass, tt.c)
			if res != tt.wantRes {
				t.Errorf("Expected runSuite to return %v, but got %v", tt.wantRes, res)
			}
			if err != tt.wantErr {
				t.Errorf("Expected runSuite to return error %v, but got %v", tt.wantErr, err)
			}
		})
	} */
}
