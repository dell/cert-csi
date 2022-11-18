package suites

import (
	"bytes"
	"cert-csi/pkg/k8sclient"
	"cert-csi/pkg/k8sclient/resources/commonparams"
	"cert-csi/pkg/k8sclient/resources/node"
	"cert-csi/pkg/k8sclient/resources/pod"
	"cert-csi/pkg/k8sclient/resources/pv"
	"cert-csi/pkg/k8sclient/resources/pvc"
	"cert-csi/pkg/k8sclient/resources/replicationgroup"
	"cert-csi/pkg/k8sclient/resources/sc"
	"cert-csi/pkg/k8sclient/resources/volumesnapshot"
	"cert-csi/pkg/observer"
	"cert-csi/pkg/testcore"
	"cert-csi/pkg/utils"
	"context"
	"errors"
	"fmt"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	snapv1client "cert-csi/pkg/k8sclient/resources/volumesnapshot/v1"
	snapbetaclient "cert-csi/pkg/k8sclient/resources/volumesnapshot/v1beta1"

	"math/rand"
	"time"

	"github.com/fatih/color"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapbeta "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultSnapPrefix       = "snap"
	ControllerLogsSleepTime = 20
	maxRetryCount           = 30
)

type VolumeCreationSuite struct {
	VolumeNumber int
	Description  string
	VolumeSize   string
	CustomName   string
	AccessMode   string
	RawBlock     bool
}

func (vcs *VolumeCreationSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	if vcs.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		vcs.VolumeNumber = 10
	}
	if vcs.VolumeSize == "" {
		log.Info("Using default volume size")
		vcs.VolumeSize = "3Gi"
	}
	result := validateCustomName(vcs.CustomName, vcs.VolumeNumber)
	if result {
		log.Infof("using custom pvc-name:%s", vcs.CustomName)
	} else {
		vcs.CustomName = ""
	}

	log.Infof("Creating %s volumes with size:%s", color.YellowString(strconv.Itoa(vcs.VolumeNumber)),
		color.YellowString(vcs.VolumeSize))
	pvcClient := clients.PVCClient

	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return err, delFunc
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
		return createErr, delFunc
	}

	// Wait until all PVCs will be bound
	if !firstConsumer {
		boundErr := pvcClient.WaitForAllToBeBound(ctx)
		if boundErr != nil {
			return boundErr, delFunc
		}
	}

	return nil, delFunc
}

func shouldWaitForFirstConsumer(ctx context.Context, storageClass string, pvcClient *pvc.Client) (bool, error) {
	s, err := pvcClient.ClientSet.StorageV1().StorageClasses().Get(ctx, storageClass, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return *s.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer, nil
}

func (vcs *VolumeCreationSuite) GetName() string {
	if vcs.Description != "" {
		return vcs.Description
	}
	return "VolumeCreationSuite"
}

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

func validateCustomName(name string, volumes int) bool {
	// If no. of volumes is only 1 then we will take custom name else no.
	if volumes == 1 && len(name) != 0 {
		return true
	}
	// we will use custom name only if number of volumes is 1 else we will discard custom name
	return false
}

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

func (*VolumeCreationSuite) GetNamespace() string {
	return "vcs-test"
}

func (vcs *VolumeCreationSuite) Parameters() string {
	return fmt.Sprintf("{number: %d, size: %s, raw-block: %s}", vcs.VolumeNumber, vcs.VolumeSize, strconv.FormatBool(vcs.RawBlock))
}

type ProvisioningSuite struct {
	VolumeNumber  int
	VolumeSize    string
	PodCustomName string
	Description   string
	PodNumber     int
	RawBlock      bool
	VolAccessMode string
	ROFlag        bool
}

func (ps *ProvisioningSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	if ps.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		ps.VolumeNumber = 5
	}
	if ps.PodNumber <= 0 {
		log.Info("Using default number of pods")
		ps.PodNumber = 1
	}
	if ps.VolumeSize == "" {
		log.Info("Using default volume size 3Gi")
		ps.VolumeSize = "3Gi"
	}
	ps.validateCustomPodName()

	log.Infof("Creating %s pods, each with %s volumes", color.YellowString(strconv.Itoa(ps.PodNumber)),
		color.YellowString(strconv.Itoa(ps.VolumeNumber)))

	for i := 0; i < ps.PodNumber; i++ {
		var pvcNameList []string
		for j := 0; j < ps.VolumeNumber; j++ {
			// Create PVCs
			var volumeName string
			if ps.PodCustomName != "" {
				volumeName = ps.PodCustomName + "-pvc-" + strconv.Itoa(j)
			}
			vcconf := testcore.VolumeCreationConfig(storageClass, ps.VolumeSize, volumeName, ps.VolAccessMode)
			if ps.RawBlock {
				log.Info(color.YellowString("Creating Raw Block Volumes"))
				var mode v1.PersistentVolumeMode = pvc.Block
				vcconf.VolumeMode = &mode
			}
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return pvc.GetError(), delFunc
			}

			pvcNameList = append(pvcNameList, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, ps.PodCustomName)
		if ps.RawBlock {
			podconf.VolumeMode = pod.Block
		}
		if ps.ROFlag {
			podconf.ReadOnlyFlag = true
		}
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return pod.GetError(), delFunc
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}

	return nil, delFunc
}

func (*ProvisioningSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

func (*ProvisioningSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
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

func (*ProvisioningSuite) GetNamespace() string {
	return "prov-test"
}

func (ps *ProvisioningSuite) GetName() string {
	if ps.Description != "" {
		return ps.Description
	}
	return "ProvisioningSuite"
}

func (ps *ProvisioningSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, volumeSize: %s}", ps.PodNumber, ps.VolumeNumber, ps.VolumeSize)
}

func (ps *ProvisioningSuite) validateCustomPodName() {
	// If no. of pods is only 1 then we will take custom name else generated name will be used.
	if ps.PodNumber == 1 && len(ps.PodCustomName) != 0 {
		logrus.Infof("using custom pod-name:%s", ps.PodCustomName)
	} else {
		// we will use custom name only if number of volumes is 1 else we will discard custom name
		ps.PodCustomName = ""
	}
}

type RemoteReplicationProvisioningSuite struct {
	VolumeNumber     int
	VolumeSize       string
	Description      string
	VolAccessMode    string
	RemoteConfigPath string
	NoFailover       bool
}

func (rrps *RemoteReplicationProvisioningSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	pvClient := clients.PersistentVolumeClient
	scClient := clients.SCClient
	rgClient := clients.RgClient
	storClass, err := scClient.Interface.Get(ctx, storageClass, metav1.GetOptions{})
	if err != nil {
		return err, delFunc
	}
	isSingle := false
	if storClass.Parameters["replication.storage.dell.com/remoteClusterID"] == "self" {
		isSingle = true
	}
	var (
		remotePVCObject  v1.PersistentVolumeClaim
		remotePVClient   *pv.Client
		remoteRGClient   *replicationgroup.Client
		remoteKubeClient *k8sclient.KubeClient
	)

	if rrps.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		rrps.VolumeNumber = 1
	}
	if rrps.VolumeSize == "" {
		log.Info("Using default volume size 3Gi")
		rrps.VolumeSize = "3Gi"
	}

	if rrps.RemoteConfigPath != "" {
		// Loading config
		remoteConfig, err := k8sclient.GetConfig(rrps.RemoteConfigPath)
		if err != nil {
			log.Error(err)
			return err, nil
		}
		// Connecting to host and creating new Kubernetes Client
		remoteKubeClient, err = k8sclient.NewRemoteKubeClient(remoteConfig, pvClient.Timeout)
		if err != nil {
			log.Errorf("Couldn't create new Remote kubernetes client. Error = %v", err)
			return err, nil
		}
		remotePVClient, err = remoteKubeClient.CreatePVClient()
		if err != nil {
			return err, nil
		}
		remoteRGClient, err = remoteKubeClient.CreateRGClient()
		if err != nil {
			return err, nil
		}

		log.Info("Created remote kube client")
	} else {
		remotePVClient = pvClient
		remoteRGClient = rgClient
	}

	scObject, scErr := scClient.Interface.Get(ctx, storageClass, metav1.GetOptions{})
	if scErr != nil {
		return scErr, nil
	}

	rpEnabled := scObject.Parameters[sc.IsReplicationEnabled]
	if rpEnabled != "true" {
		return fmt.Errorf("replication is not enabled on this storage class and please provide valid sc"), nil
	}

	log.Infof("Creating %s volumes", color.YellowString(strconv.Itoa(rrps.VolumeNumber)))
	var pvcNames []string
	for i := 0; i < rrps.VolumeNumber; i++ {
		// Create PVCs
		var volumeName string
		vcconf := testcore.VolumeCreationConfig(storageClass, rrps.VolumeSize, volumeName, rrps.VolAccessMode)
		volTmpl := pvcClient.MakePVC(vcconf)
		pvc := pvcClient.Create(ctx, volTmpl)
		if pvc.HasError() {
			return pvc.GetError(), delFunc
		}
		pvcNames = append(pvcNames, pvc.Object.Name)
	}

	err = pvcClient.WaitForAllToBeBound(ctx)
	if err != nil {
		return err, delFunc
	}

	var pvNames []string
	// We can get actual pv names only after all pvc are bound
	pvcList, err := pvcClient.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err, delFunc
	}
	for _, p := range pvcList.Items {
		pvNames = append(pvNames, p.Spec.VolumeName)
	}

	log.Info("Creating pod for each volume")
	for _, name := range pvcNames {
		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig([]string{name}, "")
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl).Sync(ctx)
		if pod.HasError() {
			return pod.GetError(), delFunc
		}

		// write data to files and calculate checksum.
		file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
		sum := fmt.Sprintf("%s0/writer-%d.sha512", podconf.MountPath, 0)

		// Write random blob to pvc
		ddRes := bytes.NewBufferString("")
		if err := podClient.Exec(ctx, pod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
			return err, delFunc
		}

		log.Info("Writer pod: ", pod.Object.GetName())
		log.Debug(ddRes.String())
		log.Info("Written the values successfully ", ddRes)
		log.Info(ddRes.String())

		// Write hash sum of blob
		if err := podClient.Exec(ctx, pod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return err, delFunc
		}
		log.Info("Checksum value: ", sum)

		// sync to be sure
		if err := podClient.Exec(ctx, pod.Object, []string{"/bin/bash", "-c", "sync " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return err, delFunc
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}

	log.Infof("Checking annotations for all PVC's")

	// Wait until to get annotations for all PVCs

	boundErr := pvcClient.CheckAnnotationsForVolumes(ctx, scObject)
	if boundErr != nil {
		return boundErr, delFunc
	}

	log.Infof("Successfully checked annotations On PVC")

	log.Infof("Checking annotation on pv's")
	for _, pvName := range pvNames {
		// Check on local cluster
		pvObject, pvErr := pvClient.Interface.Get(ctx, pvName, metav1.GetOptions{})
		if pvErr != nil {
			return pvErr, delFunc
		}
		err := pvClient.CheckReplicationAnnotationsForPV(ctx, pvObject)
		if err != nil {
			return fmt.Errorf("replication Annotations and Labels are not added for PV %s and %s", pvName, err), delFunc
		}

		// Check on remote cluster
		if !isSingle {
			remotePvName := pvName
			remotePVObject, remotePVErr := remotePVClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return remotePVErr, delFunc
			}
			err = remotePVClient.CheckReplicationAnnotationsForRemotePV(ctx, remotePVObject)
			if err != nil {
				return fmt.Errorf("replication Annotations and Labels are not added for Remote PV %s and %s", remotePvName, err), delFunc
			}
		} else {
			remotePvName := "replicated-" + pvName
			remotePVObject, remotePVErr := pvClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return remotePVErr, delFunc
			}
			err = pvClient.CheckReplicationAnnotationsForRemotePV(ctx, remotePVObject)
			if err != nil {
				return fmt.Errorf("replication Annotations and Labels are not added for Remote PV %s and %s", remotePvName, err), delFunc
			}
		}
	}
	log.Infof("Successfully checked annotations on local and remote pv")

	// List PVCs once again, since here we can be sure that all annotations will be correctly set
	pvcList, err = pvcClient.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err, delFunc
	}

	rgName := pvcList.Items[0].Annotations[commonparams.ReplicationGroupName]
	log.Infof("The replication group name from pvc is %s ", rgName)

	// Add remote RG deletion step to deletion callback function
	delFunc = func(f func() error) func() error {
		return func() error {
			log.Info("Deleting local RG")
			rgObject := rgClient.Get(context.Background(), rgName)
			deletedLocalRG := rgClient.Delete(context.Background(), rgObject.Object)
			if deletedLocalRG.HasError() {
				log.Warnf("error when deleting local RG: %s", deletedLocalRG.GetError().Error())
			}

			log.Info("Deleting remote RG")
			remoteRgObject := remoteRGClient.Get(context.Background(), rgName)
			deletedRemoteRG := remoteRGClient.Delete(context.Background(), remoteRgObject.Object)
			if deletedRemoteRG.HasError() {
				log.Warnf("error when deleting remote RG: %s", deletedRemoteRG.GetError().Error())
			}

			log.Info("Sleeping for 1 minute...") // TODO: maybe do some polling instead, idk rn
			// Sleeping for 1 minutes
			time.Sleep(1 * time.Minute)

			return nil
		}
	}(nil)

	log.Infof("Checking Annotations and Labels on ReplicationGroup")

	rgObject := rgClient.Get(ctx, rgName)
	var remoteRgName string
	if !isSingle {
		remoteRgName = rgName
	} else {
		remoteRgName = "replicated-" + rgName
	}

	remoteRgObject := remoteRGClient.Get(ctx, remoteRgName)

	if rgObject.Object.Annotations["replication.storage.dell.com/remoteReplicationGroupName"] !=
		remoteRgObject.Object.Labels["replication.storage.dell.com/remoteReplicationGroupName"] &&
		rgObject.Object.Annotations[commonparams.RemoteClusterID] != scObject.Parameters[sc.RemoteClusterID] {
		return fmt.Errorf("expected Annotations are not added to the replication group %s", rgName), delFunc
	}

	if rgObject.Object.Labels[commonparams.RemoteClusterID] != scObject.Parameters[sc.RemoteClusterID] {
		return fmt.Errorf("expected Labels are not added to the replication group %s", rgName), delFunc
	}
	log.Infof("Successfully Checked Annotations and Labels on ReplicationGroup")

	// Return if failover is not requested
	if rrps.NoFailover {
		return nil, delFunc
	}

	// Failover to the target site
	log.Infof("Executing failover action on ReplicationGroup %s", remoteRgName)
	actionErr := rgObject.ExecuteAction(ctx, "FAILOVER_REMOTE")
	if actionErr != nil {
		return actionErr, delFunc
	}

	log.Infof("Executing reprotect action on ReplicationGroup %s", remoteRgName)
	remoteRgObject = remoteRGClient.Get(ctx, remoteRgName)
	actionErr = remoteRgObject.ExecuteAction(ctx, "REPROTECT_LOCAL")
	if actionErr != nil {
		return actionErr, delFunc
	}

	log.Infof("Creating pvc and pods on remote cluster")
	if !isSingle {
		ns, err := remoteKubeClient.CreateNamespace(ctx, clients.PVCClient.Namespace)
		if err != nil {
			return err, delFunc
		}
		remoteNamespace := ns.Name
		remotePVCClient, err := remoteKubeClient.CreatePVCClient(ns.Name)
		if err != nil {
			return err, delFunc
		}
		remotePodClient, err := remoteKubeClient.CreatePodClient(ns.Name)
		if err != nil {
			return err, delFunc
		}

		var pvcNameList []string

		for _, pvc := range pvcList.Items {
			log.Infof("Creating remote pvc %s", pvc.Name)

			remotePvName := pvc.Spec.VolumeName
			remotePVObject, remotePVErr := remotePVClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return remotePVErr, delFunc
			}
			pvcName := remotePVObject.Annotations["replication.storage.dell.com/remotePVC"]
			pvcNameList = append(pvcNameList, pvcName)

			remotePVCObject = pvcClient.CreatePVCObject(ctx, remotePVObject, remoteNamespace)
			remotePvc := remotePVCClient.Create(ctx, &remotePVCObject)
			if remotePvc.HasError() {
				return remotePvc.GetError(), delFunc
			}

		}
		for _, pvcName := range pvcNameList {
			log.Infof("Verifying data from pvc %s", pvcName)

			podConf := testcore.ProvisioningPodConfig([]string{pvcName}, "")
			podTemp := remotePodClient.MakePod(podConf)

			remotePod := remotePodClient.Create(ctx, podTemp).Sync(ctx)
			if remotePod.HasError() {
				return remotePod.GetError(), delFunc
			}

			sum := fmt.Sprintf("%s0/writer-%d.sha512", podConf.MountPath, 0)
			writer := bytes.NewBufferString(" ")
			log.Info("Checker pod: ", remotePod.Object.GetName())
			if err := remotePodClient.Exec(ctx, remotePod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
				log.Error("Failed to verify hash")
				return err, delFunc
			}

			if strings.Contains(writer.String(), "OK") {
				log.Info("Hashes match")
			} else {
				return fmt.Errorf("hashes don't match"), delFunc
			}
		}
	} else {
		ns, err := clients.KubeClient.CreateNamespace(ctx, "replicated-"+clients.PVCClient.Namespace)
		if err != nil {
			return err, delFunc
		}

		remoteNamespace := ns.Name
		remotePVCClient, err := clients.KubeClient.CreatePVCClient(ns.Name)
		if err != nil {
			return err, delFunc
		}
		remotePodClient, err := clients.KubeClient.CreatePodClient(ns.Name)
		if err != nil {
			return err, delFunc
		}

		var pvcNameList []string

		for _, pvc := range pvcList.Items {
			log.Infof("Creating remote pvc %s", pvc.Name)

			remotePvName := "replicated" + pvc.Spec.VolumeName
			remotePVObject, remotePVErr := pvClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return remotePVErr, delFunc
			}
			pvcName := remotePVObject.Annotations["replication.storage.dell.com/remotePVC"]
			pvcNameList = append(pvcNameList, pvcName)

			remotePVCObject = pvcClient.CreatePVCObject(ctx, remotePVObject, remoteNamespace)
			remotePvc := remotePVCClient.Create(ctx, &remotePVCObject)
			if remotePvc.HasError() {
				return remotePvc.GetError(), delFunc
			}

		}
		for _, pvcName := range pvcNameList {
			log.Infof("Verifying data from pvc %s", pvcName)

			podConf := testcore.ProvisioningPodConfig([]string{pvcName}, "")
			podTemp := remotePodClient.MakePod(podConf)

			remotePod := remotePodClient.Create(ctx, podTemp).Sync(ctx)
			if remotePod.HasError() {
				return remotePod.GetError(), delFunc
			}

			sum := fmt.Sprintf("%s0/writer-%d.sha512", podConf.MountPath, 0)
			writer := bytes.NewBufferString(" ")
			log.Info("Checker pod: ", remotePod.Object.GetName())
			if err := remotePodClient.Exec(ctx, remotePod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
				log.Error("Failed to verify hash")
				return err, delFunc
			}

			if strings.Contains(writer.String(), "OK") {
				log.Info("Hashes match")
			} else {
				return fmt.Errorf("hashes don't match"), delFunc
			}
		}

	}
	// Initializing deletion callback function
	delFunc = func(f func() error) func() error {
		return func() error {
			err := f()
			if err != nil {
				return err
			}
			if !isSingle {
				log.Infof("Deleting remote namespace %s", clients.PVCClient.Namespace)
				err = remoteKubeClient.DeleteNamespace(context.Background(), clients.PVCClient.Namespace)
				if err != nil {
					log.Warnf("error deleting remote namespace: %s", err.Error())
				}
			} else {
				log.Infof("Deleting remote namespace %s", "replicated-"+clients.PVCClient.Namespace)
				err = remoteKubeClient.DeleteNamespace(context.Background(), "replicated-"+clients.PVCClient.Namespace)
				if err != nil {
					log.Warnf("error deleting remote namespace: %s", err.Error())
				}
			}

			return nil
		}
	}(delFunc)

	// failover back to source side to ease deletion of resources
	remoteRgObject = remoteRGClient.Get(ctx, remoteRgName)
	log.Infof("Executing failover action using remote RG %s", remoteRgName)
	actionErr = remoteRgObject.ExecuteAction(ctx, "FAILOVER_REMOTE")
	if actionErr != nil {
		return actionErr, delFunc
	}

	log.Infof("Executing reprotect action using local RG %s", remoteRgName)
	rgObject = rgClient.Get(ctx, remoteRgName)
	actionErr = rgObject.ExecuteAction(ctx, "REPROTECT_LOCAL")
	if actionErr != nil {
		return actionErr, delFunc
	}

	return nil, delFunc
}

func (*RemoteReplicationProvisioningSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

func (*RemoteReplicationProvisioningSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
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
	scClient, scErr := client.CreateSCClient()
	if scErr != nil {
		return nil, scErr
	}
	rgClient, rgErr := client.CreateRGClient()
	if rgErr != nil {
		return nil, rgErr
	}
	return &k8sclient.Clients{
		PVCClient:              pvcClient,
		PodClient:              podClient,
		VaClient:               vaClient,
		MetricsClient:          metricsClient,
		PersistentVolumeClient: pvClient,
		SCClient:               scClient,
		RgClient:               rgClient,
		KubeClient:             client,
	}, nil
}

func (*RemoteReplicationProvisioningSuite) GetNamespace() string {
	return "repl-prov-test"
}

func (rrps *RemoteReplicationProvisioningSuite) GetName() string {
	if rrps.Description != "" {
		return rrps.Description
	}
	return "RemoteReplicationProvisioningSuite"
}
func (rrps *RemoteReplicationProvisioningSuite) Parameters() string {
	return fmt.Sprintf("{volumes: %d, volumeSize: %s, remoteConfig: %s}", rrps.VolumeNumber, rrps.VolumeSize, rrps.RemoteConfigPath)
}

type ScalingSuite struct {
	ReplicaNumber    int
	VolumeNumber     int
	GradualScaleDown bool
	PodPolicy        string
	VolumeSize       string
}

func (ss *ScalingSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
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

	stsconf := testcore.ScalingStsConfig(storageClass, ss.VolumeSize, ss.VolumeNumber, ss.PodPolicy)
	stsTmpl := stsClient.MakeStatefulSet(stsconf)

	// Creating new statefulset
	sts := stsClient.Create(ctx, stsTmpl)
	if sts.HasError() {
		return sts.GetError(), delFunc
	}

	sts = sts.Sync(ctx)
	if sts.HasError() {
		return sts.GetError(), delFunc
	}

	// Scaling to needed number of replicas
	sts = stsClient.Scale(ctx, sts.Set, int32(ss.ReplicaNumber))
	if sts.HasError() {
		return sts.GetError(), delFunc
	}
	sts.Sync(ctx)

	// Scaling to zero
	if !ss.GradualScaleDown {
		sts := stsClient.Scale(ctx, sts.Set, 0)
		if sts.HasError() {
			return sts.GetError(), delFunc
		}
		sts.Sync(ctx)
	} else {
		log.Info("Gradually scaling down sts")
		for i := ss.ReplicaNumber - 1; i >= 0; i-- {
			sts := stsClient.Scale(ctx, sts.Set, int32(i))
			if sts.HasError() {
				return sts.GetError(), delFunc
			}
			sts.Sync(ctx)
		}
	}

	return nil, delFunc
}

func (ss *ScalingSuite) GetName() string {
	return "ScalingSuite"
}

func (ss *ScalingSuite) Parameters() string {
	return fmt.Sprintf("{replicas: %d, volumes: %d, volumeSize: %s}", ss.ReplicaNumber, ss.VolumeNumber, ss.VolumeSize)
}

func (ss *ScalingSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (ss *ScalingSuite) GetNamespace() string {
	return "scale-test"
}

type VolumeIoSuite struct {
	VolumeNumber int
	VolumeSize   string
	ChainNumber  int
	ChainLength  int
}

func (vis *VolumeIoSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
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

	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return err, delFunc
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
			return pvc.GetError(), delFunc
		}

		pvcNameList = append(pvcNameList, pvc.Object.Name)

		if !firstConsumer {
			err := pvcClient.WaitForAllToBeBound(errCtx)
			if err != nil {
				return err, delFunc
			}
		}

		gotPvc, err := pvcClient.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
		if err != nil {
			return err, delFunc
		}

		pvName := gotPvc.Spec.VolumeName
		// Create Pod, and attach PVC
		podconf := testcore.IoWritePodConfig(pvcNameList, "")
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

	return errs.Wait(), delFunc
}

func (*VolumeIoSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (*VolumeIoSuite) GetNamespace() string {
	return "volumeio-test"
}

func (*VolumeIoSuite) GetName() string {
	return "VolumeIoSuite"
}

func (vis *VolumeIoSuite) Parameters() string {
	return fmt.Sprintf("{volumes: %d, volumeSize: %s chains: %d-%d}", vis.VolumeNumber, vis.VolumeSize,
		vis.ChainNumber, vis.ChainLength)
}

type VolumeGroupSnapSuite struct {
	SnapClass       string
	VolumeSize      string
	AccessMode      string
	VolumeGroupName string
	VolumeLabel     string
	ReclaimPolicy   string
	VolumeNumber    int
	Driver          string
}

func (vgs *VolumeGroupSnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	var namespace string

	if vgs.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		vgs.VolumeSize = "3Gi"
	}

	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	vgsClient := clients.VgsClient

	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return err, delFunc
	}

	// Create PVCs
	var pvcNameList []string
	vcconf := testcore.VolumeCreationConfig(storageClass, vgs.VolumeSize, "", vgs.AccessMode)
	vcconf.Labels = map[string]string{"volume-group": vgs.VolumeLabel}

	for i := 0; i <= vgs.VolumeNumber; i++ {
		volTmpl := pvcClient.MakePVC(vcconf)
		pvc := pvcClient.Create(ctx, volTmpl)
		if pvc.HasError() {
			return pvc.GetError(), delFunc
		}
		pvcNameList = append(pvcNameList, pvc.Object.Name)
		// we will take namespace from the PVC created, As the namespace is dynamically generated
		if namespace == "" {
			namespace = pvc.Object.Namespace
		}
		if !firstConsumer {
			err := pvcClient.WaitForAllToBeBound(ctx)
			if err != nil {
				return err, delFunc
			}
		}
	}

	// Create Pod, and attach PVC
	podconf := testcore.IoWritePodConfig(pvcNameList, "")
	podTmpl := podClient.MakePod(podconf)

	writerPod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if writerPod.HasError() {
		return writerPod.GetError(), delFunc
	}

	// create volume group snapshot using CRD.
	vgsConfig := testcore.VolumeGroupSnapConfig(vgs.VolumeGroupName, vgs.Driver, vgs.ReclaimPolicy, vgs.SnapClass, vgs.VolumeLabel, namespace)
	vgsTemplate := vgsClient.MakeVGS(vgsConfig)
	vg := vgsClient.Create(ctx, vgsTemplate)
	if vg.HasError() {
		return vg.GetError(), delFunc
	}

	// Poll until the status is complete
	err = vgsClient.WaitForComplete(ctx, vgsConfig.Name, vgsConfig.Namespace)
	if err != nil {
		return err, delFunc
	}
	return nil, delFunc
}

func (*VolumeGroupSnapSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

	return &k8sclient.Clients{
		PVCClient: pvcClient,
		PodClient: podClient,
		VaClient:  vaClient,
		VgsClient: vgsClient,
	}, nil
}

func (*VolumeGroupSnapSuite) GetNamespace() string {
	return "vgs-snap-test"
}

func (vgs *VolumeGroupSnapSuite) GetName() string {
	return "VolumeGroupSnapSuite"
}

func (vgs *VolumeGroupSnapSuite) Parameters() string {
	return fmt.Sprintf("{volumes: %d, volumeSize: %s}", vgs.VolumeNumber, vgs.VolumeSize)
}

// end of VGS

type SnapSuite struct {
	SnapAmount         int
	SnapClass          string
	VolumeSize         string
	Description        string
	CustomSnapName     string
	AccessModeOriginal string
	AccessModeRestored string
}

func (vis *SnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	if vis.SnapAmount <= 0 {
		log.Info("Using default number of snapshots")
		vis.SnapAmount = 3
	}
	if vis.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		vis.VolumeSize = "3Gi"
	}

	result := validateCustomSnapName(vis.CustomSnapName, vis.SnapAmount)
	if result {
		log.Infof("using custom snap-name:%s", vis.CustomSnapName)
	} else {
		vis.CustomSnapName = ""
	}

	pvcClient := clients.PVCClient
	podClient := clients.PodClient

	snapvolname := ""
	snappodname := ""

	result1 := validateCustomSnapName(vis.CustomSnapName, vis.SnapAmount)
	if result1 {
		snapvolname = vis.CustomSnapName + "-pvc"
		snappodname = vis.CustomSnapName + "-pod"
		log.Infof("using custom snap-name:%s"+" and PVC name:%s"+" and POD name:%s", vis.CustomSnapName, snapvolname, snappodname)
	} else {
		snapvolname = ""
		snappodname = ""
	}

	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return err, delFunc
	}
	log.Info("Creating Snapshot pod")
	// Create first PVC
	var pvcNameList []string
	vcconf := testcore.VolumeCreationConfig(storageClass, vis.VolumeSize, snapvolname, vis.AccessModeOriginal)
	volTmpl := pvcClient.MakePVC(vcconf)
	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return pvc.GetError(), delFunc
	}
	pvcNameList = append(pvcNameList, pvc.Object.Name)
	if !firstConsumer {
		err := pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return err, delFunc
		}
	}

	// Create Pod, and attach PVC
	podconf := testcore.IoWritePodConfig(pvcNameList, snappodname)
	podTmpl := podClient.MakePod(podconf)

	file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
	sum := fmt.Sprintf("%s0/writer-%d.sha512", podconf.MountPath, 0)
	writerPod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if writerPod.HasError() {
		return writerPod.GetError(), delFunc
	}

	// Write random blob to pvc
	ddRes := bytes.NewBufferString("")
	if err := podClient.Exec(ctx, writerPod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
		return err, delFunc
	}
	log.Info("Writer pod: ", writerPod.Object.GetName())
	log.Debug(ddRes.String())

	// Write hash sum of blob
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
		return err, delFunc
	}
	podClient.Delete(ctx, writerPod.Object).Sync(ctx)
	if writerPod.HasError() {
		return writerPod.GetError(), delFunc
	}

	// Get PVC for creating snapshot
	gotPvc, err := pvcClient.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
	if err != nil {
		return err, delFunc
	}

	snapPrefix := DefaultSnapPrefix

	if vis.CustomSnapName != "" {
		snapPrefix = vis.CustomSnapName
	}

	var snaps []volumesnapshot.Interface
	log.Infof("Creating %d snapshots", vis.SnapAmount)
	for i := 0; i < vis.SnapAmount; i++ {
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
						VolumeSnapshotClassName: &vis.SnapClass,
					},
				})
			if createSnap.HasError() {
				return createSnap.GetError(), delFunc
			}

			// Wait for snapshot to be created
			err := createSnap.WaitForRunning(ctx)
			if err != nil {
				return err, delFunc
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
						VolumeSnapshotClassName: &vis.SnapClass,
					},
				})
			if createSnap.HasError() {
				return createSnap.GetError(), delFunc
			}

			// Wait for snapshot to be created
			err := createSnap.WaitForRunning(ctx)
			if err != nil {
				return err, delFunc
			}
		} else {
			return fmt.Errorf("can't get alpha or beta snapshot client"), delFunc
		}
		snaps = append(snaps, createSnap)
	}
	// Create second PVC from snapshot
	var pvcFromSnapNameList []string
	rand.Seed(time.Now().Unix())
	n := rand.Int() % len(snaps) // #nosec
	vcconf.SnapName = snaps[n].Name()
	log.Infof("Restoring from %s", vcconf.SnapName)
	vcconf.Name = vcconf.SnapName + "-restore"
	accessModeRestoredVolume := testcore.GetAccessMode(vis.AccessModeRestored)
	vcconf.AccessModes = accessModeRestoredVolume
	log.Infof("Creating pvc %s", vcconf.Name)
	volRestored := pvcClient.MakePVC(vcconf)
	pvcRestored := pvcClient.Create(ctx, volRestored)
	if pvc.HasError() {
		return pvc.GetError(), delFunc
	}
	pvcFromSnapNameList = append(pvcFromSnapNameList, pvcRestored.Object.Name)
	if !firstConsumer {
		err = pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return err, delFunc
		}
	}

	// Create Pod, and attach PVC from snapshot

	podRestored := testcore.IoWritePodConfig(pvcFromSnapNameList, pvcRestored.Object.Name+"-pod")
	podTmplRestored := podClient.MakePod(podRestored)

	writerPod = podClient.Create(ctx, podTmplRestored).Sync(ctx)
	if writerPod.HasError() {
		return writerPod.GetError(), delFunc
	}

	// Check if hash sum is correct
	sum = fmt.Sprintf("%s0/writer-%d.sha512", podRestored.MountPath, 0)
	writer := bytes.NewBufferString("")
	log.Info("Checker pod: ", writerPod.Object.GetName())
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
		return err, delFunc
	}

	if strings.Contains(writer.String(), "OK") {
		log.Info("Hashes match")
	} else {
		return fmt.Errorf("hashes don't match"), delFunc
	}
	return nil, delFunc
}

func validateCustomSnapName(name string, snapshotAmount int) bool {
	// If no. of snapshots is only 1 then we will take custom name else no.
	if snapshotAmount == 1 && len(name) != 0 {
		return true
	}
	// we will use custom name only if number of snapshots is 1 else we will discard custom name
	return false
}

func (*SnapSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (*SnapSuite) GetNamespace() string {
	return "snap-test"
}

func (ss *SnapSuite) GetName() string {
	if ss.Description != "" {
		return ss.Description
	}
	return "SnapSuite"
}

func (ss *SnapSuite) Parameters() string {
	return fmt.Sprintf("{snapshots: %d, volumeSize; %s}", ss.SnapAmount, ss.VolumeSize)
}

func getAllObservers(obsType observer.Type) []observer.Interface {
	if obsType == observer.EVENT {
		return []observer.Interface{
			&observer.PvcObserver{},
			&observer.VaObserver{},
			&observer.PodObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	} else if obsType == observer.LIST {
		return []observer.Interface{
			&observer.PvcListObserver{},
			&observer.VaListObserver{},
			&observer.PodListObserver{},
			&observer.EntityNumberObserver{},
			&observer.ContainerMetricsObserver{},
		}
	}
	return []observer.Interface{}
}

type ReplicationSuite struct {
	VolumeNumber int
	VolumeSize   string
	PodNumber    int
	SnapClass    string
}

func (rs *ReplicationSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient

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
				return pvc.GetError(), delFunc
			}

			pvcNameList[j] = pvc.Object.Name
			allPvcNames = append(allPvcNames, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "")
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return pod.GetError(), delFunc
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}
	log.Info("Creating a snapshot on each of the volumes")
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
					return err, delFunc
				}
				snapName := fmt.Sprintf("snap-%s", gotPvc.Name)
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
					return createSnap.GetError(), delFunc
				}
			}
			snapReadyError := clients.SnapClientGA.WaitForAllToBeReady(ctx)
			if snapReadyError != nil {
				return snapReadyError, delFunc
			}
		}
	} else if clients.SnapClientBeta != nil {
		for _, pvc := range allPvcNames {
			gotPvc, err := pvcClient.Interface.Get(ctx, pvc, metav1.GetOptions{})
			if err != nil {
				return err, delFunc
			}
			snapName := fmt.Sprintf("snap-%s", gotPvc.Name)
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
				return createSnap.GetError(), delFunc
			}
		}
		// Wait for snapshot to be created
		snapReadyError := clients.SnapClientBeta.WaitForAllToBeReady(ctx)
		if snapReadyError != nil {
			return snapReadyError, delFunc
		}
	} else {
		return fmt.Errorf("can't get alpha or beta snapshot client"), delFunc
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
				return pvc.GetError(), delFunc
			}
			pvcNameList[j] = pvc.Object.Name
		}
		// Create Pod, and attach restored PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "")
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return pod.GetError(), delFunc
		}
	}
	readyErr = podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}
	return nil, delFunc
}

func (rs *ReplicationSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (*ReplicationSuite) GetNamespace() string {
	return "replication-suite"
}

func (*ReplicationSuite) GetName() string {
	return "ReplicationSuite"
}

func (rs *ReplicationSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, volumeSize: %s}", rs.PodNumber, rs.VolumeNumber, rs.VolumeSize)
}

type VolumeExpansionSuite struct {
	VolumeNumber int
	PodNumber    int
	IsBlock      bool
	InitialSize  string
	ExpandedSize string
	Description  string
	AccessMode   string
}

func (ves *VolumeExpansionSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
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
				return pvc.GetError(), delFunc
			}

			pvcNameList = append(pvcNameList, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, "")
		if ves.IsBlock {
			podconf.VolumeMode = pod.Block
		}
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return pod.GetError(), delFunc
		}
		podObjectList = append(podObjectList, pod.Object)
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
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
				return nil, delFunc
			}

			gotSize, err := checkSize(ctx, podClient, p, v, true)
			if err != nil {
				return err, delFunc
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
		return err, delFunc
	}

	for i := range pvcList.Items {
		pvcList.Items[i].Spec.Resources.Requests = v1.ResourceList{
			v1.ResourceStorage: resource.MustParse(ves.ExpandedSize),
		}
		updatedPVC := pvcClient.Update(ctx, &pvcList.Items[i])
		if updatedPVC.HasError() {
			return updatedPVC.GetError(), delFunc
		}
	}
	if ves.IsBlock {
		// We can't compare deltas for RawBlock so just end it here
		return nil, delFunc
	}

	// Give some time for driver to expand
	time.Sleep(10 * time.Second)

	for _, p := range podObjectList {
		for _, v := range p.Spec.Containers[0].VolumeMounts {
			if !strings.Contains(v.Name, "vol") { // we can get token volume
				continue
			}

			log.Infof("Checking if %s was properly resized", p.Name+v.MountPath)

			wantSize, err := convertSpecSize(ves.ExpandedSize)
			if err != nil {
				return nil, delFunc
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
				return fmt.Errorf("sizes don't match: %s", pollErr.Error()), delFunc
			}
		}
	}

	return nil, delFunc
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

func (*VolumeExpansionSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (*VolumeExpansionSuite) GetNamespace() string {
	return "volume-expansion-suite"
}

func (ves *VolumeExpansionSuite) GetName() string {
	if ves.Description != "" {
		return ves.Description
	}
	return "VolumeExpansionSuite"
}

func (ves *VolumeExpansionSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, size: %s, expSize: %s, block: %s}", ves.PodNumber,
		ves.VolumeNumber, ves.InitialSize, ves.ExpandedSize, strconv.FormatBool(ves.IsBlock))
}

type VolumeHealthMetricsSuite struct {
	VolumeNumber int
	PodNumber    int
	VolumeSize   string
	Description  string
	AccessMode   string
	Namespace    string
}

func FindDriverLogs(command []string) (string, error) {

	cmd := exec.Command(command[0], command[1:]...) // #nosec G204
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	str := string(output)
	return str, nil
}

func (vh *VolumeHealthMetricsSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
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

	//Create a PVC
	vcconf := testcore.VolumeCreationConfig(storageClass, vh.VolumeSize, "", vh.AccessMode)
	volTmpl := pvcClient.MakePVC(vcconf)
	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return pvc.GetError(), delFunc
	}
	PVCNamespace := pvcClient.Namespace

	// Create Pod, and attach PVC
	podconf := testcore.VolumeHealthPodConfig([]string{pvc.Object.Name}, "")
	podTmpl := podClient.MakePod(podconf)

	pod := podClient.Create(ctx, podTmpl)
	if pod.HasError() {
		return pod.GetError(), delFunc
	}

	// Waiting for the Pod to be ready
	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}

	// Finding the node on which the pod is scheduled
	pods, podErr := podClient.Interface.List(ctx, metav1.ListOptions{})
	if podErr != nil {
		return podErr, delFunc
	}
	podScheduledNode := pods.Items[0].Spec.NodeName

	// Finding the PV created
	persistentVolumes, pvErr := pvClient.Interface.List(ctx, metav1.ListOptions{})
	if pvErr != nil {
		return pvErr, delFunc
	}

	// Finding the Volume Handle of the PV created
	var PersistentVolumeId string
	for _, p := range persistentVolumes.Items {
		if p.Spec.ClaimRef.Namespace == PVCNamespace {

			PersistentVolumeId = p.Spec.CSI.VolumeHandle
		}
	}

	// Finding the node pods and controller pods in the driver namespace
	driverpodClient, driverpoderr := clients.KubeClient.CreatePodClient(vh.Namespace)
	if driverpoderr != nil {
		return driverpoderr, delFunc
	}

	driverPods, podErr := driverpodClient.Interface.List(ctx, metav1.ListOptions{})
	if podErr != nil {
		return podErr, delFunc
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
		return err, delFunc
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

	var NodeLogs = false
	var ControllerLogs = false

	var retryCount = 1
	for i := 0; i < maxRetryCount; i++ {

		if !ControllerLogs {
			exe = []string{"bash", "-c", fmt.Sprintf("kubectl logs %s -n %s -c driver | grep ControllerGetVolume ", driverControllerPod, vh.Namespace)}
			str, err = FindDriverLogs(exe)
			if err == nil && strings.Contains(str, PersistentVolumeId) {
				ControllerLogs = true
			}
		}

		if !NodeLogs {
			exe = []string{"bash", "-c", fmt.Sprintf("kubectl logs %s -n %s -c driver | grep NodeGetVolumeStats ", driverNodePod, vh.Namespace)}
			str, err = FindDriverLogs(exe)
			if err == nil && strings.Contains(str, PersistentVolumeId) {
				NodeLogs = true
			}
		}

		if ControllerLogs && NodeLogs {
			return nil, delFunc
		}

		time.Sleep(ControllerLogsSleepTime * time.Second)
		retryCount = retryCount + 1
	}

	return errors.New("volume health metrics error"), delFunc
}

func (*VolumeHealthMetricsSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (*VolumeHealthMetricsSuite) GetNamespace() string {
	return "volume-health-metrics"
}

func (vh *VolumeHealthMetricsSuite) GetName() string {
	if vh.Description != "" {
		return vh.Description
	}
	return "VolumeHealthMetricSuite"
}

func (vh *VolumeHealthMetricsSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, size: %s}", vh.PodNumber, vh.VolumeNumber, vh.VolumeSize)
}

type CloneVolumeSuite struct {
	VolumeNumber  int
	VolumeSize    string
	PodNumber     int
	CustomPvcName string
	CustomPodName string
	Description   string
	AccessMode    string
}

func (cs *CloneVolumeSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	if cs.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		cs.VolumeNumber = 5
	}
	if cs.PodNumber <= 0 {
		log.Info("Using default number of pods")
		cs.PodNumber = 1
	}
	if cs.VolumeSize == "" {
		log.Info("Using default volume size:3Gi")
		cs.VolumeSize = "3Gi"
	}
	clonedVolName := ""
	result := validateCustomName(cs.CustomPvcName, cs.VolumeNumber)
	if result {
		clonedVolName = cs.CustomPvcName + "-cloned"
		log.Infof("using custom pvc-name:%s"+" and cloned PVC name:%s", cs.CustomPvcName, clonedVolName)
	} else {
		cs.CustomPvcName = ""
	}

	clonedPodName := ""
	result = validateCustomName(cs.CustomPodName, cs.VolumeNumber)
	if result {
		clonedPodName = cs.CustomPodName + "-cloned"
		log.Infof("using custom pod-name:%s"+" and cloned PVC pod-name:%s", cs.CustomPodName, clonedPodName)
	} else {
		cs.CustomPodName = ""
	}

	log.Infof("Creating %s pods, each with %s volumes", color.YellowString(strconv.Itoa(cs.PodNumber)),
		color.YellowString(strconv.Itoa(cs.VolumeNumber)))
	var allPvcNames []string
	for i := 0; i < cs.PodNumber; i++ {
		pvcNameList := make([]string, cs.VolumeNumber)
		for j := 0; j < cs.VolumeNumber; j++ {
			// Create PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, cs.VolumeSize, cs.CustomPvcName, cs.AccessMode)
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return pvc.GetError(), delFunc
			}

			pvcNameList[j] = pvc.Object.Name
			allPvcNames = append(allPvcNames, pvc.Object.Name)
		}

		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, cs.CustomPodName)
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return pod.GetError(), delFunc
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}

	log.Info("Creating new pods with cloned volumes mounted on them")
	for i := 0; i < cs.PodNumber; i++ {
		pvcNameList := make([]string, cs.VolumeNumber)
		for j := 0; j < cs.VolumeNumber; j++ {
			// Restore PVCs
			vcconf := testcore.VolumeCreationConfig(storageClass, cs.VolumeSize, clonedVolName, cs.AccessMode)
			vcconf.SourceVolumeName = allPvcNames[j+(i*cs.VolumeNumber)]
			volTmpl := pvcClient.MakePVC(vcconf)

			pvc := pvcClient.Create(ctx, volTmpl)
			if pvc.HasError() {
				return pvc.GetError(), delFunc
			}
			pvcNameList[j] = pvc.Object.Name
		}
		// Create Pod, and attach restored PVC
		podconf := testcore.ProvisioningPodConfig(pvcNameList, clonedPodName)
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl)
		if pod.HasError() {
			return pod.GetError(), delFunc
		}
	}
	readyErr = podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}
	return nil, delFunc
}

func (cs *CloneVolumeSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

func (cs *CloneVolumeSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
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
		SnapClientGA:      nil,
		SnapClientBeta:    nil,
	}, nil
}

func (*CloneVolumeSuite) GetNamespace() string {
	return "clonevolume-suite"
}

func (cs *CloneVolumeSuite) GetName() string {
	if cs.Description != "" {
		return cs.Description
	}
	return "CloneVolumeSuite"
}

func (cs *CloneVolumeSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, volumes: %d, volumeSize: %s}", cs.PodNumber, cs.VolumeNumber, cs.VolumeSize)
}

type MultiAttachSuite struct {
	PodNumber   int
	RawBlock    bool
	Description string
	AccessMode  string
	VolumeSize  string
}

func (mas *MultiAttachSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	nodeClient := clients.NodeClient

	if mas.PodNumber <= 0 {
		log.Info("Using default number of pods")
		mas.PodNumber = 2
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
		return pvc.GetError(), delFunc
	}

	log.Info("Attaching Volume to original pod")
	// Create Pod, and attach PVC
	podconf := testcore.MultiAttachPodConfig([]string{pvc.Object.Name})

	if mas.RawBlock {
		podconf.VolumeMode = pod.Block
	}
	podTmpl := podClient.MakePod(podconf)

	originalPod := podClient.Create(ctx, podTmpl)
	if originalPod.HasError() {
		return originalPod.GetError(), delFunc
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}

	var spreadConstraint []v1.TopologySpreadConstraint
	var labels map[string]string
	if nodeClient != nil {
		nodeList, ncErr := nodeClient.Interface.List(ctx, metav1.ListOptions{
			LabelSelector: "!node-role.kubernetes.io/master",
		})
		if ncErr != nil {
			return ncErr, delFunc
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
			return pod.GetError(), delFunc
		}
		newPods = append(newPods, pod)
	}

	if mas.AccessMode == "ReadWriteOncePod" {
		time.Sleep(1 * time.Minute)
		readyPodCount, err := podClient.ReadyPodsCount(ctx)

		if err != nil {
			return err, delFunc
		}
		if readyPodCount == 1 {
			return nil, delFunc
		}
	}

	readyErr = podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return readyErr, delFunc
	}

	if !mas.RawBlock {

		log.Info("Writing to Volume on 1st originalPod")
		file := fmt.Sprintf("%s0/blob.data", podconf.MountPath)
		sum := fmt.Sprintf("%s0/blob.sha512", podconf.MountPath)
		// Write random blob to pvc
		ddRes := bytes.NewBufferString("")
		if err := podClient.Exec(ctx, originalPod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
			return err, delFunc
		}
		log.Info("Writer originalPod: ", originalPod.Object.GetName())
		log.Debug(ddRes.String())

		// Write hash sum of blob
		if err := podClient.Exec(ctx, originalPod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return err, delFunc
		}

		log.Info("Checking hash sum on all of the other pods")
		for _, p := range newPods {
			writer := bytes.NewBufferString("")
			if err := podClient.Exec(ctx, p.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
				return err, delFunc
			}
			if strings.Contains(writer.String(), "OK") {
				log.Info("Hashes match")
			} else {
				return fmt.Errorf("hashes don't match"), delFunc
			}
		}
	} else {
		device := fmt.Sprintf("/dev%s0", podconf.MountPath)
		file := fmt.Sprintf("/tmp/blob.data")

		hash := bytes.NewBufferString("")

		// Write random data to block device
		if err := podClient.Exec(ctx, originalPod.Object, []string{"dd", "if=/dev/urandom", "of=" + device, "bs=1M", "count=128", "oflag=sync"}, os.Stdout, os.Stderr, false); err != nil {
			return err, delFunc
		}
		// Calculate hashsum of first 128MB
		// We can't pipe anything here so just write to file
		if err := podClient.Exec(ctx, originalPod.Object, []string{"dd", "if=" + device, "of=" + file, "bs=1M", "count=128"}, os.Stdout, os.Stderr, false); err != nil {
			return err, delFunc
		}
		// Calculate hash sum of that file
		if err := podClient.Exec(ctx, originalPod.Object, []string{"sha512sum", file}, hash, os.Stderr, false); err != nil {
			return err, delFunc
		}
		log.Info("OriginalPod: ", originalPod.Object.GetName())
		log.Info("hash sum is:", hash.String())

		log.Info("Checking hash sum on all of the other pods")
		for _, p := range newPods {
			newHash := bytes.NewBufferString("")
			if err := podClient.Exec(ctx, p.Object, []string{"blockdev", "--flushbufs", device}, os.Stdout, os.Stderr, false); err != nil {
				return err, delFunc
			}
			if err := podClient.Exec(ctx, p.Object, []string{"dd", "if=" + device, "of=" + file, "bs=1M", "count=128"}, os.Stdout, os.Stderr, false); err != nil {
				return err, delFunc
			}
			if err := podClient.Exec(ctx, p.Object, []string{"sha512sum", file}, newHash, os.Stderr, false); err != nil {
				return err, delFunc
			}

			log.Info("Pod: ", p.Object.GetName())
			log.Info("hash sum is:", newHash.String())

			if newHash.String() == hash.String() {
				log.Info("Hashes match")
			} else {
				return fmt.Errorf("hashes don't match"), delFunc
			}
		}
	}

	return nil, delFunc
}

func (mas *MultiAttachSuite) GenerateTopologySpreadConstraints(nodeCount int, labels map[string]string) []v1.TopologySpreadConstraint {
	// Calculate MaxSkew parameter
	maxSkew, remainder := mas.PodNumber/nodeCount, mas.PodNumber%nodeCount
	// in case of odd pod/node count - increase the maxSkew
	if remainder != 0 {
		maxSkew += 1
	}
	spreadConstraints := []v1.TopologySpreadConstraint{{
		MaxSkew:           int32(maxSkew),
		TopologyKey:       "kubernetes.io/hostname",
		WhenUnsatisfiable: v1.ScheduleAnyway,
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
	},
	}
	return spreadConstraints
}

func (mas *MultiAttachSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (*MultiAttachSuite) GetNamespace() string {
	return "mas-test"
}

func (mas *MultiAttachSuite) GetName() string {
	if mas.Description != "" {
		return mas.Description
	}
	return "MultiAttachSuite"
}

func (mas *MultiAttachSuite) Parameters() string {
	return fmt.Sprintf("{pods: %d, rawBlock: %s, size: %s, accMode: %s}",
		mas.PodNumber, strconv.FormatBool(mas.RawBlock), mas.VolumeSize, mas.AccessMode)
}

type BlockSnapSuite struct {
	SnapClass   string
	VolumeSize  string
	Description string
	AccessMode  string
}

func (bss *BlockSnapSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)
	if bss.VolumeSize == "" {
		log.Info("Using default volume size : 3Gi")
		bss.VolumeSize = "3Gi"
	}
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	log.Info("Creating BlockSnap pod")
	firstConsumer, err := shouldWaitForFirstConsumer(ctx, storageClass, pvcClient)
	if err != nil {
		return err, delFunc
	}
	// Create first PVC
	var pvcNameList []string
	vcconf := testcore.VolumeCreationConfig(storageClass, bss.VolumeSize, "", "")
	volTmpl := pvcClient.MakePVC(vcconf)
	pvc := pvcClient.Create(ctx, volTmpl)
	if pvc.HasError() {
		return pvc.GetError(), delFunc
	}
	pvcNameList = append(pvcNameList, pvc.Object.Name)
	if !firstConsumer {
		err := pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return err, delFunc
		}
	}

	// Create Pod, and attach PVC
	podconf := testcore.BlockSnapPodConfig(pvcNameList)
	podTmpl := podClient.MakePod(podconf)

	file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
	writerPod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if writerPod.HasError() {
		return writerPod.GetError(), delFunc
	}

	// Write random blob to pvc
	ddRes := bytes.NewBufferString("")
	if err := podClient.Exec(ctx, writerPod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
		return err, delFunc
	}
	log.Info("Writer pod: ", writerPod.Object.GetName())
	log.Debug(ddRes.String())

	originalHash := bytes.NewBufferString("")

	// Calculate hash sum of that file
	if err := podClient.Exec(ctx, writerPod.Object, []string{"sha512sum", file}, originalHash, os.Stderr, false); err != nil {
		return err, delFunc
	}
	log.Info("hash sum is: ", originalHash.String())

	// Get PVC for creating snapshot
	gotPvc, err := pvcClient.Interface.Get(ctx, pvc.Object.Name, metav1.GetOptions{})
	if err != nil {
		return err, delFunc
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
			return createSnap.GetError(), delFunc
		}

		// Wait for snapshot to be created
		err := createSnap.WaitForRunning(ctx)
		if err != nil {
			return err, delFunc
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
			return createSnap.GetError(), delFunc
		}

		// Wait for snapshot to be created
		err := createSnap.WaitForRunning(ctx)
		if err != nil {
			return err, delFunc
		}
	} else {
		return fmt.Errorf("can't get alpha or beta snapshot client"), delFunc
	}

	// Create second PVC from snapshot
	vcconf.SnapName = createSnap.Name()
	var mode v1.PersistentVolumeMode = pod.Block
	vcconf.VolumeMode = &mode

	vcconf.AccessModes = testcore.GetAccessMode(bss.AccessMode)

	log.Infof("Restoring from %s", vcconf.SnapName)
	volRestored := pvcClient.MakePVC(vcconf)
	pvcRestored := pvcClient.Create(ctx, volRestored)
	if pvc.HasError() {
		return pvc.GetError(), delFunc
	}
	if !firstConsumer {
		err := pvcClient.WaitForAllToBeBound(ctx)
		if err != nil {
			return err, delFunc
		}
	}

	// Create Pod, and attach PVC from snapshot
	podRestored := testcore.BlockSnapPodConfig([]string{pvcRestored.Object.Name})
	podRestored.VolumeMode = pod.Block
	podTmplRestored := podClient.MakePod(podRestored)

	writerPod = podClient.Create(ctx, podTmplRestored).Sync(ctx)
	if writerPod.HasError() {
		return writerPod.GetError(), delFunc
	}

	// Check the data persistence
	writer := bytes.NewBufferString("")
	log.Info("Checker pod: ", writerPod.Object.GetName())

	// flush the buffer
	if err := podClient.Exec(ctx, writerPod.Object, []string{"blockdev", "--flushbufs", "/dev/data0"}, os.Stdout, os.Stderr, false); err != nil {
		return err, delFunc
	}
	// create folder
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "mkdir /data0"}, writer, os.Stderr, false); err != nil {
		return err, delFunc
	}
	// mount rawblock volume to created folder
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "mount /dev/data0 /data0"}, writer, os.Stderr, false); err != nil {
		return err, delFunc
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
		return pollErr, delFunc
	}

	log.Info("new hash sum is: ", newHash.String())

	if newHash.String() == originalHash.String() {
		log.Info("Hashes match")
	} else {
		return fmt.Errorf("hashes don't match"), delFunc
	}

	return nil, delFunc
}

func (*BlockSnapSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (*BlockSnapSuite) GetNamespace() string {
	return "block-snap-test"
}

func (bss *BlockSnapSuite) GetName() string {
	if bss.Description != "" {
		return bss.Description
	}
	return "BlockSnapSuite"
}

func (bss *BlockSnapSuite) Parameters() string {
	return fmt.Sprintf("{size: %s, accMode: %s}", bss.VolumeSize, bss.AccessMode)
}

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

type VolumeMigrateSuite struct {
	TargetSC     string
	Description  string
	VolumeNumber int
	PodNumber    int
	Flag         bool
}

func (vms *VolumeMigrateSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	log := utils.GetLoggerFromContext(ctx)

	if vms.VolumeNumber <= 0 {
		log.Println("Using default number of volumes")
		vms.VolumeNumber = 1
	}
	if vms.PodNumber <= 0 {
		log.Println("Using default number of pods")
		vms.PodNumber = 3
	}

	log.Println("Volumes:", vms.VolumeNumber, "pods:", vms.PodNumber)

	scClient := clients.SCClient
	pvcClient := clients.PVCClient
	pvClient := clients.PersistentVolumeClient
	podClient := clients.PodClient
	stsClient := clients.StatefulSetClient

	sourceSC := scClient.Get(ctx, storageClass)
	if sourceSC.HasError() {
		return sourceSC.GetError(), delFunc
	}
	targetSC := scClient.Get(ctx, vms.TargetSC)
	if targetSC.HasError() {
		return targetSC.GetError(), delFunc
	}

	stsConf := testcore.VolumeMigrateStsConfig(storageClass, "1Gi", vms.VolumeNumber, int32(vms.PodNumber), "")
	stsTmpl := stsClient.MakeStatefulSet(stsConf)
	// Creating Statefulset
	log.Println("Creating Statefulset")
	sts := stsClient.Create(ctx, stsTmpl)
	if sts.HasError() {
		return sts.GetError(), delFunc
	}
	sts = sts.Sync(ctx)
	if sts.HasError() {
		return sts.GetError(), delFunc
	}

	var pvNames []string
	podList, err := sts.GetPodList(ctx)
	if err != nil {
		return err, delFunc
	}
	g, _ := errgroup.WithContext(ctx)
	for _, pod := range podList.Items {
		pod := pod
		for _, volume := range pod.Spec.Volumes {
			volume := volume
			if volume.PersistentVolumeClaim != nil {
				g.Go(func() error {
					log.Println("Getting PVC")
					pvc := pvcClient.Get(ctx, volume.PersistentVolumeClaim.ClaimName)
					if pvc.HasError() {
						return pvc.GetError()
					}
					err = pvcClient.WaitForAllToBeBound(ctx)
					if err != nil {
						return err
					}

					log.Println("Getting PV")
					pvName := pvc.Object.Spec.VolumeName
					pvNames = append(pvNames, pvName)
					pv := pvClient.Get(ctx, pvName)
					if pv.HasError() {
						return pv.GetError()
					}

					if !vms.Flag {
						file := fmt.Sprintf("%s0/writer-%d.data", stsConf.MountPath, 0)
						sum := fmt.Sprintf("%s0/writer-%d.sha512", stsConf.MountPath, 0)
						// Write random blob
						ddRes := bytes.NewBufferString("")
						if err := podClient.Exec(ctx, &pod, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
							return err
						}
						log.Info("Writer pod: ", pod.Name)
						log.Debug(ddRes.String())
						log.Info("Written the values successfully ", ddRes)
						log.Info(ddRes.String())

						// Write hash sum of blob
						if err := podClient.Exec(ctx, &pod, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
							log.Println("write hash sum err")
							return err
						}
						log.Info("Checksum value: ", sum)
						// sync to be sure
						if err := podClient.Exec(ctx, &pod, []string{"/bin/bash", "-c", "sync " + sum}, os.Stdout, os.Stderr, false); err != nil {
							return err
						}
					}

					pv.Object.Annotations["migration.storage.dell.com/migrate-to"] = vms.TargetSC
					log.Println("Updating PV")
					updatedPV := pvClient.Update(ctx, pv.Object)
					if updatedPV.HasError() {
						return updatedPV.GetError()
					}

					log.Println("Waiting PV to create")
					err = pvClient.WaitPV(ctx, pvName+"-to-"+vms.TargetSC)
					if err != nil {
						return err
					}
					log.Println("pv", pvName+"-to-"+vms.TargetSC, "seems good")
					return nil
				})
			}
		}
	}

	delFunc = func(f func() error) func() error {
		return func() error {
			log.Info("Deleting pvs")
			pvs, err := pvClient.Interface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}
			for _, p := range pvs.Items {
				for _, name := range pvNames {
					p := p
					if strings.Contains(p.Name, name) {
						pvClient.Delete(ctx, &p)
					}
				}
			}
			return nil
		}
	}(nil)

	if err := g.Wait(); err != nil {
		log.Println("g.wait err")
		return err, delFunc
	}

	if vms.Flag {
		return nil, delFunc
	}

	log.Println("Deleting old Statefulset")
	deletionOrphan := metav1.DeletePropagationOrphan
	delSts := stsClient.DeleteWithOptions(ctx, sts.Set, metav1.DeleteOptions{PropagationPolicy: &deletionOrphan})
	if delSts.HasError() {
		return delSts.GetError(), delFunc
	}

	log.Println("Deleting pods")
	for _, pod := range podList.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				log.Println("Deleting PVC")
				pvc := pvcClient.Get(ctx, volume.PersistentVolumeClaim.ClaimName)
				if pvc.HasError() {
					return pvc.GetError(), delFunc
				}
				delPVC := pvcClient.Delete(ctx, pvc.Object)
				if delPVC.HasError() {
					return delPVC.GetError(), delFunc
				}
			}
		}
		pod := pod
		podClient.Delete(ctx, &pod)
	}

	newStsConf := testcore.VolumeMigrateStsConfig(vms.TargetSC, "1Gi", vms.VolumeNumber, int32(vms.PodNumber), "")
	newStsTmpl := stsClient.MakeStatefulSet(newStsConf)
	// Creating new Statefulset
	log.Println("Creating new Statefulset")
	newSts := stsClient.Create(ctx, newStsTmpl)
	if newSts.HasError() {
		return newSts.GetError(), delFunc
	}
	newSts = newSts.Sync(ctx)
	if newSts.HasError() {
		return newSts.GetError(), delFunc
	}

	newPodList, err := newSts.GetPodList(ctx)
	if err != nil {
		return err, delFunc
	}

	for _, pod := range newPodList.Items {
		// Check if hash sum is correct
		sum := fmt.Sprintf("%s0/writer-%d.sha512", newStsConf.MountPath, 0)
		writer := bytes.NewBufferString("")
		log.Info("Checker pod: ", pod.Name)
		pod := pod
		if err := podClient.Exec(ctx, &pod, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
			return err, delFunc
		}
		if strings.Contains(writer.String(), "OK") {
			log.Info("Hashes match")
		} else {
			return fmt.Errorf("hashes don't match"), delFunc
		}
	}

	return nil, delFunc
}

func (*VolumeMigrateSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

func (vms *VolumeMigrateSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	if ok, err := client.StorageClassExists(context.Background(), vms.TargetSC); !ok {
		return nil, fmt.Errorf("target storage class doesn't exist; error = %v", err)
	}

	pvClient, pvErr := client.CreatePVClient()
	if pvErr != nil {
		return nil, pvErr
	}

	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	scClient, scErr := client.CreateSCClient()
	if scErr != nil {
		return nil, scErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	stsClient, stsErr := client.CreateStatefulSetClient(namespace)
	if stsErr != nil {
		return nil, stsErr
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
		PersistentVolumeClient: pvClient,
		PVCClient:              pvcClient,
		PodClient:              podClient,
		SCClient:               scClient,
		StatefulSetClient:      stsClient,
		VaClient:               vaClient,
		MetricsClient:          metricsClient,
	}, nil
}

func (*VolumeMigrateSuite) GetNamespace() string {
	return "migration-test"
}

func (vms *VolumeMigrateSuite) GetName() string {
	if vms.Description != "" {
		return vms.Description
	}
	return "VolumeMigrationSuite"
}

func (vms *VolumeMigrateSuite) Parameters() string {
	return fmt.Sprintf("{Target storageclass: %s, volumes: %s, pods: %s}", vms.TargetSC, vms.VolumeNumber, vms.PodNumber)
}
