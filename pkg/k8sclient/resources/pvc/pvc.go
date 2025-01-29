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

package pvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient/resources/commonparams"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/sc"
	"github.com/dell/cert-csi/pkg/utils"

	"github.com/fatih/color"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	v2 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	// Poll is a poll interval for PVC tests
	Poll = 2 * time.Second
	// Timeout is a timeout interval for PVC operations
	Timeout = 1800 * time.Second
	// Block volume mode
	Block = "Block"
)

// Config describes PersistentVolumeClaim
type Config struct {
	// Name defaults to "" if unspecified
	Name string

	// NamePrefix defaults to "pvc-" if unspecified
	NamePrefix string

	// SnapName
	SnapName string

	// SourceVolumeName
	SourceVolumeName string

	// ClaimSize must be specified in the Quantity format. Defaults to 2Gi if
	// unspecified
	ClaimSize string
	// AccessModes defaults to RWO if unspecified
	AccessModes      []v1.PersistentVolumeAccessMode
	Annotations      map[string]string
	Selector         *metav1.LabelSelector
	StorageClassName *string
	// VolumeMode defaults to nil if unspecified or specified as the empty
	// string
	VolumeMode *v1.PersistentVolumeMode
	// Labels to group the volumes for vgs
	Labels map[string]string
}

// Client conatins pvc interface and kubeclient
type Client struct {
	// KubeClient *core.KubeClient
	Interface tcorev1.PersistentVolumeClaimInterface
	ClientSet kubernetes.Interface
	Namespace string
	Timeout   int
	//Name      string //todo: remove if adding a name to Client struct doesn't work
}

// PersistentVolumeClaim conatins pvc client and claim
type PersistentVolumeClaim struct {
	Client  *Client
	Object  *v1.PersistentVolumeClaim
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

// MakePVC creates new PersistentVolumeClaim object from Config
func (c *Client) MakePVC(cfg *Config) *v1.PersistentVolumeClaim {
	if len(cfg.AccessModes) == 0 {
		cfg.AccessModes = append(cfg.AccessModes, v1.ReadWriteOnce)
	}

	if len(cfg.ClaimSize) == 0 {
		cfg.ClaimSize = "3Gi"
	}

	if len(cfg.NamePrefix) == 0 {
		cfg.NamePrefix = "pvc-"
	}

	if cfg.VolumeMode != nil && *cfg.VolumeMode == "" {
		logrus.Warningf("Making PVC: VolumeMode specified as invalid empty string, treating as nil")
		cfg.VolumeMode = nil
	}

	var dataSource *v1.TypedLocalObjectReference
	if cfg.SnapName != "" {
		apiGroup := "snapshot.storage.k8s.io"
		dataSource = &v1.TypedLocalObjectReference{
			APIGroup: &apiGroup,
			Kind:     "VolumeSnapshot",
			Name:     cfg.SnapName,
		}
	} else if cfg.SourceVolumeName != "" {
		dataSource = &v1.TypedLocalObjectReference{
			Kind: "PersistentVolumeClaim",
			Name: cfg.SourceVolumeName,
		}
	}
	objMeta := metav1.ObjectMeta{
		GenerateName: cfg.NamePrefix,
		Namespace:    c.Namespace,
		Annotations:  cfg.Annotations,
		Labels:       cfg.Labels,
	}

	// If custom name is given then create PVC with Name attribute not Generated Name
	if len(cfg.Name) != 0 {
		objMeta = metav1.ObjectMeta{
			Name:        cfg.Name,
			Namespace:   c.Namespace,
			Annotations: cfg.Annotations,
		}
	}

	return &v1.PersistentVolumeClaim{
		ObjectMeta: objMeta,
		Spec: v1.PersistentVolumeClaimSpec{
			Selector:    cfg.Selector,
			AccessModes: cfg.AccessModes,
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse(cfg.ClaimSize),
				},
			},
			StorageClassName: cfg.StorageClassName,
			VolumeMode:       cfg.VolumeMode,
			DataSource:       dataSource,
		},
	}
}

// MakePVCFromYaml makes PersistentVolumeClaim object from provided yaml file
func (c *Client) MakePVCFromYaml(ctx context.Context, filePath string) (*v1.PersistentVolumeClaim, error) {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Making PVC from yaml file at %s", filePath)
	pvc := &v1.PersistentVolumeClaim{}

	file, _ := os.ReadFile(filepath.Clean(filePath))
	log.Debugf("%s \n", string(file))

	errUnmarshal := yaml.Unmarshal(file, pvc)
	if errUnmarshal != nil {
		return nil, errUnmarshal
	}
	log.Infof("Successfully made PVC from yaml file at %s", filePath)
	return pvc, nil
}

// Create uses client interface to make API call for creating provided PersistentVolumeClaim
func (c *Client) Create(ctx context.Context, pvc *v1.PersistentVolumeClaim, _ ...int) *PersistentVolumeClaim {
	log := utils.GetLoggerFromContext(ctx)
	var funcErr error
	newPVC, err := c.Interface.Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		funcErr = err
	}

	log.Debugf("Created PVC %s", newPVC.GetName())
	return &PersistentVolumeClaim{
		Client:  c,
		Object:  newPVC,
		Deleted: false,
		error:   funcErr,
	}
}

// Get uses client interface to make API call for getting provided PersistentVolumeClaim
func (c *Client) Get(ctx context.Context, name string) *PersistentVolumeClaim {
	log := utils.GetLoggerFromContext(ctx)
	var funcErr error
	newPVC, err := c.Interface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		funcErr = err
	}

	log.Debugf("Got PVC %s", newPVC.GetName())
	return &PersistentVolumeClaim{
		Client:  c,
		Object:  newPVC,
		Deleted: false,
		error:   funcErr,
	}
}

// CreateMultiple creates multiple instances of provided PersistentVolumeClaim
func (c *Client) CreateMultiple(ctx context.Context, pvc *v1.PersistentVolumeClaim, pvcNum int, pvcSize string) error {
	if pvcNum <= 0 {
		return errors.New("number of pvcs can't be less or equal than zero")
	}
	if pvcSize == "" {
		return errors.New("volume size cannot be nulls")
	}
	for i := 0; i < pvcNum; i++ {
		_, err := c.Interface.Create(ctx, pvc, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	logrus.Debugf("Created %d PVCs of size:%s", pvcNum, pvcSize)
	return nil
}

// Delete deletes PersistentVolumeClaim from Kubernetes
func (c *Client) Delete(ctx context.Context, pvc *v1.PersistentVolumeClaim) *PersistentVolumeClaim {
	var funcErr error
	err := c.Interface.Delete(ctx, pvc.GetName(), metav1.DeleteOptions{})
	if err != nil {
		funcErr = err
	}
	logrus.Debugf("Deleted PVC %s", pvc.GetName())
	return &PersistentVolumeClaim{
		Client:  c,
		Object:  pvc,
		Deleted: true,
		error:   funcErr,
	}
}

// Update updates a PersistentVolumeClaim
func (c *Client) Update(ctx context.Context, pvc *v1.PersistentVolumeClaim) *PersistentVolumeClaim {
	var funcErr error
	updatedPVC, err := c.Interface.Update(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		funcErr = err
	}
	logrus.Debugf("Updated PVC %s", pvc.GetName())
	return &PersistentVolumeClaim{
		Client:  c,
		Object:  updatedPVC,
		Deleted: false,
		error:   funcErr,
	}
}

// DeleteAll deletes all client pvcs in timely manner, using event subscription model
func (c *Client) DeleteAll(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	podList, podErr := c.Interface.List(ctx, metav1.ListOptions{})
	if podErr != nil {
		return podErr
	}
	log.Debugf("Deleting all PVC")
	for i, pvc := range podList.Items {
		log.Debugf("Deleting pvc [%d/%d]", i+1, len(podList.Items)-1)
		err := c.Delete(ctx, &podList.Items[i]).Sync(ctx).GetError()
		if err != nil {
			log.Errorf("Can't delete pvc %s; error=%v", pvc.Name, err)
		}
	}

	return nil
}

// WaitForAllToBeBound waits for every pvc, that belongs to PVCClient, to be Bound
func (c *Client) WaitForAllToBeBound(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Debugf("Waiting for all PVCs in %s to be %s", c.Namespace, color.GreenString("BOUND"))
	startTime := time.Now()
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pvc wait polling")
				return true, fmt.Errorf("stopped waiting to be bound")
			default:
				break
			}

			pvcList, err := c.Interface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			for _, pvc := range pvcList.Items {
				if pvc.Status.Phase != v1.ClaimBound {
					log.Debugf("PVC %s is still not bound", pvc.Name)
					return false, nil
				}
			}

			return true, nil
		})
	if pollErr != nil {
		return pollErr
	}

	yellow := color.New(color.FgHiYellow)
	log.Debugf("All PVCs in %s are bound in %s", c.Namespace, yellow.Sprint(time.Since(startTime)))
	return nil
}

// CheckAnnotationsForVolumes checking annotations for  every pvc, that belongs to PVCClient, to be Bound
func (c *Client) CheckAnnotationsForVolumes(ctx context.Context, scObject *v2.StorageClass) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Checking annotation and Labels on all PVCs in %s namespace ", c.Namespace)
	startTime := time.Now()
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pvc annotations wait polling")
				return true, fmt.Errorf("stopped waiting for annotations and labels")
			default:
				break
			}
			found := true
			log.Debugf("Getting PVC list")
			pvcList, err := c.Interface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			for _, pvc := range pvcList.Items {
				log.Debugf("Checking annotations and Labels for PVC %s and it has pv %s", pvc.Name, pvc.Spec.VolumeName)
				for _, v := range commonparams.LocalPVCAnnotations {
					if v == "replication.storage.dell.com/remoteClusterID" && pvc.Annotations[v] != scObject.Parameters[sc.RemoteClusterID] {
						log.Debugf("Annotations %s are not added for PVC %s", v, pvc.Name)
						found = false
					} else if v == "replication.storage.dell.com/remoteStorageClassName" && pvc.Annotations[v] != scObject.Parameters[sc.RemoteStorageClassName] {
						log.Debugf("Annotations %s are not added for PVC %s", v, pvc.Name)
						found = false
					} else if pvc.Annotations[v] == "" {
						log.Debugf("Annotations %s are not added for PVC %s", v, pvc.Name)
						found = false
					}
				}
				for _, v := range commonparams.LocalPVCLabels {
					if pvc.Labels[v] == "" {
						log.Debugf("Labels %s are not added for PVC %s", v, pvc.Name)
						found = false
					}
				}
				if !found {
					return false, nil
				}

			}
			return true, nil
		})
	if pollErr != nil {
		return pollErr
	}
	yellow := color.New(color.FgHiYellow)
	log.Infof("Annotations and labels are added for all PVC's in  %s in %s", c.Namespace, yellow.Sprint(time.Since(startTime)))
	return nil
}

// CreatePVCObject will read the remotepvobject and return pvc object for the same
func (c *Client) CreatePVCObject(_ context.Context, remotePVObject *v1.PersistentVolume, remoteNamespace string) v1.PersistentVolumeClaim {
	var (
		pvcName        string
		accessMode     []v1.PersistentVolumeAccessMode
		volMode        *v1.PersistentVolumeMode
		resObj         string
		scName         string
		requestsResReq v1.ResourceList
		remotePVName   string
		pvcObject      v1.PersistentVolumeClaim
	)
	pvcLabels := make(map[string]string, 0)
	pvcAnnotations := make(map[string]string, 0)
	// Iterate through PV labels and apply all replication specific labels
	for key, value := range remotePVObject.Labels {
		if strings.Contains(key, "replication.storage.dell.com") {
			pvcLabels[key] = value
		}
	}
	// Iterate through PV annotations and apply all replication specific annotations
	for key, value := range remotePVObject.Annotations {
		if strings.Contains(key, "replication.storage.dell.com") {
			pvcAnnotations[key] = value
		}
	}

	scName = string(remotePVObject.Spec.StorageClassName)
	pvcName = remotePVObject.Annotations["replication.storage.dell.com/remotePVC"]
	accessMode = remotePVObject.Spec.AccessModes
	volMode = remotePVObject.Spec.VolumeMode
	resObj = remotePVObject.Annotations["replication.storage.dell.com/resourceRequest"]
	remotePVName = remotePVObject.Name
	err := json.Unmarshal([]byte(resObj), &requestsResReq)
	if err != nil {
		fmt.Printf("Failed to unmarshal json for resource requirements. Error: %s\n", err.Error())
		return pvcObject
	}

	pvcObject = v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   remoteNamespace,
			Name:        pvcName,
			Labels:      pvcLabels,
			Annotations: pvcAnnotations,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: accessMode,
			Resources: v1.VolumeResourceRequirements{
				Requests: requestsResReq,
			},
			VolumeName:       remotePVName,
			StorageClassName: &scName,
			VolumeMode:       volMode,
		},
	}
	return pvcObject
}

// WaitToBeBound waits until PVC is Bound
func (pvc *PersistentVolumeClaim) WaitToBeBound(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Debugf("Waiting for  PVC %s to be %s", pvc.Object.Name, color.GreenString("BOUND"))
	startTime := time.Now()
	timeout := Timeout
	if pvc.Client.Timeout != 0 {
		timeout = time.Duration(pvc.Client.Timeout) * time.Second
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pvc wait polling")
				return true, fmt.Errorf("stopped waiting to be bound")
			default:
				break
			}

			if pvc.Object.Status.Phase != v1.ClaimBound {
				log.Debugf("PVC %s is still not bound", pvc.Object.Name)
				return false, nil
			}

			return true, nil
		})
	if pollErr != nil {
		return pollErr
	}

	yellow := color.New(color.FgHiYellow)
	log.Debugf("All PVCs in %s are bound in %s", pvc.Object.Namespace, yellow.Sprint(time.Since(startTime)))
	return nil
}

func watchUntilAllGone(ctx context.Context, finished chan bool, pvcNum int, pvcClient tcorev1.PersistentVolumeClaimInterface) {
	logrus.Debug("Watcher: started watching")
	watch, watchErr := pvcClient.Watch(ctx, metav1.ListOptions{})
	if watchErr != nil {
		logrus.Errorf("Can't watch pvcClient; error = %v", watchErr)
		return
	}
	defer watch.Stop()

	cnt := 0
	for {
		data := <-watch.ResultChan()
		event := data.Type

		if event == "DELETED" {
			cnt++
		}

		if cnt == pvcNum {
			break
		}
	}
	logrus.Debug("Watcher: finished watching")
	finished <- true
}

func (pvc *PersistentVolumeClaim) pollWait(ctx context.Context) (bool, error) {
	log := utils.GetLoggerFromContext(ctx)
	select {
	case <-ctx.Done():
		log.Infof("Stopping pvc wait polling")
		return true, fmt.Errorf("stopped waiting to be bound")
	default:
		break
	}
	if _, err := pvc.Client.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{}); err != nil {
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		log.Errorf("Error while waiting for pvc to be terminated: %v", err)
		return false, err
	}
	return false, nil
}

// WaitUntilGone stalls until said PVC no longer can be found in Kubernetes
func (pvc *PersistentVolumeClaim) WaitUntilGone(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	startTime := time.Now()
	timeout := Timeout
	if pvc.Client.Timeout != 0 {
		timeout = time.Duration(pvc.Client.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout/100*95, func() (done bool, err error) {
		done, err = pvc.pollWait(ctx)
		return done, err
	})
	if pollErr != nil {
		gotpvc, err := pvc.Client.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		log.Errorf("Failed to delete pvc: %v \n", pollErr)
		log.Debug("Forcing finalizers cleanup")
		gotpvc.SetFinalizers([]string{})
		_, er := pvc.Client.Interface.Update(ctx, gotpvc, metav1.UpdateOptions{})
		if er != nil {
			return er
		}
		pollErr = wait.PollImmediate(Poll, timeout/2, func() (done bool, err error) {
			done, err = pvc.pollWait(ctx)
			return done, err
		})

		if pollErr != nil {
			log.Errorf("Failed to delete pvc: %v \n", pollErr)
			log.Info("Forcing finalizers cleanup")
			return errors.New("failed to delete even with finalizers cleaned up")
		}
	}
	yellow := color.New(color.FgHiYellow)
	log.Debugf("Pvc %s was deleted in %s", pvc.Object.Name, yellow.Sprint(time.Since(startTime)))
	return nil
}

// Sync waits until PVC is deleted or bound
func (pvc *PersistentVolumeClaim) Sync(ctx context.Context) *PersistentVolumeClaim {
	if pvc.Deleted {
		pvc.error = pvc.WaitUntilGone(ctx)
	} else {
		pvc.error = pvc.WaitToBeBound(ctx)
	}
	return pvc
}

// HasError checks if PVC contains error
func (pvc *PersistentVolumeClaim) HasError() bool {
	return pvc.error != nil
}

// GetError returns PVC error
func (pvc *PersistentVolumeClaim) GetError() error {
	return pvc.error
}
