// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/dell/cert-csi/pkg/k8sclient (interfaces: KubeClientInterface)
//
// Generated by this command:
//
//	mockgen -destination=mocks/kubeclientinterface.go -package=mocks github.com/dell/cert-csi/pkg/k8sclient KubeClientInterface
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	csistoragecapacity "github.com/dell/cert-csi/pkg/k8sclient/resources/csistoragecapacity"
	metrics "github.com/dell/cert-csi/pkg/k8sclient/resources/metrics"
	pod "github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	pvc "github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	replicationgroup "github.com/dell/cert-csi/pkg/k8sclient/resources/replicationgroup"
	sc "github.com/dell/cert-csi/pkg/k8sclient/resources/sc"
	statefulset "github.com/dell/cert-csi/pkg/k8sclient/resources/statefulset"
	va "github.com/dell/cert-csi/pkg/k8sclient/resources/va"
	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
)

// MockKubeClientInterface is a mock of KubeClientInterface interface.
type MockKubeClientInterface struct {
	ctrl     *gomock.Controller
	recorder *MockKubeClientInterfaceMockRecorder
	isgomock struct{}
}

// MockKubeClientInterfaceMockRecorder is the mock recorder for MockKubeClientInterface.
type MockKubeClientInterfaceMockRecorder struct {
	mock *MockKubeClientInterface
}

// NewMockKubeClientInterface creates a new mock instance.
func NewMockKubeClientInterface(ctrl *gomock.Controller) *MockKubeClientInterface {
	mock := &MockKubeClientInterface{ctrl: ctrl}
	mock.recorder = &MockKubeClientInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKubeClientInterface) EXPECT() *MockKubeClientInterfaceMockRecorder {
	return m.recorder
}

// CreateCSISCClient mocks base method.
func (m *MockKubeClientInterface) CreateCSISCClient(namespace string) (*csistoragecapacity.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCSISCClient", namespace)
	ret0, _ := ret[0].(*csistoragecapacity.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCSISCClient indicates an expected call of CreateCSISCClient.
func (mr *MockKubeClientInterfaceMockRecorder) CreateCSISCClient(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCSISCClient", reflect.TypeOf((*MockKubeClientInterface)(nil).CreateCSISCClient), namespace)
}

// CreateMetricsClient mocks base method.
func (m *MockKubeClientInterface) CreateMetricsClient(namespace string) (*metrics.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateMetricsClient", namespace)
	ret0, _ := ret[0].(*metrics.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateMetricsClient indicates an expected call of CreateMetricsClient.
func (mr *MockKubeClientInterfaceMockRecorder) CreateMetricsClient(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateMetricsClient", reflect.TypeOf((*MockKubeClientInterface)(nil).CreateMetricsClient), namespace)
}

// CreateNamespace mocks base method.
func (m *MockKubeClientInterface) CreateNamespace(namespace string) (*v1.Namespace, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNamespace", namespace)
	ret0, _ := ret[0].(*v1.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNamespace indicates an expected call of CreateNamespace.
func (mr *MockKubeClientInterfaceMockRecorder) CreateNamespace(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNamespace", reflect.TypeOf((*MockKubeClientInterface)(nil).CreateNamespace), namespace)
}

// CreateNamespaceWithSuffix mocks base method.
func (m *MockKubeClientInterface) CreateNamespaceWithSuffix(namespace string) (*v1.Namespace, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNamespaceWithSuffix", namespace)
	ret0, _ := ret[0].(*v1.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNamespaceWithSuffix indicates an expected call of CreateNamespaceWithSuffix.
func (mr *MockKubeClientInterfaceMockRecorder) CreateNamespaceWithSuffix(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNamespaceWithSuffix", reflect.TypeOf((*MockKubeClientInterface)(nil).CreateNamespaceWithSuffix), namespace)
}

// CreatePVCClient mocks base method.
func (m *MockKubeClientInterface) CreatePVCClient(namespace string) (*pvc.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePVCClient", namespace)
	ret0, _ := ret[0].(*pvc.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreatePVCClient indicates an expected call of CreatePVCClient.
func (mr *MockKubeClientInterfaceMockRecorder) CreatePVCClient(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePVCClient", reflect.TypeOf((*MockKubeClientInterface)(nil).CreatePVCClient), namespace)
}

// CreatePodClient mocks base method.
func (m *MockKubeClientInterface) CreatePodClient(namespace string) (*pod.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePodClient", namespace)
	ret0, _ := ret[0].(*pod.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreatePodClient indicates an expected call of CreatePodClient.
func (mr *MockKubeClientInterfaceMockRecorder) CreatePodClient(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePodClient", reflect.TypeOf((*MockKubeClientInterface)(nil).CreatePodClient), namespace)
}

// CreateRGClient mocks base method.
func (m *MockKubeClientInterface) CreateRGClient(namespace string) (*replicationgroup.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRGClient", namespace)
	ret0, _ := ret[0].(*replicationgroup.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRGClient indicates an expected call of CreateRGClient.
func (mr *MockKubeClientInterfaceMockRecorder) CreateRGClient(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRGClient", reflect.TypeOf((*MockKubeClientInterface)(nil).CreateRGClient), namespace)
}

// CreateSCClient mocks base method.
func (m *MockKubeClientInterface) CreateSCClient(namespace string) (*sc.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSCClient", namespace)
	ret0, _ := ret[0].(*sc.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSCClient indicates an expected call of CreateSCClient.
func (mr *MockKubeClientInterfaceMockRecorder) CreateSCClient(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSCClient", reflect.TypeOf((*MockKubeClientInterface)(nil).CreateSCClient), namespace)
}

// CreateStatefulSetClient mocks base method.
func (m *MockKubeClientInterface) CreateStatefulSetClient(namespace string) (*statefulset.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateStatefulSetClient", namespace)
	ret0, _ := ret[0].(*statefulset.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateStatefulSetClient indicates an expected call of CreateStatefulSetClient.
func (mr *MockKubeClientInterfaceMockRecorder) CreateStatefulSetClient(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateStatefulSetClient", reflect.TypeOf((*MockKubeClientInterface)(nil).CreateStatefulSetClient), namespace)
}

// CreateVaClient mocks base method.
func (m *MockKubeClientInterface) CreateVaClient(namespace string) (*va.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVaClient", namespace)
	ret0, _ := ret[0].(*va.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVaClient indicates an expected call of CreateVaClient.
func (mr *MockKubeClientInterfaceMockRecorder) CreateVaClient(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVaClient", reflect.TypeOf((*MockKubeClientInterface)(nil).CreateVaClient), namespace)
}

// DeleteNamespace mocks base method.
func (m *MockKubeClientInterface) DeleteNamespace(namespace string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNamespace", namespace)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNamespace indicates an expected call of DeleteNamespace.
func (mr *MockKubeClientInterfaceMockRecorder) DeleteNamespace(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNamespace", reflect.TypeOf((*MockKubeClientInterface)(nil).DeleteNamespace), namespace)
}

// NamespaceExists mocks base method.
func (m *MockKubeClientInterface) NamespaceExists(namespace string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NamespaceExists", namespace)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NamespaceExists indicates an expected call of NamespaceExists.
func (mr *MockKubeClientInterfaceMockRecorder) NamespaceExists(namespace any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NamespaceExists", reflect.TypeOf((*MockKubeClientInterface)(nil).NamespaceExists), namespace)
}

// StorageClassExists mocks base method.
func (m *MockKubeClientInterface) StorageClassExists(storageClass string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StorageClassExists", storageClass)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StorageClassExists indicates an expected call of StorageClassExists.
func (mr *MockKubeClientInterfaceMockRecorder) StorageClassExists(storageClass any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StorageClassExists", reflect.TypeOf((*MockKubeClientInterface)(nil).StorageClassExists), storageClass)
}

// Timeout mocks base method.
func (m *MockKubeClientInterface) Timeout() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Timeout")
	ret0, _ := ret[0].(int)
	return ret0
}

// Timeout indicates an expected call of Timeout.
func (mr *MockKubeClientInterfaceMockRecorder) Timeout() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Timeout", reflect.TypeOf((*MockKubeClientInterface)(nil).Timeout))
}
