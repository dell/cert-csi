/*
 *
 * Copyright © 2025 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	mocks "github.com/dell/cert-csi/pkg/testmocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"k8s.io/client-go/kubernetes/fake"
	fakeClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
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
	mockKube := mocks.NewMockKubeClientInterface(gomock.NewController(t))
	mockKube.EXPECT().StorageClassExists(gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)
	mockKube.EXPECT().NamespaceExists(gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)
	mock := mocks.NewMockK8sClientInterface(gomock.NewController(t))
	mock.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(&rest.Config{
		Host: "localhost",
	}, nil)
	mock.EXPECT().NewKubeClient(gomock.Any(), gomock.Any()).AnyTimes().Return(mockKube, nil)

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
	client := fakeClient.NewSimpleClientset()
	client.Fake.PrependReactor("*", "*", func(_ clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("error listing volumes")
	})

	tests := []struct {
		name                    string
		suites                  func() []suites.Interface
		expectedSucceededSuites float64
		tr                      *store.TestRun
		wantErr                 bool
	}{
		{
			name: "Successful test run",
			suites: func() []suites.Interface {
				suite := mocks.NewMockInterface(gomock.NewController(t))
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
			wantErr: false,
		},
		{
			name: "Obs error test run",
			suites: func() []suites.Interface {
				suite := mocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run - obs error")
				suite.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{
					PVCClient: &pvc.Client{
						ClientSet: client,
					},
				}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				return []suites.Interface{suite}
			},

			tr: &store.TestRun{
				Name:           "test run - obs error",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			wantErr: true,
		},
		{
			name: "No namespace error test run",
			suites: func() []suites.Interface {
				suite := mocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run 2")
				suite.EXPECT().GetNamespace().AnyTimes().Return("")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				return []suites.Interface{suite}
			},

			tr: &store.TestRun{
				Name:           "test run 2",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			wantErr: true,
		},
		{
			name: "Delete namespace error test run",
			suites: func() []suites.Interface {
				suite := mocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run 3")
				suite.EXPECT().GetNamespace().AnyTimes().Return("")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				return []suites.Interface{suite}
			},

			tr: &store.TestRun{
				Name:           "test run 3",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			wantErr: true,
		},
		{
			name: "Save test case error test run",
			suites: func() []suites.Interface {
				suite := mocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run 4")
				suite.EXPECT().GetNamespace().AnyTimes().Return("")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				return []suites.Interface{suite}
			},

			tr: &store.TestRun{
				Name:           "test run 4",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			wantErr: true,
		},
		{
			name: "Save test run error test run",
			suites: func() []suites.Interface {
				suite := mocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run 5")
				suite.EXPECT().GetNamespace().AnyTimes().Return("")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				return []suites.Interface{suite}
			},

			tr: &store.TestRun{
				Name:           "test run 5",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			wantErr: true,
		},
		{
			name: "Successful test run error test run",
			suites: func() []suites.Interface {
				suite := mocks.NewMockInterface(gomock.NewController(t))
				suite.EXPECT().GetName().AnyTimes().Return("test run 5")
				suite.EXPECT().GetNamespace().AnyTimes().Return("")
				suite.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				suite.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				suite.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				return []suites.Interface{suite}
			},

			tr: &store.TestRun{
				Name:           "test run 5",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := mocks.NewMockInterface(gomock.NewController(t))
			mockStore := mocks.NewMockStore(gomock.NewController(t))

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
			if tt.name == "Successful test run" {
				mockRunner.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				mockRunner.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				mockRunner.EXPECT().GetName().AnyTimes().Return("test run 1")
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), "test-namespace").AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockRunner.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
			}
			if tt.name == "Obs error test run" {
				mockRunner.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				mockRunner.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().FailedTestCase(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				mockRunner.EXPECT().GetName().AnyTimes().Return("test run - obs error")
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockRunner.EXPECT().GetNamespace().AnyTimes().Return("test-namespace")
				oldRunnerTimeout := observer.RunnerTimeout
				observer.RunnerTimeout = 0 * time.Minute
				defer func() { observer.RunnerTimeout = oldRunnerTimeout }()
			}
			if tt.name == "No namespace error test run" {
				mockRunner.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				mockRunner.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().FailedTestCase(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				mockRunner.EXPECT().GetName().AnyTimes().Return("test run 2")
				mockRunner.EXPECT().GetNamespace().AnyTimes().Return("")
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, fmt.Errorf("Mock Error"))
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(fmt.Errorf("Mock Error"))
				r := &Runner{
					Config:     &rest.Config{},
					KubeClient: mockKubeClient,
				}
				fr.Runner = r

				fr.RunFunctionalSuites(tt.suites())
			}
			if tt.name == "Delete namespace error test run" {
				mockRunner.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				mockRunner.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().FailedTestCase(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				mockRunner.EXPECT().GetName().AnyTimes().Return("test run 3")
				mockRunner.EXPECT().GetNamespace().AnyTimes().Return("")
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(fmt.Errorf("Mock Error"))
				r := &Runner{
					Config:     &rest.Config{},
					KubeClient: mockKubeClient,
				}
				fr.Runner = r

				fr.RunFunctionalSuites(tt.suites())
			}
			if tt.name == "Save test case error test run" {
				mockRunner.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				mockRunner.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(fmt.Errorf("new error"))
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				mockRunner.EXPECT().GetName().AnyTimes().Return("test run 4")
				mockRunner.EXPECT().GetNamespace().AnyTimes().Return("")
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				r := &Runner{
					Config:     &rest.Config{},
					KubeClient: mockKubeClient,
				}
				fr.Runner = r

				fr.RunFunctionalSuites(tt.suites())
			}
			if tt.name == "Save test run error test run" {
				mockRunner.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				mockRunner.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(fmt.Errorf("new error"))
				mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				mockRunner.EXPECT().GetName().AnyTimes().Return("test run 5")
				mockRunner.EXPECT().GetNamespace().AnyTimes().Return("")
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				r := &Runner{
					Config:     &rest.Config{},
					KubeClient: mockKubeClient,
				}
				fr.Runner = r

				fr.RunFunctionalSuites(tt.suites())
			}
			if tt.name == "Successful test run error test run" {
				mockRunner.EXPECT().GetClients(gomock.Any(), gomock.Any()).AnyTimes().Return(&k8sclient.Clients{}, nil)
				mockRunner.EXPECT().GetObservers(gomock.Any()).AnyTimes().Return([]observer.Interface{})
				mockStore.EXPECT().SaveTestCase(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SaveTestRun(gomock.Any()).AnyTimes().Return(nil)
				mockStore.EXPECT().SuccessfulTestCase(gomock.Any(), gomock.Any()).AnyTimes().Return(fmt.Errorf("new error"))
				mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
				mockRunner.EXPECT().GetName().AnyTimes().Return("test run 6")
				mockRunner.EXPECT().GetNamespace().AnyTimes().Return("")
				mockKubeClient.EXPECT().CreateNamespaceWithSuffix(gomock.Any(), gomock.Any()).AnyTimes().Return(newNameSpace, nil)
				mockKubeClient.EXPECT().DeleteNamespace(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				r := &Runner{
					Config:     &rest.Config{},
					KubeClient: mockKubeClient,
				}
				fr.Runner = r

				fr.RunFunctionalSuites(tt.suites())
			}
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
	assert.False(suite.T(), suite.runner.ShouldClean(FAILURE), "Expected ShouldClean to return false for FAILURE")
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
