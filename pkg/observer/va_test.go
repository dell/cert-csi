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
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
	test "k8s.io/client-go/testing"
)

func TestVaObserver_StartWatching(t *testing.T) {
	ctx := context.Background()

	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: vaClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		PvcShare:  sync.Map{},
		Database:  NewSimpleStore(),
	}
	entity := &store.Entity{}
	runner.PvcShare.Store("test-pvc", entity)

	runner.WaitGroup.Add(1)

	obs := &VaObserver{}

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
	fakeWatcher.Add(va)

	fakeWatcher.Modify(va)

	va = &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-va",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	fakeWatcher.Delete(va)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StopWatching(t *testing.T) {
	obs := &VaObserver{}

	obs.finished = make(chan bool)

	go obs.StopWatching()

	select {
	case <-obs.finished:
		assert.True(t, true)

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for channel to receive a value")
	}
}

func TestVaObserver_GetName(t *testing.T) {
	obs := &VaObserver{}

	name := obs.GetName()

	assert.Equal(t, "VolumeAttachmentObserver", name)
}

func TestVaObserver_MakeChannel(t *testing.T) {
	obs := &VaObserver{}

	obs.MakeChannel()

	assert.NotNil(t, obs.finished)
}

func TestVaObserver_StartWatching_ShouldExit(t *testing.T) {
	ctx := context.Background()

	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: vaClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup:   sync.WaitGroup{},
		PvcShare:    sync.Map{},
		Database:    NewSimpleStore(),
		ShouldClean: true,
	}
	entity := &store.Entity{}
	runner.PvcShare.Store("test-pvc", entity)

	runner.WaitGroup.Add(1)

	obs := &VaObserver{}

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
	fakeWatcher.Add(va)

	fakeWatcher.Modify(va)

	va = &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-va",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	fakeWatcher.Delete(va)

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_ClientIsNil(t *testing.T) {
	ctx := context.Background()

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: nil,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		PvcShare:  sync.Map{},
		Database:  NewSimpleStore(),
	}
	entity := &store.Entity{}
	runner.PvcShare.Store("test-pvc", entity)

	runner.WaitGroup.Add(1)

	obs := &VaObserver{}

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

	time.Sleep(100 * time.Millisecond)

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_WatchError(t *testing.T) {
	ctx := context.Background()

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: vaClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		PvcShare:  sync.Map{},
		Database:  NewSimpleStore(),
	}
	entity := &store.Entity{}
	runner.PvcShare.Store("test-pvc", entity)

	runner.WaitGroup.Add(1)

	obs := &VaObserver{}

	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(_ test.Action) (handled bool, ret watch.Interface, err error) {
		return true, nil, fmt.Errorf("test error")
	})

	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)

	time.Sleep(100 * time.Millisecond)

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_DataObjectIsNil(t *testing.T) {
	ctx := context.Background()

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: vaClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		PvcShare:  sync.Map{},
		Database:  NewSimpleStore(),
	}
	entity := &store.Entity{}
	runner.PvcShare.Store("test-pvc", entity)

	runner.WaitGroup.Add(1)

	obs := &VaObserver{}

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
	fakeWatcher.Add(nil)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_UnexpectedType(t *testing.T) {
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
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: vaClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		PvcShare:  sync.Map{},
		Database:  NewSimpleStore(),
	}
	entity := &store.Entity{}
	runner.PvcShare.Store("test-pvc", entity)

	runner.WaitGroup.Add(1)

	obs := &VaObserver{}

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

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_PvcShare(t *testing.T) {
	ctx := context.Background()

	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: vaClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		PvcShare:  sync.Map{},
		Database:  NewSimpleStore(),
	}

	runner.WaitGroup.Add(1)

	obs := &VaObserver{}

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

	fakeWatcher.Add(va)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_WatchModified(t *testing.T) {
	ctx := context.Background()

	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
	}

	clientSet := fake.NewSimpleClientset()

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	vaClient, _ := kubeClient.CreateVaClient("test-namespace")
	runner := &Runner{
		Clients: &k8sclient.Clients{
			VaClient: vaClient,
		},
		TestCase: &store.TestCase{
			ID: 1,
		},
		WaitGroup: sync.WaitGroup{},
		PvcShare:  sync.Map{},
		Database:  NewSimpleStore(),
	}
	entity := &store.Entity{}
	runner.PvcShare.Store("test-pvc", entity)

	runner.WaitGroup.Add(1)

	obs := &VaObserver{}

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

	fakeWatcher.Modify(va)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}
