package replication

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/commonparams"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/blockvolsnapshot"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/fatih/color"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapbeta "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"time"
)

// ReplicationSuite is used to manage replication test suite
type ReplicationSuite struct {
	VolumeNumber int
	VolumeSize   string
	PodNumber    int
	SnapClass    string
	Image        string
}

// Run executes replication test suite
func (rs *ReplicationSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	rgClient := clients.RgClient

	if rs.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		rs.VolumeNumber = 5
	}
	if rs.PodNumber <= 0 {
		log.Info("Using default number of pods")
		rs.PodNumber = 1
	}
	if rs.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		rs.VolumeSize = "3Gi"
	}
	if rs.Image == "" {
		rs.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", rs.Image)
	}

	log.Infof("Creating %s pods, each with %s volumes", color.YellowString(strconv.Itoa(rs.PodNumber)),
		color.YellowString(strconv.Itoa(rs.VolumeNumber)))
	var allPvcNames []string
	for i := 0; i < rs.PodNumber; i++ {
		pvcNameList := make([]string, rs.VolumeNumber)
		for j := 0; j < rs.VolumeNumber; j++ {
			// Create PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, rs.VolumeSize, "", "")
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return delFunc, pvc.GetError()
			}

			pvcNameList[j] = pvc.Object.Name
			allPvcNames = append(allPvcNames, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "", rs.Image)
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
	log.Info("Creating a blockvolsnapshot on each of the volumes")
	var snapNameList []string
	if clients.SnapClientGA != nil {
		lenPvcList := len(allPvcNames)
		iters := lenPvcList / 10
		if lenPvcList%10 != 0 {
			iters++
		}
		for iter := 0; iter < iters; iter++ {
			initial := iter * 10
			final := initial + 10
			if final > lenPvcList {
				final = lenPvcList
			}
			for _, pvc := range allPvcNames[initial:final] {
				gotPvc, err := pvcClient.Interface.Get(ctx, pvc, metav1.GetOptions{})
				if err != nil {
					return delFunc, err
				}
				snapName := fmt.Sprintf("volsnap-%s", gotPvc.Name)
				snapNameList = append(snapNameList, snapName)
				createSnap := clients.SnapClientGA.Create(ctx,
					&snapv1.VolumeSnapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:         snapName,
							GenerateName: "",
							Namespace:    gotPvc.Namespace,
						},
						Spec: snapv1.VolumeSnapshotSpec{
							Source: snapv1.VolumeSnapshotSource{
								PersistentVolumeClaimName: &gotPvc.Name,
							},
							VolumeSnapshotClassName: &rs.SnapClass,
						},
					})
				if createSnap.HasError() {
					return delFunc, createSnap.GetError()
				}
			}
			snapReadyError := clients.SnapClientGA.WaitForAllToBeReady(ctx)
			if snapReadyError != nil {
				return delFunc, snapReadyError
			}
		}
	} else if clients.SnapClientBeta != nil {
		for _, pvc := range allPvcNames {
			gotPvc, err := pvcClient.Interface.Get(ctx, pvc, metav1.GetOptions{})
			if err != nil {
				return delFunc, err
			}
			snapName := fmt.Sprintf("volsnap-%s", gotPvc.Name)
			snapNameList = append(snapNameList, snapName)
			createSnap := clients.SnapClientBeta.Create(ctx,
				&snapbeta.VolumeSnapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:         snapName,
						GenerateName: "",
						Namespace:    gotPvc.Namespace,
					},
					Spec: snapbeta.VolumeSnapshotSpec{
						Source: snapbeta.VolumeSnapshotSource{
							PersistentVolumeClaimName: &gotPvc.Name,
						},
						VolumeSnapshotClassName: &rs.SnapClass,
					},
				})
			if createSnap.HasError() {
				return delFunc, createSnap.GetError()
			}
		}
		// Wait for blockvolsnapshot to be created
		snapReadyError := clients.SnapClientBeta.WaitForAllToBeReady(ctx)
		if snapReadyError != nil {
			return delFunc, snapReadyError
		}
	} else {
		return delFunc, fmt.Errorf("can't get alpha or beta blockvolsnapshot client")
	}

	log.Info("Creating new pods with replicated volumes mounted on them")
	for i := 0; i < rs.PodNumber; i++ {
		pvcNameList := make([]string, rs.VolumeNumber)
		for j := 0; j < rs.VolumeNumber; j++ {
			// Restore PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, rs.VolumeSize, "", "")
			vcconf.SnapName = snapNameList[j+(i*rs.VolumeNumber)]
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return delFunc, pvc.GetError()
			}
			pvcNameList[j] = pvc.Object.Name
		}
		// Create Pod, and attach restored PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "", rs.Image)
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}
	}
	readyErr = podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	// TODO: List PVCs once again, since here we can be sure that all annotations will be correctly set
	pvcList, err := pvcClient.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return delFunc, err
	}

	rgName := pvcList.Items[0].Annotations[commonparams.ReplicationGroupName]
	log.Infof("The replication group name from pvc is %s ", rgName)

	// Add remote RG deletion step to deletion callback function
	delFunc = func(_ func() error) func() error {
		return func() error {
			log.Info("Deleting local RG")
			rgObject := rgClient.Get(context.Background(), rgName)
			deletedLocalRG := rgClient.Delete(context.Background(), rgObject.Object)
			if deletedLocalRG.HasError() {
				log.Warnf("error when deleting local RG: %s", deletedLocalRG.GetError().Error())
			}

			log.Info("Sleeping for 1 minute...")
			time.Sleep(1 * time.Minute)

			return nil
		}
	}(nil)

	return delFunc, nil
}

// GetObservers returns all observers
func (rs *ReplicationSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return common.GetAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics, blockvolsnapshot clients
func (rs *ReplicationSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.SnapshotClassExists(rs.SnapClass); !ok {
		return nil, fmt.Errorf("snasphot class doesn't exist; error = %v", err)
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

	rgClient, rgErr := client.CreateRGClient()
	if rgErr != nil {
		return nil, rgErr
	}

	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
		SnapClientGA:      snapGA,
		SnapClientBeta:    snapBeta,
		RgClient:          rgClient,
	}, nil
}

// GetNamespace returns replication suite namespace
func (*ReplicationSuite) GetNamespace() string {
	return "replication-suite"
}

// GetName returns replication suite name
func (*ReplicationSuite) GetName() string {
	return "ReplicationSuite"
}

// Parameters returns formatted string of parameters
func (rs *ReplicationSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, volumeSize: %s}", rs.PodNumber, rs.VolumeNumber, rs.VolumeSize)
}
