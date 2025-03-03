package runner

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/mocks"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/store"
	storemocks "github.com/dell/cert-csi/pkg/store/mocks"
	runnermocks "github.com/dell/cert-csi/pkg/testcore/runner/mocks"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

type FunctionalSuiteRunnerTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
	runner     *FunctionalSuiteRunner
}

func (suite *FunctionalSuiteRunnerTestSuite) SetupSuite() {
	client := fake.NewSimpleClientset()

	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)

	scDB := &store.StorageClassDB{}
	suite.runner = &FunctionalSuiteRunner{
		Runner: &Runner{
			KubeClient: suite.kubeClient,
		},
		ScDB: scDB,
	}
}

func TestNewFunctionalSuiteRunner(t *testing.T) {
	mock_kube := mocks.NewMockKubeClientInterface(gomock.NewController(t))
	mock_kube.EXPECT().StorageClassExists(gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)
	mock_kube.EXPECT().NamespaceExists(gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)
	mock := runnermocks.NewMockK8sClientInterface(gomock.NewController(t))
	mock.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(&rest.Config{
		Host: "localhost",
	}, nil)
	mock.EXPECT().NewKubeClient(gomock.Any(), gomock.Any()).AnyTimes().Return(mock_kube, nil)

	mockScDB := &store.StorageClassDB{
		StorageClass: "sc1",
	}
	mockConfigPath := "config.yaml"
	mockNamespace := "namespace"
	mockTimeout := 30
	mockNoCleanup := true
	mockNoCleanupOnFail := true
	mockNoreport := true

	result := NewFunctionalSuiteRunner(
		mockConfigPath,
		mockNamespace,
		mockTimeout,
		mockNoCleanup,
		mockNoCleanupOnFail,
		mockNoreport,
		mockScDB,
		mock,
	)

	// Assert the result
	if result.DriverNamespace != mockNamespace {
		t.Errorf("Expected DriverNamespace to be %s, got %s", mockNamespace, result.DriverNamespace)
	}
	if result.Timeout != mockTimeout {
		t.Errorf("Expected Timeout to be %d, got %d", mockTimeout, result.Timeout)
	}
	if result.NoCleanupOnFail != mockNoCleanupOnFail {
		t.Errorf("Expected NoCleanupOnFail to be %t, got %t", mockNoCleanupOnFail, result.NoCleanupOnFail)
	}
	if result.noreport != mockNoreport {
		t.Errorf("Expected noreport to be %t, got %t", mockNoreport, result.noreport)
	}
	if result.ScDB != mockScDB {
		t.Errorf("Expected ScDB to be %v, got %v", mockScDB, result.ScDB)
	}
}

func TestRunFunctionalSuites(t *testing.T) {
	tests := []struct {
		name                    string
		suites                  func() []suites.Interface
		expectedSucceededSuites float64
		tr                      *store.TestRun
	}{
		{
			name: "Successful test run",
			suites: func() []suites.Interface {
				suite := runnermocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run 1")
				suite.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				return []suites.Interface{suite}
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
			mockRunner := runnermocks.NewMockInterface(gomock.NewController(t))
			mockStore := storemocks.NewMockStore(gomock.NewController(t))

			fr := &FunctionalSuiteRunner{
				ScDB: &store.StorageClassDB{
					StorageClass: "sc1",
					DB:           mockStore,
					TestRun:      *tt.tr,
				},
			}

			mockKubeClient := mocks.NewMockKubeClientInterface(gomock.NewController(t))
			newNameSpace := &corev1.Namespace{}
			newNameSpace.Name = "new-ns"
			mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), "test-namespace").AnyTimes().Return(newNameSpace, nil)
			mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockRunner.EXPECT().GetName().AnyTimes().Return("test run 1")
			mockRunner.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
			mockRunner.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
			mockRunner.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
			mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
			mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
			mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

			r := &Runner{
				Config:     &rest.Config{},
				KubeClient: mockKubeClient,
			}
			fr.Runner = r

			fr.RunFunctionalSuites(tt.suites())
		})
	}
}

func (suite *FunctionalSuiteRunnerTestSuite) TestIsStopped() {
	suite.runner.Stop()
	assert.True(suite.T(), suite.runner.IsStopped(), "Expected runner to be stopped")
}

func (suite *FunctionalSuiteRunnerTestSuite) TestNoCleaning() {
	suite.runner.NoCleaning()
	assert.True(suite.T(), suite.runner.noCleaning, "Expected noCleaning to be true")
}

// todo: fix
func (suite *FunctionalSuiteRunnerTestSuite) TestShouldClean() {
	assert.True(suite.T(), suite.runner.ShouldClean(SUCCESS), "Expected ShouldClean to return true for SUCCESS")
	//assert.False(suite.T(), suite.runner.ShouldClean(FAILURE), "Expected ShouldClean to return false for FAILURE")
}

// MockSuite is a mock implementation of the suites.Interface for testing purposes
type MockSuite struct {
	mock.Mock
}

func (m *MockSuite) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSuite) GetNamespace() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSuite) GetClients(namespace string, kubeClient k8sclient.KubeClientInterface) (*k8sclient.Clients, error) {
	args := m.Called(namespace, kubeClient)
	return args.Get(0).(*k8sclient.Clients), args.Error(1)
}

func (m *MockSuite) GetObservers(observerType observer.Type) []observer.Interface {
	args := m.Called(observerType)
	return args.Get(0).([]observer.Interface)
}

func (m *MockSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (func() error, error) {
	args := m.Called(ctx, storageClass, clients)
	return args.Get(0).(func() error), args.Error(1)
}

func (m *MockSuite) Parameters() string {
	args := m.Called()
	return args.String(0)
}
