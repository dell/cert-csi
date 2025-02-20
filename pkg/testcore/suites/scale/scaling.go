package scale

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
)

// ScalingSuite is used to manage scaling test suite
type ScalingSuite struct {
	ReplicaNumber    int
	VolumeNumber     int
	GradualScaleDown bool
	PodPolicy        string
	VolumeSize       string
	Image            string
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
	if ss.Image == "" {
		ss.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", ss.Image)
	}

	stsconf := testcore.ScalingStsConfig(storageClass, ss.VolumeSize, ss.VolumeNumber, ss.PodPolicy, ss.Image)
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
	sts = stsClient.Scale(ctx, sts.Set, int32(ss.ReplicaNumber)) // #nosec G115
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
			sts := stsClient.Scale(ctx, sts.Set, int32(i)) // #nosec G115
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
	return common.GetAllObservers(obsType)
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
