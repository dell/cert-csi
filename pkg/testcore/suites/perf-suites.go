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

package suites

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/commonparams"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/node"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pv"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/replicationgroup"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/sc"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshot"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/utils"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"

	snapv1client "github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshot/v1"
	snapbetaclient "github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshot/v1beta1"

	"math/rand"
	"time"

	"github.com/fatih/color"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapbeta "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultSnapPrefix is snapshot prefix
	DefaultSnapPrefix = "snap"
	// ControllerLogsSleepTime is controller logs sleep time
	ControllerLogsSleepTime = 20
	maxRetryCount           = 30
)

// VolumeCreationSuite is used to manage volume creation test suite
type VolumeCreationSuite struct {
	VolumeNumber int
	Description  string
	VolumeSize   string
	CustomName   string
	AccessMode   string
	RawBlock     bool
}

// Run executes volume creation test suite
func (vcs *VolumeCreationSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	if vcs.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		vcs.VolumeNumber = 10
	}
	if vcs.VolumeSize == "" {
		log.Info("Using default volume size")
		vcs.VolumeSize = "3Gi"
	}
	result := validateCustomName(vcs.CustomName, vcs.VolumeNumber)
	if result {
		log.Infof("using custom pvc-name:%s", vcs.CustomName)
	} else {
		vcs.CustomName = ""
	}

	log.Infof("Creating %s volumes with size:%s", color.YellowString(strconv.Itoa(vcs.VolumeNumber)),
		color.YellowString(vcs.VolumeSize))
	pvcClient := clients.PVCClient

	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return delFunc, err
	}
	// Making PVC template from golden testing config
	vcconf := testcore.VolumeCreationConfig(storageClass, vcs.VolumeSize, vcs.CustomName, vcs.AccessMode)
	if vcs.RawBlock {
		log.Info(color.YellowString("Creating Raw Block Volume"))
		var mode v1.PersistentVolumeMode = pvc.Block
		vcconf.VolumeMode = &mode
	}
	tmpl := pvcClient.MakePVC(vcconf)

	// Making API call to create `VolumeNumber` of PVCS
	createErr := pvcClient.CreateMultiple(ctx, tmpl, vcs.VolumeNumber, vcs.VolumeSize)
	if createErr != nil {
		return delFunc, createErr
	}

	// Wait until all PVCs will be bound
	if !firstConsumer {
		boundErr := pvcClient.WaitForAllToBeBound(ctx)
		if boundErr != nil {
			return delFunc, boundErr
		}
	}

	return delFunc, nil
}

func shouldWaitForFirstConsumer(ctx context.Context, storageClass string, pvcClient *pvc.Client) (bool, error) {
	s, err := pvcClient.ClientSet.StorageV1().StorageClasses().Get(ctx, storageClass, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return *s.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer, nil
}

// GetName returns volume creation suite name
func (vcs *VolumeCreationSuite) GetName() string {
	if vcs.Description != "" {
		return vcs.Description
	}
	return "VolumeCreationSuite"
}

// GetObservers returns pvc, entity number, container metrics observers
func (*VolumeCreationSuite) GetObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PvcObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	} else if obsType == observer.LIST {
		return []observer.Interface{
			&observer.PvcListObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

func validateCustomName(name string, volumes int) bool {
	// If no. of volumes is only 1 then we will take custom name else no.
	if volumes == 1 && len(name) != 0 {
		return true
	}
	// we will use custom name only if number of volumes is 1 else we will discard custom name
	return false
}

// GetClients creates and returns pvc and metrics clients
func (*VolumeCreationSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         nil,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
	}, nil
}

// GetNamespace returns volume creation suite name
func (*VolumeCreationSuite) GetNamespace() string {
	return "vcs-test"
}

// Parameters returns formatted string of parameters
func (vcs *VolumeCreationSuite) Parameters() string {
	return fmt.Sprintf("{number: %d, size: %s, raw-block: %s}", vcs.VolumeNumber, vcs.VolumeSize, strconv.FormatBool(vcs.RawBlock))
}

// ProvisioningSuite is used to manage provisioning test suite
type ProvisioningSuite struct {
	VolumeNumber  int
	VolumeSize    string
	PodCustomName string
	Description   string
	PodNumber     int
	RawBlock      bool
	VolAccessMode string
	ROFlag        bool
}

// Run executes provisioning test suite
func (ps *ProvisioningSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	if ps.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		ps.VolumeNumber = 5
	}
	if ps.PodNumber <= 0 {
		log.Info("Using default number of pods")
		ps.PodNumber = 1
	}
	if ps.VolumeSize == "" {
		log.Info("Using default volume size 3Gi")
		ps.VolumeSize = "3Gi"
	}
	ps.validateCustomPodName()

	log.Infof("Creating %s pods, each with %s volumes", color.YellowString(strconv.Itoa(ps.PodNumber)),
		color.YellowString(strconv.Itoa(ps.VolumeNumber)))

	for i := 0; i < ps.PodNumber; i++ {
		var pvcNameList []string
		for j := 0; j < ps.VolumeNumber; j++ {
			// Create PVCs
			var volumeName string
			if ps.PodCustomName != "" {
				volumeName = ps.PodCustomName + "-pvc-" + strconv.Itoa(j)
			}
			vcconf := testcore.VolumeCreationConfig(storageClass, ps.VolumeSize, volumeName, ps.VolAccessMode)
			if ps.RawBlock {
				log.Info(color.YellowString("Creating Raw Block Volumes"))
				var mode v1.PersistentVolumeMode = pvc.Block
				vcconf.VolumeMode = &mode
			}
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return delFunc, pvc.GetError()
			}

			pvcNameList = append(pvcNameList, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, ps.PodCustomName)
		if ps.RawBlock {
			podconf.VolumeMode = pod.Block
		}
		if ps.ROFlag {
			podconf.ReadOnlyFlag = true
		}
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	return delFunc, nil
}

// GetObservers returns all observers
func (*ProvisioningSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients returns pvc, pod, va, metrics clients
func (*ProvisioningSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
	}, nil
}

// GetNamespace returns provisioning suite namespace
func (*ProvisioningSuite) GetNamespace() string {
	return "prov-test"
}

// GetName returns provisioning suite name
func (ps *ProvisioningSuite) GetName() string {
	if ps.Description != "" {
		return ps.Description
	}
	return "ProvisioningSuite"
}

// Parameters returns formatted string of parameters
func (ps *ProvisioningSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, volumeSize: %s}", ps.PodNumber, ps.VolumeNumber, ps.VolumeSize)
}

func (ps *ProvisioningSuite) validateCustomPodName() {
	// If no. of pods is only 1 then we will take custom name else generated name will be used.
	if ps.PodNumber == 1 && len(ps.PodCustomName) != 0 {
		logrus.Infof("using custom pod-name:%s", ps.PodCustomName)
	} else {
		// we will use custom name only if number of volumes is 1 else we will discard custom name
		ps.PodCustomName = ""
	}
}

// RemoteReplicationProvisioningSuite is used to manage remote replication provisioning test suite
type RemoteReplicationProvisioningSuite struct {
	VolumeNumber     int
	VolumeSize       string
	Description      string
	VolAccessMode    string
	RemoteConfigPath string
	NoFailover       bool
}

// Run executes remote replication provisioning test suite
func (rrps *RemoteReplicationProvisioningSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	pvClient := clients.PersistentVolumeClient
	scClient := clients.SCClient
	rgClient := clients.RgClient
	storClass, err := scClient.Interface.Get(ctx, storageClass, metav1.GetOptions{})
	if err != nil {
		return delFunc, err
	}

	var (
		remotePVCObject  v1.PersistentVolumeClaim
		remotePVClient   *pv.Client
		remoteRGClient   *replicationgroup.Client
		remoteKubeClient *k8sclient.KubeClient
	)

	isSingle := false
	if storClass.Parameters["replication.storage.dell.com/remoteClusterID"] == "self" {
		isSingle = true
	}

	if rrps.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		rrps.VolumeNumber = 1
	}
	if rrps.VolumeSize == "" {
		log.Info("Using default volume size 3Gi")
		rrps.VolumeSize = "3Gi"
	}

	if rrps.RemoteConfigPath != "" && !isSingle {
		// Loading config
		remoteConfig, err := k8sclient.GetConfig(rrps.RemoteConfigPath)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		// Connecting to host and creating new Kubernetes Client
		remoteKubeClient, err = k8sclient.NewRemoteKubeClient(remoteConfig, pvClient.Timeout)
		if err != nil {
			log.Errorf("Couldn't create new Remote kubernetes client. Error = %v", err)
			return nil, err
		}
		remotePVClient, err = remoteKubeClient.CreatePVClient()
		if err != nil {
			return nil, err
		}
		remoteRGClient, err = remoteKubeClient.CreateRGClient()
		if err != nil {
			return nil, err
		}

		log.Info("Created remote kube client")
	} else {
		remotePVClient = pvClient
		remoteRGClient = rgClient
		remoteKubeClient = clients.KubeClient
	}

	scObject, scErr := scClient.Interface.Get(ctx, storageClass, metav1.GetOptions{})
	if scErr != nil {
		return nil, scErr
	}

	rpEnabled := scObject.Parameters[sc.IsReplicationEnabled]
	if rpEnabled != "true" {
		return nil, fmt.Errorf("replication is not enabled on this storage class and please provide valid sc")
	}

	log.Infof("Creating %s volumes", color.YellowString(strconv.Itoa(rrps.VolumeNumber)))
	var pvcNames []string
	for i := 0; i < rrps.VolumeNumber; i++ {
		// Create PVCs
		var volumeName string
		vcconf := testcore.VolumeCreationConfig(storageClass, rrps.VolumeSize, volumeName, rrps.VolAccessMode)
		volTmpl := pvcClient.MakePVC(vcconf)
		pvc := pvcClient.Create(ctx, volTmpl)
		if pvc.HasError() {
			return delFunc, pvc.GetError()
		}
		pvcNames = append(pvcNames, pvc.Object.Name)
	}

	err = pvcClient.WaitForAllToBeBound(ctx)
	if err != nil {
		return delFunc, err
	}

	var pvNames []string
	// We can get actual pv names only after all pvc are bound
	pvcList, err := pvcClient.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return delFunc, err
	}
	for _, p := range pvcList.Items {
		pvNames = append(pvNames, p.Spec.VolumeName)
	}

	log.Info("Creating pod for each volume")
	for _, name := range pvcNames {
		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig([]string{name}, "")
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl).Sync(ctx)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}

		// write data to files and calculate checksum.
		file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
		sum := fmt.Sprintf("%s0/writer-%d.sha512", podconf.MountPath, 0)

		// Write random blob to pvc
		ddRes := bytes.NewBufferString("")
		if err := podClient.Exec(ctx, pod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
			return delFunc, err
		}

		log.Info("Writer pod: ", pod.Object.GetName())
		log.Debug(ddRes.String())
		log.Info("Written the values successfully ", ddRes)
		log.Info(ddRes.String())

		// Write hash sum of blob
		if err := podClient.Exec(ctx, pod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return delFunc, err
		}
		log.Info("Checksum value: ", sum)

		// sync to be sure
		if err := podClient.Exec(ctx, pod.Object, []string{"/bin/bash", "-c", "sync " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return delFunc, err
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	log.Infof("Checking annotations for all PVC's")

	// Wait until to get annotations for all PVCs

	boundErr := pvcClient.CheckAnnotationsForVolumes(ctx, scObject)
	if boundErr != nil {
		return delFunc, boundErr
	}

	log.Infof("Successfully checked annotations On PVC")

	log.Infof("Checking annotation on pv's")
	for _, pvName := range pvNames {
		// Check on local cluster
		pvObject, pvErr := pvClient.Interface.Get(ctx, pvName, metav1.GetOptions{})
		if pvErr != nil {
			return delFunc, pvErr
		}
		err := pvClient.CheckReplicationAnnotationsForPV(ctx, pvObject)
		if err != nil {
			return delFunc, fmt.Errorf("replication Annotations and Labels are not added for PV %s and %s", pvName, err)
		}

		// Check on remote cluster
		if !isSingle {
			remotePvName := pvName
			remotePVObject, remotePVErr := remotePVClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return delFunc, remotePVErr
			}
			err = remotePVClient.CheckReplicationAnnotationsForRemotePV(ctx, remotePVObject)
			if err != nil {
				return delFunc, fmt.Errorf("replication Annotations and Labels are not added for Remote PV %s and %s", remotePvName, err)
			}
		} else {
			remotePvName := "replicated-" + pvName
			remotePVObject, remotePVErr := pvClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return delFunc, remotePVErr
			}
			err = pvClient.CheckReplicationAnnotationsForRemotePV(ctx, remotePVObject)
			if err != nil {
				return delFunc, fmt.Errorf("replication Annotations and Labels are not added for Remote PV %s and %s", remotePvName, err)
			}
		}
	}
	log.Infof("Successfully checked annotations on local and remote pv")

	// List PVCs once again, since here we can be sure that all annotations will be correctly set
	pvcList, err = pvcClient.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return delFunc, err
	}

	rgName := pvcList.Items[0].Annotations[commonparams.ReplicationGroupName]
	log.Infof("The replication group name from pvc is %s ", rgName)

	// Add remote RG deletion step to deletion callback function
	delFunc = func(f func() error) func() error {
		return func() error {
			log.Info("Deleting local RG")
			rgObject := rgClient.Get(context.Background(), rgName)
			deletedLocalRG := rgClient.Delete(context.Background(), rgObject.Object)
			if deletedLocalRG.HasError() {
				log.Warnf("error when deleting local RG: %s", deletedLocalRG.GetError().Error())
			}

			log.Info("Deleting remote RG")
			remoteRgObject := remoteRGClient.Get(context.Background(), rgName)
			deletedRemoteRG := remoteRGClient.Delete(context.Background(), remoteRgObject.Object)
			if deletedRemoteRG.HasError() {
				log.Warnf("error when deleting remote RG: %s", deletedRemoteRG.GetError().Error())
			}

			log.Info("Sleeping for 1 minute...") // TODO: maybe do some polling instead, idk rn
			// Sleeping for 1 minutes
			time.Sleep(1 * time.Minute)

			return nil
		}
	}(nil)

	log.Infof("Checking Annotations and Labels on ReplicationGroup")

	rgObject := rgClient.Get(ctx, rgName)
	var remoteRgName string
	if !isSingle {
		remoteRgName = rgName
	} else {
		remoteRgName = "replicated-" + rgName
	}

	remoteRgObject := remoteRGClient.Get(ctx, remoteRgName)

	if rgObject.Object.Annotations["replication.storage.dell.com/remoteReplicationGroupName"] !=
		remoteRgObject.Object.Labels["replication.storage.dell.com/remoteReplicationGroupName"] &&
		rgObject.Object.Annotations[commonparams.RemoteClusterID] != scObject.Parameters[sc.RemoteClusterID] {
		return delFunc, fmt.Errorf("expected Annotations are not added to the replication group %s", rgName)
	}

	if rgObject.Object.Labels[commonparams.RemoteClusterID] != scObject.Parameters[sc.RemoteClusterID] {
		return delFunc, fmt.Errorf("expected Labels are not added to the replication group %s", rgName)
	}
	log.Infof("Successfully Checked Annotations and Labels on ReplicationGroup")

	// Return if failover is not requested
	if rrps.NoFailover {
		return delFunc, nil
	}

	// Failover to the target site
	log.Infof("Executing failover action on ReplicationGroup %s", remoteRgName)
	actionErr := rgObject.ExecuteAction(ctx, "FAILOVER_REMOTE")
	if actionErr != nil {
		return delFunc, actionErr
	}

	log.Infof("Executing reprotect action on ReplicationGroup %s", remoteRgName)
	remoteRgObject = remoteRGClient.Get(ctx, remoteRgName)
	actionErr = remoteRgObject.ExecuteAction(ctx, "REPROTECT_LOCAL")
	if actionErr != nil {
		return delFunc, actionErr
	}

	log.Infof("Creating pvc and pods on remote cluster")
	if !isSingle {
		ns, err := remoteKubeClient.CreateNamespace(ctx, clients.PVCClient.Namespace)
		if err != nil {
			return delFunc, err
		}
		remoteNamespace := ns.Name
		remotePVCClient, err := remoteKubeClient.CreatePVCClient(ns.Name)
		if err != nil {
			return delFunc, err
		}
		remotePodClient, err := remoteKubeClient.CreatePodClient(ns.Name)
		if err != nil {
			return delFunc, err
		}

		var pvcNameList []string

		for _, pvc := range pvcList.Items {
			log.Infof("Creating remote pvc %s", pvc.Name)

			remotePvName := pvc.Spec.VolumeName
			remotePVObject, remotePVErr := remotePVClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return delFunc, remotePVErr
			}
			pvcName := remotePVObject.Annotations["replication.storage.dell.com/remotePVC"]
			pvcNameList = append(pvcNameList, pvcName)

			remotePVCObject = pvcClient.CreatePVCObject(ctx, remotePVObject, remoteNamespace)
			remotePvc := remotePVCClient.Create(ctx, &remotePVCObject)
			if remotePvc.HasError() {
				return delFunc, remotePvc.GetError()
			}

		}
		for _, pvcName := range pvcNameList {
			log.Infof("Verifying data from pvc %s", pvcName)

			podConf := testcore.ProvisioningPodConfig([]string{pvcName}, "")
			podTemp := remotePodClient.MakePod(podConf)

			remotePod := remotePodClient.Create(ctx, podTemp).Sync(ctx)
			if remotePod.HasError() {
				return delFunc, remotePod.GetError()
			}

			sum := fmt.Sprintf("%s0/writer-%d.sha512", podConf.MountPath, 0)
			writer := bytes.NewBufferString(" ")
			log.Info("Checker pod: ", remotePod.Object.GetName())
			if err := remotePodClient.Exec(ctx, remotePod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
				log.Error("Failed to verify hash")
				return delFunc, err
			}

			if strings.Contains(writer.String(), "OK") {
				log.Info("Hashes match")
			} else {
				return delFunc, fmt.Errorf("hashes don't match")
			}
		}
	} else {
		ns, err := clients.KubeClient.CreateNamespace(ctx, "replicated-"+clients.PVCClient.Namespace)
		if err != nil {
			return delFunc, err
		}

		remoteNamespace := ns.Name
		remotePVCClient, err := clients.KubeClient.CreatePVCClient(ns.Name)
		if err != nil {
			return delFunc, err
		}
		remotePodClient, err := clients.KubeClient.CreatePodClient(ns.Name)
		if err != nil {
			return delFunc, err
		}

		var pvcNameList []string

		for _, pvc := range pvcList.Items {
			log.Infof("Creating remote pvc %s", pvc.Name)

			remotePvName := "replicated-" + pvc.Spec.VolumeName
			remotePVObject, remotePVErr := pvClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return delFunc, remotePVErr
			}
			pvcName := remotePVObject.Annotations["replication.storage.dell.com/remotePVC"]
			pvcNameList = append(pvcNameList, pvcName)

			remotePVCObject = pvcClient.CreatePVCObject(ctx, remotePVObject, remoteNamespace)
			remotePvc := remotePVCClient.Create(ctx, &remotePVCObject)
			if remotePvc.HasError() {
				return delFunc, remotePvc.GetError()
			}

		}
		for _, pvcName := range pvcNameList {
			log.Infof("Verifying data from pvc %s", pvcName)

			podConf := testcore.ProvisioningPodConfig([]string{pvcName}, "")
			podTemp := remotePodClient.MakePod(podConf)

			remotePod := remotePodClient.Create(ctx, podTemp).Sync(ctx)
			if remotePod.HasError() {
				return delFunc, remotePod.GetError()
			}

			sum := fmt.Sprintf("%s0/writer-%d.sha512", podConf.MountPath, 0)
			writer := bytes.NewBufferString(" ")
			log.Info("Checker pod: ", remotePod.Object.GetName())
			if err := remotePodClient.Exec(ctx, remotePod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
				log.Error("Failed to verify hash")
				return delFunc, err
			}

			if strings.Contains(writer.String(), "OK") {
				log.Info("Hashes match")
			} else {
				return delFunc, fmt.Errorf("hashes don't match")
			}
		}

	}
	// Initializing deletion callback function
	delFunc = func(f func() error) func() error {
		return func() error {
			err := f()
			if err != nil {
				return err
			}
			if !isSingle {
				log.Infof("Deleting remote namespace %s", clients.PVCClient.Namespace)
				err = remoteKubeClient.DeleteNamespace(context.Background(), clients.PVCClient.Namespace)
				if err != nil {
					log.Warnf("error deleting remote namespace: %s", err.Error())
				}
			} else {
				log.Infof("Deleting remote namespace %s", "replicated-"+clients.PVCClient.Namespace)
				err = remoteKubeClient.DeleteNamespace(context.Background(), "replicated-"+clients.PVCClient.Namespace)
				if err != nil {
					log.Warnf("error deleting remote namespace: %s", err.Error())
				}
			}

			return nil
		}
	}(delFunc)

	// failover back to source side to ease deletion of resources
	remoteRgObject = remoteRGClient.Get(ctx, remoteRgName)
	log.Infof("Executing failover action using remote RG %s", remoteRgName)
	actionErr = remoteRgObject.ExecuteAction(ctx, "FAILOVER_REMOTE")
	if actionErr != nil {
		return delFunc, actionErr
	}

	log.Infof("Executing reprotect action using local RG %s", remoteRgName)
	rgObject = rgClient.Get(ctx, remoteRgName)
	actionErr = rgObject.ExecuteAction(ctx, "REPROTECT_LOCAL")
	if actionErr != nil {
		return delFunc, actionErr
	}

	return delFunc, nil
}

// GetObservers returns all observers
func (*RemoteReplicationProvisioningSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, pv, va, metrics, sc, rg clients
func (*RemoteReplicationProvisioningSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	pvClient, pvErr := client.CreatePVClient()
	if pvErr != nil {
		return nil, pvErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	scClient, scErr := client.CreateSCClient()
	if scErr != nil {
		return nil, scErr
	}
	rgClient, rgErr := client.CreateRGClient()
	if rgErr != nil {
		return nil, rgErr
	}
	return &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		VaClient:               vaClient,
		MetricsClient:          metricsClient,
		PersistentVolumeClient: pvClient,
		SCClient:               scClient,
		RgClient:               rgClient,
		KubeClient:             client,
	}, nil
}

// GetNamespace returns remote replication provisioning suite namespace
func (*RemoteReplicationProvisioningSuite) GetNamespace() string {
	return "repl-prov-test"
}

// GetName returns remote replication provisioning suite name
func (rrps *RemoteReplicationProvisioningSuite) GetName() string {
	if rrps.Description != "" {
		return rrps.Description
	}
	return "RemoteReplicationProvisioningSuite"
}

// Parameters returns formatted string of parameters
func (rrps *RemoteReplicationProvisioningSuite) Parameters() string {
	return fmt.Sprintf("{volumes: %d, volumeSize: %s, remoteConfig: %s}", rrps.VolumeNumber, rrps.VolumeSize, rrps.RemoteConfigPath)
}

// ScalingSuite is used to manage scaling test suite
type ScalingSuite struct {
	ReplicaNumber    int
	VolumeNumber     int
	GradualScaleDown bool
	PodPolicy        string
	VolumeSize       string
}

// Run executes scaling test suite
func (ss *ScalingSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	stsClient := clients.StatefulSetClient

	if ss.ReplicaNumber < 1 {
		log.Errorf("Can't use %d replicas, using default number", ss.ReplicaNumber)
		ss.ReplicaNumber = 5
	}
	if ss.VolumeNumber < 1 {
		log.Errorf("Can't use %d volumes, using default number", ss.VolumeNumber)
		ss.VolumeNumber = 10
	}
	if ss.PodPolicy == "" {
		log.Info("Using default podManagementPolicy: Parallel")
		ss.PodPolicy = "Parallel"
	}
	if ss.VolumeSize == "" {
		log.Info("Using default volume size 3Gi")
		ss.VolumeSize = "3Gi"
	}

	stsconf := testcore.ScalingStsConfig(storageClass, ss.VolumeSize, ss.VolumeNumber, ss.PodPolicy)
	stsTmpl := stsClient.MakeStatefulSet(stsconf)

	// Creating new statefulset
	sts := stsClient.Create(ctx, stsTmpl)
	if sts.HasError() {
		return delFunc, sts.GetError()
	}

	sts = sts.Sync(ctx)
	if sts.HasError() {
		return delFunc, sts.GetError()
	}

	// Scaling to needed number of replicas
	sts = stsClient.Scale(ctx, sts.Set, int32(ss.ReplicaNumber))
	if sts.HasError() {
		return delFunc, sts.GetError()
	}
	sts.Sync(ctx)

	// Scaling to zero
	if !ss.GradualScaleDown {
		sts := stsClient.Scale(ctx, sts.Set, 0)
		if sts.HasError() {
			return delFunc, sts.GetError()
		}
		sts.Sync(ctx)
	} else {
		log.Info("Gradually scaling down sts")
		for i := ss.ReplicaNumber - 1; i >= 0; i-- {
			sts := stsClient.Scale(ctx, sts.Set, int32(i))
			if sts.HasError() {
				return delFunc, sts.GetError()
			}
			sts.Sync(ctx)
		}
	}

	return delFunc, nil
}

// GetName returns scaling test suite name
func (ss *ScalingSuite) GetName() string {
	return "ScalingSuite"
}

// Parameters returns formatted string of parameters
func (ss *ScalingSuite) Parameters() string {
	return fmt.Sprintf("{replicas: %d, volumes: %d, volumeSize: %s}", ss.ReplicaNumber, ss.VolumeNumber, ss.VolumeSize)
}

// GetObservers returns all observers
func (ss *ScalingSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, statefulset, metrics clients
func (ss *ScalingSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	stsClient, stsErr := client.CreateStatefulSetClient(namespace)
	if stsErr != nil {
		return nil, stsErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: stsClient,
		MetricsClient:     metricsClient,
	}, nil
}

// GetNamespace returns scaling test suite namespace
func (ss *ScalingSuite) GetNamespace() string {
	return "scale-test"
}

// VolumeIoSuite is used to manage volume IO test suite
type VolumeIoSuite struct {
	VolumeNumber int
	VolumeSize   string
	ChainNumber  int
	ChainLength  int
}

// Run executes volume IO test suite
func (vis *VolumeIoSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	vaClient := clients.VaClient

	if vis.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		vis.VolumeNumber = 1
	}

	if vis.ChainNumber <= 0 {
		log.Info("Using default number of chains")
		vis.ChainNumber = 5
	}

	if vis.ChainLength <= 0 {
		log.Info("Using default length of chains")
		vis.ChainLength = 5
	}

	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return delFunc, err
	}

	log.Info("Creating IO pod")
	errs, errCtx := errgroup.WithContext(ctx)
	for j := 0; j < vis.ChainNumber; j++ {
		j := j // https://golang.org/doc/faq#closures_and_goroutines
		// Create PVCs
		var pvcNameList []string
		vcconf := testcore.VolumeCreationConfig(storageClass, vis.VolumeSize, "", "")
		volTmpl := pvcClient.MakePVC(vcconf)

		pvc := pvcClient.Create(ctx, volTmpl)
		if pvc.HasError() {
			return delFunc, pvc.GetError()
		}

		pvcNameList = append(pvcNameList, pvc.Object.Name)

		if !firstConsumer {
			err := pvcClient.WaitForAllToBeBound(errCtx)
			if err != nil {
				return delFunc, err
			}
		}

		gotPvc, err := pvcClient.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
		if err != nil {
			return delFunc, err
		}

		pvName := gotPvc.Spec.VolumeName
		// Create Pod, and attach PVC
		podconf := testcore.IoWritePodConfig(pvcNameList, "")
		podTmpl := podClient.MakePod(podconf)
		errs.Go(func() error {
			for i := 0; i < vis.ChainLength; i++ {
				file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, j)
				sum := fmt.Sprintf("%s0/writer-%d.sha512", podconf.MountPath, j)
				writerPod := podClient.Create(ctx, podTmpl).Sync(errCtx)
				if writerPod.HasError() {
					return writerPod.GetError()
				}

				if i != 0 {
					writer := bytes.NewBufferString("")
					if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
						return err
					}
					if strings.Contains(writer.String(), "OK") {
						log.Info("Hashes match")
					} else {
						return fmt.Errorf("hashes don't match")
					}
				}
				ddRes := bytes.NewBufferString("")
				if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "dd if=/dev/urandom bs=1M count=128 oflag=sync > " + file}, ddRes, os.Stderr, false); err != nil {
					log.Info(err)
					return err
				}

				log.Debug(ddRes.String())
				if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
					return err
				}
				podClient.Delete(ctx, writerPod.Object).Sync(errCtx)
				if writerPod.HasError() {
					return writerPod.GetError()
				}

				// WAIT FOR VA TO BE DELETED
				err := vaClient.WaitUntilVaGone(ctx, pvName)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}

	return delFunc, errs.Wait()
}

// GetObservers returns all observers
func (*VolumeIoSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients returns pvc, pod, va, metrics clients
func (*VolumeIoSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
	}, nil
}

// GetNamespace returns volume IO test suite namespace
func (*VolumeIoSuite) GetNamespace() string {
	return "volumeio-test"
}

// GetName returns volume IO test suite name
func (*VolumeIoSuite) GetName() string {
	return "VolumeIoSuite"
}

// Parameters returns formatted string of parameters
func (vis *VolumeIoSuite) Parameters() string {
	return fmt.Sprintf("{volumes: %d, volumeSize: %s chains: %d-%d}", vis.VolumeNumber, vis.VolumeSize,
		vis.ChainNumber, vis.ChainLength)
}

// VolumeGroupSnapSuite is used to manage volume group snap test suite
type VolumeGroupSnapSuite struct {
	SnapClass       string
	VolumeSize      string
	AccessMode      string
	VolumeGroupName string
	VolumeLabel     string
	ReclaimPolicy   string
	VolumeNumber    int
	Driver          string
}

// Run executes volume group snap test suite
func (vgs *VolumeGroupSnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	var namespace string

	if vgs.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		vgs.VolumeSize = "3Gi"
	}

	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	vgsClient := clients.VgsClient

	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return delFunc, err
	}

	// Create PVCs
	var pvcNameList []string
	vcconf := testcore.VolumeCreationConfig(storageClass, vgs.VolumeSize, "", vgs.AccessMode)
	vcconf.Labels = map[string]string{"volume-group": vgs.VolumeLabel}

	for i := 0; i <= vgs.VolumeNumber; i++ {
		volTmpl := pvcClient.MakePVC(vcconf)
		pvc := pvcClient.Create(ctx, volTmpl)
		if pvc.HasError() {
			return delFunc, pvc.GetError()
		}
		pvcNameList = append(pvcNameList, pvc.Object.Name)
		// we will take namespace from the PVC created, As the namespace is dynamically generated
		if namespace == "" {
			namespace = pvc.Object.Namespace
		}
		if !firstConsumer {
			err := pvcClient.WaitForAllToBeBound(ctx)
			if err != nil {
				return delFunc, err
			}
		}
	}

	// Create Pod, and attach PVC
	podconf := testcore.IoWritePodConfig(pvcNameList, "")
	podTmpl := podClient.MakePod(podconf)

	writerPod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// create volume group snapshot using CRD.
	vgsConfig := testcore.VolumeGroupSnapConfig(vgs.VolumeGroupName, vgs.Driver, vgs.ReclaimPolicy, vgs.SnapClass, vgs.VolumeLabel, namespace)
	vgsTemplate := vgsClient.MakeVGS(vgsConfig)
	vg := vgsClient.Create(ctx, vgsTemplate)
	if vg.HasError() {
		return delFunc, vg.GetError()
	}

	// Poll until the status is complete
	err = vgsClient.WaitForComplete(ctx, vgsConfig.Name, vgsConfig.Namespace)
	if err != nil {
		return delFunc, err
	}
	return delFunc, nil
}

// GetObservers returns all observers
func (*VolumeGroupSnapSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, vgs clients
func (vgs *VolumeGroupSnapSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.SnapshotClassExists(vgs.SnapClass); !ok {
		return nil, fmt.Errorf("snapshotclass class doesn't exist; error = %v", err)
	}

	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	vgsClient, vaErr := client.CreateVGSClient()
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	return &k8sclient.Clients{
		PVCClient:     pvcClient,
		PodClient:     podClient,
		VaClient:      vaClient,
		VgsClient:     vgsClient,
		MetricsClient: metricsClient,
	}, nil
}

// GetNamespace returns volume group snap test suite namespace
func (*VolumeGroupSnapSuite) GetNamespace() string {
	return "vgs-snap-test"
}

// GetName returns volume group snap test suite name
func (vgs *VolumeGroupSnapSuite) GetName() string {
	return "VolumeGroupSnapSuite"
}

// Parameters returns formatted string of parameters
func (vgs *VolumeGroupSnapSuite) Parameters() string {
	return fmt.Sprintf("{volumes: %d, volumeSize: %s}", vgs.VolumeNumber, vgs.VolumeSize)
}

// end of VGS

// SnapSuite is used to manage snap test suite
type SnapSuite struct {
	SnapAmount         int
	SnapClass          string
	VolumeSize         string
	Description        string
	CustomSnapName     string
	AccessModeOriginal string
	AccessModeRestored string
}

// Run executes snap test suite
func (ss *SnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	if ss.SnapAmount <= 0 {
		log.Info("Using default number of snapshots")
		ss.SnapAmount = 3
	}
	if ss.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		ss.VolumeSize = "3Gi"
	}

	result := validateCustomSnapName(ss.CustomSnapName, ss.SnapAmount)
	if result {
		log.Infof("using custom snap-name:%s", ss.CustomSnapName)
	} else {
		ss.CustomSnapName = ""
	}

	pvcClient := clients.PVCClient
	podClient := clients.PodClient

	snapvolname := ""
	snappodname := ""

	result1 := validateCustomSnapName(ss.CustomSnapName, ss.SnapAmount)
	if result1 {
		snapvolname = ss.CustomSnapName + "-pvc"
		snappodname = ss.CustomSnapName + "-pod"
		log.Infof("using custom snap-name:%s"+" and PVC name:%s"+" and POD name:%s", ss.CustomSnapName, snapvolname, snappodname)
	} else {
		snapvolname = ""
		snappodname = ""
	}

	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return delFunc, err
	}
	log.Info("Creating Snapshot pod")
	// Create first PVC
	var pvcNameList []string
	vcconf := testcore.VolumeCreationConfig(storageClass, ss.VolumeSize, snapvolname, ss.AccessModeOriginal)
	volTmpl := pvcClient.MakePVC(vcconf)
	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}
	pvcNameList = append(pvcNameList, pvc.Object.Name)
	if !firstConsumer {
		err := pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return delFunc, err
		}
	}

	// Create Pod, and attach PVC
	podconf := testcore.IoWritePodConfig(pvcNameList, snappodname)
	podTmpl := podClient.MakePod(podconf)

	file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
	sum := fmt.Sprintf("%s0/writer-%d.sha512", podconf.MountPath, 0)
	writerPod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Write random blob to pvc
	ddRes := bytes.NewBufferString("")
	if err := podClient.Exec(ctx, writerPod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
		return delFunc, err
	}
	log.Info("Writer pod: ", writerPod.Object.GetName())
	log.Debug(ddRes.String())

	// Write hash sum of blob
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
		return delFunc, err
	}
	podClient.Delete(ctx, writerPod.Object).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Get PVC for creating snapshot
	gotPvc, err := pvcClient.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
	if err != nil {
		return delFunc, err
	}

	snapPrefix := DefaultSnapPrefix

	if ss.CustomSnapName != "" {
		snapPrefix = ss.CustomSnapName
	}

	var snaps []volumesnapshot.Interface
	log.Infof("Creating %d snapshots", ss.SnapAmount)
	for i := 0; i < ss.SnapAmount; i++ {
		var createSnap volumesnapshot.Interface
		// Create Interface from PVC using gotPvc name
		if clients.SnapClientGA != nil {
			name := snapPrefix + strconv.Itoa(i)
			if i == 0 {
				name = snapPrefix
			}
			createSnap = clients.SnapClientGA.Create(ctx,
				&snapv1.VolumeSnapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:         name,
						GenerateName: "",
						Namespace:    gotPvc.Namespace,
					},
					Spec: snapv1.VolumeSnapshotSpec{
						Source: snapv1.VolumeSnapshotSource{
							PersistentVolumeClaimName: &gotPvc.Name,
						},
						VolumeSnapshotClassName: &ss.SnapClass,
					},
				})
			if createSnap.HasError() {
				return delFunc, createSnap.GetError()
			}

			// Wait for snapshot to be created
			err := createSnap.WaitForRunning(ctx)
			if err != nil {
				return delFunc, err
			}
		} else if clients.SnapClientBeta != nil {
			name := snapPrefix + strconv.Itoa(i)
			if i == 0 {
				name = snapPrefix
			}
			createSnap = clients.SnapClientBeta.Create(ctx,
				&snapbeta.VolumeSnapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:         name,
						GenerateName: "",
						Namespace:    gotPvc.Namespace,
					},
					Spec: snapbeta.VolumeSnapshotSpec{
						Source: snapbeta.VolumeSnapshotSource{
							PersistentVolumeClaimName: &gotPvc.Name,
						},
						VolumeSnapshotClassName: &ss.SnapClass,
					},
				})
			if createSnap.HasError() {
				return delFunc, createSnap.GetError()
			}

			// Wait for snapshot to be created
			err := createSnap.WaitForRunning(ctx)
			if err != nil {
				return delFunc, err
			}
		} else {
			return delFunc, fmt.Errorf("can't get alpha or beta snapshot client")
		}
		snaps = append(snaps, createSnap)
	}
	// Create second PVC from snapshot
	var pvcFromSnapNameList []string
	rand.Seed(time.Now().Unix())
	n := rand.Int() % len(snaps) // #nosec
	vcconf.SnapName = snaps[n].Name()
	log.Infof("Restoring from %s", vcconf.SnapName)
	vcconf.Name = vcconf.SnapName + "-restore"
	accessModeRestoredVolume := testcore.GetAccessMode(ss.AccessModeRestored)
	vcconf.AccessModes = accessModeRestoredVolume
	log.Infof("Creating pvc %s", vcconf.Name)
	volRestored := pvcClient.MakePVC(vcconf)
	pvcRestored := pvcClient.Create(ctx, volRestored)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}
	pvcFromSnapNameList = append(pvcFromSnapNameList, pvcRestored.Object.Name)
	if !firstConsumer {
		err = pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return delFunc, err
		}
	}

	// Create Pod, and attach PVC from snapshot

	podRestored := testcore.IoWritePodConfig(pvcFromSnapNameList, pvcRestored.Object.Name+"-pod")
	podTmplRestored := podClient.MakePod(podRestored)

	writerPod = podClient.Create(ctx, podTmplRestored).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Check if hash sum is correct
	sum = fmt.Sprintf("%s0/writer-%d.sha512", podRestored.MountPath, 0)
	writer := bytes.NewBufferString("")
	log.Info("Checker pod: ", writerPod.Object.GetName())
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
		return delFunc, err
	}

	if strings.Contains(writer.String(), "OK") {
		log.Info("Hashes match")
	} else {
		return delFunc, fmt.Errorf("hashes don't match")
	}
	return delFunc, nil
}

func validateCustomSnapName(name string, snapshotAmount int) bool {
	// If no. of snapshots is only 1 then we will take custom name else no.
	if snapshotAmount == 1 && len(name) != 0 {
		return true
	}
	// we will use custom name only if number of snapshots is 1 else we will discard custom name
	return false
}

// GetObservers returns all observers
func (*SnapSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics, snapsnot clients
func (ss *SnapSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.SnapshotClassExists(ss.SnapClass); !ok {
		return nil, fmt.Errorf("snapshotclass class doesn't exist; error = %v", err)
	}

	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	snapGA, snapBeta, snErr := GetSnapshotClient(namespace, client)
	if snErr != nil {
		return nil, snErr
	}
	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      snapGA,
		SnapClientBeta:    snapBeta,
	}, nil
}

// GetNamespace returns snap suite namespaces
func (*SnapSuite) GetNamespace() string {
	return "snap-test"
}

// GetName returns snap suite name
func (ss *SnapSuite) GetName() string {
	if ss.Description != "" {
		return ss.Description
	}
	return "SnapSuite"
}

// Parameters returns formatted string of paramters
func (ss *SnapSuite) Parameters() string {
	return fmt.Sprintf("{snapshots: %d, volumeSize; %s}", ss.SnapAmount, ss.VolumeSize)
}

func getAllObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PvcObserver{},
			&observer.VaObserver{},
			&observer.PodObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	} else if obsType == observer.LIST {
		return []observer.Interface{
			&observer.PvcListObserver{},
			&observer.VaListObserver{},
			&observer.PodListObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

// ReplicationSuite is used to manage replication test suite
type ReplicationSuite struct {
	VolumeNumber int
	VolumeSize   string
	PodNumber    int
	SnapClass    string
}

// Run executes replication test suite
func (rs *ReplicationSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	rgClient := clients.RgClient

	if rs.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		rs.VolumeNumber = 5
	}
	if rs.PodNumber <= 0 {
		log.Info("Using default number of pods")
		rs.PodNumber = 1
	}
	if rs.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		rs.VolumeSize = "3Gi"
	}

	log.Infof("Creating %s pods, each with %s volumes", color.YellowString(strconv.Itoa(rs.PodNumber)),
		color.YellowString(strconv.Itoa(rs.VolumeNumber)))
	var allPvcNames []string
	for i := 0; i < rs.PodNumber; i++ {
		pvcNameList := make([]string, rs.VolumeNumber)
		for j := 0; j < rs.VolumeNumber; j++ {
			// Create PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, rs.VolumeSize, "", "")
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return delFunc, pvc.GetError()
			}

			pvcNameList[j] = pvc.Object.Name
			allPvcNames = append(allPvcNames, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "")
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}
	log.Info("Creating a snapshot on each of the volumes")
	var snapNameList []string
	if clients.SnapClientGA != nil {
		lenPvcList := len(allPvcNames)
		iters := lenPvcList / 10
		if lenPvcList%10 != 0 {
			iters++
		}
		for iter := 0; iter < iters; iter++ {
			initial := iter * 10
			final := initial + 10
			if final > lenPvcList {
				final = lenPvcList
			}
			for _, pvc := range allPvcNames[initial:final] {
				gotPvc, err := pvcClient.Interface.Get(ctx, pvc, metav1.GetOptions{})
				if err != nil {
					return delFunc, err
				}
				snapName := fmt.Sprintf("snap-%s", gotPvc.Name)
				snapNameList = append(snapNameList, snapName)
				createSnap := clients.SnapClientGA.Create(ctx,
					&snapv1.VolumeSnapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:         snapName,
							GenerateName: "",
							Namespace:    gotPvc.Namespace,
						},
						Spec: snapv1.VolumeSnapshotSpec{
							Source: snapv1.VolumeSnapshotSource{
								PersistentVolumeClaimName: &gotPvc.Name,
							},
							VolumeSnapshotClassName: &rs.SnapClass,
						},
					})
				if createSnap.HasError() {
					return delFunc, createSnap.GetError()
				}
			}
			snapReadyError := clients.SnapClientGA.WaitForAllToBeReady(ctx)
			if snapReadyError != nil {
				return delFunc, snapReadyError
			}
		}
	} else if clients.SnapClientBeta != nil {
		for _, pvc := range allPvcNames {
			gotPvc, err := pvcClient.Interface.Get(ctx, pvc, metav1.GetOptions{})
			if err != nil {
				return delFunc, err
			}
			snapName := fmt.Sprintf("snap-%s", gotPvc.Name)
			snapNameList = append(snapNameList, snapName)
			createSnap := clients.SnapClientBeta.Create(ctx,
				&snapbeta.VolumeSnapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:         snapName,
						GenerateName: "",
						Namespace:    gotPvc.Namespace,
					},
					Spec: snapbeta.VolumeSnapshotSpec{
						Source: snapbeta.VolumeSnapshotSource{
							PersistentVolumeClaimName: &gotPvc.Name,
						},
						VolumeSnapshotClassName: &rs.SnapClass,
					},
				})
			if createSnap.HasError() {
				return delFunc, createSnap.GetError()
			}
		}
		// Wait for snapshot to be created
		snapReadyError := clients.SnapClientBeta.WaitForAllToBeReady(ctx)
		if snapReadyError != nil {
			return delFunc, snapReadyError
		}
	} else {
		return delFunc, fmt.Errorf("can't get alpha or beta snapshot client")
	}

	log.Info("Creating new pods with replicated volumes mounted on them")
	for i := 0; i < rs.PodNumber; i++ {
		pvcNameList := make([]string, rs.VolumeNumber)
		for j := 0; j < rs.VolumeNumber; j++ {
			// Restore PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, rs.VolumeSize, "", "")
			vcconf.SnapName = snapNameList[j+(i*rs.VolumeNumber)]
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return delFunc, pvc.GetError()
			}
			pvcNameList[j] = pvc.Object.Name
		}
		// Create Pod, and attach restored PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "")
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
	}
	readyErr = podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	// TODO: List PVCs once again, since here we can be sure that all annotations will be correctly set
	pvcList, err := pvcClient.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return delFunc, err
	}

	rgName := pvcList.Items[0].Annotations[commonparams.ReplicationGroupName]
	log.Infof("The replication group name from pvc is %s ", rgName)

	// Add remote RG deletion step to deletion callback function
	delFunc = func(f func() error) func() error {
		return func() error {
			log.Info("Deleting local RG")
			rgObject := rgClient.Get(context.Background(), rgName)
			deletedLocalRG := rgClient.Delete(context.Background(), rgObject.Object)
			if deletedLocalRG.HasError() {
				log.Warnf("error when deleting local RG: %s", deletedLocalRG.GetError().Error())
			}

			log.Info("Sleeping for 1 minute...")
			time.Sleep(1 * time.Minute)

			return nil
		}
	}(nil)

	return delFunc, nil
}

// GetObservers returns all observers
func (rs *ReplicationSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics, snapshot clients
func (rs *ReplicationSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.SnapshotClassExists(rs.SnapClass); !ok {
		return nil, fmt.Errorf("snasphot class doesn't exist; error = %v", err)
	}

	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	snapGA, snapBeta, snErr := GetSnapshotClient(namespace, client)
	if snErr != nil {
		return nil, snErr
	}

	rgClient, rgErr := client.CreateRGClient()
	if rgErr != nil {
		return nil, rgErr
	}

	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      snapGA,
		SnapClientBeta:    snapBeta,
		RgClient:          rgClient,
	}, nil
}

// GetNamespace returns replication suite namespace
func (*ReplicationSuite) GetNamespace() string {
	return "replication-suite"
}

// GetName returns replication suite name
func (*ReplicationSuite) GetName() string {
	return "ReplicationSuite"
}

// Parameters returns formatted string of parameters
func (rs *ReplicationSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, volumeSize: %s}", rs.PodNumber, rs.VolumeNumber, rs.VolumeSize)
}

// VolumeExpansionSuite is used to manage volume expansion test suite
type VolumeExpansionSuite struct {
	VolumeNumber int
	PodNumber    int
	IsBlock      bool
	InitialSize  string
	ExpandedSize string
	Description  string
	AccessMode   string
}

// Run executes volume expansion test suite
func (ves *VolumeExpansionSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	if ves.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		ves.VolumeNumber = 5
	}
	if ves.PodNumber <= 0 {
		log.Info("Using default number of volumes")
		ves.PodNumber = 1
	}

	log.Infof("Creating %s pods, each with %s volumes of size (%s)", color.YellowString(strconv.Itoa(ves.PodNumber)),
		color.YellowString(strconv.Itoa(ves.VolumeNumber)), ves.InitialSize)

	var podObjectList []*v1.Pod
	for i := 0; i < ves.PodNumber; i++ {
		var pvcNameList []string
		for j := 0; j < ves.VolumeNumber; j++ {
			// Create PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, ves.InitialSize, "", ves.AccessMode)
			if ves.IsBlock {
				var mode v1.PersistentVolumeMode = "Block"
				vcconf.VolumeMode = &mode
			}
			volTmpl := pvcClient.MakePVC(vcconf)
			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return delFunc, pvc.GetError()
			}

			pvcNameList = append(pvcNameList, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "")
		if ves.IsBlock {
			podconf.VolumeMode = pod.Block
		}
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
		podObjectList = append(podObjectList, pod.Object)
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	// Check the sizes of created volumes on the nodes
	// We need to do that because sometimes drivers could create volumes of different size than requested
	deltas := make(map[string]int)
	for _, p := range podObjectList {
		if ves.IsBlock {
			// We can't check FS info for RawBlock so just break
			break
		}
		for _, v := range p.Spec.Containers[0].VolumeMounts {
			if !strings.Contains(v.Name, "vol") { // we can get token volume
				continue
			}

			wantSize, err := convertSpecSize(ves.InitialSize)
			if err != nil {
				return delFunc, nil
			}

			gotSize, err := checkSize(ctx, podClient, p, v, true)
			if err != nil {
				return delFunc, err
			}

			delta := gotSize - wantSize
			deltas[p.Name+v.MountPath] = delta

			log.Infof("Requested %s KiB, got %s KiB -- delta is %s KiB",
				color.YellowString(strconv.Itoa(wantSize)), color.YellowString(strconv.Itoa(gotSize)),
				color.YellowString(strconv.Itoa(delta)))
		}
	}

	log.Infof("Expanding the size of volumes from %s to %s",
		color.YellowString(ves.InitialSize), color.YellowString(ves.ExpandedSize))

	pvcList, err := pvcClient.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return delFunc, err
	}

	for i := range pvcList.Items {
		pvcList.Items[i].Spec.Resources.Requests = v1.ResourceList{
			v1.ResourceStorage: resource.MustParse(ves.ExpandedSize),
		}
		updatedPVC := pvcClient.Update(ctx, &pvcList.Items[i])
		if updatedPVC.HasError() {
			return delFunc, updatedPVC.GetError()
		}
	}
	if ves.IsBlock {
		// We can't compare deltas for RawBlock so just end it here
		return delFunc, nil
	}

	// Give some time for driver to expand
	time.Sleep(10 * time.Second)

	for _, p := range podObjectList {
		for _, v := range p.Spec.Containers[0].VolumeMounts {
			if !strings.Contains(v.Name, "vol") { // we can get token volume
				continue
			}

			log.Infof("Checking if %s was properly resized", p.Name+v.MountPath)

			wantSize, err := convertSpecSize(ves.ExpandedSize)
			if err != nil {
				return delFunc, nil
			}
			oldDelta := deltas[p.Name+v.MountPath]

			pollErr := wait.PollImmediate(10*time.Second, time.Duration(pvcClient.Timeout)*time.Second,
				func() (bool, error) {
					select {
					case <-ctx.Done():
						log.Infof("Stopping pod wait polling")
						return true, fmt.Errorf("stopped waiting to be ready")
					default:
						break
					}

					gotSize, err := checkSize(ctx, podClient, p, v, true)
					if err != nil {
						return false, err
					}

					newDelta := gotSize - wantSize
					log.Debugf("old delta: %d; new delta: %d", oldDelta, newDelta)
					if newDelta == oldDelta {
						log.Infof("%s was successfully expanded", p.Name+v.MountPath)
						return true, nil
					} else if math.Abs(float64(newDelta))/float64(wantSize)*100 < 5 {
						// If new delta is in 5% of requested size we count it as successfully expanded
						log.Warnf("%s expanded within ±5%c of request size", p.Name+v.MountPath, '%')
						return true, nil
					}

					return false, nil
				})

			if pollErr != nil {
				return delFunc, fmt.Errorf("sizes don't match: %s", pollErr.Error())
			}
		}
	}

	return delFunc, nil
}

func checkSize(ctx context.Context, podClient *pod.Client, p *v1.Pod, v v1.VolumeMount, quiet bool) (int, error) {
	log := utils.GetLoggerFromContext(ctx)
	res := bytes.NewBufferString("")
	if err := podClient.Exec(ctx,
		p, []string{"/bin/bash", "-c", "df"},
		res, os.Stderr, quiet); err != nil {
		log.Error(res.String())
		return 0, err
	}

	size := ""
	// pipes won't work for some reason, try to find size here
	sliced := strings.FieldsFunc(res.String(), func(r rune) bool { return strings.ContainsRune(" \n", r) })
	// Check `mounted on` column
	// i := 12 needed to skip header
	for i := 12; i < len(sliced); i += 6 {
		fs := sliced[i]
		if strings.Contains(fs, v.MountPath) {
			size = sliced[i-4]
		}
	}

	log.Debug(res.String())
	if size == "" {
		return 0, nil
	}
	gotSize, err := strconv.Atoi(size)
	if err != nil {
		return 0, fmt.Errorf("can't convert df output to KiB: %s", err.Error())
	}

	return gotSize, nil
}

func convertSpecSize(specSize string) (int, error) {
	quantity := resource.MustParse(specSize)
	initSize, ok := quantity.AsInt64()
	if !ok {
		return 0, fmt.Errorf("can't convert initial size quantity %v", quantity)
	}

	wantSize := int(initSize / 1024) // convert to KiB
	return wantSize, nil
}

// GetObservers returns all observers
func (*VolumeExpansionSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics clients
func (*VolumeExpansionSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
	}, nil
}

// GetNamespace returns volume expansion suite namespace
func (*VolumeExpansionSuite) GetNamespace() string {
	return "volume-expansion-suite"
}

// GetName returns volume expansion suite name
func (ves *VolumeExpansionSuite) GetName() string {
	if ves.Description != "" {
		return ves.Description
	}
	return "VolumeExpansionSuite"
}

// Parameters returns formatted string of parameters
func (ves *VolumeExpansionSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, size: %s, expSize: %s, block: %s}", ves.PodNumber,
		ves.VolumeNumber, ves.InitialSize, ves.ExpandedSize, strconv.FormatBool(ves.IsBlock))
}

// VolumeHealthMetricsSuite is used to manage volume health metrics test suite
type VolumeHealthMetricsSuite struct {
	VolumeNumber int
	PodNumber    int
	VolumeSize   string
	Description  string
	AccessMode   string
	Namespace    string
}

// FindDriverLogs executes command and returns the output
func FindDriverLogs(command []string) (string, error) {

	cmd := exec.Command(command[0], command[1:]...) // #nosec G204
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	str := string(output)
	return str, nil
}

// Run executes volume health metrics test suite
func (vh *VolumeHealthMetricsSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	pvClient := clients.PersistentVolumeClient

	if vh.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		vh.VolumeNumber = 1
	}
	if vh.PodNumber <= 0 {
		log.Info("Using default number of pods")
		vh.PodNumber = 1
	}

	//Create a PVC
	vcconf := testcore.VolumeCreationConfig(storageClass, vh.VolumeSize, "", vh.AccessMode)
	volTmpl := pvcClient.MakePVC(vcconf)
	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}
	PVCNamespace := pvcClient.Namespace

	// Create Pod, and attach PVC
	podconf := testcore.VolumeHealthPodConfig([]string{pvc.Object.Name}, "")
	podTmpl := podClient.MakePod(podconf)

	pod := podClient.Create(ctx, podTmpl)
	if pod.HasError() {
		return delFunc, pod.GetError()
	}

	// Waiting for the Pod to be ready
	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	// Finding the node on which the pod is scheduled
	pods, podErr := podClient.Interface.List(ctx, metav1.ListOptions{})
	if podErr != nil {
		return delFunc, podErr
	}
	podScheduledNode := pods.Items[0].Spec.NodeName

	// Finding the PV created
	persistentVolumes, pvErr := pvClient.Interface.List(ctx, metav1.ListOptions{})
	if pvErr != nil {
		return delFunc, pvErr
	}

	// Finding the Volume Handle of the PV created
	var PersistentVolumeID string
	for _, p := range persistentVolumes.Items {
		if p.Spec.ClaimRef.Namespace == PVCNamespace {

			PersistentVolumeID = p.Spec.CSI.VolumeHandle
		}
	}

	// Finding the node pods and controller pods in the driver namespace
	driverpodClient, driverpoderr := clients.KubeClient.CreatePodClient(vh.Namespace)
	if driverpoderr != nil {
		return delFunc, driverpoderr
	}

	driverPods, podErr := driverpodClient.Interface.List(ctx, metav1.ListOptions{})
	if podErr != nil {
		return delFunc, podErr
	}

	// Finding the driver node pod scheduled on the node, on which the above volume and pod are scheduled
	var driverNodePod string
	var driverControllerPod string
	for _, p := range driverPods.Items {
		if strings.Contains(p.Name, "node") && p.Spec.NodeName == podScheduledNode {
			driverNodePod = p.Name
		}
	}

	// Finding the driver controller pod
	exe := []string{"bash", "-c", fmt.Sprintf("kubectl get leases -n %s ", vh.Namespace)}
	str, err := FindDriverLogs(exe)
	if err != nil {
		return delFunc, err
	}
	driverLeases := strings.Split(str, "\n")
	for _, leases := range driverLeases {
		temp := strings.Fields(leases)
		if (len(temp) > 0) && (strings.Contains(temp[0], "health")) {
			driverControllerPod = temp[1]
		}
	}

	log.Info("Driver Node Pod ", driverNodePod)
	log.Info("Driver Controller Pod ", driverControllerPod)

	var NodeLogs = false
	var ControllerLogs = false

	var retryCount = 1
	for i := 0; i < maxRetryCount; i++ {

		if !ControllerLogs {
			exe = []string{"bash", "-c", fmt.Sprintf("kubectl logs %s -n %s -c driver | grep ControllerGetVolume ", driverControllerPod, vh.Namespace)}
			str, err = FindDriverLogs(exe)
			if err == nil && strings.Contains(str, PersistentVolumeID) {
				ControllerLogs = true
			}
		}

		if !NodeLogs {
			exe = []string{"bash", "-c", fmt.Sprintf("kubectl logs %s -n %s -c driver | grep NodeGetVolumeStats ", driverNodePod, vh.Namespace)}
			str, err = FindDriverLogs(exe)
			if err == nil && strings.Contains(str, PersistentVolumeID) {
				NodeLogs = true
			}
		}

		if ControllerLogs && NodeLogs {
			return delFunc, nil
		}

		time.Sleep(ControllerLogsSleepTime * time.Second)
		retryCount = retryCount + 1
	}

	return delFunc, errors.New("volume health metrics error")
}

// GetObservers returns all observers
func (*VolumeHealthMetricsSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, pv, va, metrics clients
func (*VolumeHealthMetricsSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	pvClient, pvErr := client.CreatePVClient()
	if pvErr != nil {
		return nil, pvErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	return &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		VaClient:               vaClient,
		StatefulSetClient:      nil,
		MetricsClient:          metricsClient,
		KubeClient:             client,
		PersistentVolumeClient: pvClient,
	}, nil
}

// GetNamespace returns volume health metrics test suite namespace
func (*VolumeHealthMetricsSuite) GetNamespace() string {
	return "volume-health-metrics"
}

// GetName returns volume health metrics suite name
func (vh *VolumeHealthMetricsSuite) GetName() string {
	if vh.Description != "" {
		return vh.Description
	}
	return "VolumeHealthMetricSuite"
}

// Parameters returns formatted string of parameters
func (vh *VolumeHealthMetricsSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, size: %s}", vh.PodNumber, vh.VolumeNumber, vh.VolumeSize)
}

// CloneVolumeSuite is used to manage clone volume suite test suite
type CloneVolumeSuite struct {
	VolumeNumber  int
	VolumeSize    string
	PodNumber     int
	CustomPvcName string
	CustomPodName string
	Description   string
	AccessMode    string
}

// Run executes clone volume test suite
func (cs *CloneVolumeSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	if cs.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		cs.VolumeNumber = 5
	}
	if cs.PodNumber <= 0 {
		log.Info("Using default number of pods")
		cs.PodNumber = 1
	}
	if cs.VolumeSize == "" {
		log.Info("Using default volume size:3Gi")
		cs.VolumeSize = "3Gi"
	}
	clonedVolName := ""
	result := validateCustomName(cs.CustomPvcName, cs.VolumeNumber)
	if result {
		clonedVolName = cs.CustomPvcName + "-cloned"
		log.Infof("using custom pvc-name:%s"+" and cloned PVC name:%s", cs.CustomPvcName, clonedVolName)
	} else {
		cs.CustomPvcName = ""
	}

	clonedPodName := ""
	result = validateCustomName(cs.CustomPodName, cs.VolumeNumber)
	if result {
		clonedPodName = cs.CustomPodName + "-cloned"
		log.Infof("using custom pod-name:%s"+" and cloned PVC pod-name:%s", cs.CustomPodName, clonedPodName)
	} else {
		cs.CustomPodName = ""
	}

	log.Infof("Creating %s pods, each with %s volumes", color.YellowString(strconv.Itoa(cs.PodNumber)),
		color.YellowString(strconv.Itoa(cs.VolumeNumber)))
	var allPvcNames []string
	for i := 0; i < cs.PodNumber; i++ {
		pvcNameList := make([]string, cs.VolumeNumber)
		for j := 0; j < cs.VolumeNumber; j++ {
			// Create PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, cs.VolumeSize, cs.CustomPvcName, cs.AccessMode)
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return delFunc, pvc.GetError()
			}

			pvcNameList[j] = pvc.Object.Name
			allPvcNames = append(allPvcNames, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, cs.CustomPodName)
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	log.Info("Creating new pods with cloned volumes mounted on them")
	for i := 0; i < cs.PodNumber; i++ {
		pvcNameList := make([]string, cs.VolumeNumber)
		for j := 0; j < cs.VolumeNumber; j++ {
			// Restore PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, cs.VolumeSize, clonedVolName, cs.AccessMode)
			vcconf.SourceVolumeName = allPvcNames[j+(i*cs.VolumeNumber)]
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return delFunc, pvc.GetError()
			}
			pvcNameList[j] = pvc.Object.Name
		}
		// Create Pod, and attach restored PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, clonedPodName)
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
	}
	readyErr = podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}
	return delFunc, nil
}

// GetObservers returns all observers
func (cs *CloneVolumeSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics clients
func (cs *CloneVolumeSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      nil,
		SnapClientBeta:    nil,
	}, nil
}

// GetNamespace returns clone volume suite test namespace
func (*CloneVolumeSuite) GetNamespace() string {
	return "clonevolume-suite"
}

// GetName returns clone volume suite test name
func (cs *CloneVolumeSuite) GetName() string {
	if cs.Description != "" {
		return cs.Description
	}
	return "CloneVolumeSuite"
}

// Parameters returns formatted string of parameters
func (cs *CloneVolumeSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, volumeSize: %s}", cs.PodNumber, cs.VolumeNumber, cs.VolumeSize)
}

// MultiAttachSuite is used to manage multi attach test suite
type MultiAttachSuite struct {
	PodNumber   int
	RawBlock    bool
	Description string
	AccessMode  string
	VolumeSize  string
}

// Run executes multi attach test suite
func (mas *MultiAttachSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	nodeClient := clients.NodeClient

	if mas.PodNumber <= 0 {
		log.Info("Using default number of pods")
		mas.PodNumber = 2
	}

	log.Info("Creating Volume")
	// Create PVCs
	vcconf := testcore.MultiAttachVolumeConfig(storageClass, mas.VolumeSize, mas.AccessMode)
	if mas.RawBlock {
		var mode v1.PersistentVolumeMode = "Block"
		vcconf.VolumeMode = &mode
	}

	volTmpl := pvcClient.MakePVC(vcconf)

	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}

	log.Info("Attaching Volume to original pod")
	// Create Pod, and attach PVC
	podconf := testcore.MultiAttachPodConfig([]string{pvc.Object.Name})

	if mas.RawBlock {
		podconf.VolumeMode = pod.Block
	}
	podTmpl := podClient.MakePod(podconf)

	originalPod := podClient.Create(ctx, podTmpl)
	if originalPod.HasError() {
		return delFunc, originalPod.GetError()
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	var spreadConstraint []v1.TopologySpreadConstraint
	var labels map[string]string
	if nodeClient != nil {
		nodeList, ncErr := nodeClient.Interface.List(ctx, metav1.ListOptions{
			LabelSelector: "!node-role.kubernetes.io/master",
		})
		if ncErr != nil {
			return delFunc, ncErr
		}
		labels = map[string]string{"ts": "mas"}
		spreadConstraint = mas.GenerateTopologySpreadConstraints(len(nodeList.Items), labels)
	}

	log.Info("Creating new pods with original Volume attached to them")
	var newPods []*pod.Pod
	for i := 0; i < mas.PodNumber; i++ {
		podTmpl := podClient.MakePod(podconf)
		podTmpl.Spec.TopologySpreadConstraints = spreadConstraint
		podTmpl.Labels = labels

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
		newPods = append(newPods, pod)
	}

	if mas.AccessMode == "ReadWriteOncePod" {
		time.Sleep(1 * time.Minute)
		readyPodCount, err := podClient.ReadyPodsCount(ctx)

		if err != nil {
			return delFunc, err
		}
		if readyPodCount == 1 {
			return delFunc, nil
		}
	}

	readyErr = podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	if !mas.RawBlock {

		log.Info("Writing to Volume on 1st originalPod")
		file := fmt.Sprintf("%s0/blob.data", podconf.MountPath)
		sum := fmt.Sprintf("%s0/blob.sha512", podconf.MountPath)
		// Write random blob to pvc
		ddRes := bytes.NewBufferString("")
		if err := podClient.Exec(ctx, originalPod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
			return delFunc, err
		}
		log.Info("Writer originalPod: ", originalPod.Object.GetName())
		log.Debug(ddRes.String())

		// Write hash sum of blob
		if err := podClient.Exec(ctx, originalPod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return delFunc, err
		}

		log.Info("Checking hash sum on all of the other pods")
		for _, p := range newPods {
			writer := bytes.NewBufferString("")
			if err := podClient.Exec(ctx, p.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
				return delFunc, err
			}
			if strings.Contains(writer.String(), "OK") {
				log.Info("Hashes match")
			} else {
				return delFunc, fmt.Errorf("hashes don't match")
			}
		}
	} else {
		device := fmt.Sprintf("/dev%s0", podconf.MountPath)
		file := fmt.Sprintf("/tmp/blob.data")

		hash := bytes.NewBufferString("")

		// Write random data to block device
		if err := podClient.Exec(ctx, originalPod.Object, []string{"dd", "if=/dev/urandom", "of=" + device, "bs=1M", "count=128", "oflag=sync"}, os.Stdout, os.Stderr, false); err != nil {
			return delFunc, err
		}
		// Calculate hashsum of first 128MB
		// We can't pipe anything here so just write to file
		if err := podClient.Exec(ctx, originalPod.Object, []string{"dd", "if=" + device, "of=" + file, "bs=1M", "count=128"}, os.Stdout, os.Stderr, false); err != nil {
			return delFunc, err
		}
		// Calculate hash sum of that file
		if err := podClient.Exec(ctx, originalPod.Object, []string{"sha512sum", file}, hash, os.Stderr, false); err != nil {
			return delFunc, err
		}
		log.Info("OriginalPod: ", originalPod.Object.GetName())
		log.Info("hash sum is:", hash.String())

		log.Info("Checking hash sum on all of the other pods")
		for _, p := range newPods {
			newHash := bytes.NewBufferString("")
			if err := podClient.Exec(ctx, p.Object, []string{"blockdev", "--flushbufs", device}, os.Stdout, os.Stderr, false); err != nil {
				return delFunc, err
			}
			if err := podClient.Exec(ctx, p.Object, []string{"dd", "if=" + device, "of=" + file, "bs=1M", "count=128"}, os.Stdout, os.Stderr, false); err != nil {
				return delFunc, err
			}
			if err := podClient.Exec(ctx, p.Object, []string{"sha512sum", file}, newHash, os.Stderr, false); err != nil {
				return delFunc, err
			}

			log.Info("Pod: ", p.Object.GetName())
			log.Info("hash sum is:", newHash.String())

			if newHash.String() == hash.String() {
				log.Info("Hashes match")
			} else {
				return delFunc, fmt.Errorf("hashes don't match")
			}
		}
	}

	return delFunc, nil
}

// GenerateTopologySpreadConstraints creates and returns topology spread constraints
func (mas *MultiAttachSuite) GenerateTopologySpreadConstraints(nodeCount int, labels map[string]string) []v1.TopologySpreadConstraint {
	// Calculate MaxSkew parameter
	maxSkew, remainder := mas.PodNumber/nodeCount, mas.PodNumber%nodeCount
	// in case of odd pod/node count - increase the maxSkew
	if remainder != 0 {
		maxSkew++
	}
	spreadConstraints := []v1.TopologySpreadConstraint{{
		MaxSkew:           int32(maxSkew),
		TopologyKey:       "kubernetes.io/hostname",
		WhenUnsatisfiable: v1.ScheduleAnyway,
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
	},
	}
	return spreadConstraints
}

// GetObservers returns all observers
func (mas *MultiAttachSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics (and node) clients
func (mas *MultiAttachSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	var nodeClient *node.Client
	var ncErr error
	if client.Minor >= 19 {
		// TopologySpreadConstraints supported from k8s version 1.19
		nodeClient, ncErr = client.CreateNodeClient()
		if ncErr != nil {
			return nil, ncErr
		}
	}
	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      nil,
		SnapClientBeta:    nil,
		NodeClient:        nodeClient,
	}, nil
}

// GetNamespace returns multi attach suite namespace
func (*MultiAttachSuite) GetNamespace() string {
	return "mas-test"
}

// GetName returns multi attach suite name
func (mas *MultiAttachSuite) GetName() string {
	if mas.Description != "" {
		return mas.Description
	}
	return "MultiAttachSuite"
}

// Parameters returns formatted string of parameters
func (mas *MultiAttachSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, rawBlock: %s, size: %s, accMode: %s}",
		mas.PodNumber, strconv.FormatBool(mas.RawBlock), mas.VolumeSize, mas.AccessMode)
}

// BlockSnapSuite is used to manage block snapshot test suite
type BlockSnapSuite struct {
	SnapClass   string
	VolumeSize  string
	Description string
	AccessMode  string
}

// Run executes block snapshot test suite
func (bss *BlockSnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	if bss.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		bss.VolumeSize = "3Gi"
	}
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	log.Info("Creating BlockSnap pod")
	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return delFunc, err
	}
	// Create first PVC
	var pvcNameList []string
	vcconf := testcore.VolumeCreationConfig(storageClass, bss.VolumeSize, "", "")
	volTmpl := pvcClient.MakePVC(vcconf)
	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}
	pvcNameList = append(pvcNameList, pvc.Object.Name)
	if !firstConsumer {
		err := pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return delFunc, err
		}
	}

	// Create Pod, and attach PVC
	podconf := testcore.BlockSnapPodConfig(pvcNameList)
	podTmpl := podClient.MakePod(podconf)

	file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
	writerPod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Write random blob to pvc
	ddRes := bytes.NewBufferString("")
	if err := podClient.Exec(ctx, writerPod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
		return delFunc, err
	}
	log.Info("Writer pod: ", writerPod.Object.GetName())
	log.Debug(ddRes.String())

	originalHash := bytes.NewBufferString("")

	// Calculate hash sum of that file
	if err := podClient.Exec(ctx, writerPod.Object, []string{"sha512sum", file}, originalHash, os.Stderr, false); err != nil {
		return delFunc, err
	}
	log.Info("hash sum is: ", originalHash.String())

	// Get PVC for creating snapshot
	gotPvc, err := pvcClient.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
	if err != nil {
		return delFunc, err
	}

	var createSnap volumesnapshot.Interface
	// Create Interface from PVC using gotPvc name
	if clients.SnapClientGA != nil {
		createSnap = clients.SnapClientGA.Create(ctx,
			&snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "snap0",
					GenerateName: "",
					Namespace:    gotPvc.Namespace,
				},
				Spec: snapv1.VolumeSnapshotSpec{
					Source: snapv1.VolumeSnapshotSource{
						PersistentVolumeClaimName: &gotPvc.Name,
					},
					VolumeSnapshotClassName: &bss.SnapClass,
				},
			})
		if createSnap.HasError() {
			return delFunc, createSnap.GetError()
		}

		// Wait for snapshot to be created
		err := createSnap.WaitForRunning(ctx)
		if err != nil {
			return delFunc, err
		}
	} else if clients.SnapClientBeta != nil {
		createSnap = clients.SnapClientBeta.Create(ctx,
			&snapbeta.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "snap0",
					GenerateName: "",
					Namespace:    gotPvc.Namespace,
				},
				Spec: snapbeta.VolumeSnapshotSpec{
					Source: snapbeta.VolumeSnapshotSource{
						PersistentVolumeClaimName: &gotPvc.Name,
					},
					VolumeSnapshotClassName: &bss.SnapClass,
				},
			})
		if createSnap.HasError() {
			return delFunc, createSnap.GetError()
		}

		// Wait for snapshot to be created
		err := createSnap.WaitForRunning(ctx)
		if err != nil {
			return delFunc, err
		}
	} else {
		return delFunc, fmt.Errorf("can't get alpha or beta snapshot client")
	}

	// Create second PVC from snapshot
	vcconf.SnapName = createSnap.Name()
	var mode v1.PersistentVolumeMode = pod.Block
	vcconf.VolumeMode = &mode

	vcconf.AccessModes = testcore.GetAccessMode(bss.AccessMode)

	log.Infof("Restoring from %s", vcconf.SnapName)
	volRestored := pvcClient.MakePVC(vcconf)
	pvcRestored := pvcClient.Create(ctx, volRestored)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}
	if !firstConsumer {
		err := pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return delFunc, err
		}
	}

	// Create Pod, and attach PVC from snapshot
	podRestored := testcore.BlockSnapPodConfig([]string{pvcRestored.Object.Name})
	podRestored.VolumeMode = pod.Block
	podTmplRestored := podClient.MakePod(podRestored)

	writerPod = podClient.Create(ctx, podTmplRestored).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Check the data persistence
	writer := bytes.NewBufferString("")
	log.Info("Checker pod: ", writerPod.Object.GetName())

	// flush the buffer
	if err := podClient.Exec(ctx, writerPod.Object, []string{"blockdev", "--flushbufs", "/dev/data0"}, os.Stdout, os.Stderr, false); err != nil {
		return delFunc, err
	}
	// create folder
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "mkdir /data0"}, writer, os.Stderr, false); err != nil {
		return delFunc, err
	}
	// mount rawblock volume to created folder
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "mount /dev/data0 /data0"}, writer, os.Stderr, false); err != nil {
		return delFunc, err
	}

	newHash := bytes.NewBufferString("")
	pollErr := wait.PollImmediate(10*time.Second, 5*time.Minute,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pod wait polling")
				return true, fmt.Errorf("stopped waiting to be ready")
			default:
				break
			}

			// Check if file exists
			if err := podClient.Exec(ctx, writerPod.Object, []string{"ls", file}, os.Stdout, os.Stderr, false); err != nil {
				return false, nil
			}

			// Calculate hash sum of that file
			if err := podClient.Exec(ctx, writerPod.Object, []string{"sha512sum", file}, newHash, os.Stderr, false); err != nil {
				return false, err
			}

			return true, nil
		})

	if pollErr != nil {
		return delFunc, pollErr
	}

	log.Info("new hash sum is: ", newHash.String())

	if newHash.String() == originalHash.String() {
		log.Info("Hashes match")
	} else {
		return delFunc, fmt.Errorf("hashes don't match")
	}

	return delFunc, nil
}

// GetObservers returns all observers
func (*BlockSnapSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics, snapshot clients
func (bss *BlockSnapSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.SnapshotClassExists(bss.SnapClass); !ok {
		return nil, fmt.Errorf("snapshotclass class doesn't exist; error = %v", err)
	}

	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	snapGA, snapBeta, snErr := GetSnapshotClient(namespace, client)
	if snErr != nil {
		return nil, snErr
	}
	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      snapGA,
		SnapClientBeta:    snapBeta,
	}, nil
}

// GetNamespace returns block snap test suite namespace
func (*BlockSnapSuite) GetNamespace() string {
	return "block-snap-test"
}

// GetName returns block snap test suite name
func (bss *BlockSnapSuite) GetName() string {
	if bss.Description != "" {
		return bss.Description
	}
	return "BlockSnapSuite"
}

// Parameters returns formatted string of parameters
func (bss *BlockSnapSuite) Parameters() string {
	return fmt.Sprintf("{size: %s, accMode: %s}", bss.VolumeSize, bss.AccessMode)
}

// GetSnapshotClient returns snapshot client
func GetSnapshotClient(namespace string, client *k8sclient.KubeClient) (*snapv1client.SnapshotClient, *snapbetaclient.SnapshotClient, error) {
	gaClient, snErr := client.CreateSnapshotGAClient(namespace)
	_, err := gaClient.Interface.List(context.Background(), metav1.ListOptions{})
	if err != nil || snErr != nil {
		betaClient, snErr := client.CreateSnapshotBetaClient(namespace)
		if snErr != nil {
			return nil, nil, snErr
		}
		return nil, betaClient, nil
	}
	return gaClient, nil, nil
}

// VolumeMigrateSuite is used to manage volume migrate test suite
type VolumeMigrateSuite struct {
	TargetSC     string
	Description  string
	VolumeNumber int
	PodNumber    int
	Flag         bool
}

// Run executes volume migrate test suite
func (vms *VolumeMigrateSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)

	if vms.VolumeNumber <= 0 {
		log.Println("Using default number of volumes")
		vms.VolumeNumber = 1
	}
	if vms.PodNumber <= 0 {
		log.Println("Using default number of pods")
		vms.PodNumber = 3
	}

	log.Println("Volumes:", vms.VolumeNumber, "pods:", vms.PodNumber)

	scClient := clients.SCClient
	pvcClient := clients.PVCClient
	pvClient := clients.PersistentVolumeClient
	podClient := clients.PodClient
	stsClient := clients.StatefulSetClient

	sourceSC := scClient.Get(ctx, storageClass)
	if sourceSC.HasError() {
		return delFunc, sourceSC.GetError()
	}
	targetSC := scClient.Get(ctx, vms.TargetSC)
	if targetSC.HasError() {
		return delFunc, targetSC.GetError()
	}

	stsConf := testcore.VolumeMigrateStsConfig(storageClass, "1Gi", vms.VolumeNumber, int32(vms.PodNumber), "")
	stsTmpl := stsClient.MakeStatefulSet(stsConf)
	// Creating Statefulset
	log.Println("Creating Statefulset")
	sts := stsClient.Create(ctx, stsTmpl)
	if sts.HasError() {
		return delFunc, sts.GetError()
	}
	sts = sts.Sync(ctx)
	if sts.HasError() {
		return delFunc, sts.GetError()
	}

	var pvNames []string
	podList, err := sts.GetPodList(ctx)
	if err != nil {
		return delFunc, err
	}
	g, _ := errgroup.WithContext(ctx)
	for _, pod := range podList.Items {
		pod := pod
		for _, volume := range pod.Spec.Volumes {
			volume := volume
			if volume.PersistentVolumeClaim != nil {
				g.Go(func() error {
					log.Println("Getting PVC")
					pvc := pvcClient.Get(ctx, volume.PersistentVolumeClaim.ClaimName)
					if pvc.HasError() {
						return pvc.GetError()
					}
					err = pvcClient.WaitForAllToBeBound(ctx)
					if err != nil {
						return err
					}

					log.Println("Getting PV")
					pvName := pvc.Object.Spec.VolumeName
					pvNames = append(pvNames, pvName)
					pv := pvClient.Get(ctx, pvName)
					if pv.HasError() {
						return pv.GetError()
					}

					if !vms.Flag {
						file := fmt.Sprintf("%s0/writer-%d.data", stsConf.MountPath, 0)
						sum := fmt.Sprintf("%s0/writer-%d.sha512", stsConf.MountPath, 0)
						// Write random blob
						ddRes := bytes.NewBufferString("")
						if err := podClient.Exec(ctx, &pod, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
							return err
						}
						log.Info("Writer pod: ", pod.Name)
						log.Debug(ddRes.String())
						log.Info("Written the values successfully ", ddRes)
						log.Info(ddRes.String())

						// Write hash sum of blob
						if err := podClient.Exec(ctx, &pod, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
							log.Println("write hash sum err")
							return err
						}
						log.Info("Checksum value: ", sum)
						// sync to be sure
						if err := podClient.Exec(ctx, &pod, []string{"/bin/bash", "-c", "sync " + sum}, os.Stdout, os.Stderr, false); err != nil {
							return err
						}
					}

					pv.Object.Annotations["migration.storage.dell.com/migrate-to"] = vms.TargetSC
					log.Println("Updating PV")
					updatedPV := pvClient.Update(ctx, pv.Object)
					if updatedPV.HasError() {
						return updatedPV.GetError()
					}

					log.Println("Waiting PV to create")
					err = pvClient.WaitPV(ctx, pvName+"-to-"+vms.TargetSC)
					if err != nil {
						return err
					}
					log.Println("pv", pvName+"-to-"+vms.TargetSC, "seems good")
					return nil
				})
			}
		}
	}

	delFunc = func(f func() error) func() error {
		return func() error {
			log.Info("Deleting pvs")
			pvs, err := pvClient.Interface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}
			for _, p := range pvs.Items {
				for _, name := range pvNames {
					p := p
					if strings.Contains(p.Name, name) {
						pvClient.Delete(ctx, &p)
					}
				}
			}
			return nil
		}
	}(nil)

	if err := g.Wait(); err != nil {
		log.Println("g.wait err")
		return delFunc, err
	}

	if vms.Flag {
		return delFunc, nil
	}

	log.Println("Deleting old Statefulset")
	deletionOrphan := metav1.DeletePropagationOrphan
	delSts := stsClient.DeleteWithOptions(ctx, sts.Set, metav1.DeleteOptions{PropagationPolicy: &deletionOrphan})
	if delSts.HasError() {
		return delFunc, delSts.GetError()
	}

	log.Println("Deleting pods")
	for _, pod := range podList.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				log.Println("Deleting PVC")
				pvc := pvcClient.Get(ctx, volume.PersistentVolumeClaim.ClaimName)
				if pvc.HasError() {
					return delFunc, pvc.GetError()
				}
				delPVC := pvcClient.Delete(ctx, pvc.Object)
				if delPVC.HasError() {
					return delFunc, delPVC.GetError()
				}
			}
		}
		pod := pod
		podClient.Delete(ctx, &pod)
	}

	newStsConf := testcore.VolumeMigrateStsConfig(vms.TargetSC, "1Gi", vms.VolumeNumber, int32(vms.PodNumber), "")
	newStsTmpl := stsClient.MakeStatefulSet(newStsConf)
	// Creating new Statefulset
	log.Println("Creating new Statefulset")
	newSts := stsClient.Create(ctx, newStsTmpl)
	if newSts.HasError() {
		return delFunc, newSts.GetError()
	}
	newSts = newSts.Sync(ctx)
	if newSts.HasError() {
		return delFunc, newSts.GetError()
	}

	newPodList, err := newSts.GetPodList(ctx)
	if err != nil {
		return delFunc, err
	}

	for _, pod := range newPodList.Items {
		// Check if hash sum is correct
		sum := fmt.Sprintf("%s0/writer-%d.sha512", newStsConf.MountPath, 0)
		writer := bytes.NewBufferString("")
		log.Info("Checker pod: ", pod.Name)
		pod := pod
		if err := podClient.Exec(ctx, &pod, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
			return delFunc, err
		}
		if strings.Contains(writer.String(), "OK") {
			log.Info("Hashes match")
		} else {
			return delFunc, fmt.Errorf("hashes don't match")
		}
	}

	return delFunc, nil
}

// GetObservers returns all observers
func (*VolumeMigrateSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetClients creates and returns pvc, pv, sc, pod, statefulset, va, metrics clients
func (vms *VolumeMigrateSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.StorageClassExists(context.Background(), vms.TargetSC); !ok {
		return nil, fmt.Errorf("target storage class doesn't exist; error = %v", err)
	}

	pvClient, pvErr := client.CreatePVClient()
	if pvErr != nil {
		return nil, pvErr
	}

	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	scClient, scErr := client.CreateSCClient()
	if scErr != nil {
		return nil, scErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	stsClient, stsErr := client.CreateStatefulSetClient(namespace)
	if stsErr != nil {
		return nil, stsErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	return &k8sclient.Clients{
		PersistentVolumeClient: pvClient,
		PVCClient:              pvcClient,
		PodClient:              podClient,
		SCClient:               scClient,
		StatefulSetClient:      stsClient,
		VaClient:               vaClient,
		MetricsClient:          metricsClient,
	}, nil
}

// GetNamespace returns volume migrate test suite namespace
func (*VolumeMigrateSuite) GetNamespace() string {
	return "migration-test"
}

// GetName returns volume migrate test suite name
func (vms *VolumeMigrateSuite) GetName() string {
	if vms.Description != "" {
		return vms.Description
	}
	return "VolumeMigrationSuite"
}

// Parameters returns formatted string of parameters
func (vms *VolumeMigrateSuite) Parameters() string {
	return fmt.Sprintf("{Target storageclass: %s, volumes: %d, pods: %d}", vms.TargetSC, vms.VolumeNumber, vms.PodNumber)
}
