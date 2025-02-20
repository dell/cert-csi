package volumeio

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
)

type VolumeIoSuite struct {
	VolumeNumber int
	VolumeSize   string
	ChainNumber  int
	ChainLength  int
	Image        string
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

	if vis.Image == "" {
		vis.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", vis.Image)
	}

	firstConsumer, err := common.ShouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
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
		podconf := testcore.IoWritePodConfig(pvcNameList, "", vis.Image)
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
	return common.GetAllObservers(obsType)
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
