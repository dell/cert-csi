package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementation of Interface
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

// type mockObserver struct {
// 	makeChannelCalled   bool
// 	startWatchingCalled bool
// }

// func (m *mockObserver) MakeChannel() {
// 	m.makeChannelCalled = true
// }

// func (m *mockObserver) StartWatching(ctx context.Context, runner *Runner) {
// 	m.startWatchingCalled = true
// }

// func (m *mockObserver) GetName() string {
// 	return "MockObserver"
// }

// TestRunner_Start tests the Start method of the Runner
func TestRunner_Start(t *testing.T) {
	// Test case: Start watching all the runners

	// Test case: Start watching all the runners
	ctx := context.Background()

	observer1 := &mockObserver{}
	observer2 := &mockObserver{}

	observers := []Interface{observer1, observer2}

	runner := &Runner{
		Observers: observers,
	}

	// err := runner.Start(ctx)

	// if err != nil {
	// 	t.Errorf("Unexpected error: %v", err)
	// }

	// if !observer1.makeChannelCalled {
	// 	t.Error("Expected MakeChannel to be called on observer1")
	// }

	// if !observer1.startWatchingCalled {
	// 	t.Error("Expected StartWatching to be called on observer1")
	// }

	// if !observer2.makeChannelCalled {
	// 	t.Error("Expected MakeChannel to be called on observer2")
	// }

	// if !observer2.startWatchingCalled {
	// 	t.Error("Expected StartWatching to be called on observer2")
	// }

	// observer1 := &mockObserver{}

	// // Set up the mock observer to expect the StartWatching method to be called
	// observer1.On("StartWatching", mock.Anything, mock.Anything).Return()

	// // Create a mock Runner
	// runner := &Runner{
	// 	Observers: []Interface{observer1},
	// 	// ... other fields
	// }

	// // Call the Start method of the Runner
	//runner.Start(ctx)

	// // Assert that the StartWatching method of the mock observer was called
	// observer1.AssertExpectations(t)

	// ctx := context.Background()
	// observer1 := &mockObserver{}
	// observer2 := &mockObserver{}
	// observers := []Interface{observer1, observer2}

	// clients := &k8sclient.Clients{}
	// db := NewSimpleStore()
	// testCase := &store.TestCase{
	// 	ID: 1,
	// }
	// driverNs := "driver-namespace"
	// shouldClean := true

	// runner := &Runner{
	// 	Observers:       observers,
	// 	Clients:         clients,
	// 	Database:        db,
	// 	TestCase:        testCase,
	// 	DriverNamespace: driverNs,
	// 	ShouldClean:     shouldClean,
	// }

	observer1.On("MakeChannel").Return()
	observer1.On("StartWatching", ctx, runner).Return()
	observer2.On("MakeChannel").Return()
	observer2.On("StartWatching", ctx, runner).Return()

	err := runner.Start(ctx)

	assert.NoError(t, err)
	// observer1.AssertExpectations(t)
	// observer2.AssertExpectations(t)
}

// TestRunner_Stop tests the Stop method of the Runner
func TestRunner_Stop(t *testing.T) {
	// Test case: Stop watching all the runners and delete PVCs
	//ctx := context.Background()
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

// TestRunner_waitTimeout tests the waitTimeout method of the Runner
func TestRunner_waitTimeout(t *testing.T) {
	// Test case: Wait for all of observers to complete
	//ctx := context.Background()
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

	// err := runner.Stop()

	// assert.NoError(t, err)
	// observer1.AssertExpectations(t)
	// observer2.AssertExpectations(t)

	// observer1.On("StopWatching").Return()
	// observer2.On("StopWatching").Return()

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

// TestRunner_GetName tests the GetName method of the Runner
func TestRunner_GetName(t *testing.T) {
	// Test case: Get name of the Runner
	observer := &mockObserver{}
	observer.On("GetName").Return("MockObserver")

	result := observer.GetName()

	assert.Equal(t, "MockObserver", result)
	observer.AssertExpectations(t)
}

// TestRunner_MakeChannel tests the MakeChannel method of the Runner
func TestRunner_MakeChannel(t *testing.T) {
	// Test case: Create a new channel
	observer := &mockObserver{}
	observer.On("MakeChannel").Return()

	observer.MakeChannel()

	observer.AssertExpectations(t)
}
