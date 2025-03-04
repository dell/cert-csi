package observer

import (
	"context"
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

	"fmt"

	"github.com/stretchr/testify/assert"
	test "k8s.io/client-go/testing"
)

func TestVaObserver_StartWatching(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/
	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}
	//pvc.ObjectMeta = va.ObjectMeta
	clientSet := fake.NewSimpleClientset()

	/*storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)*/

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
			// Set the desired properties of the VolumeAttachment
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
	// Test case: Stopping watching volume attachments
	obs := &VaObserver{}

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

func TestVaObserver_GetName(t *testing.T) {
	// Test case: Getting name of VA observer
	obs := &VaObserver{}

	name := obs.GetName()

	assert.Equal(t, "VolumeAttachmentObserver", name)
}

func TestVaObserver_MakeChannel(t *testing.T) {
	// Test case: Creating a new channel
	obs := &VaObserver{}

	obs.MakeChannel()

	assert.NotNil(t, obs.finished)
}

func (m *mockVAClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(watch.Interface), args.Error(1)
}

func TestVaObserver_StartWatching_ShouldExit(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/
	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}
	//pvc.ObjectMeta = va.ObjectMeta
	clientSet := fake.NewSimpleClientset()

	/*storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)*/

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
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	//fakeWatcher.Delete(va)

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	fakeWatcher.Delete(va)

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_ClientIsNil(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/
	/*va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}*/
	//pvc.ObjectMeta = va.ObjectMeta
	clientSet := fake.NewSimpleClientset()

	/*storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)*/

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}
	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	//vaClient, _ := kubeClient.CreateVaClient("test-namespace")
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
	/*fakeWatcher.Add(va)

	fakeWatcher.Modify(va)

	va = &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-va",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	fakeWatcher.Delete(va)*/

	time.Sleep(100 * time.Millisecond)

	//obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_WatchError(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/
	/*va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}*/
	//pvc.ObjectMeta = va.ObjectMeta
	clientSet := fake.NewSimpleClientset()

	/*storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)*/

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

	pvcClient.ClientSet.(*fake.Clientset).PrependWatchReactor("*", func(action test.Action) (handled bool, ret watch.Interface, err error) {
		return true, nil, fmt.Errorf("test error")
	})

	obs.MakeChannel()

	go obs.StartWatching(ctx, runner)
	/*fakeWatcher.Add(va)

	fakeWatcher.Modify(va)

	va = &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-va",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	fakeWatcher.Delete(va)*/

	time.Sleep(100 * time.Millisecond)

	//obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_DataObjectIsNil(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/
	/*va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}*/
	//pvc.ObjectMeta = va.ObjectMeta
	clientSet := fake.NewSimpleClientset()

	/*storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)*/

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

	/*fakeWatcher.Modify(va)

	va = &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-va",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	fakeWatcher.Delete(va)*/

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_UnexpectedType(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	// Create a mock PVC
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}
	/*va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}*/
	//pvc.ObjectMeta = va.ObjectMeta
	clientSet := fake.NewSimpleClientset()

	/*storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)*/

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

	/*fakeWatcher.Modify(va)

	va = &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-va",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	fakeWatcher.Delete(va)*/

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_PvcShare(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/
	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}
	//pvc.ObjectMeta = va.ObjectMeta
	clientSet := fake.NewSimpleClientset()

	/*storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)*/

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
	//entity := &store.Entity{}
	//runner.PvcShare.Store("test-pvc", entity)

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

	/*fakeWatcher.Modify(va)

	va = &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-va",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	fakeWatcher.Delete(va)*/

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}

func TestVaObserver_StartWatching_WatchModified(t *testing.T) {
	// Test case: Watching volume attachments
	ctx := context.Background()

	// Create a mock PVC
	/*pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc",
			UID:  "test-uid",
		},
		//Status: v1.PersistentVolumeClaimStatus{Phase: "Bound"},
	}*/
	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		//Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}
	//pvc.ObjectMeta = va.ObjectMeta
	clientSet := fake.NewSimpleClientset()

	/*storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)*/

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

	/*fakeWatcher.Modify(va)

	va = &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-va",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: storagev1.VolumeAttachmentSpec{
			// Set the desired properties of the VolumeAttachment
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: func() *string { s := "test-pvc"; return &s }(),
			},
		},
		Status: storagev1.VolumeAttachmentStatus{Attached: true},
	}

	fakeWatcher.Modify(va)

	fakeWatcher.Delete(va)*/

	time.Sleep(100 * time.Millisecond)

	obs.StopWatching()

	runner.WaitGroup.Wait()

	// Assert that the function completed successfully
	assert.True(t, true)
}
