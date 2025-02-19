package vgs

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
)

// VolumeGroupSnapSuite is used to manage volume group volsnap test suite
type VolumeGroupSnapSuite struct {
	SnapClass       string
	VolumeSize      string
	AccessMode      string
	VolumeGroupName string
	VolumeLabel     string
	ReclaimPolicy   string
	VolumeNumber    int
	Driver          string
	Image           string
}

// Run executes volume group volsnap test suite
func (vgs *VolumeGroupSnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	var namespace string

	if vgs.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		vgs.VolumeSize = "3Gi"
	}

	if vgs.Image == "" {
		vgs.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", vgs.Image)
	}

	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	vgsClient := clients.VgsClient

	firstConsumer, err := common.ShouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
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
	podconf := testcore.IoWritePodConfig(pvcNameList, "", vgs.Image)
	podTmpl := podClient.MakePod(podconf)

	writerPod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// create volume group blockvolsnapshot using CRD.
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
	return common.GetAllObservers(obsType)
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

// GetNamespace returns volume group volsnap test suite namespace
func (*VolumeGroupSnapSuite) GetNamespace() string {
	return "vgs-volsnap-test"
}

// GetName returns volume group volsnap test suite name
func (vgs *VolumeGroupSnapSuite) GetName() string {
	return "VolumeGroupSnapSuite"
}

// Parameters returns formatted string of parameters
func (vgs *VolumeGroupSnapSuite) Parameters() string {
	return fmt.Sprintf("{volumes: %d, volumeSize: %s}", vgs.VolumeNumber, vgs.VolumeSize)
}
