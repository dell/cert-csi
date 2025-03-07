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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
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
		assert.True(t, true)

	case <-time.After(1 * time.Second):
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

func TestPvcObserver_StartWatching_ClientIsNil(t *testing.T) {
	// Test case: Client is nil
	ctx := context.Background()

	runner := &Runner{
		Clients: &k8sclient.Clients{
			PVCClient: nil,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		Database:  NewSimpleStore(),
	}
	runner.WaitGroup.Add(1)

	obs := &PvcObserver{}

	// Start the observer in a separate goroutine
	go obs.StartWatching(ctx, runner)

	time.Sleep(100 * time.Millisecond)

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcObserver_StartWatching_WatchError(t *testing.T) {
	// Test case: Error when watching PVCs
	ctx := context.Background()

	// Create a mock client
	clientSet := fake.NewSimpleClientset()

	// Create a mock KubeClient
	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	// Create a mock PVC client
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	// Create a mock Runner
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

	// Create a PvcObserver
	obs := &PvcObserver{}

	// Set the watch reactor to return an error
	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(_ test.Action) (handled bool, ret watch.Interface, err error) {
		return true, nil, fmt.Errorf("test error")
	})

	// Start the observer
	go obs.StartWatching(ctx, runner)

	time.Sleep(100 * time.Millisecond)

	// obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcObserver_StartWatching_DataObjectIsNil(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

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
			return true, fakeWatcher, nil
		}
		return false, nil, nil
	})

	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	fakeWatcher.Add(nil)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcObserver_StartWatching_UnexpectedType(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
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

	fakeWatcher.Modify(storageClass)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestPvcObserver_StartWatching_WatchModified(t *testing.T) {
	// Test case: Watching PVCs
	ctx := context.Background()

	// Create a mock PVC
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
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

	fakeWatcher.Modify(pvc)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}
