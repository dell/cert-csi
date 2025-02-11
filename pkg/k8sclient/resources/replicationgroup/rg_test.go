package replicationgroup

import (
	"context"
	replv1 "github.com/dell/csm-replication/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

func TestClient_Delete(t *testing.T) {
	// Add the necessary schemes to the runtime scheme
	_ = replv1.AddToScheme(scheme.Scheme)
	// Create a new context
	ctx := context.Background()

	// Create a new fake runtime client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&replv1.DellCSIReplicationGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rg",
					Namespace: "default",
				},
			},
		).
		Build()

	// Create a new replication group
	rg := &replv1.DellCSIReplicationGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rg",
			Namespace: "default",
		},
	}

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
}

func TestClient_Get(t *testing.T) {
	// Add the necessary schemes to the runtime scheme
	_ = replv1.AddToScheme(scheme.Scheme)
	// Create a new context
	ctx := context.Background()

	// Create a new fake runtime client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&replv1.DellCSIReplicationGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rg1",
					Namespace: "default",
				},
			},
		).
		Build()

	// Create a new replication group
	rg := &replv1.DellCSIReplicationGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rg1",
			Namespace: "default",
		},
	}

	// Create a new Client instance with the fake client
	c := &Client{
		Interface: fakeClient,
	}

	// Call the Get function
	result := c.Get(ctx, "test-rg1")

	// Check the returned RG object
	assert.Equal(t, c, result.Client)
	assert.NotNil(t, rg, result.Object)
	//assert.NoError(t, result.error)
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
				State:       "",
				RemoteState: "",
				ReplicationLinkState: replv1.ReplicationLinkState{
					State:                "SYNCHRONIZED",
					IsSource:             false,
					LastSuccessfulUpdate: nil,
					ErrorMessage:         "",
				},
				LastAction: replv1.LastAction{},
				Conditions: nil,
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
			Timeout:   0,
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
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		go rgClient.ExecuteAction(timeoutCtx, rgAction)

		rg := rgClient.Client.Get(ctx, "test-rg2")

		if rg.Object.Status.ReplicationLinkState.State != expectedState {
			t.Errorf("Expected: %v, got: %v", expectedState, rg.Object.Status.ReplicationLinkState.State)
		}
	})

	// Test case: Testing RG action "FAILOVER_LOCAL"
	t.Run("RG action FAILOVER_LOCAL", func(t *testing.T) {
		rgAction := "FAILOVER_LOCAL"
		//driverName := "powermax"
		//expectedState := "SUSPENDED"
		expectedState := "SYNCHRONIZED"
		rgObj.Object.Status.ReplicationLinkState.State = "SUSPENDED"
		// Create a new context with a timeout
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		go rgClient.ExecuteAction(timeoutCtx, rgAction)

		rg := rgClient.Client.Get(ctx, "test-rg2")

		if rg.Object.Status.ReplicationLinkState.State != expectedState {
			t.Errorf("Expected: %v, got: %v", expectedState, rg.Object.Status.ReplicationLinkState.State)
		}
	})

	// Test case: Testing RG action "REPROTECT"
	t.Run("RG action REPROTECT", func(t *testing.T) {
		rgAction := "REPROTECT"
		//expectedState := "REPROTECT"
		expectedState := "SYNCHRONIZED"
		// Create a new context with a timeout
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		go rgClient.ExecuteAction(timeoutCtx, rgAction)

		rg := rgClient.Client.Get(ctx, "test-rg2")

		if rg.Object.Status.ReplicationLinkState.State != expectedState {
			t.Errorf("Expected: %v, got: %v", expectedState, rg.Object.Status.ReplicationLinkState.State)
		}
	})

	// Test case: Testing RG action "invalid"
	t.Run("RG action invalid", func(t *testing.T) {
		rgAction := "invalid"
		expectedState := "SYNCHRONIZED"
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		go rgClient.ExecuteAction(timeoutCtx, rgAction)

		rg := rgClient.Client.Get(ctx, "test-rg2")

		if rg.Object.Status.ReplicationLinkState.State != expectedState {
			t.Errorf("Expected: %v, got: %v", expectedState, rg.Object.Status.ReplicationLinkState.State)
		}
	})

}
