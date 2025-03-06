package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	k8sclientV1 "github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestPvcListObserver_StartWatching(t *testing.T) {
	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}

	clientSet := NewFakeClientsetWithRestClient(storageClass)

	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")

	pvcsList := &v1.PersistentVolumeClaimList{
		Items: []v1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pvc",
					Namespace:         "test-namespace",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: v1.PersistentVolumeClaimStatus{
					Phase: v1.ClaimBound,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pvc",
					Namespace:         "test-namespace",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: v1.PersistentVolumeClaimStatus{
					Phase: v1.ClaimBound,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pvc",
					Namespace:         "test-namespace",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: v1.PersistentVolumeClaimStatus{
					Phase: v1.ClaimBound,
				},
			},
		},
	}

	tests := []struct {
		name    string
		runner  *Runner
		pvcList *v1.PersistentVolumeClaimList
	}{
		{
			name: "Test case: Watching container pvc with driver namespace and metricList",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					PVCClient: pvcClient,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup:       sync.WaitGroup{},
				DriverNamespace: "test-namespace",
				Database:        NewSimpleStore(),
			},
			pvcList: pvcsList,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var pvcListWg sync.WaitGroup

			ctx := context.Background()

			test.runner.WaitGroup.Add(1)

			po := &PvcListObserver{}
			po.MakeChannel()

			if test.pvcList != nil {
				pvcListWg.Add(1)
				originalGetPvcsList := getPvcsList
				getPvcsList = func(ctx context.Context, client *k8sclientV1.Client, opts metav1.ListOptions) (*v1.PersistentVolumeClaimList, error) {
					pvcListWg.Done()

					return test.pvcList, nil
				}
				defer func() {
					getPvcsList = originalGetPvcsList
				}()
			}

			go po.StartWatching(ctx, test.runner)
			if test.pvcList != nil {
				pvcListWg.Wait()
				go func() {
					po.finished <- true
				}()
			}
			po.StopWatching()
			test.runner.WaitGroup.Wait()

			assert.True(t, true)
		})
	}
}

func TestPvcListObserver_StopWatching(t *testing.T) {

	obs := &PvcListObserver{}

	obs.finished = make(chan bool)

	go obs.StopWatching()

	select {
	case <-obs.finished:
		assert.True(t, true)

	case <-time.After(1 * time.Second):
		// Timeout waiting for channel to receive a value
		t.Error("Timeout waiting for channel to receive a value")
	}
}

func TestPvcListObserver_GetName(t *testing.T) {
	obs := &PvcListObserver{}

	name := obs.GetName()

	assert.Equal(t, "PersistentVolumeClaimObserver", name)
}

func TestPvcListObserver_MakeChannel(t *testing.T) {
	obs := &PvcListObserver{}

	obs.MakeChannel()

	assert.NotNil(t, obs.finished)
}

// Mock implementation of PVCClient
type mockPVCClient struct {
	mock.Mock
}

func (m *mockPVCClient) List(ctx context.Context, opts metav1.ListOptions) (*v1.PersistentVolumeClaimList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*v1.PersistentVolumeClaimList), args.Error(1)
}

// Mock implementation of PVC
type mockPVC struct {
	mock.Mock
}

func (m *mockPVC) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockPVC) GetUID() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockPVC) GetDeletionTimestamp() *metav1.Time {
	args := m.Called()
	return args.Get(0).(*metav1.Time)
}

func (m *mockPVC) GetStatusPhase() v1.PersistentVolumeClaimPhase {
	args := m.Called()
	return args.Get(0).(v1.PersistentVolumeClaimPhase)
}

func (m *mockPVC) GetSpecVolumeName() string {
	args := m.Called()
	return args.String(0)
}
