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

package volumegroupsnapshot

import (
	"context"
	"fmt"
	"time"

	"github.com/dell/cert-csi/pkg/utils"

	vgsAlpha "github.com/dell/csi-volumegroup-snapshotter/api/v1"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Poll is a poll interval for Pod
	Poll = 2 * time.Second
	// Timeout is a timeout for Pod operations
	Timeout = 1800 * time.Second
	// StatusComplete represents Complete status
	StatusComplete = "Complete"
)

// Client is a client for managing RGs
type Client struct {
	Interface runtimeclient.Client
	ClientSet kubernetes.Interface
	Timeout   int
}

// Config contains parameters specific to VGS
type Config struct {
	Name          string
	Namespace     string
	DriverName    string
	ReclaimPolicy string
	SnapClass     string
	VolumeLabel   string
}

// VGS is contains information specific to a VG snapshot
type VGS struct {
	Client  *Client
	Object  *vgsAlpha.DellCsiVolumeGroupSnapshot
	Deleted bool

	// Used when error arises in sync methods
	error error
}

// Create creates a VGS
func (c *Client) Create(ctx context.Context, vgs *vgsAlpha.DellCsiVolumeGroupSnapshot) *VGS {
	var funcErr error

	err := c.Interface.Create(ctx, vgs)
	if err != nil {
		funcErr = err
	}
	logrus.Debugf("Created VGS %s", vgs.GetName())
	return &VGS{
		Client:  c,
		Object:  vgs,
		Deleted: false,
		error:   funcErr,
	}
}

// Delete deletes a VGS
func (c *Client) Delete(ctx context.Context, vgs *vgsAlpha.DellCsiVolumeGroupSnapshot) *VGS {
	var funcErr error

	err := c.Interface.Delete(ctx, vgs)
	if err != nil {
		funcErr = err
	}
	logrus.Debugf("Deleted VGS %s", vgs.GetName())
	return &VGS{
		Client:  c,
		Object:  vgs,
		Deleted: true,
		error:   funcErr,
	}
}

// Get returns a requested VGS
func (c *Client) Get(ctx context.Context, name, namespace string) *VGS {
	var funcErr error

	vgsObject := &vgsAlpha.DellCsiVolumeGroupSnapshot{}

	err := c.Interface.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, vgsObject)
	if err != nil {
		funcErr = err
	}

	logrus.Debugf("Got the VGS  %s", vgsObject.GetName())
	return &VGS{
		Client:  c,
		Object:  vgsObject,
		Deleted: false,
		error:   funcErr,
	}
}

// MakeVGS returns a VGS object
func (c *Client) MakeVGS(cfg *Config) *vgsAlpha.DellCsiVolumeGroupSnapshot {
	vgObj := &vgsAlpha.DellCsiVolumeGroupSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
		},
		Spec: vgsAlpha.DellCsiVolumeGroupSnapshotSpec{
			DriverName:          cfg.DriverName,
			Volumesnapshotclass: cfg.SnapClass,
			PvcLabel:            cfg.VolumeLabel,
			MemberReclaimPolicy: vgsAlpha.MemberReclaimDelete,
		},
	}
	if cfg.ReclaimPolicy == vgsAlpha.MemberReclaimRetain {
		vgObj.Spec.MemberReclaimPolicy = vgsAlpha.MemberReclaimRetain
	}
	return vgObj
}

// WaitForComplete waits until VGS is in completed state
func (c *Client) WaitForComplete(ctx context.Context, name, namespace string) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Waiting for VGS to be in %s state", color.GreenString("COMPLETE"))
	startTime := time.Now()
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}
	var snapList string

	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping vgs wait polling")
				return true, fmt.Errorf("stopped waiting to be completed")
			default:
				break
			}

			gotVg := c.Get(ctx, name, namespace)
			if gotVg.Object.Status.Status != StatusComplete {
				return false, nil
			}
			snapList = gotVg.Object.Status.Snapshots
			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}

	yellow := color.New(color.FgHiYellow)
	log.Infof("VGS is in %s state in %s seconds", color.GreenString("COMPLETE"), yellow.Sprint(time.Since(startTime)))
	log.Infof("Snapshots under the vgs %s", yellow.Sprint(snapList))
	return nil
}

// Name return VGS name
func (vgs *VGS) Name() string {
	return vgs.Object.Name
}

// HasError checks whether VGS has error
func (vgs *VGS) HasError() bool {
	return vgs.error != nil
}

// GetError returns VGS error
func (vgs *VGS) GetError() error {
	return vgs.error
}
