package podprovision

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"strconv"
)

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
	Image         string
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
	if ps.Image == "" {
		ps.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", ps.Image)
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
		podconf := testcore.ProvisioningPodConfig(pvcNameList, ps.PodCustomName, ps.Image)
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
	return common.GetAllObservers(obsType)
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
