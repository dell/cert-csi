package replicationgroup

import (
	"context"
	"errors"
	"testing"
	"time"

	replv1 "github.com/dell/csm-replication/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestClient_Delete(t *testing.T) {
	// Add the necessary schemes to the runtime scheme
	_ = replv1.AddToScheme(scheme.Scheme)
	// Create a new context
	ctx := context.Background()

	// Create a new replication group
	rg := &replv1.DellCSIReplicationGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rg",
			Namespace: "default",
		},
	}

	// Create a new fake runtime client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(rg).
		Build()

	// Create a new Client instance with the fake client
	c := &Client{
		Interface: fakeClient,
	}

	// Call the Delete function
	result := c.Delete(ctx, rg)

	// Check the returned RG object
	assert.Equal(t, c, result.Client)
	assert.Equal(t, rg, result.Object)
	assert.True(t, result.Deleted)
	assert.NoError(t, result.error)

	// Call the Delete function
	result = c.Delete(ctx, rg)
	assert.Error(t, result.error)
}

func TestClient_Get(t *testing.T) {
	// Add the necessary schemes to the runtime scheme
	_ = replv1.AddToScheme(scheme.Scheme)
	// Create a new context
	ctx := context.Background()

	// Create a new replication group
	rg := &replv1.DellCSIReplicationGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rg1",
			Namespace: "default",
		},
	}

	// Create a new fake runtime client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(rg).
		Build()

	// Create a new Client instance with the fake client
	c := &Client{
		Interface: fakeClient,
	}

	// Call the Get function
	result := c.Get(ctx, "test-rg1")

	// Check the returned RG object
	assert.Equal(t, c, result.Client)
	assert.NotNil(t, rg, result.Object)
}

func TestGetName(t *testing.T) {
	// Create a new replication group object
	rg := &RG{
		Object: &replv1.DellCSIReplicationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-rg2",
			},
		},
	}

	// Test the Name function
	expectedName := "test-rg2"
	actualName := rg.Name()

	// Check the returned name
	assert.Equal(t, expectedName, actualName)
}

func TestAction(t *testing.T) {
	_ = replv1.AddToScheme(scheme.Scheme)
	// Create a new context
	ctx := context.Background()
	rgObj := &RG{
		Object: &replv1.DellCSIReplicationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-rg2",
			},
			Status: replv1.DellCSIReplicationGroupStatus{
				ReplicationLinkState: replv1.ReplicationLinkState{
					State:    "SYNCHRONIZED",
					IsSource: false,
				},
			},
		},
	}

	// Create a new fake runtime client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			rgObj.Object,
		).
		Build()
	rgClient := RG{
		Client: &Client{
			Interface: fakeClient,
			ClientSet: nil,
			Timeout:   10,
		},
		Object:  rgObj.Object,
		Deleted: false,
		error:   nil,
	}

	// Test case: Testing RG action "FAILOVER_REMOTE"
	t.Run("RG action FAILOVER_REMOTE", func(t *testing.T) {
		rgAction := "FAILOVER_REMOTE"
		expectedState := "SYNCHRONIZED"

		// Create a new context with a timeout
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go rgClient.ExecuteAction(timeoutCtx, rgAction)

		rg := rgClient.Client.Get(ctx, "test-rg2")

		if rg.Object.Status.ReplicationLinkState.State != expectedState {
			t.Errorf("Expected: %v, got: %v", expectedState, rg.Object.Status.ReplicationLinkState.State)
		}
	})
}

func TestRG_GetError(t *testing.T) {
	// Test case: RG with error
	t.Run("RG with error", func(t *testing.T) {
		testError := errors.New("test error")
		rg := &RG{
			error: testError,
		}

		result := rg.GetError()

		// Check the returned error
		assert.Equal(t, testError, result)
	})

	// Test case: RG without error
	t.Run("RG without error", func(t *testing.T) {
		rg := &RG{
			error: nil,
		}

		result := rg.GetError()

		// Check the returned error
		assert.Nil(t, result)
	})
}

func TestHasError(t *testing.T) {
	// Test case: RG with error
	t.Run("RG with error", func(t *testing.T) {
		testError := errors.New("test error")
		rg := &RG{
			error: testError,
		}

		result := rg.HasError()

		// Check the returned error
		assert.True(t, result)
	})

	// Test case: RG without error
	t.Run("RG without error", func(t *testing.T) {
		rg := &RG{
			error: nil,
		}

		result := rg.HasError()

		// Check the returned error
		assert.False(t, result)
	})
}

// Test case: RG.selectDesiredState
func TestRGSelectDesiredState(t *testing.T) {
	// Test case: RG.selectDesiredState
	t.Run("RG.selectDesiredState", func(t *testing.T) {
		testCases := []struct {
			rgAction   string
			driverName string
			expected   string
		}{
			{
				rgAction: "FAILOVER_REMOTE",
				expected: "FAILEDOVER",
			},
			{
				rgAction:   "FAILOVER_LOCAL",
				driverName: "unity",
				expected:   "FAILEDOVER",
			},
			{
				rgAction:   "FAILOVER_REMOTE",
				driverName: "powermax",
				expected:   "SUSPENDED",
			},
			{
				rgAction:   "REPROTECT_LOCAL",
				driverName: "powermax",
				expected:   "SYNCHRONIZED",
			},
			{
				rgAction:   "REPROTECT_LOCAL",
				driverName: "unity",
				expected:   "SYNCHRONIZED",
			},
			{
				rgAction: "",
				expected: "SYNCHRONIZED",
			},
		}
		for _, tc := range testCases {
			rg := &RG{}
			actual := rg.selectDesiredState(tc.rgAction, tc.driverName)
			assert.Equal(t, tc.expected, actual)
		}
	})
}

func TestGetPreDesiredState(t *testing.T) {
	testCases := []struct {
		rgAction   string
		driverName string
		expected   string
	}{
		{
			rgAction: "FAILOVER_REMOTE",
			expected: "SYNCHRONIZED",
		},
		{
			rgAction: "FAILOVER_LOCAL",
			expected: "SYNCHRONIZED",
		},
		{
			rgAction:   "FAILOVER_REMOTE",
			driverName: "powermax",
			expected:   "SYNCHRONIZED",
		},
		{
			rgAction:   "REPROTECT_LOCAL",
			driverName: "powermax",
			expected:   "SUSPENDED",
		},
		{
			rgAction:   "REPROTECT_REMOTE",
			driverName: "unity",
			expected:   "FAILEDOVER",
		},
	}
	for _, tc := range testCases {
		rg := &RG{}
		actual := rg.getPreDesiredState(tc.rgAction, tc.driverName)
		assert.Equal(t, tc.expected, actual)
	}
}
