package volumeclone

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/fatih/color"
	"strconv"
)

// CloneVolumeSuite is used to manage clone volume suite test suite
type CloneVolumeSuite struct {
	VolumeNumber  int
	VolumeSize    string
	PodNumber     int
	CustomPvcName string
	CustomPodName string
	Description   string
	AccessMode    string
	Image         string
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
	if cs.Image == "" {
		cs.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", cs.Image)
	}
	clonedVolName := ""
	result := common.ValidateCustomName(cs.CustomPvcName, cs.VolumeNumber)
	if result {
		clonedVolName = cs.CustomPvcName + "-cloned"
		log.Infof("using custom pvc-name:%s"+" and cloned PVC name:%s", cs.CustomPvcName, clonedVolName)
	} else {
		cs.CustomPvcName = ""
	}

	clonedPodName := ""
	result = common.ValidateCustomName(cs.CustomPodName, cs.VolumeNumber)
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
		podconf := testcore.ProvisioningPodConfig(pvcNameList, cs.CustomPodName, cs.Image)
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
		podconf := testcore.ProvisioningPodConfig(pvcNameList, clonedPodName, cs.Image)
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
	return common.GetAllObservers(obsType)
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
