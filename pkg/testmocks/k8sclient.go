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
// Source: github.com/dell/cert-csi/pkg/testcore/runner (interfaces: K8sClientInterface)
//
// Generated by this command:
//
//	mockgen -destination=mocks/k8sclient.go -package=mocks github.com/dell/cert-csi/pkg/testcore/runner K8sClientInterface
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	k8sclient "github.com/dell/cert-csi/pkg/k8sclient"
	gomock "go.uber.org/mock/gomock"
	rest "k8s.io/client-go/rest"
)

// MockK8sClientInterface is a mock of K8sClientInterface interface.
type MockK8sClientInterface struct {
	ctrl     *gomock.Controller
	recorder *MockK8sClientInterfaceMockRecorder
	isgomock struct{}
}

// MockK8sClientInterfaceMockRecorder is the mock recorder for MockK8sClientInterface.
type MockK8sClientInterfaceMockRecorder struct {
	mock *MockK8sClientInterface
}

// NewMockK8sClientInterface creates a new mock instance.
func NewMockK8sClientInterface(ctrl *gomock.Controller) *MockK8sClientInterface {
	mock := &MockK8sClientInterface{ctrl: ctrl}
	mock.recorder = &MockK8sClientInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockK8sClientInterface) EXPECT() *MockK8sClientInterfaceMockRecorder {
	return m.recorder
}

// GetConfig mocks base method.
func (m *MockK8sClientInterface) GetConfig(arg0 string) (*rest.Config, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConfig", arg0)
	ret0, _ := ret[0].(*rest.Config)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConfig indicates an expected call of GetConfig.
func (mr *MockK8sClientInterfaceMockRecorder) GetConfig(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfig", reflect.TypeOf((*MockK8sClientInterface)(nil).GetConfig), arg0)
}

// NewKubeClient mocks base method.
func (m *MockK8sClientInterface) NewKubeClient(config *rest.Config, timeout int) (k8sclient.KubeClientInterface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewKubeClient", config, timeout)
	ret0, _ := ret[0].(k8sclient.KubeClientInterface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewKubeClient indicates an expected call of NewKubeClient.
func (mr *MockK8sClientInterfaceMockRecorder) NewKubeClient(config, timeout any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewKubeClient", reflect.TypeOf((*MockK8sClientInterface)(nil).NewKubeClient), config, timeout)
}
