package volsnap

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshot"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/blockvolsnapshot"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapbeta "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	"helm.sh/helm/v3/pkg/time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// DefaultSnapPrefix is blockvolsnapshot prefix
const DefaultSnapPrefix = "volsnap"

// SnapSuite is used to manage volsnap test suite
type SnapSuite struct {
	SnapAmount         int
	SnapClass          string
	VolumeSize         string
	Description        string
	CustomSnapName     string
	AccessModeOriginal string
	AccessModeRestored string
	Image              string
}

// Run executes volsnap test suite
func (ss *SnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	if ss.SnapAmount <= 0 {
		log.Info("Using default number of snapshots")
		ss.SnapAmount = 3
	}
	if ss.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		ss.VolumeSize = "3Gi"
	}
	if ss.Image == "" {
		ss.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", ss.Image)
	}

	result := validateCustomSnapName(ss.CustomSnapName, ss.SnapAmount)
	if result {
		log.Infof("using custom volsnap-name:%s", ss.CustomSnapName)
	} else {
		ss.CustomSnapName = ""
	}

	pvcClient := clients.PVCClient
	podClient := clients.PodClient

	snapvolname := ""
	snappodname := ""

	result1 := validateCustomSnapName(ss.CustomSnapName, ss.SnapAmount)
	if result1 {
		snapvolname = ss.CustomSnapName + "-pvc"
		snappodname = ss.CustomSnapName + "-pod"
		log.Infof("using custom volsnap-name:%s"+" and PVC name:%s"+" and POD name:%s", ss.CustomSnapName, snapvolname, snappodname)
	} else {
		snapvolname = ""
		snappodname = ""
	}

	firstConsumer, err := common.ShouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return delFunc, err
	}
	log.Info("Creating Snapshot pod")
	// Create first PVC
	var pvcNameList []string
	vcconf := testcore.VolumeCreationConfig(storageClass, ss.VolumeSize, snapvolname, ss.AccessModeOriginal)
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
	podconf := testcore.IoWritePodConfig(pvcNameList, snappodname, ss.Image)
	podTmpl := podClient.MakePod(podconf)

	file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
	sum := fmt.Sprintf("%s0/writer-%d.sha512", podconf.MountPath, 0)
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

	// Write hash sum of blob
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
		return delFunc, err
	}
	podClient.Delete(ctx, writerPod.Object).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Get PVC for creating blockvolsnapshot
	gotPvc, err := pvcClient.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
	if err != nil {
		return delFunc, err
	}

	snapPrefix := DefaultSnapPrefix

	if ss.CustomSnapName != "" {
		snapPrefix = ss.CustomSnapName
	}

	var snaps []volumesnapshot.Interface
	log.Infof("Creating %d snapshots", ss.SnapAmount)
	for i := 0; i < ss.SnapAmount; i++ {
		var createSnap volumesnapshot.Interface
		// Create Interface from PVC using gotPvc name
		if clients.SnapClientGA != nil {
			name := snapPrefix + strconv.Itoa(i)
			if i == 0 {
				name = snapPrefix
			}
			createSnap = clients.SnapClientGA.Create(ctx,
				&snapv1.VolumeSnapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:         name,
						GenerateName: "",
						Namespace:    gotPvc.Namespace,
					},
					Spec: snapv1.VolumeSnapshotSpec{
						Source: snapv1.VolumeSnapshotSource{
							PersistentVolumeClaimName: &gotPvc.Name,
						},
						VolumeSnapshotClassName: &ss.SnapClass,
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
			name := snapPrefix + strconv.Itoa(i)
			if i == 0 {
				name = snapPrefix
			}
			createSnap = clients.SnapClientBeta.Create(ctx,
				&snapbeta.VolumeSnapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:         name,
						GenerateName: "",
						Namespace:    gotPvc.Namespace,
					},
					Spec: snapbeta.VolumeSnapshotSpec{
						Source: snapbeta.VolumeSnapshotSource{
							PersistentVolumeClaimName: &gotPvc.Name,
						},
						VolumeSnapshotClassName: &ss.SnapClass,
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
		snaps = append(snaps, createSnap)
	}
	// Create second PVC from blockvolsnapshot
	var pvcFromSnapNameList []string
	rand.Seed(time.Now().Unix())
	n := rand.Int() % len(snaps) // #nosec
	vcconf.SnapName = snaps[n].Name()
	log.Infof("Restoring from %s", vcconf.SnapName)
	vcconf.Name = vcconf.SnapName + "-restore"
	accessModeRestoredVolume := testcore.GetAccessMode(ss.AccessModeRestored)
	vcconf.AccessModes = accessModeRestoredVolume
	log.Infof("Creating pvc %s", vcconf.Name)
	volRestored := pvcClient.MakePVC(vcconf)
	pvcRestored := pvcClient.Create(ctx, volRestored)
	if pvc.HasError() {
		return delFunc, pvc.GetError()
	}
	pvcFromSnapNameList = append(pvcFromSnapNameList, pvcRestored.Object.Name)
	if !firstConsumer {
		err = pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return delFunc, err
		}
	}

	// Create Pod, and attach PVC from blockvolsnapshot

	podRestored := testcore.IoWritePodConfig(pvcFromSnapNameList, pvcRestored.Object.Name+"-pod", ss.Image)
	podTmplRestored := podClient.MakePod(podRestored)

	writerPod = podClient.Create(ctx, podTmplRestored).Sync(ctx)
	if writerPod.HasError() {
		return delFunc, writerPod.GetError()
	}

	// Check if hash sum is correct
	sum = fmt.Sprintf("%s0/writer-%d.sha512", podRestored.MountPath, 0)
	writer := bytes.NewBufferString("")
	log.Info("Checker pod: ", writerPod.Object.GetName())
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
		return delFunc, err
	}

	if strings.Contains(writer.String(), "OK") {
		log.Info("Hashes match")
	} else {
		return delFunc, fmt.Errorf("hashes don't match")
	}
	return delFunc, nil
}

func validateCustomSnapName(name string, snapshotAmount int) bool {
	// If no. of snapshots is only 1 then we will take custom name else no.
	if snapshotAmount == 1 && len(name) != 0 {
		return true
	}
	// we will use custom name only if number of snapshots is 1 else we will discard custom name
	return false
}

// GetObservers returns all observers
func (*SnapSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return common.GetAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics, snapsnot clients
func (ss *SnapSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.SnapshotClassExists(ss.SnapClass); !ok {
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

	snapGA, snapBeta, snErr := blockvolsnapshot.GetSnapshotClient(namespace, client)
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

// GetNamespace returns volsnap suite namespaces
func (*SnapSuite) GetNamespace() string {
	return "volsnap-test"
}

// GetName returns volsnap suite name
func (ss *SnapSuite) GetName() string {
	if ss.Description != "" {
		return ss.Description
	}
	return "SnapSuite"
}

// Parameters returns formatted string of paramters
func (ss *SnapSuite) Parameters() string {
	return fmt.Sprintf("{snapshots: %d, volumeSize; %s}", ss.SnapAmount, ss.VolumeSize)
}
