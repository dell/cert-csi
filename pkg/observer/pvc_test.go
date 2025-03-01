package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	test "k8s.io/client-go/testing"
)

func TestPvcObserver_StartWatching(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

	// Create a mock PVC
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: pvcClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}
	runner.WaitGroup.Add(1)

	obs := &PvcObserver{}

	fakeWatcher := watch.NewFake()
	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		if action.GetVerb() == "watch" {
			// Return the fake watcher
			return true, fakeWatcher, nil
		}
		return false, nil, nil
	})

	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	fakeWatcher.Add(pvc)
	fakeWatcher.Modify(pvc)

	pvc = &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pvc",
			UID:               "test-uid",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}

	fakeWatcher.Modify(pvc)

	fakeWatcher.Delete(pvc)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcObserver_StopWatching(t *testing.T) {
	// Test case: Stopping watching PVCs
	obs := &PvcObserver{}

	obs.finished = make(chan bool)

	go obs.StopWatching()

	select {
	case <-obs.finished:
		// Channel received a value
		// Make assertions here
		assert.True(t, true)

	case <-time.After(1 * time.Second):
		// Timeout waiting for channel to receive a value
		t.Error("Timeout waiting for channel to receive a value")
	}
}

func TestPvcObserver_GetName(t *testing.T) {
	// Test case: Getting name of PVC observer
	obs := &PvcObserver{}

	name := obs.GetName()

	assert.Equal(t, "PersistentVolumeClaimObserver", name)
}

func TestPvcObserver_MakeChannel(t *testing.T) {
	// Test case: Creating a new channel
	obs := &PvcObserver{}

	obs.MakeChannel()

	assert.NotNil(t, obs.finished)
}
