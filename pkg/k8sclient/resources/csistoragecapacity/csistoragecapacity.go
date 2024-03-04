/*
 *
 * Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package csistoragecapacity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	tcorev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
)

const (
	// Poll represents poll time interval
	Poll = 2 * time.Second
	// Timeout represents timeout
	Timeout = 5 * time.Minute
)

// Client contains CSIStorageCapacity interface
type Client struct {
	Interface tcorev1.CSIStorageCapacityInterface
	Namespace string
	Timeout   int
}

// CSIStorageCapacity contains pvc client and claim
type CSIStorageCapacity struct {
	Client *Client
	Object *storagev1.CSIStorageCapacity
}

// GetByStorageClass uses client interface to make API call for getting CSIStorageCapacity for provided StorageClass
func (c *Client) GetByStorageClass(ctx context.Context, scName string) ([]*CSIStorageCapacity, error) {
	var capacities []*CSIStorageCapacity
	capacityList, err := c.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, capacity := range capacityList.Items {
		if capacity.StorageClassName == scName {
			log.Debugf("Got %s CSIStorageCapacity", capacity.GetName())
			capacities = append(capacities, &CSIStorageCapacity{Client: c, Object: capacity.DeepCopy()})
		}
	}

	log.Debugf("Got %d CSIStorageCapacity for %s storage class", len(capacities), scName)
	return capacities, nil
}

// SetCapacityToZero sets capacity to 0
func (c *Client) SetCapacityToZero(ctx context.Context, capacities []*CSIStorageCapacity) ([]*CSIStorageCapacity, error) {
	var updatedCapacities []*CSIStorageCapacity
	for _, capacity := range capacities {
		capacity.Object.Capacity = resource.NewQuantity(0, resource.BinarySI)
		capacity, err := c.Update(ctx, capacity.Object)
		if err != nil {
			return nil, err
		}
		updatedCapacities = append(updatedCapacities, capacity)
	}
	return updatedCapacities, nil
}

// Update updates storage capacity object
func (c *Client) Update(ctx context.Context, capacity *storagev1.CSIStorageCapacity) (*CSIStorageCapacity, error) {
	capacity, err := c.Interface.Update(ctx, capacity, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	log.Debugf("Updated %s CSIStorageCapacity", capacity.GetName())

	return &CSIStorageCapacity{
		Client: c,
		Object: capacity,
	}, nil
}

// WaitForAllToBeCreated waits for CSIStorageCapacity objects to be created for a storage class
func (c *Client) WaitForAllToBeCreated(ctx context.Context, scName string, expectedCount int) error {
	log.Infof("Waiting for CSIStorageCapacity to be %s for %s storage class in %s namespace", color.GreenString("CREATED"), color.YellowString(scName), color.BlueString(c.Namespace))
	startTime := time.Now()
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping CSIStorageCapacity wait polling")
				return true, fmt.Errorf("stopped waiting to be created")
			default:
				break
			}

			csiscList, err := c.Interface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}

			createdCount := 0
			for _, csisc := range csiscList.Items {
				if strings.Compare(csisc.StorageClassName, scName) == 0 {
					createdCount++
					log.Debugf("Found %d CSIStorageCapacity object, expected %d", createdCount, expectedCount)
				}
			}

			return createdCount == expectedCount, nil
		})

	if pollErr != nil {
		return pollErr
	}

	yellow := color.New(color.FgHiYellow)
	log.Infof("All CSIStorageCapacity are created in %s", yellow.Sprint(time.Since(startTime)))
	return nil
}

// WaitForAllToBeDeleted waits for CSIStorageCapacity objects to be deleted for a storage class
func (c *Client) WaitForAllToBeDeleted(ctx context.Context, scName string) error {
	log.Infof("Waiting for CSIStorageCapacity to be %s for %s storage class in %s namespace", color.GreenString("DELETED"), color.YellowString(scName), color.BlueString(c.Namespace))
	startTime := time.Now()
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping CSIStorageCapacity wait polling")
				return true, fmt.Errorf("stopped waiting to be deleted")
			default:
				break
			}

			csiscList, err := c.Interface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}

			foundCount := 0
			for _, csisc := range csiscList.Items {
				if strings.Compare(csisc.StorageClassName, scName) == 0 {
					foundCount++
					log.Debugf("Found %d CSIStorageCapacity object, expected %d", foundCount, 0)
				}
			}

			return foundCount == 0, nil
		})

	if pollErr != nil {
		return pollErr
	}

	yellow := color.New(color.FgHiYellow)
	log.Infof("All CSIStorageCapacity are deleted in %s", yellow.Sprint(time.Since(startTime)))
	return nil
}

// WatchUntilUpdated polls till CSIStorageCapacity object is updated
func (csisc *CSIStorageCapacity) WatchUntilUpdated(ctx context.Context, pollInterval time.Duration) error {
	meta := csisc.Object.ObjectMeta
	timeout := int64(pollInterval.Seconds())

	watcher, err := csisc.Client.Interface.Watch(ctx,
		metav1.ListOptions{
			FieldSelector:   fields.OneTermEqualSelector("metadata.name", meta.Name).String(),
			ResourceVersion: meta.ResourceVersion,
			TimeoutSeconds:  &timeout,
		})
	if err != nil {
		return err
	}

	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Infof("Stopping CSIStorageCapacity wait polling")
			return fmt.Errorf("stopped waiting for CSIStorageCapacity %s to be updated", color.YellowString(csisc.Object.Name))
		case event := <-watcher.ResultChan():
			if len(event.Type) == 0 {
				return fmt.Errorf("%s CSIStorageCapacity object was not updated in %s", color.YellowString(csisc.Object.Name), color.HiYellowString(pollInterval.String()))
			}

			capacity, ok := event.Object.(*storagev1.CSIStorageCapacity)
			if !ok {
				return fmt.Errorf("unexpected type in %v", event)
			}

			if event.Type == watch.Modified && !capacity.Capacity.IsZero() {
				return nil
			}
		}
	}
}
