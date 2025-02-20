package blockvolsnapshot

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshot"
	snapv1client "github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshot/v1"
	snapbetaclient "github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshot/v1beta1"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapbeta "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"time"
)

// BlockSnapSuite is used to manage block blockvolsnapshot test suite
type BlockSnapSuite struct {
	SnapClass   string
	VolumeSize  string
	Description string
	AccessMode  string
	Image       string
}

// Run executes block blockvolsnapshot test suite
func (bss *BlockSnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	if bss.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		bss.VolumeSize = "3Gi"
	}
	if bss.Image == "" {
		bss.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", bss.Image)
	}
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	log.Info("Creating BlockSnap pod")
	firstConsumer, err := common.ShouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return delFunc, err
	}
	// Create first PVC
	var pvcNameList []string
	vcconf := testcore.VolumeCreationConfig(storageClass, bss.VolumeSize, "", "")
	volTmpl := pvcClient.MakePVC(vcconf)
	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}
	pvcNameList = append(pvcNameList, pvc.Object.Name)
	if !firstConsumer {
		err := pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return delFunc, err
		}
	}

	// Create Pod, and attach PVC
	podconf := testcore.BlockSnapPodConfig(pvcNameList, bss.Image)
	podTmpl := podClient.MakePod(podconf)

	file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
	writerPod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Write random blob to pvc
	ddRes := bytes.NewBufferString("")
	if err := podClient.Exec(ctx, writerPod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
		return delFunc, err
	}
	log.Info("Writer pod: ", writerPod.Object.GetName())
	log.Debug(ddRes.String())

	originalHash := bytes.NewBufferString("")

	// Calculate hash sum of that file
	if err := podClient.Exec(ctx, writerPod.Object, []string{"sha512sum", file}, originalHash, os.Stderr, false); err != nil {
		return delFunc, err
	}
	log.Info("hash sum is: ", originalHash.String())

	// Get PVC for creating blockvolsnapshot
	gotPvc, err := pvcClient.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
	if err != nil {
		return delFunc, err
	}

	var createSnap volumesnapshot.Interface
	// Create Interface from PVC using gotPvc name
	if clients.SnapClientGA != nil {
		createSnap = clients.SnapClientGA.Create(ctx,
			&snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "snap0",
					GenerateName: "",
					Namespace:    gotPvc.Namespace,
				},
				Spec: snapv1.VolumeSnapshotSpec{
					Source: snapv1.VolumeSnapshotSource{
						PersistentVolumeClaimName: &gotPvc.Name,
					},
					VolumeSnapshotClassName: &bss.SnapClass,
				},
			})
		if createSnap.HasError() {
			return delFunc, createSnap.GetError()
		}

		// Wait for blockvolsnapshot to be created
		err := createSnap.WaitForRunning(ctx)
		if err != nil {
			return delFunc, err
		}
	} else if clients.SnapClientBeta != nil {
		createSnap = clients.SnapClientBeta.Create(ctx,
			&snapbeta.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "snap0",
					GenerateName: "",
					Namespace:    gotPvc.Namespace,
				},
				Spec: snapbeta.VolumeSnapshotSpec{
					Source: snapbeta.VolumeSnapshotSource{
						PersistentVolumeClaimName: &gotPvc.Name,
					},
					VolumeSnapshotClassName: &bss.SnapClass,
				},
			})
		if createSnap.HasError() {
			return delFunc, createSnap.GetError()
		}

		// Wait for blockvolsnapshot to be created
		err := createSnap.WaitForRunning(ctx)
		if err != nil {
			return delFunc, err
		}
	} else {
		return delFunc, fmt.Errorf("can't get alpha or beta blockvolsnapshot client")
	}

	// Create second PVC from blockvolsnapshot
	vcconf.SnapName = createSnap.Name()
	var mode v1.PersistentVolumeMode = pod.Block
	vcconf.VolumeMode = &mode

	vcconf.AccessModes = testcore.GetAccessMode(bss.AccessMode)

	log.Infof("Restoring from %s", vcconf.SnapName)
	volRestored := pvcClient.MakePVC(vcconf)
	pvcRestored := pvcClient.Create(ctx, volRestored)
	if pvcRestored.HasError() {
		return delFunc, pvcRestored.GetError()
	}
	if !firstConsumer {
		err := pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return delFunc, err
		}
	}

	// Create Pod, and attach PVC from blockvolsnapshot
	podRestored := testcore.BlockSnapPodConfig([]string{pvcRestored.Object.Name}, bss.Image)
	podRestored.VolumeMode = pod.Block
	podTmplRestored := podClient.MakePod(podRestored)

	writerPod = podClient.Create(ctx, podTmplRestored).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Check the data persistence
	writer := bytes.NewBufferString("")
	log.Info("Checker pod: ", writerPod.Object.GetName())

	// flush the buffer
	if err := podClient.Exec(ctx, writerPod.Object, []string{"blockdev", "--flushbufs", "/dev/data0"}, os.Stdout, os.Stderr, false); err != nil {
		return delFunc, err
	}
	// create folder
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "mkdir /data0"}, writer, os.Stderr, false); err != nil {
		return delFunc, err
	}
	// mount rawblock volume to created folder
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "mount /dev/data0 /data0"}, writer, os.Stderr, false); err != nil {
		return delFunc, err
	}

	newHash := bytes.NewBufferString("")
	pollErr := wait.PollImmediate(10*time.Second, 5*time.Minute,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pod wait polling")
				return true, fmt.Errorf("stopped waiting to be ready")
			default:
				break
			}

			// Check if file exists
			if err := podClient.Exec(ctx, writerPod.Object, []string{"ls", file}, os.Stdout, os.Stderr, false); err != nil {
				return false, nil
			}

			// Calculate hash sum of that file
			if err := podClient.Exec(ctx, writerPod.Object, []string{"sha512sum", file}, newHash, os.Stderr, false); err != nil {
				return false, err
			}

			return true, nil
		})

	if pollErr != nil {
		return delFunc, pollErr
	}

	log.Info("new hash sum is: ", newHash.String())

	if newHash.String() == originalHash.String() {
		log.Info("Hashes match")
	} else {
		return delFunc, fmt.Errorf("hashes don't match")
	}

	return delFunc, nil
}

// GetObservers returns all observers
func (*BlockSnapSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return common.GetAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics, blockvolsnapshot clients
func (bss *BlockSnapSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.SnapshotClassExists(bss.SnapClass); !ok {
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

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	snapGA, snapBeta, snErr := GetSnapshotClient(namespace, client)
	if snErr != nil {
		return nil, snErr
	}
	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      snapGA,
		SnapClientBeta:    snapBeta,
	}, nil
}

// GetNamespace returns block volsnap test suite namespace
func (*BlockSnapSuite) GetNamespace() string {
	return "block-volsnap-test"
}

// GetName returns block volsnap test suite name
func (bss *BlockSnapSuite) GetName() string {
	if bss.Description != "" {
		return bss.Description
	}
	return "BlockSnapSuite"
}

// Parameters returns formatted string of parameters
func (bss *BlockSnapSuite) Parameters() string {
	return fmt.Sprintf("{size: %s, accMode: %s}", bss.VolumeSize, bss.AccessMode)
}

// GetSnapshotClient returns blockvolsnapshot client
func GetSnapshotClient(namespace string, client *k8sclient.KubeClient) (*snapv1client.SnapshotClient, *snapbetaclient.SnapshotClient, error) {
	gaClient, snErr := client.CreateSnapshotGAClient(namespace)
	_, err := gaClient.Interface.List(context.Background(), metav1.ListOptions{})
	if err != nil || snErr != nil {
		betaClient, snErr := client.CreateSnapshotBetaClient(namespace)
		if snErr != nil {
			return nil, nil, snErr
		}
		return nil, betaClient, nil
	}
	return gaClient, nil, nil
}
