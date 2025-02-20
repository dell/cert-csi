package expansion

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/fatih/color"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/pkg/kubelet/events"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// VolumeExpansionSuite is used to manage volume expansion test suite
type VolumeExpansionSuite struct {
	VolumeNumber int
	PodNumber    int
	IsBlock      bool
	InitialSize  string
	ExpandedSize string
	Description  string
	AccessMode   string
	Image        string
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
	if ves.Image == "" {
		ves.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", ves.Image)
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
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "", ves.Image)
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

	// Give some time for driver to expand
	time.Sleep(10 * time.Second)

	if ves.IsBlock {
		// Check for "FileSystemResizeSuccessful" event to confirm successful resizing
		log.Infof("Waiting for 'FileSystemResizeSuccessful' event for each pod")

		for _, pod := range podObjectList {
			pollErr := wait.PollImmediate(10*time.Second, time.Duration(pvcClient.Timeout)*time.Second,
				func() (bool, error) {
					eventList, err := podClient.ClientSet.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
						FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", pod.Name),
					})
					if err != nil {
						log.Errorf("Failed to list events for pod %s: %v", pod.Name, err)
						return false, err
					}

					for _, event := range eventList.Items {
						if event.Reason == events.FileSystemResizeSuccess {
							log.Infof("Pod %s: 'FileSystemResizeSuccessful' event detected", pod.Name)
							return true, nil
						}
					}
					return false, nil
				})

			if pollErr != nil {
				return delFunc, fmt.Errorf("timed out waiting for 'FileSystemResizeSuccessful' event for pod %s", pod.Name)
			}
		}

		log.Infof("'FileSystemResizeSuccessful' event detected for all pods")
		return delFunc, nil
	}

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
	return common.GetAllObservers(obsType)
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
