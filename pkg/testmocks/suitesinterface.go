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

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/dell/cert-csi/pkg/testcore/suites (interfaces: Interface)
//
// Generated by this command:
//
//	mockgen -destination=mocks/suitesinterface.go -package=mocks github.com/dell/cert-csi/pkg/testcore/suites Interface
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	k8sclient "github.com/dell/cert-csi/pkg/k8sclient"
	observer "github.com/dell/cert-csi/pkg/observer"
	gomock "go.uber.org/mock/gomock"
)

// MockInterface is a mock of Interface interface.
type MockInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceMockRecorder
	isgomock struct{}
}

// MockInterfaceMockRecorder is the mock recorder for MockInterface.
type MockInterfaceMockRecorder struct {
	mock *MockInterface
}

// NewMockInterface creates a new mock instance.
func NewMockInterface(ctrl *gomock.Controller) *MockInterface {
	mock := &MockInterface{ctrl: ctrl}
	mock.recorder = &MockInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterface) EXPECT() *MockInterfaceMockRecorder {
	return m.recorder
}

// GetClients mocks base method.
func (m *MockInterface) GetClients(arg0 string, arg1 k8sclient.KubeClientInterface) (*k8sclient.Clients, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClients", arg0, arg1)
	ret0, _ := ret[0].(*k8sclient.Clients)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClients indicates an expected call of GetClients.
func (mr *MockInterfaceMockRecorder) GetClients(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClients", reflect.TypeOf((*MockInterface)(nil).GetClients), arg0, arg1)
}

// GetName mocks base method.
func (m *MockInterface) GetName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetName indicates an expected call of GetName.
func (mr *MockInterfaceMockRecorder) GetName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetName", reflect.TypeOf((*MockInterface)(nil).GetName))
}

// GetNamespace mocks base method.
func (m *MockInterface) GetNamespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNamespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetNamespace indicates an expected call of GetNamespace.
func (mr *MockInterfaceMockRecorder) GetNamespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNamespace", reflect.TypeOf((*MockInterface)(nil).GetNamespace))
}

// GetObservers mocks base method.
func (m *MockInterface) GetObservers(obsType observer.Type) []observer.Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetObservers", obsType)
	ret0, _ := ret[0].([]observer.Interface)
	return ret0
}

// GetObservers indicates an expected call of GetObservers.
func (mr *MockInterfaceMockRecorder) GetObservers(obsType any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetObservers", reflect.TypeOf((*MockInterface)(nil).GetObservers), obsType)
}

// Parameters mocks base method.
func (m *MockInterface) Parameters() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Parameters")
	ret0, _ := ret[0].(string)
	return ret0
}

// Parameters indicates an expected call of Parameters.
func (mr *MockInterfaceMockRecorder) Parameters() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Parameters", reflect.TypeOf((*MockInterface)(nil).Parameters))
}

// Run mocks base method.
func (m *MockInterface) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (func() error, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", ctx, storageClass, clients)
	ret0, _ := ret[0].(func() error)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Run indicates an expected call of Run.
func (mr *MockInterfaceMockRecorder) Run(ctx, storageClass, clients any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockInterface)(nil).Run), ctx, storageClass, clients)
}
