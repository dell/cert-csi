package healthmetric

import (
	"context"
	"errors"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
	"strings"
	"time"
)

const (
	// ControllerLogsSleepTime is controller logs sleep time
	ControllerLogsSleepTime = 20
	maxRetryCount           = 30
)

// VolumeHealthMetricsSuite is used to manage volume health metrics test suite
type VolumeHealthMetricsSuite struct {
	VolumeNumber int
	PodNumber    int
	VolumeSize   string
	Description  string
	AccessMode   string
	Namespace    string
	Image        string
}

// FindDriverLogs executes command and returns the output
var FindDriverLogs = func(command []string) (string, error) {
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
	if vh.Image == "" {
		vh.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", vh.Image)
	}

	// Create a PVC
	vcconf := testcore.VolumeCreationConfig(storageClass, vh.VolumeSize, "", vh.AccessMode)
	volTmpl := pvcClient.MakePVC(vcconf)
	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}
	PVCNamespace := pvcClient.Namespace

	// Create Pod, and attach PVC
	podconf := testcore.VolumeHealthPodConfig([]string{pvc.Object.Name}, "", vh.Image)
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

	NodeLogs := false
	ControllerLogs := false

	retryCount := 1
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
	return common.GetAllObservers(obsType)
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
