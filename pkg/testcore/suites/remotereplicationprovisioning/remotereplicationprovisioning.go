package remotereplicationprovisioning

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/commonparams"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pv"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/replicationgroup"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/sc"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	"github.com/fatih/color"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RemoteReplicationProvisioningSuite is used to manage remote replication provisioning test suite
type RemoteReplicationProvisioningSuite struct {
	VolumeNumber     int
	VolumeSize       string
	Description      string
	VolAccessMode    string
	RemoteConfigPath string
	NoFailover       bool
	Image            string
}

// Run executes remote replication provisioning test suite
func (rrps *RemoteReplicationProvisioningSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)
	pvcClient := clients.PVCClient
	podClient := clients.PodClient
	pvClient := clients.PersistentVolumeClient
	scClient := clients.SCClient
	rgClient := clients.RgClient
	storClass, err := scClient.Interface.Get(ctx, storageClass, metav1.GetOptions{})
	if err != nil {
		return delFunc, err
	}

	var (
		remotePVCObject  v1.PersistentVolumeClaim
		remotePVClient   *pv.Client
		remoteRGClient   *replicationgroup.Client
		remoteKubeClient *k8sclient.KubeClient
	)

	isSingle := false
	if storClass.Parameters["replication.storage.dell.com/remoteClusterID"] == "self" {
		isSingle = true
	}

	if rrps.VolumeNumber <= 0 {
		log.Info("Using default number of volumes")
		rrps.VolumeNumber = 1
	}
	if rrps.VolumeSize == "" {
		log.Info("Using default volume size 3Gi")
		rrps.VolumeSize = "3Gi"
	}
	if rrps.Image == "" {
		rrps.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", rrps.Image)
	}

	if rrps.RemoteConfigPath != "" && !isSingle {
		// Loading config
		remoteConfig, err := k8sclient.GetConfig(rrps.RemoteConfigPath)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		// Connecting to host and creating new Kubernetes Client
		remoteKubeClient, err = k8sclient.NewRemoteKubeClient(remoteConfig, pvClient.Timeout)
		if err != nil {
			log.Errorf("Couldn't create new Remote kubernetes client. Error = %v", err)
			return nil, err
		}
		remotePVClient, err = remoteKubeClient.CreatePVClient()
		if err != nil {
			return nil, err
		}
		remoteRGClient, err = remoteKubeClient.CreateRGClient()
		if err != nil {
			return nil, err
		}

		log.Info("Created remote kube client")
	} else {
		remotePVClient = pvClient
		remoteRGClient = rgClient
		remoteKubeClient = clients.KubeClient
	}

	scObject, scErr := scClient.Interface.Get(ctx, storageClass, metav1.GetOptions{})
	if scErr != nil {
		return nil, scErr
	}

	rpEnabled := scObject.Parameters[sc.IsReplicationEnabled]
	if rpEnabled != "true" {
		return nil, fmt.Errorf("replication is not enabled on this storage class and please provide valid sc")
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
			return delFunc, pvc.GetError()
		}
		pvcNames = append(pvcNames, pvc.Object.Name)
	}

	err = pvcClient.WaitForAllToBeBound(ctx)
	if err != nil {
		return delFunc, err
	}

	var pvNames []string
	// We can get actual pv names only after all pvc are bound
	pvcList, err := pvcClient.Interface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return delFunc, err
	}
	for _, p := range pvcList.Items {
		pvNames = append(pvNames, p.Spec.VolumeName)
	}

	log.Info("Creating pod for each volume")
	for _, name := range pvcNames {
		// Create Pod, and attach PVC
		podconf := testcore.ProvisioningPodConfig([]string{name}, "", rrps.Image)
		podTmpl := podClient.MakePod(podconf)

		pod := podClient.Create(ctx, podTmpl).Sync(ctx)
		if pod.HasError() {
			return delFunc, pod.GetError()
		}

		// write data to files and calculate checksum.
		file := fmt.Sprintf("%s0/writer-%d.data", podconf.MountPath, 0)
		sum := fmt.Sprintf("%s0/writer-%d.sha512", podconf.MountPath, 0)

		// Write random blob to pvc
		ddRes := bytes.NewBufferString("")
		if err := podClient.Exec(ctx, pod.Object, []string{"dd", "if=/dev/urandom", "of=" + file, "bs=1M", "count=128", "oflag=sync"}, ddRes, os.Stderr, false); err != nil {
			return delFunc, err
		}

		log.Info("Writer pod: ", pod.Object.GetName())
		log.Debug(ddRes.String())
		log.Info("Written the values successfully ", ddRes)
		log.Info(ddRes.String())

		// Write hash sum of blob
		if err := podClient.Exec(ctx, pod.Object, []string{"/bin/bash", "-c", "sha512sum " + file + " > " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return delFunc, err
		}
		log.Info("Checksum value: ", sum)

		// sync to be sure
		if err := podClient.Exec(ctx, pod.Object, []string{"/bin/bash", "-c", "sync " + sum}, os.Stdout, os.Stderr, false); err != nil {
			return delFunc, err
		}
	}

	readyErr := podClient.WaitForAllToBeReady(ctx)
	if readyErr != nil {
		return delFunc, readyErr
	}

	log.Infof("Checking annotations for all PVC's")

	// Wait until to get annotations for all PVCs

	boundErr := pvcClient.CheckAnnotationsForVolumes(ctx, scObject)
	if boundErr != nil {
		return delFunc, boundErr
	}

	log.Infof("Successfully checked annotations On PVC")

	log.Infof("Checking annotation on pv's")
	for _, pvName := range pvNames {
		// Check on local cluster
		pvObject, pvErr := pvClient.Interface.Get(ctx, pvName, metav1.GetOptions{})
		if pvErr != nil {
			return delFunc, pvErr
		}
		err := pvClient.CheckReplicationAnnotationsForPV(ctx, pvObject)
		if err != nil {
			return delFunc, fmt.Errorf("replication Annotations and Labels are not added for PV %s and %s", pvName, err)
		}

		// Check on remote cluster
		if !isSingle {
			remotePvName := pvName
			remotePVObject, remotePVErr := remotePVClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return delFunc, remotePVErr
			}
			err = remotePVClient.CheckReplicationAnnotationsForRemotePV(ctx, remotePVObject)
			if err != nil {
				return delFunc, fmt.Errorf("replication Annotations and Labels are not added for Remote PV %s and %s", remotePvName, err)
			}
		} else {
			remotePvName := "replicated-" + pvName
			remotePVObject, remotePVErr := pvClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return delFunc, remotePVErr
			}
			err = pvClient.CheckReplicationAnnotationsForRemotePV(ctx, remotePVObject)
			if err != nil {
				return delFunc, fmt.Errorf("replication Annotations and Labels are not added for Remote PV %s and %s", remotePvName, err)
			}
		}
	}
	log.Infof("Successfully checked annotations on local and remote pv")

	// List PVCs once again, since here we can be sure that all annotations will be correctly set
	pvcList, err = pvcClient.Interface.List(ctx, metav1.ListOptions{})
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
		return delFunc, fmt.Errorf("expected Annotations are not added to the replication group %s", rgName)
	}

	if rgObject.Object.Labels[commonparams.RemoteClusterID] != scObject.Parameters[sc.RemoteClusterID] {
		return delFunc, fmt.Errorf("expected Labels are not added to the replication group %s", rgName)
	}
	log.Infof("Successfully Checked Annotations and Labels on ReplicationGroup")

	// Return if failover is not requested
	if rrps.NoFailover {
		return delFunc, nil
	}

	// Failover to the target site
	log.Infof("Executing failover action on ReplicationGroup %s", remoteRgName)
	actionErr := rgObject.ExecuteAction(ctx, "FAILOVER_REMOTE")
	if actionErr != nil {
		return delFunc, actionErr
	}

	log.Infof("Executing reprotect action on ReplicationGroup %s", remoteRgName)
	remoteRgObject = remoteRGClient.Get(ctx, remoteRgName)
	actionErr = remoteRgObject.ExecuteAction(ctx, "REPROTECT_LOCAL")
	if actionErr != nil {
		return delFunc, actionErr
	}

	log.Infof("Creating pvc and pods on remote cluster")
	if !isSingle {
		ns, err := remoteKubeClient.CreateNamespace(ctx, clients.PVCClient.Namespace)
		if err != nil {
			return delFunc, err
		}
		remoteNamespace := ns.Name
		remotePVCClient, err := remoteKubeClient.CreatePVCClient(ns.Name)
		if err != nil {
			return delFunc, err
		}
		remotePodClient, err := remoteKubeClient.CreatePodClient(ns.Name)
		if err != nil {
			return delFunc, err
		}

		var pvcNameList []string

		for _, pvc := range pvcList.Items {
			log.Infof("Creating remote pvc %s", pvc.Name)

			remotePvName := pvc.Spec.VolumeName
			remotePVObject, remotePVErr := remotePVClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return delFunc, remotePVErr
			}
			pvcName := remotePVObject.Annotations["replication.storage.dell.com/remotePVC"]
			pvcNameList = append(pvcNameList, pvcName)

			remotePVCObject = pvcClient.CreatePVCObject(ctx, remotePVObject, remoteNamespace)
			remotePvc := remotePVCClient.Create(ctx, &remotePVCObject)
			if remotePvc.HasError() {
				return delFunc, remotePvc.GetError()
			}

		}
		for _, pvcName := range pvcNameList {
			log.Infof("Verifying data from pvc %s", pvcName)

			podConf := testcore.ProvisioningPodConfig([]string{pvcName}, "", rrps.Image)
			podTemp := remotePodClient.MakePod(podConf)

			remotePod := remotePodClient.Create(ctx, podTemp).Sync(ctx)
			if remotePod.HasError() {
				return delFunc, remotePod.GetError()
			}

			sum := fmt.Sprintf("%s0/writer-%d.sha512", podConf.MountPath, 0)
			writer := bytes.NewBufferString(" ")
			log.Info("Checker pod: ", remotePod.Object.GetName())
			if err := remotePodClient.Exec(ctx, remotePod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
				log.Error("Failed to verify hash")
				return delFunc, err
			}

			if strings.Contains(writer.String(), "OK") {
				log.Info("Hashes match")
			} else {
				return delFunc, fmt.Errorf("hashes don't match")
			}
		}
	} else {
		ns, err := clients.KubeClient.CreateNamespace(ctx, "replicated-"+clients.PVCClient.Namespace)
		if err != nil {
			return delFunc, err
		}

		remoteNamespace := ns.Name
		remotePVCClient, err := clients.KubeClient.CreatePVCClient(ns.Name)
		if err != nil {
			return delFunc, err
		}
		remotePodClient, err := clients.KubeClient.CreatePodClient(ns.Name)
		if err != nil {
			return delFunc, err
		}

		var pvcNameList []string

		for _, pvc := range pvcList.Items {
			log.Infof("Creating remote pvc %s", pvc.Name)

			remotePvName := "replicated-" + pvc.Spec.VolumeName
			remotePVObject, remotePVErr := pvClient.Interface.Get(ctx, remotePvName, metav1.GetOptions{})
			if remotePVErr != nil {
				return delFunc, remotePVErr
			}
			pvcName := remotePVObject.Annotations["replication.storage.dell.com/remotePVC"]
			pvcNameList = append(pvcNameList, pvcName)

			remotePVCObject = pvcClient.CreatePVCObject(ctx, remotePVObject, remoteNamespace)
			remotePvc := remotePVCClient.Create(ctx, &remotePVCObject)
			if remotePvc.HasError() {
				return delFunc, remotePvc.GetError()
			}

		}
		for _, pvcName := range pvcNameList {
			log.Infof("Verifying data from pvc %s", pvcName)

			podConf := testcore.ProvisioningPodConfig([]string{pvcName}, "", rrps.Image)
			podTemp := remotePodClient.MakePod(podConf)

			remotePod := remotePodClient.Create(ctx, podTemp).Sync(ctx)
			if remotePod.HasError() {
				return delFunc, remotePod.GetError()
			}

			sum := fmt.Sprintf("%s0/writer-%d.sha512", podConf.MountPath, 0)
			writer := bytes.NewBufferString(" ")
			log.Info("Checker pod: ", remotePod.Object.GetName())
			if err := remotePodClient.Exec(ctx, remotePod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
				log.Error("Failed to verify hash")
				return delFunc, err
			}

			if strings.Contains(writer.String(), "OK") {
				log.Info("Hashes match")
			} else {
				return delFunc, fmt.Errorf("hashes don't match")
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
		return delFunc, actionErr
	}

	log.Infof("Executing reprotect action using local RG %s", remoteRgName)
	rgObject = rgClient.Get(ctx, remoteRgName)
	actionErr = rgObject.ExecuteAction(ctx, "REPROTECT_LOCAL")
	if actionErr != nil {
		return delFunc, actionErr
	}

	return delFunc, nil
}

// GetObservers returns all observers
func (*RemoteReplicationProvisioningSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return common.GetAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, pv, va, metrics, sc, rg clients
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

// GetNamespace returns remote replication provisioning suite namespace
func (*RemoteReplicationProvisioningSuite) GetNamespace() string {
	return "repl-prov-test"
}

// GetName returns remote replication provisioning suite name
func (rrps *RemoteReplicationProvisioningSuite) GetName() string {
	if rrps.Description != "" {
		return rrps.Description
	}
	return "RemoteReplicationProvisioningSuite"
}

// Parameters returns formatted string of parameters
func (rrps *RemoteReplicationProvisioningSuite) Parameters() string {
	return fmt.Sprintf("{volumes: %d, volumeSize: %s, remoteConfig: %s}", rrps.VolumeNumber, rrps.VolumeSize, rrps.RemoteConfigPath)
}

//todo: get this uncommented and working, it will be used to overload the NewClient call when the test is ran
/* var clientNewFunc = func(config *rest.Config, options Options) (Client, error) {
	return client.NewClient(config, options)
} */

// Options are creation options for a Client.
type Options struct {
	// Scheme, if provided, will be used to map go structs to GroupVersionKinds
	Scheme *runtime.Scheme

	// Mapper, if provided, will be used to map GroupVersionKinds to Resources
	Mapper meta.RESTMapper

	// Opts is used to configure the warning handler responsible for
	// surfacing and handling warnings messages sent by the API server.
	Opts WarningHandlerOptions
}

type WarningHandlerOptions struct {
	// SuppressWarnings decides if the warnings from the
	// API server are suppressed or surfaced in the client.
	SuppressWarnings bool
	// AllowDuplicateLogs does not deduplicate the to-be
	// logged surfaced warnings messages. See
	// log.WarningHandlerOptions for considerations
	// regarding deduplication
	AllowDuplicateLogs bool
}
