/* package runner

import (
	"context"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes/fake"
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

func (suite *FunctionalSuiteRunnerTestSuite)
() {
	mockSuite := new(MockSuite)
	mockSuite.On("GetName").Return("Mock Suite")
	mockSuite.On("GetNamespace").Return("mock-namespace")
	mockSuite.On("GetClients", mock.Anything, mock.Anything).Return(&k8sclient.Clients{}, nil)
	mockSuite.On("GetObservers", mock.Anything).Return([]observer.Interface{})
	mockSuite.On("Run", mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, storageClass string, clients *k8sclient.Clients) (func() error, error) {
		return func() error {
			return nil
		}, nil
	}, nil)
	mockSuite.On("Parameters").Return("")

	suites := []suites.Interface{mockSuite}

	suite.runner.RunFunctionalSuites(suites)

	assert.Equal(suite.T(), 1.0, suite.runner.SucceededSuites, "Expected all suites to succeed")

}

func (suite *FunctionalSuiteRunnerTestSuite) TestIsStopped() {
	suite.runner.Stop()
	assert.True(suite.T(), suite.runner.IsStopped(), "Expected runner to be stopped")
}

func (suite *FunctionalSuiteRunnerTestSuite) TestNoCleaning() {
	suite.runner.NoCleaning()
	assert.True(suite.T(), suite.runner.noCleaning, "Expected noCleaning to be true")
}

func (suite *FunctionalSuiteRunnerTestSuite) TestShouldClean() {
	assert.True(suite.T(), suite.runner.ShouldClean(SUCCESS), "Expected ShouldClean to return true for SUCCESS")
	assert.False(suite.T(), suite.runner.ShouldClean(FAILURE), "Expected ShouldClean to return false for FAILURE")
}

func TestFunctionalSuiteRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuiteRunnerTestSuite))
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
}*/
