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

package volumesnapshotcontent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dell/cert-csi/pkg/utils"

	"github.com/fatih/color"
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned/typed/volumesnapshot/v1"
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

// SnapshotContentClient is a client for managing volume blockvolsnapshot content
type SnapshotContentClient struct {
	Interface snapshotv1.VolumeSnapshotContentInterface
	Namespace string
	Timeout   int
}

// SnapshotContent contains parameters needed for managing blockvolsnapshot content
type SnapshotContent struct {
	Client  *SnapshotContentClient
	Object  *v1.VolumeSnapshotContent
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

// Delete deletes a volume blockvolsnapshot content
func (scc *SnapshotContentClient) Delete(ctx context.Context, snap *v1.VolumeSnapshotContent) *SnapshotContent {
	var funcErr error

	err := scc.Interface.Delete(ctx, snap.Name, metav1.DeleteOptions{})
	if err != nil {
		funcErr = err
	}
	logrus.Debugf("Deleted SnapshotContent %s", snap.Name)
	return &SnapshotContent{
		Client:  scc,
		Object:  snap,
		Deleted: true,
		error:   funcErr,
	}
}

// DeleteAll deletes all blockvolsnapshot contents for a PVC
func (scc *SnapshotContentClient) DeleteAll(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	snapList, snapErr := scc.Interface.List(ctx, metav1.ListOptions{})
	if snapErr != nil {
		return snapErr
	}
	log.Debugf("Deleting all SnapContents")
	for i, pvc := range snapList.Items {

		log.Debugf("Deleting snapContent %s", pvc.Name)
		err := scc.Delete(ctx, &snapList.Items[i]).Sync(ctx).GetError()
		if err != nil {
			log.Errorf("Can't delete snapContent %s; error=%v", pvc.Name, err)
		}

	}

	return nil
}

// WaitUntilGone waits until blockvolsnapshot content is deleted
func (cont *SnapshotContent) WaitUntilGone(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	startTime := time.Now()
	timeout := Timeout
	if cont.Client.Timeout != 0 {
		timeout = time.Duration(cont.Client.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout/2, func() (done bool, err error) {
		done, err = cont.pollWait(ctx)
		return done, err
	})
	if pollErr != nil {

		log.Errorf("Failed to delete volsnap: %v \n", pollErr)
		log.Info("Forcing finalizers cleanup")
		gotsnap, err := cont.Client.Interface.Get(ctx, cont.Object.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		gotsnap.SetFinalizers([]string{})
		_, er := cont.Client.Interface.Update(ctx, gotsnap, metav1.UpdateOptions{})
		if er != nil {
			return er
		}
		pollErr = wait.PollImmediate(Poll, timeout/100*95, func() (done bool, err error) {
			done, err = cont.pollWait(ctx)
			return done, err
		})

		if pollErr != nil {
			log.Errorf("Failed to delete volsnap: %v \n", pollErr)
			log.Info("Forcing finalizers cleanup")
			return errors.New("failed to delete even with finalizers cleaned up")
		}
	}
	yellow := color.New(color.FgHiYellow)
	log.Debugf("volsnap %s was deleted in %s", cont.Object.Name, yellow.Sprint(time.Since(startTime)))
	return nil
}

func (cont *SnapshotContent) pollWait(ctx context.Context) (bool, error) {
	log := utils.GetLoggerFromContext(ctx)
	select {
	case <-ctx.Done():
		log.Infof("Stopping Snap wait polling")
		return true, fmt.Errorf("stopped waiting to be ready")
	default:
		break
	}
	if _, err := cont.Client.Interface.Get(ctx, cont.Object.Name, metav1.GetOptions{}); err != nil {
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		log.Errorf("Error while waiting for Snap to be terminated: %v", err)
		return false, err
	}
	return false, nil
}

// WaitForRunning stalls until pod is ready
func (cont *SnapshotContent) WaitForRunning(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Waiting for SnapshotCont '%s' to be Presented", cont.Object.Name)
	timeout := Timeout
	if cont.Client.Timeout != 0 {
		timeout = time.Duration(cont.Client.Timeout) * time.Second
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

			_, err := cont.Client.Interface.Get(ctx, cont.Object.Name, metav1.GetOptions{})
			if err != nil {
				log.Errorf("Can't find volsnap %s", cont.Object.Name)
				return false, err
			}

			ready := true
			return ready, nil
		})

	if pollErr != nil {
		return pollErr
	}
	return nil
}

// HasError checks if blockvolsnapshot content has error
func (cont *SnapshotContent) HasError() bool {
	return cont.error != nil
}

// GetError returns blockvolsnapshot content error
func (cont *SnapshotContent) GetError() error {
	return cont.error
}

// Sync updates blockvolsnapshot content state
func (cont *SnapshotContent) Sync(ctx context.Context) *SnapshotContent {
	if cont.Deleted {
		cont.error = cont.WaitUntilGone(ctx)
	} else {
		cont.error = cont.WaitForRunning(ctx)
	}
	return cont
}
