package observer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/metrics"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func TestContainerMetricsObserver_StartWatching(t *testing.T) {
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
	kubeClient.SetTimeout(3000)

	podClient, _ := kubeClient.CreatePodClient("test-namespace")
	metricsClient, _ := kubeClient.CreateMetricsClient("test-namespace")

	// Create a PodMetrics object
	podMetrics := &v1beta1.PodMetrics{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMetrics",
			APIVersion: "metrics.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-metrics-example",
			Namespace: "default",
		},
		Timestamp: metav1.Time{Time: time.Now()},
		Window:    metav1.Duration{Duration: time.Minute},
		Containers: []v1beta1.ContainerMetrics{
			{
				Name:  "container-1",
				Usage: v1.ResourceList{},
			},
		},
	}

	// Create a PodMetricsList object
	podMetricsList := &v1beta1.PodMetricsList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMetricsList",
			APIVersion: "metrics.k8s.io/v1beta1",
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
		Items: []v1beta1.PodMetrics{
			*podMetrics,
		},
	}

	tests := []struct {
		name        string
		runner      *Runner
		metricsList *v1beta1.PodMetricsList
	}{
		{
			name: "Test case: Watching container metrics without driver namespace",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					PodClient:     podClient,
					MetricsClient: metricsClient,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup: sync.WaitGroup{},
				Database:  NewSimpleStore(),
			},
			metricsList: nil,
		},
		{
			name: "Test case: Watching container metrics with driver namespace and error getMetricsList",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					PodClient:     podClient,
					MetricsClient: metricsClient,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup:       sync.WaitGroup{},
				DriverNamespace: "test-namespace",
				Database:        NewSimpleStore(),
			},
			metricsList: nil,
		},
		{
			name: "Test case: Watching container metrics with driver namespace and metricList",
			runner: &Runner{
				Clients: &k8sclient.Clients{
					PodClient:     podClient,
					MetricsClient: metricsClient,
				},
				TestCase: &store.TestCase{
					ID: 1,
				},
				WaitGroup:       sync.WaitGroup{},
				DriverNamespace: "test-namespace",
				Database:        NewSimpleStore(),
			},
			metricsList: podMetricsList,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var metricListWg sync.WaitGroup

			ctx := context.Background()

			test.runner.WaitGroup.Add(1)

			cmo := &ContainerMetricsObserver{}
			defer func() {
				cmo.Interrupt()
				cmo.StopWatching()
			}()
			cmo.MakeChannel()

			if test.metricsList != nil {
				metricListWg.Add(1)
				originalGetMetricsList := getMetricsList
				getMetricsList = func(_ *metrics.Client, _ string, _ context.Context, _ metav1.ListOptions) (*v1beta1.PodMetricsList, error) {
					metricListWg.Done()

					return test.metricsList, nil
				}
				defer func() {
					getMetricsList = originalGetMetricsList
				}()
			}

			go cmo.StartWatching(ctx, test.runner)
			if test.metricsList != nil {
				metricListWg.Wait()
				go func() {
					cmo.mutex.Lock()
					cmo.finished <- true
					cmo.mutex.Unlock()
				}()
			}
			test.runner.WaitGroup.Wait()

			// Assert that the function completed successfully
			assert.True(t, true)
		})
	}
}

func TestContainerMetricsObserver_StopWatching(_ *testing.T) {
	cmo := &ContainerMetricsObserver{}

	cmo.finished = make(chan bool)
	go func() {
		cmo.StopWatching()
	}()
	time.Sleep(1 * time.Second)

	cmo.Interrupt()
	cmo.StopWatching()
}

func TestContainerMetricsObserver_GetName(t *testing.T) {
	cmo := &ContainerMetricsObserver{}
	assert.Equal(t, "ContainerMetricsObserver", cmo.GetName())
}

func TestContainerMetricsObserver_MakeChannel(t *testing.T) {
	cmo := &ContainerMetricsObserver{}
	cmo.MakeChannel()
	assert.NotNil(t, cmo.finished)
}
