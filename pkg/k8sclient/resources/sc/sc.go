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

package sc

import (
	"context"

	"github.com/dell/cert-csi/pkg/utils"

	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
)

const (
	// IsReplicationEnabled represents if replication is enabled
	IsReplicationEnabled = "replication.storage.dell.com/isReplicationEnabled"
	// RemoteClusterID represents remote cluster ID
	RemoteClusterID = "replication.storage.dell.com/remoteClusterID"
	// RemoteStorageClassName represents remote sc name
	RemoteStorageClassName = "replication.storage.dell.com/remoteStorageClassName"
)

// Client conatins sc interface and kubeclient
type Client struct {
	// KubeClient *core.KubeClient
	Interface tcorev1.StorageClassInterface
	ClientSet kubernetes.Interface
	Timeout   int
}

// StorageClass conatins pvc client and claim
type StorageClass struct {
	Client  *Client
	Object  *v1.StorageClass
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

// Get uses client interface to make API call for getting provided StorageClass
func (c *Client) Get(ctx context.Context, name string) *StorageClass {
	log := utils.GetLoggerFromContext(ctx)
	var funcErr error
	newSC, err := c.Interface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		funcErr = err
	}

	log.Debugf("Got Storage Class %s", newSC.GetName())
	return &StorageClass{
		Client:  c,
		Object:  newSC,
		Deleted: false,
		error:   funcErr,
	}
}

// Create creates a new storage class
func (c *Client) Create(ctx context.Context, sc *v1.StorageClass) error {
	log := utils.GetLoggerFromContext(ctx)
	_, err := c.Interface.Create(ctx, sc, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Debugf("Created %s Storage Class ", sc.GetName())
	return nil
}

// Delete deletes a storage class
func (c *Client) Delete(ctx context.Context, name string) error {
	log := utils.GetLoggerFromContext(ctx)
	err := c.Interface.Delete(ctx, name, *metav1.NewDeleteOptions(0))
	if err != nil {
		return err
	}
	log.Debugf("Deleted %s Storage Class", name)
	return nil
}

// MakeStorageClass returns a storage class
func (c *Client) MakeStorageClass(name string, provisioner string) *v1.StorageClass {
	WaitForFirstConsumer := v1.VolumeBindingWaitForFirstConsumer
	return &v1.StorageClass{
		ObjectMeta:        metav1.ObjectMeta{Name: name},
		Provisioner:       provisioner,
		VolumeBindingMode: &WaitForFirstConsumer}

}

// DuplicateStorageClass creates a copy of storage class
func (c *Client) DuplicateStorageClass(name string, sourceSc *v1.StorageClass) *v1.StorageClass {
	newSc := sourceSc.DeepCopy()
	newSc.ObjectMeta = metav1.ObjectMeta{Name: name}
	return newSc
}

// HasError checks whether storage class has error
func (sc *StorageClass) HasError() bool {
	if sc.error != nil {
		return true
	}
	return false
}

// GetError returns storage class error
func (sc *StorageClass) GetError() error {
	return sc.error
}
