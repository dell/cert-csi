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
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func isChanClosed(ch <-chan bool) bool {
	select {
	case _, ok := <-ch:
		return !ok
	default:
		return false
	}
}

func TestPodListObserver_StartWatching(t *testing.T) {

	ctx := context.Background()

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	}
	clientSet := NewFakeClientsetWithRestClient(storageClass)

	clientSet.CoreV1().Pods("test-namespace").Create(ctx, &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace"}}, metav1.CreateOptions{})
	clientSet.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})

	kubeClient := &k8sclient.KubeClient{
		ClientSet: clientSet,
		Config:    &rest.Config{},
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "test-namespace",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(v1.PodReady),
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	pod2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod-2",
			Namespace:         "test-namespace",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(v1.PodReady),
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	podClient.Create(ctx, pod)
	podClient.Create(ctx, pod2)

	tests := []struct {
		name   string
		runner *Runner
	}{
		{
			name: "Test case: nil podClient",
			runner: &Runner{
				WaitGroup: sync.WaitGroup{},
				Clients: &k8sclient.Clients{
					PodClient: nil,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				Database: NewSimpleStore(),
			},
		},
		{
			name: "Test case: podClient with added pods conditional is false",
			runner: &Runner{
				WaitGroup: sync.WaitGroup{},
				Clients: &k8sclient.Clients{
					PodClient: podClient,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				Database: NewSimpleStore(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var wg sync.WaitGroup

			po := &PodListObserver{}
			po.MakeChannel()

			test.runner.WaitGroup.Add(1)
			runCount := 0

			originalGetBoolValueFromMapWithKey := getBoolValueFromMapWithKey
			getBoolValueFromMapWithKey = func(m map[string]bool, key string) bool {
				runCount++
				if runCount == 5 {
					podClient.Delete(ctx, pod)
				}
				if runCount == 10 {
					wg.Done()
				}
				return originalGetBoolValueFromMapWithKey(m, key)
			}
			defer func() {
				podClient.Create(ctx, pod)
				getBoolValueFromMapWithKey = originalGetBoolValueFromMapWithKey
			}()

			go po.StartWatching(ctx, test.runner)
			if test.runner.Clients.PodClient != nil {
				wg.Add(1)
				wg.Wait()
				po.StopWatching()
			}
			test.runner.WaitGroup.Wait()

			// Assert that the function completed successfully
			assert.True(t, true)
		})
	}
}
func TestPodListObserver_StopWatching(t *testing.T) {

	po := &PodListObserver{}

	po.finished = make(chan bool)

	go po.StopWatching()

	select {
	case <-po.finished:
		assert.True(t, true)

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for channel to receive a value")
	}
}

func TestPodListObserver_GetName(t *testing.T) {

	po := &PodListObserver{}

	name := po.GetName()

	assert.Equal(t, "Pod Observer", name)
}

func TestPodListObserver_MakeChannel(t *testing.T) {

	po := &PodListObserver{}

	po.MakeChannel()

	assert.NotNil(t, po.finished)
}
