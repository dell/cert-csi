package multivolattach

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/node"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strconv"
	"strings"
	"time"
)

// MultiAttachSuite is used to manage multi attach test suite
type MultiAttachSuite struct {
	PodNumber   int
	RawBlock    bool
	Description string
	AccessMode  string
	VolumeSize  string
	Image       string
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
	if mas.Image == "" {
		mas.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", mas.Image)
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
	podconf := testcore.MultiAttachPodConfig([]string{pvc.Object.Name}, mas.Image)

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
		file := "/tmp/blob.data"

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
	spreadConstraints := []v1.TopologySpreadConstraint{
		{
			MaxSkew:           int32(maxSkew), // #nosec G115
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
	return common.GetAllObservers(obsType)
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
