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

package observer

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

type mockObserver struct {
	mock.Mock
}

func (m *mockObserver) StartWatching(ctx context.Context, runner *Runner) {
	m.Called(ctx, runner)
}

func (m *mockObserver) StopWatching() {
	m.Called()
}

func (m *mockObserver) GetName() string {
	return m.Called().String(0)
}

func (m *mockObserver) MakeChannel() {
	m.Called()
}

func TestRunner_Start(t *testing.T) {
	// Test case: Start watching all the runners
	ctx := context.Background()

	observer1 := &mockObserver{}
	observer2 := &mockObserver{}

	observers := []Interface{observer1, observer2}

	runner := &Runner{
		Observers: observers,
	}

	observer1.On("MakeChannel").Return()
	observer1.On("StartWatching", ctx, runner).Return()
	observer2.On("MakeChannel").Return()
	observer2.On("StartWatching", ctx, runner).Return()

	err := runner.Start(ctx)

	assert.NoError(t, err)
}

func TestRunner_Stop(t *testing.T) {
	// Test case: Stop watching all the runners and delete PVCs
	observer1 := &mockObserver{}
	observer2 := &mockObserver{}
	observers := []Interface{observer1, observer2}

	clients := &k8sclient.Clients{}
	db := NewSimpleStore()
	testCase := &store.TestCase{
		ID: 1,
	}
	driverNs := "driver-namespace"
	shouldClean := true

	runner := &Runner{
		Observers:       observers,
		Clients:         clients,
		Database:        db,
		TestCase:        testCase,
		DriverNamespace: driverNs,
		ShouldClean:     shouldClean,
	}

	observer1.On("StopWatching").Return()
	observer2.On("StopWatching").Return()

	err := runner.Stop()

	assert.NoError(t, err)
	observer1.AssertExpectations(t)
	observer2.AssertExpectations(t)
}

func TestRunner_WaitTimeout(t *testing.T) {
	// Test case: Wait for all of observers to complete
	observer1 := &mockObserver{}
	observer2 := &mockObserver{}
	observers := []Interface{observer1, observer2}

	clients := &k8sclient.Clients{}
	db := NewSimpleStore()
	testCase := &store.TestCase{
		ID: 1,
	}
	driverNs := "driver-namespace"
	shouldClean := true

	runner := &Runner{
		Observers:       observers,
		Clients:         clients,
		Database:        db,
		TestCase:        testCase,
		DriverNamespace: driverNs,
		ShouldClean:     shouldClean,
	}

	timeout := time.Duration(5 * time.Second)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		time.Sleep(3 * time.Second)
		wg.Done()
	}()

	result := runner.waitTimeout(timeout)

	wg.Wait()
	assert.False(t, result)
	observer1.AssertExpectations(t)
	observer2.AssertExpectations(t)
}

func TestRunner_GetName(t *testing.T) {
	// Test case: Get name of the Runner
	observer := &mockObserver{}
	observer.On("GetName").Return("MockObserver")

	result := observer.GetName()

	assert.Equal(t, "MockObserver", result)
	observer.AssertExpectations(t)
}

func TestRunner_MakeChannel(t *testing.T) {
	// Test case: Create a new channel
	observer := &mockObserver{}
	observer.On("MakeChannel").Return()

	observer.MakeChannel()

	observer.AssertExpectations(t)
}

func TestRunner_NewObserverRunner(t *testing.T) {
	// Test case: Create a new Runner instance
	observers := []Interface{}
	clients := &k8sclient.Clients{}
	db := NewSimpleStore()
	testCase := &store.TestCase{
		ID: 1,
	}
	driverNs := "test-driver-namespace"
	shouldClean := true

	runner := NewObserverRunner(observers, clients, db, testCase, driverNs, shouldClean)

	// Assert that the Runner instance has the correct values for its fields
	if !reflect.DeepEqual(runner.Observers, observers) {
		t.Errorf("Expected Observers to be %v, got %v", observers, runner.Observers)
	}
	if runner.Clients != clients {
		t.Errorf("Expected Clients to be %v, got %v", clients, runner.Clients)
	}
	if runner.Database != db {
		t.Errorf("Expected Database to be %v, got %v", db, runner.Database)
	}
	if runner.TestCase != testCase {
		t.Errorf("Expected TestCase to be %v, got %v", testCase, runner.TestCase)
	}
	if runner.DriverNamespace != driverNs {
		t.Errorf("Expected DriverNamespace to be %v, got %v", driverNs, runner.DriverNamespace)
	}
	if runner.ShouldClean != shouldClean {
		t.Errorf("Expected ShouldClean to be %v, got %v", shouldClean, runner.ShouldClean)
	}
}

func TestRunner_WaitTimeout_Timeout(t *testing.T) {
	// Test case: timeout is reached
	runner := &Runner{
		WaitGroup: sync.WaitGroup{},
	}
	runner.WaitGroup.Add(10)
	go func() {
		time.Sleep(1 * time.Second)
		runner.WaitGroup.Done()
	}()
	if !runner.waitTimeout(500 * time.Millisecond) {
		t.Errorf("Expected timeout to be reached")
	}
}

func TestRunner_Stop_WaitTimeout(t *testing.T) {
	// Test case: Stop watching all the runners and delete PVCs
	ctx := context.Background()

	observer1 := &mockObserver{}
	observer2 := &mockObserver{}
	observers := []Interface{observer1, observer2}

	db := NewSimpleStore()
	testCase := &store.TestCase{
		ID: 1,
	}
	driverNs := "driver-namespace"
	shouldClean := true

	clientSet := fake.NewSimpleClientset()

	// Create a fake PV
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
	}
	clientSet.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})

	clientSet.CoreV1().PersistentVolumeClaims("test-namespace").Create(ctx, &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "test-pvc", Namespace: "test-namespace"}}, metav1.CreateOptions{})
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: pvcClient,
		},
		Observers:       observers,
		Database:        db,
		TestCase:        testCase,
		PvcShare:        sync.Map{},
		DriverNamespace: driverNs,
		ShouldClean:     shouldClean,
	}

	entity := &store.Entity{}
	runner.PvcShare.Store("test-pv", entity)

	runner.WaitGroup.Add(1)

	observer1.On("StopWatching").Return()
	observer2.On("StopWatching").Return()

	err := runner.Stop()

	expectedError := errors.New("pvs are in hanging state, something's wrong")
	if err.Error() != expectedError.Error() {
		t.Errorf("Expected error: %v, but got: %v", expectedError, err)
	}
	observer1.AssertExpectations(t)
	observer2.AssertExpectations(t)
}
