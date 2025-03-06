/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
package csistoragecapacity_test

import (
	"context"
	"errors"
	testing2 "testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/csistoragecapacity"
	"github.com/stretchr/testify/suite"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

func TestCSIStorageCapacity(t *testing2.T) {
	suite.Run(t, new(TestCSIStorageCapacitySuite))
}

type TestCSIStorageCapacitySuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
}

func (suite *TestCSIStorageCapacitySuite) SetupSuite() {
	client := fake.NewSimpleClientset()
	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
}

func (suite *TestCSIStorageCapacitySuite) TestGetByStorageClass() {
	ctx := context.TODO()
	scName := "test-storage-class"

	// Mock CSIStorageCapacity objects
	capacity1 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity1",
		},
		StorageClassName: scName,
	}
	capacity2 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity2",
		},
		StorageClassName: "other-storage-class",
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &storagev1.CSIStorageCapacityList{
			Items: []storagev1.CSIStorageCapacity{*capacity1, *capacity2},
		}, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	capacities, err := client.GetByStorageClass(ctx, scName)

	// Debugging
	suite.T().Logf("Capacities: %v", capacities)
	suite.T().Logf("Error: %v", err)

	// Assertions
	suite.NoError(err)
	suite.Len(capacities, 1)
	suite.Equal(capacity1.ObjectMeta.Name, capacities[0].Object.GetName())
}

func (suite *TestCSIStorageCapacitySuite) TestGetByStorageClass_NoMatch() {
	ctx := context.TODO()
	scName := "non-existent-storage-class"

	// Mock CSIStorageCapacity objects
	capacity1 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity1",
		},
		StorageClassName: "test-storage-class",
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &storagev1.CSIStorageCapacityList{
			Items: []storagev1.CSIStorageCapacity{*capacity1},
		}, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	capacities, err := client.GetByStorageClass(ctx, scName)

	// Assertions
	suite.NoError(err)
	suite.Len(capacities, 0)
}

func (suite *TestCSIStorageCapacitySuite) TestGetByStorageClass_Error() {
	ctx := context.TODO()
	scName := "test-storage-class"

	// Add reactor to return an error
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("failed to list CSIStorageCapacities")
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	capacities, err := client.GetByStorageClass(ctx, scName)

	// Assertions
	suite.Error(err)
	suite.Nil(capacities)
}

func (suite *TestCSIStorageCapacitySuite) TestSetCapacityToZero() {
	ctx := context.TODO()

	// Mock CSIStorageCapacity objects
	capacity1 := &csistoragecapacity.CSIStorageCapacity{
		Object: &storagev1.CSIStorageCapacity{
			ObjectMeta: metav1.ObjectMeta{
				Name: "capacity1",
			},
			Capacity: resource.NewQuantity(100, resource.BinarySI),
		},
	}
	capacity2 := &csistoragecapacity.CSIStorageCapacity{
		Object: &storagev1.CSIStorageCapacity{
			ObjectMeta: metav1.ObjectMeta{
				Name: "capacity2",
			},
			Capacity: resource.NewQuantity(200, resource.BinarySI),
		},
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("update", "csistoragecapacities", func(action testing.Action) (bool, runtime.Object, error) {
		updateAction := action.(testing.UpdateAction)
		obj := updateAction.GetObject().(*storagev1.CSIStorageCapacity)
		return true, obj, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	updatedCapacities, err := client.SetCapacityToZero(ctx, []*csistoragecapacity.CSIStorageCapacity{
		capacity1, capacity2,
	})

	// Assertions
	suite.NoError(err)
	suite.Len(updatedCapacities, 2)
	suite.Equal(resource.NewQuantity(0, resource.BinarySI), updatedCapacities[0].Object.Capacity)
	suite.Equal(resource.NewQuantity(0, resource.BinarySI), updatedCapacities[1].Object.Capacity)
}

func (suite *TestCSIStorageCapacitySuite) TestSetCapacityToZero_Error() {
	ctx := context.TODO()

	// Mock CSIStorageCapacity objects
	capacity1 := &csistoragecapacity.CSIStorageCapacity{
		Object: &storagev1.CSIStorageCapacity{
			ObjectMeta: metav1.ObjectMeta{
				Name: "capacity1",
			},
			Capacity: resource.NewQuantity(100, resource.BinarySI),
		},
	}

	// Add reactor to return an error
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("update", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("failed to update CSIStorageCapacity")
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	updatedCapacities, err := client.SetCapacityToZero(ctx, []*csistoragecapacity.CSIStorageCapacity{
		capacity1,
	})

	// Assertions
	suite.Error(err)
	suite.Nil(updatedCapacities)
}

func (suite *TestCSIStorageCapacitySuite) TestWaitForAllToBeCreated_Success() {
	ctx := context.TODO()
	scName := "test-storage-class"
	expectedCount := 2

	// Mock CSIStorageCapacity objects
	capacity1 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity1",
		},
		StorageClassName: scName,
	}
	capacity2 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity2",
		},
		StorageClassName: scName,
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &storagev1.CSIStorageCapacityList{
			Items: []storagev1.CSIStorageCapacity{*capacity1, *capacity2},
		}, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	err := client.WaitForAllToBeCreated(ctx, scName, expectedCount)

	// Assertions
	suite.NoError(err)
}

func (suite *TestCSIStorageCapacitySuite) TestWaitForAllToBeCreated_Timeout() {
	ctx := context.TODO()
	scName := "test-storage-class"
	expectedCount := 2

	// Mock CSIStorageCapacity objects
	capacity1 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity1",
		},
		StorageClassName: scName,
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &storagev1.CSIStorageCapacityList{
			Items: []storagev1.CSIStorageCapacity{*capacity1},
		}, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
		Timeout:   1, // Set a short timeout for testing
	}

	// Call the function
	err := client.WaitForAllToBeCreated(ctx, scName, expectedCount)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "timed out")
}

func (suite *TestCSIStorageCapacitySuite) TestWaitForAllToBeCreated_ContextCancelled() {
	ctx, cancel := context.WithCancel(context.TODO())
	scName := "test-storage-class"
	expectedCount := 2

	// Mock CSIStorageCapacity objects
	capacity1 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity1",
		},
		StorageClassName: scName,
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &storagev1.CSIStorageCapacityList{
			Items: []storagev1.CSIStorageCapacity{*capacity1},
		}, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Cancel the context to simulate context cancellation
	cancel()

	// Call the function
	err := client.WaitForAllToBeCreated(ctx, scName, expectedCount)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "stopped waiting to be created")
}

func (suite *TestCSIStorageCapacitySuite) TestWaitForAllToBeCreated_ListError() {
	ctx := context.TODO()
	scName := "test-storage-class"
	expectedCount := 2

	// Add reactor to return an error
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("failed to list CSIStorageCapacities")
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	err := client.WaitForAllToBeCreated(ctx, scName, expectedCount)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "failed to list CSIStorageCapacities")
}

func (suite *TestCSIStorageCapacitySuite) TestWaitForAllToBeDeleted_Success() {
	ctx := context.TODO()
	scName := "test-storage-class"

	// Mock CSIStorageCapacity objects
	_ = &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity1",
		},
		StorageClassName: scName,
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &storagev1.CSIStorageCapacityList{
			Items: []storagev1.CSIStorageCapacity{},
		}, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	err := client.WaitForAllToBeDeleted(ctx, scName)

	// Assertions
	suite.NoError(err)
}

func (suite *TestCSIStorageCapacitySuite) TestWaitForAllToBeDeleted_Timeout() {
	ctx := context.TODO()
	scName := "test-storage-class"

	// Mock CSIStorageCapacity objects
	capacity1 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity1",
		},
		StorageClassName: scName,
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &storagev1.CSIStorageCapacityList{
			Items: []storagev1.CSIStorageCapacity{*capacity1},
		}, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
		Timeout:   1, // Set a short timeout for testing
	}

	// Call the function
	err := client.WaitForAllToBeDeleted(ctx, scName)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "timed out")
}

func (suite *TestCSIStorageCapacitySuite) TestWaitForAllToBeDeleted_ContextCancelled() {
	ctx, cancel := context.WithCancel(context.TODO())
	scName := "test-storage-class"

	// Mock CSIStorageCapacity objects
	capacity1 := &storagev1.CSIStorageCapacity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capacity1",
		},
		StorageClassName: scName,
	}

	// Add mock objects to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &storagev1.CSIStorageCapacityList{
			Items: []storagev1.CSIStorageCapacity{*capacity1},
		}, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Cancel the context to simulate context cancellation
	cancel()

	// Call the function
	err := client.WaitForAllToBeDeleted(ctx, scName)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "stopped waiting to be deleted")
}

func (suite *TestCSIStorageCapacitySuite) TestWaitForAllToBeDeleted_ListError() {
	ctx := context.TODO()
	scName := "test-storage-class"

	// Add reactor to return an error
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "csistoragecapacities", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("failed to list CSIStorageCapacities")
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}

	// Call the function
	err := client.WaitForAllToBeDeleted(ctx, scName)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "failed to list CSIStorageCapacities")
}

func (suite *TestCSIStorageCapacitySuite) TestWatchUntilUpdated_Success() {
	ctx := context.TODO()
	pollInterval := 2 * time.Second

	// Mock CSIStorageCapacity object
	capacity := &csistoragecapacity.CSIStorageCapacity{
		Object: &storagev1.CSIStorageCapacity{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "capacity1",
				ResourceVersion: "1",
			},
		},
	}

	// Add mock watcher to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependWatchReactor("csistoragecapacities", func(_ testing.Action) (bool, watch.Interface, error) {
		w := watch.NewFake()
		go func() {
			time.Sleep(1 * time.Second)
			w.Modify(&storagev1.CSIStorageCapacity{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "capacity1",
					ResourceVersion: "2",
				},
				Capacity: resource.NewQuantity(100, resource.BinarySI),
			})
		}()
		return true, w, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}
	capacity.Client = client

	// Call the function
	err := capacity.WatchUntilUpdated(ctx, pollInterval)

	// Assertions
	suite.NoError(err)
}

func (suite *TestCSIStorageCapacitySuite) TestWatchUntilUpdated_ContextCancelled() {
	ctx, cancel := context.WithCancel(context.TODO())
	pollInterval := 2 * time.Second

	// Mock CSIStorageCapacity object
	capacity := &csistoragecapacity.CSIStorageCapacity{
		Object: &storagev1.CSIStorageCapacity{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "capacity1",
				ResourceVersion: "1",
			},
		},
	}

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}
	capacity.Client = client

	// Cancel the context to simulate context cancellation
	cancel()

	// Call the function
	err := capacity.WatchUntilUpdated(ctx, pollInterval)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "stopped waiting for CSIStorageCapacity capacity1 to be updated")
}

func (suite *TestCSIStorageCapacitySuite) TestWatchUntilUpdated_WatchError() {
	ctx := context.TODO()
	pollInterval := 2 * time.Second

	// Mock CSIStorageCapacity object
	capacity := &csistoragecapacity.CSIStorageCapacity{
		Object: &storagev1.CSIStorageCapacity{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "capacity1",
				ResourceVersion: "1",
			},
		},
	}

	// Add reactor to return an error
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependWatchReactor("csistoragecapacities", func(_ testing.Action) (bool, watch.Interface, error) {
		return true, nil, errors.New("failed to watch CSIStorageCapacities")
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}
	capacity.Client = client

	// Call the function
	err := capacity.WatchUntilUpdated(ctx, pollInterval)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "failed to watch CSIStorageCapacities")
}

func (suite *TestCSIStorageCapacitySuite) TestWatchUntilUpdated_UnexpectedType() {
	ctx := context.TODO()
	pollInterval := 2 * time.Second

	// Mock CSIStorageCapacity object
	capacity := &csistoragecapacity.CSIStorageCapacity{
		Object: &storagev1.CSIStorageCapacity{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "capacity1",
				ResourceVersion: "1",
			},
		},
	}

	// Add mock watcher to the fake client
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependWatchReactor("csistoragecapacities", func(_ testing.Action) (bool, watch.Interface, error) {
		w := watch.NewFake()
		go func() {
			time.Sleep(1 * time.Second)
			w.Add(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "unexpected",
				},
			})
		}()
		return true, w, nil
	})

	client := &csistoragecapacity.Client{
		Interface: suite.kubeClient.ClientSet.StorageV1().CSIStorageCapacities("default"),
	}
	capacity.Client = client

	// Call the function
	err := capacity.WatchUntilUpdated(ctx, pollInterval)

	// Assertions
	suite.Error(err)
	suite.Contains(err.Error(), "unexpected type in")
}
