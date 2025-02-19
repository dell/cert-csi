package volcreation

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/fatih/color"
	v1 "k8s.io/api/core/v1"
	"strconv"
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
		vcs.VolumeNumber = 1
	}
	if vcs.VolumeSize == "" {
		log.Info("Using default volume size")
		vcs.VolumeSize = "3Gi"
	}
	result := common.ValidateCustomName(vcs.CustomName, vcs.VolumeNumber)
	if result {
		log.Infof("using custom pvc-name:%s", vcs.CustomName)
	} else {
		vcs.CustomName = ""
	}

	log.Infof("Creating %s volumes with size:%s", color.YellowString(strconv.Itoa(vcs.VolumeNumber)),
		color.YellowString(vcs.VolumeSize))
	pvcClient := clients.PVCClient

	firstConsumer, err := common.ShouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
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
