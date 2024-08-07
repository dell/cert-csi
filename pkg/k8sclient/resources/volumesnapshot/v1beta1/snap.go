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

package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dell/cert-csi/pkg/utils"

	"github.com/fatih/color"
	"github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	snapshotv1beta1 "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned/typed/volumesnapshot/v1beta1"
	"github.com/sirupsen/logrus"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Poll is a poll interval for Pod
	Poll = 2 * time.Second
	// Timeout is a timeout for Pod operations
	Timeout = 300 * time.Second
)

// SnapshotClient is a client for managing Snapshots
type SnapshotClient struct {
	Interface snapshotv1beta1.VolumeSnapshotInterface
	Namespace string
	Timeout   int
}

// Snapshot contains parameters needed for managing volume snapshot
type Snapshot struct {
	Client  *SnapshotClient
	Object  *v1beta1.VolumeSnapshot
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

// Create creates a snapshot
func (sc *SnapshotClient) Create(ctx context.Context, snap *v1beta1.VolumeSnapshot) *Snapshot {
	var funcErr error
	newSnap, err := sc.Interface.Create(ctx, snap, metav1.CreateOptions{})

	if err != nil {
		funcErr = err
	} else {
		logrus.Debugf("Created Snapshot %s", newSnap.GetName())
	}

	return &Snapshot{
		Client:  sc,
		Object:  newSnap,
		Deleted: false,
		error:   funcErr,
	}
}

// DeleteAll deletes all snapshots associated to a PVC
func (sc *SnapshotClient) DeleteAll(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	snapList, snapErr := sc.Interface.List(ctx, metav1.ListOptions{})
	if snapErr != nil {
		return snapErr
	}
	log.Debugf("Deleting all Snapshots")
	for i, pvc := range snapList.Items {

		log.Debugf("Deleting snap %s", pvc.Name)
		err := sc.Delete(ctx, &snapList.Items[i]).Sync(ctx).GetError()
		if err != nil {
			log.Errorf("Can't delete snap %s; error=%v", pvc.Name, err)
		}

	}

	return nil
}

// Delete deletes a snapshot
func (sc *SnapshotClient) Delete(ctx context.Context, snap *v1beta1.VolumeSnapshot) *Snapshot {
	var funcErr error
	err := sc.Interface.Delete(ctx, snap.Name, metav1.DeleteOptions{})
	if err != nil {
		funcErr = err
	}
	logrus.Debugf("Deleted Snapshot %s", snap.GetName())
	return &Snapshot{
		Client:  sc,
		Object:  snap,
		Deleted: true,
		error:   funcErr,
	}
}

// WaitForAllToBeReady waits until all snapshots are in ReadyToUse state
func (sc *SnapshotClient) WaitForAllToBeReady(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Waiting for the snapshots in %s to be %s", sc.Namespace, color.GreenString("READY"))
	timeout := Timeout
	if sc.Timeout != 0 {
		timeout = time.Duration(sc.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping Snap wait polling")
				return true, fmt.Errorf("stopped waiting to be ready")
			default:
				break
			}

			snapList, err := sc.Interface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			for i := range snapList.Items {
				isReady := IsSnapReady(&snapList.Items[i])
				if !isReady {
					return false, nil
				}
			}

			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}
	return nil
}

// IsSnapReady checks whether snapshot is in ReadyToUse state
func IsSnapReady(sn *v1beta1.VolumeSnapshot) bool {
	var ready bool
	if sn.Status == nil || sn.Status.ReadyToUse == nil {
		ready = false
	} else {
		ready = *sn.Status.ReadyToUse
	}
	logrus.Debugf("Check snapshot %s is ready: %t", sn.Name, ready)
	return ready
}

// WaitUntilGone waits until snapshot is deleted
func (snap *Snapshot) WaitUntilGone(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	startTime := time.Now()
	timeout := Timeout
	if snap.Client.Timeout != 0 {
		timeout = time.Duration(snap.Client.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout/100*95, func() (done bool, err error) {
		done, err = snap.pollWait(ctx)
		return done, err
	})
	if pollErr != nil {
		gotsnap, err := snap.Client.Interface.Get(ctx, snap.Object.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		log.Errorf("Failed to delete snap: %v \n", pollErr)
		log.Info("Forcing finalizers cleanup")
		gotsnap.SetFinalizers([]string{})
		_, er := snap.Client.Interface.Update(ctx, gotsnap, metav1.UpdateOptions{})
		if er != nil {
			return er
		}
		pollErr = wait.PollImmediate(Poll, timeout/2, func() (done bool, err error) {
			done, err = snap.pollWait(ctx)
			return done, err
		})

		if pollErr != nil {
			log.Errorf("Failed to delete snap: %v \n", pollErr)
			log.Info("Forcing finalizers cleanup")
			return errors.New("failed to delete even with finalizers cleaned up")
		}
	}
	yellow := color.New(color.FgHiYellow)
	log.Debugf("snap %s was deleted in %s", snap.Object.Name, yellow.Sprint(time.Since(startTime)))
	return nil
}

func (snap *Snapshot) pollWait(ctx context.Context) (bool, error) {
	log := utils.GetLoggerFromContext(ctx)
	select {
	case <-ctx.Done():
		log.Infof("Stopping Snap wait polling")
		return true, fmt.Errorf("stopped waiting to be ready")
	default:
		break
	}
	if _, err := snap.Client.Interface.Get(ctx, snap.Object.Name, metav1.GetOptions{}); err != nil {
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		log.Errorf("Error while waiting for Snap to be terminated: %v", err)
		return false, err
	}
	return false, nil
}

// WaitForRunning stalls until pod is ready
func (snap *Snapshot) WaitForRunning(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Waiting for Snapshot '%s' to be READY", snap.Object.Name)
	timeout := Timeout
	if snap.Client.Timeout != 0 {
		timeout = time.Duration(snap.Client.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping Snap wait polling")
				return true, fmt.Errorf("stopped waiting to be ready")
			default:
				break
			}

			p, err := snap.Client.Interface.Get(ctx, snap.Object.Name, metav1.GetOptions{})
			if err != nil {
				log.Errorf("Can't find snap %s", snap.Object.Name)
				return false, err
			}

			ready := IsSnapReady(p)
			return ready, nil
		})

	if pollErr != nil {
		return pollErr
	}
	return nil
}

// HasError checks whether Snapshot has error
func (snap *Snapshot) HasError() bool {
	return snap.error != nil
}

// GetError returns snapshot error
func (snap *Snapshot) GetError() error {
	return snap.error
}

// Name returns snapshot name
func (snap *Snapshot) Name() string {
	return snap.Object.Name
}

// Sync updates snapshot state
func (snap *Snapshot) Sync(ctx context.Context) *Snapshot {
	if snap.Deleted {
		snap.error = snap.WaitUntilGone(ctx)
	} else {
		snap.error = snap.WaitForRunning(ctx)
	}
	return snap
}
