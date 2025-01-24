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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// DeletionStruct is used by volume deletion suite
type DeletionStruct struct {
	Name        string
	Namespace   string
	Description string
}

// VolumeDeletionSuite is used for managing volume deletion test suite
type VolumeDeletionSuite struct {
	*DeletionStruct
}

// Run to delete the volume created by name and namespace as cli params
func (vds *VolumeDeletionSuite) Run(ctx context.Context, _ string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	if vds.Name == "" {
		log.Fatalf("Error PVC name is required parameter")
	}

	log.Infof("Deleting volume with name:%s", color.YellowString(vds.Name))
	pvcClient := clients.PVCClient
	pvcObj, _ := pvcClient.Interface.Get(ctx, vds.Name, metav1.GetOptions{})
	err := pvcClient.Delete(ctx, pvcObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}
	return delFunc, nil
}

// GetName returns volume deletion suite name
func (vds *VolumeDeletionSuite) GetName() string {
	if vds.Description != "" {
		return vds.Description
	}
	return "VolumeDeletionSuite"
}

// GetObservers returns pvc, entitynumber and container observers
func (*VolumeDeletionSuite) GetObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PvcObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

// GetClients creates and returns PVC and Metrics clients
func (vds *VolumeDeletionSuite) GetClients(_ string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(vds.Namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(vds.Namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	return &k8sclient.Clients{
		PVCClient:     pvcClient,
		MetricsClient: metricsClient,
	}, nil
}

// GetNamespace returns volume deletion suite namespace
func (vds *VolumeDeletionSuite) GetNamespace() string {
	return vds.Namespace
}

// Parameters is returns format string
func (vds *VolumeDeletionSuite) Parameters() string {
	return "{}"
}

// PodDeletionSuite is used for managing pod deletion test suite
type PodDeletionSuite struct {
	*DeletionStruct
}

// Run to delete the volume created by name and namespace as cli params
func (pds *PodDeletionSuite) Run(ctx context.Context, _ string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	if pds.Name == "" {
		log.Fatalf("Error Pod name is required parameter")
	}

	log.Infof("Deleting pod with name:%s", color.YellowString(pds.Name))
	podClient := clients.PodClient
	podObj, _ := podClient.Interface.Get(ctx, pds.Name, metav1.GetOptions{})
	err := podClient.Delete(ctx, podObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}
	// getting the list of volumes attached to the pod in order to delete them upon pod deletion
	attachedVols := podObj.Spec.Volumes
	pvcClient := clients.PVCClient
	vaClient := clients.VaClient

	for i := 0; i < len(attachedVols); i++ {
		pvc := attachedVols[i].VolumeSource.PersistentVolumeClaim
		if pvc != nil {
			pvcObj, _ := pvcClient.Interface.Get(ctx, pvc.ClaimName, metav1.GetOptions{})
			err := pvcClient.Delete(ctx, pvcObj).Sync(ctx).GetError()
			if err != nil {
				return delFunc, err
			}
			err = vaClient.WaitUntilVaGone(ctx, pvcObj.Spec.VolumeName)
			if err != nil {
				return delFunc, err
			}
		}
	}
	return delFunc, nil
}

// GetName returns pod deletion suite name
func (pds *PodDeletionSuite) GetName() string {
	if pds.Description != "" {
		return pds.Description
	}
	return "PodDeletionSuite"
}

// GetObservers returns pod, pvc, va, entitynumber, containermetrics observers
func (*PodDeletionSuite) GetObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PodObserver{},
			&observer.PvcObserver{},
			&observer.VaObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

// GetClients creates and returns pod, pvc, va, metrics clients
func (pds *PodDeletionSuite) GetClients(_ string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	podClient, podErr := client.CreatePodClient(pds.Namespace)
	if podErr != nil {
		return nil, podErr
	}
	pvcClient, pvcErr := client.CreatePVCClient(pds.Namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}
	vaClient, vaErr := client.CreateVaClient(pds.Namespace)
	if vaErr != nil {
		return nil, pvcErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(pds.Namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	return &k8sclient.Clients{
		PodClient:         podClient,
		PVCClient:         pvcClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
	}, nil
}

// GetNamespace returns pod deletion suite namespace
func (pds *PodDeletionSuite) GetNamespace() string {
	return pds.Namespace
}

// Parameters returns format string
func (pds *PodDeletionSuite) Parameters() string {
	return "{}"
}

// ClonedVolDeletionSuite is used to manage cloned volume deletion test suite
type ClonedVolDeletionSuite struct {
	*DeletionStruct
	PodName string
}

// Run to delete the volume created by name and namespace as cli params
func (pds *ClonedVolDeletionSuite) Run(ctx context.Context, _ string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	if pds.Name == "" {
		log.Fatalf("Error PVC name is required parameter")
	}
	if pds.PodName == "" {
		log.Fatalf("Error Pod name is required parameter")
	}

	log.Infof("Deleting pod with name:%s", color.YellowString(pds.PodName))
	podClient := clients.PodClient
	podObj, _ := podClient.Interface.Get(ctx, pds.PodName, metav1.GetOptions{})
	err := podClient.Delete(ctx, podObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}

	// Deleting corresponding pod attached to cloned volume
	cPodObj, _ := podClient.Interface.Get(ctx, pds.PodName+"-cloned", metav1.GetOptions{})
	log.Infof("Deleting pod with name:%s", color.YellowString(cPodObj.GetName()))
	err = podClient.Delete(ctx, cPodObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}

	log.Infof("Deleting volume with name:%s", color.YellowString(pds.Name))
	pvcClient := clients.PVCClient

	pvcObj, _ := pvcClient.Interface.Get(ctx, pds.Name, metav1.GetOptions{})
	err = pvcClient.Delete(ctx, pvcObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}

	// Deleting corresponding cloned volume
	cPvcObj, _ := pvcClient.Interface.Get(ctx, pds.Name+"-cloned", metav1.GetOptions{})
	log.Infof("Deleting volume with name:%s", color.YellowString(cPvcObj.GetName()))
	err = pvcClient.Delete(ctx, cPvcObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}
	// wait for volume attachments to be deleted
	vaClient := clients.VaClient
	err = vaClient.WaitUntilVaGone(ctx, pvcObj.Spec.VolumeName)
	if err != nil {
		return delFunc, err
	}
	cPVCErr := vaClient.WaitUntilVaGone(ctx, cPvcObj.Spec.VolumeName)
	if cPVCErr != nil {
		return delFunc, cPVCErr
	}

	return delFunc, nil
}

// GetName returns cloned volume deletion test suite name
func (pds *ClonedVolDeletionSuite) GetName() string {
	if pds.Description != "" {
		return pds.Description
	}
	return "ClonedVolumeDeletionSuite"
}

// GetObservers returns pod, pvc, va, entitynumber, containermetrics observers
func (*ClonedVolDeletionSuite) GetObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PodObserver{},
			&observer.PvcObserver{},
			&observer.VaObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

// GetClients creates and returns pod, pvc, metrics, va clients
func (pds *ClonedVolDeletionSuite) GetClients(_ string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	podClient, pvcErr := client.CreatePodClient(pds.Namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}
	pvcClient, pvcErr := client.CreatePVCClient(pds.Namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}
	metricsClient, mcErr := client.CreateMetricsClient(pds.Namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	vaClient, vaErr := client.CreateVaClient(pds.Namespace)
	if vaErr != nil {
		return nil, pvcErr
	}
	return &k8sclient.Clients{
		PodClient:         podClient,
		PVCClient:         pvcClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
	}, nil
}

// GetNamespace returns cloned volume deletion suite namespace
func (pds *ClonedVolDeletionSuite) GetNamespace() string {
	return pds.Namespace
}

// Parameters returns format string
func (pds *ClonedVolDeletionSuite) Parameters() string {
	return "{}"
}

// SnapshotDeletionSuite is used for managing snapshot deletion test suite
type SnapshotDeletionSuite struct {
	*DeletionStruct
}

// Run to snaphot the volume created by name and namespace as cli params
func (sds *SnapshotDeletionSuite) Run(ctx context.Context, _ string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	if sds.Name == "" {
		log.Fatalf("Error snap name is required parameter")
	}

	log.Infof("Deleting snapshot with name:%s", color.YellowString(sds.Name))
	snapClient := clients.SnapClientGA
	snapObj, _ := snapClient.Interface.Get(ctx, sds.Name, metav1.GetOptions{})
	err := snapClient.Delete(ctx, snapObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	// Deleting pod attached to restored volume
	sPodObj, _ := podClient.Interface.Get(ctx, sds.Name+"-restore-pod", metav1.GetOptions{})
	log.Infof("Deleting restored pod with name:%s", color.YellowString(sPodObj.GetName()))
	err = podClient.Delete(ctx, sPodObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}

	// Deleting restored volume from snapshot
	sPvcObj, _ := pvcClient.Interface.Get(ctx, sds.Name+"-restore", metav1.GetOptions{})
	log.Infof("Deleting restored volume from snapshot with name:%s", color.YellowString(sPvcObj.GetName()))
	err = pvcClient.Delete(ctx, sPvcObj).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}
	// wait for volume attachments to be deleted
	vaClient := clients.VaClient
	err = vaClient.WaitUntilVaGone(ctx, sPvcObj.Spec.VolumeName)
	if err != nil {
		return delFunc, err
	}

	// Deleting volume created using custom snapname before Snapshot creation
	sPvcObj1, _ := pvcClient.Interface.Get(ctx, sds.Name+"-pvc", metav1.GetOptions{})
	log.Infof("Deleting volume created using custom snapname before Snapshot creation:%s", color.YellowString(sPvcObj1.GetName()))
	err = pvcClient.Delete(ctx, sPvcObj1).Sync(ctx).GetError()
	if err != nil {
		return delFunc, err
	}
	// wait for volume attachments to be deleted
	err = vaClient.WaitUntilVaGone(ctx, sPvcObj1.Spec.VolumeName)
	if err != nil {
		return delFunc, err
	}
	return delFunc, nil
}

// GetName returns snapshot deletion suite name
func (sds *SnapshotDeletionSuite) GetName() string {
	if sds.Description != "" {
		return sds.Description
	}
	return "SnapshotDeletionSuite"
}

// GetObservers returns pod, pvc, va, entitynumber and containermetrics observers
func (*SnapshotDeletionSuite) GetObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PodObserver{},
			&observer.PvcObserver{},
			&observer.VaObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

// GetClients creates and returns pod, pvc, va, metrics, snapshot clients
func (sds *SnapshotDeletionSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	podClient, podErr := client.CreatePodClient(sds.Namespace)
	if podErr != nil {
		return nil, podErr
	}

	pvcClient, pvcErr := client.CreatePVCClient(sds.Namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}
	metricsClient, mcErr := client.CreateMetricsClient(sds.Namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	if client.Minor > 17 {
		snapClient, snErr := client.CreateSnapshotGAClient(namespace)
		if snErr != nil {
			return nil, snErr
		}

		return &k8sclient.Clients{
			PVCClient:         pvcClient,
			PodClient:         podClient,
			VaClient:          vaClient,
			StatefulSetClient: nil,
			MetricsClient:     metricsClient,
			SnapClientGA:      snapClient,
			SnapClientBeta:    nil,
		}, nil
	}

	snapClient, snErr := client.CreateSnapshotBetaClient(namespace)
	if snErr != nil {
		return nil, snErr
	}

	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      nil,
		SnapClientBeta:    snapClient,
	}, nil
}

// GetNamespace returns snapshot deletion suite namespace
func (sds *SnapshotDeletionSuite) GetNamespace() string {
	return sds.Namespace
}

// Parameters returns format string
func (sds *SnapshotDeletionSuite) Parameters() string {
	return "{}"
}

// EphemeralVolumeSuite is used to manage ephemeral volume test suite
type EphemeralVolumeSuite struct {
	PodCustomName    string
	Description      string
	PodNumber        int
	Driver           string
	FSType           string
	Image            string
	VolumeAttributes map[string]string
}

// Run runs ephemeral volume test suite
func (ep *EphemeralVolumeSuite) Run(ctx context.Context, _ string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	podClient := clients.PodClient

	if ep.PodNumber <= 0 {
		log.Info("Using default number of pods")
		ep.PodNumber = 3
	}

	if ep.Image == "" {
		ep.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", ep.Image)
	}

	log.Infof("Creating %s pods, each with 1 volumes", color.YellowString(strconv.Itoa(ep.PodNumber)))

	csiVolSrc := v1.CSIVolumeSource{
		Driver:           ep.Driver,
		FSType:           &ep.FSType,
		VolumeAttributes: ep.VolumeAttributes,
	}
	EphemeralVolumeName := ""
	if ep.Driver == "csi-vxflexos.dellemc.com" {
		if value, exists := ep.VolumeAttributes["volumeName"]; exists {
			EphemeralVolumeName = value
		}
	}
	var podConf *pod.Config
	var ephPods []*pod.Pod
	for i := 0; i < ep.PodNumber; i++ {
		name := ""
		if len(ep.PodCustomName) != 0 {
			name = ep.PodCustomName + "-" + strconv.Itoa(i)
		}
		// Create pod with ephemeral inline volume
		if _, exists := ep.VolumeAttributes["volumeName"]; exists && ep.Driver == "csi-vxflexos.dellemc.com" {
			csiVolSrc.VolumeAttributes["volumeName"] = EphemeralVolumeName + "-" + k8sclient.RandomSuffix()
		}
		podConf = testcore.EphemeralPodConfig(name, csiVolSrc, ep.Image)
		podTmpl := podClient.MakeEphemeralPod(podConf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
		ephPods = append(ephPods, pod)
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	for _, ephPod := range ephPods {
		// write data to pod
		log.Infof("Writing to Volume on %s", ephPod.Object.GetName())
		file := fmt.Sprintf("%s/blob.data", podConf.MountPath)
		sum := fmt.Sprintf("%s/blob.sha512", podConf.MountPath)
		// Write random blob to pvc
		ddRes := bytes.NewBufferString("")
		if err := podClient.Exec(ctx, ephPod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
			return delFunc, err
		}
		log.Info("Writer originalPod: ", ephPod.Object.GetName())
		log.Debug(ddRes.String())

		// Write hash sum of blob
		if err := podClient.Exec(ctx, ephPod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return delFunc, err
		}
		// check hash
		writer := bytes.NewBufferString("")
		if err := podClient.Exec(ctx, ephPod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
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

// GetObservers returns pod, va, containermetrics observers
func (*EphemeralVolumeSuite) GetObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PodObserver{},
			&observer.VaObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

// GetClients creates and returns pod, va, metrics clients
func (*EphemeralVolumeSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
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
		PodClient:     podClient,
		VaClient:      vaClient,
		MetricsClient: metricsClient,
	}, nil
}

// GetNamespace returns ephemeral volume suite namespace
func (*EphemeralVolumeSuite) GetNamespace() string {
	return "functional-test"
}

// GetName returns ephemeral volume suite name
func (ep *EphemeralVolumeSuite) GetName() string {
	if ep.Description != "" {
		return ep.Description
	}
	return "EphemeralVolumeSuite"
}

// Parameters returns parameters string
func (ep *EphemeralVolumeSuite) Parameters() string {
	return fmt.Sprintf("{driver: %s, podNumber: %s, volAttributes: %s}",
		ep.Driver,
		strconv.Itoa(ep.PodNumber),
		fmt.Sprint(ep.VolumeAttributes))
}

// NodeDrainSuite is used to manage node drain test suite
type NodeDrainSuite struct {
	Name               string
	Namespace          string
	Description        string
	DisableEviction    bool
	GracePeriodSeconds int
}

// Run to delete the volume created by name and namespace as cli params
func (nds *NodeDrainSuite) Run(ctx context.Context, _ string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	if nds.Name == "" {
		log.Fatalf("Error Node name is required parameter")
	}

	log.Infof("Draining node with name:%s", color.YellowString(nds.Name))
	nodeClient := clients.NodeClient
	_ = nodeClient.NodeCordon(ctx, nds.Name)

	podClient := clients.PodClient
	err := podClient.DeleteOrEvictPods(ctx, nds.Name, nds.GracePeriodSeconds)
	if err != nil {
		return delFunc, err
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	podList, podErr := podClient.Interface.List(ctx, metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nds.Name}).String(),
	})
	if podErr != nil {
		return delFunc, podErr
	}
	if len(podList.Items) == 0 {
		log.Infof("The node %v have been drained successfully", nds.Name)
	}

	return delFunc, nil
}

// GetName returns node drain suite name
func (nds *NodeDrainSuite) GetName() string {
	if nds.Description != "" {
		return nds.Description
	}
	return "NodeDrainSuite"
}

// GetObservers returns pod, pvc, va, entitynumber, containermetrics observers
func (*NodeDrainSuite) GetObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PodObserver{},
			&observer.PvcObserver{},
			&observer.VaObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

// GetClients creates and returns node, pod, pvc, va, statefulset, metrics clients
func (nds *NodeDrainSuite) GetClients(_ string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	nodeClient, nodeErr := client.CreateNodeClient()
	if nodeErr != nil {
		return nil, nodeErr
	}
	podClient, podErr := client.CreatePodClient(nds.Namespace)
	if podErr != nil {
		return nil, podErr
	}
	pvcClient, pvcErr := client.CreatePVCClient(nds.Namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}
	vaClient, vaErr := client.CreateVaClient(nds.Namespace)
	if vaErr != nil {
		return nil, vaErr
	}
	statefulClient, stateErr := client.CreateStatefulSetClient(nds.Namespace)
	if stateErr != nil {
		return nil, stateErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(nds.Namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	return &k8sclient.Clients{
		PodClient:         podClient,
		PVCClient:         pvcClient,
		VaClient:          vaClient,
		StatefulSetClient: statefulClient,
		MetricsClient:     metricsClient,
		NodeClient:        nodeClient,
	}, nil
}

// GetNamespace returns node drain suite namespace
func (nds *NodeDrainSuite) GetNamespace() string {
	return nds.Namespace
}

// Parameters returns format string
func (nds *NodeDrainSuite) Parameters() string {
	return "{}"
}

// NodeUncordonSuite is used to manage node uncordon test suite
type NodeUncordonSuite struct {
	Name        string
	Description string
	Namespace   string
}

// Run to delete the volume created by name and namespace as cli params
func (nds *NodeUncordonSuite) Run(ctx context.Context, _ string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	if nds.Name == "" {
		log.Fatalf("Error Node name is required parameter")
	}

	log.Infof("Uncordoning node with name:%s", color.YellowString(nds.Name))
	nodeClient := clients.NodeClient
	err := nodeClient.NodeUnCordon(ctx, nds.Name)
	if err != nil {
		return delFunc, err
	}

	return delFunc, nil
}

// GetName returns node uncordon test suite name
func (nds *NodeUncordonSuite) GetName() string {
	if nds.Description != "" {
		return nds.Description
	}
	return "NodeUncordonSuite"
}

// GetObservers returns pod, pvc, va, entitynumber, containermetrics observers
func (*NodeUncordonSuite) GetObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PodObserver{},
			&observer.PvcObserver{},
			&observer.VaObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

// GetClients creates and returns node, pod, pvc, va, metrics clients
func (nds *NodeUncordonSuite) GetClients(_ string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	nodeClient, nodeErr := client.CreateNodeClient()
	if nodeErr != nil {
		return nil, nodeErr
	}
	podClient, podErr := client.CreatePodClient(nds.Namespace)
	if podErr != nil {
		return nil, podErr
	}
	pvcClient, pvcErr := client.CreatePVCClient(nds.Namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}
	vaClient, vaErr := client.CreateVaClient(nds.Namespace)
	if vaErr != nil {
		return nil, pvcErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(nds.Namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	return &k8sclient.Clients{
		PodClient:         podClient,
		PVCClient:         pvcClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		NodeClient:        nodeClient,
	}, nil
}

// GetNamespace returns node uncordon suite namespace
func (nds *NodeUncordonSuite) GetNamespace() string {
	return nds.Namespace
}

// Parameters returns format string
func (nds *NodeUncordonSuite) Parameters() string {
	return "{}"
}

// CapacityTrackingSuite is used to manage storage capacity tracking test suite
type CapacityTrackingSuite struct {
	DriverNamespace string
	StorageClass    string
	VolumeSize      string
	Image           string
	PollInterval    time.Duration
}

// Run runs storage capacity tracking test suite
func (cts *CapacityTrackingSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	storageClass = cts.StorageClass
	sc := clients.SCClient.Get(ctx, storageClass)
	if sc.HasError() {
		return delFunc, sc.GetError()
	}

	// Get unique topology count from csinode
	// Get topology keys to filter for when retrieving topology count
	topologyKeys := []string{}

	if len(sc.Object.AllowedTopologies) > 0 {
		matchLabelExpressions := sc.Object.AllowedTopologies[0].MatchLabelExpressions
		for _, exp := range matchLabelExpressions {
			topologyKeys = append(topologyKeys, exp.Key)
		}
	}
	topologiesCount, err := getTopologyCount(topologyKeys)
	if err != nil {
		return delFunc, err
	}
	log.Infof("Found %s topology segment(s) in csinode", color.HiYellowString(strconv.Itoa(topologiesCount)))

	if cts.Image == "" {
		cts.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", cts.Image)
	}

	if *sc.Object.VolumeBindingMode != storagev1.VolumeBindingWaitForFirstConsumer {
		return delFunc, fmt.Errorf("%s storage class does not use late binding", color.YellowString(storageClass))
	}

	// Create new storage class
	tempScName := "capacity-tracking-" + k8sclient.RandomSuffix()
	log.Infof("Creating %s storage class", color.YellowString(tempScName))
	tempScTmpl := clients.SCClient.DuplicateStorageClass(tempScName, sc.Object)
	err = clients.SCClient.Create(ctx, tempScTmpl)
	if err != nil {
		return delFunc, err
	}

	// Delete the storage class before exiting the suite, this is helpful in the case of error scenarios e.g. CSIStorageCapacity objects are not created
	defer clients.SCClient.Delete(ctx, tempScName)

	// Wait for the CSIStorageCapacity objects to be created
	err = clients.CSISCClient.WaitForAllToBeCreated(ctx, tempScName, topologiesCount)
	if err != nil {
		return delFunc, err
	}

	// Delete the storage class and check if CSIStorageCapacity objects are deleted as well
	log.Infof("Deleting %s storage class,", color.YellowString(tempScName))
	err = clients.SCClient.Delete(ctx, tempScName)
	if err != nil {
		return delFunc, err
	}

	err = clients.CSISCClient.WaitForAllToBeDeleted(ctx, tempScName)
	if err != nil {
		return delFunc, err
	}

	// POD should stay in pending state if capacity is zero
	log.Infof("Updating CSIStorageCapacity for %s storage class, setting capacity to %s", color.YellowString(storageClass), color.HiYellowString("%d", 0))
	capacities, err := clients.CSISCClient.GetByStorageClass(ctx, storageClass)

	_, err = clients.CSISCClient.SetCapacityToZero(ctx, capacities)
	if err != nil {
		return delFunc, err
	}

	pvcName := "capacity-tracking-pvc-" + k8sclient.RandomSuffix()
	podName := "capacity-tracking-pod-" + k8sclient.RandomSuffix()
	log.Infof("Creating %s pod using %s storage class", color.YellowString(podName), color.YellowString(storageClass))

	pvcConf := testcore.VolumeCreationConfig(storageClass, cts.VolumeSize, pvcName, "ReadWriteOnce")
	pvcTmpl := clients.PVCClient.MakePVC(pvcConf)
	pvc := clients.PVCClient.Create(ctx, pvcTmpl, 1)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}

	podConf := testcore.CapacityTrackingPodConfig([]string{pvc.Object.Name}, podName, cts.Image)
	podTmpl := clients.PodClient.MakePod(podConf)
	pod := clients.PodClient.Create(ctx, podTmpl)
	if pod.HasError() {
		return delFunc, pod.GetError()
	}
	time.Sleep(5 * time.Second)
	err = pod.IsInPendingState(ctx)
	if err != nil {
		return delFunc, err
	}
	log.Infof("%s pod is %s", color.YellowString(pod.Object.Name), color.GreenString("PENDING"))

	// GetCapacity should be called by provisioner based on poll interval
	err = cts.checkIfGetCapacityIsPolled(ctx, clients, storageClass, topologiesCount)
	if err != nil {
		return delFunc, err
	}

	err = pod.WaitForRunning(ctx)
	if err != nil {
		return delFunc, err
	}
	log.Infof("%s pod is %s", color.YellowString(pod.Object.Name), color.GreenString("RUNNING"))

	return delFunc, nil
}

func getTopologyCount(topologyKeys []string) (int, error) {
	exe := []string{"bash", "-c", "kubectl describe csinode | grep 'Topology Keys'"}
	str, err := FindDriverLogs(exe)
	if len(str) == 0 || err != nil {
		return 0, err
	}
	topologies := strings.Split(strings.TrimSpace(strings.ReplaceAll(str, "Topology Keys:", "")), "\n")
	topologies = removeDuplicates(topologies)
	if len(topologyKeys) > 0 {
		topologies = filterArrayForMatches(topologies, topologyKeys)
	}
	topologiesCount := len(topologies)
	return topologiesCount, nil
}

func (cts *CapacityTrackingSuite) checkIfGetCapacityIsPolled(ctx context.Context, clients *k8sclient.Clients, storageClass string, _ int) error {
	log.Infof("Waiting for provisioner to %s GetCapacity for %s storage class in %s", color.GreenString("POLL"), color.YellowString(storageClass), color.HiYellowString(cts.PollInterval.String()))

	capacities, err := clients.CSISCClient.GetByStorageClass(ctx, storageClass)
	if err != nil {
		return err
	}

	err = capacities[0].WatchUntilUpdated(ctx, cts.PollInterval)
	if err != nil {
		return err
	}

	log.Infof("Provisioner %s GetCapacity for %s storage class", color.GreenString("POLLED"), color.YellowString(storageClass))
	return nil
}

func removeDuplicates(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strSlice {
		entry = strings.TrimSpace(entry)
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func filterArrayForMatches(listToFilter []string, filterValues []string) []string {
	filteredList := []string{}
	for _, value := range listToFilter {
		for _, key := range filterValues {
			if strings.Contains(value, key) {
				filteredList = append(filteredList, value)
				break
			}
		}
	}

	return filteredList
}

// GetName returns storage capacity tracking suite name
func (cts *CapacityTrackingSuite) GetName() string {
	return "CapacityTrackingSuite"
}

// Parameters returns formatted string of parameters
func (cts *CapacityTrackingSuite) Parameters() string {
	return fmt.Sprintf("{DriverNamespace: %s, volumeSize: %s, pollInterval: %s}", cts.DriverNamespace, cts.VolumeSize, cts.PollInterval.String())
}

// GetObservers returns all observers
func (cts *CapacityTrackingSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

// GetNamespace returns storage capacity tracking suite namespace
func (cts *CapacityTrackingSuite) GetNamespace() string {
	return "capacity-tracking-test"
}

// GetClients creates and returns pvc, pod, va, metrics, storage class, CSI storage capacity clients
func (cts *CapacityTrackingSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
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
		return nil, pvcErr
	}
	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}
	scClient, scErr := client.CreateSCClient()
	if scErr != nil {
		return nil, scErr
	}
	csiScClient, csiscErr := client.CreateCSISCClient(cts.DriverNamespace)
	if csiscErr != nil {
		return nil, csiscErr
	}
	return &k8sclient.Clients{
		KubeClient:    client,
		PVCClient:     pvcClient,
		PodClient:     podClient,
		VaClient:      vaClient,
		MetricsClient: metricsClient,
		SCClient:      scClient,
		CSISCClient:   csiScClient,
	}, nil
}
