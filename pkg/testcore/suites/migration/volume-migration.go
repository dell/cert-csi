package migration

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pv"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/statefulset"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"github.com/dell/cert-csi/pkg/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
)

type VolumeMigrateSuite struct {
	TargetSC     string
	Description  string
	VolumeNumber int
	PodNumber    int
	Flag         bool
	Image        string
}

// Run executes volume migrate test suite
func (vms *VolumeMigrateSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	log := utils.GetLoggerFromContext(ctx)

	if vms.VolumeNumber <= 0 {
		log.Println("Using default number of volumes")
		vms.VolumeNumber = 1
	}
	if vms.PodNumber <= 0 {
		log.Println("Using default number of pods")
		vms.PodNumber = 3
	}
	if vms.Image == "" {
		vms.Image = "quay.io/centos/centos:latest"
		log.Infof("Using default image: %s", vms.Image)
	}

	log.Println("Volumes:", vms.VolumeNumber, "pods:", vms.PodNumber)

	scClient := clients.SCClient
	pvcClient := clients.PVCClient
	pvClient := clients.PersistentVolumeClient
	podClient := clients.PodClient
	stsClient := clients.StatefulSetClient

	sourceSC := scClient.Get(ctx, storageClass)
	if sourceSC.HasError() {
		return delFunc, sourceSC.GetError()
	}
	targetSC := scClient.Get(ctx, vms.TargetSC)
	if targetSC.HasError() {
		return delFunc, targetSC.GetError()
	}

	stsConf := testcore.VolumeMigrateStsConfig(storageClass, "1Gi", vms.VolumeNumber, int32(vms.PodNumber), "", vms.Image) // #nosec G115
	stsTmpl := stsClient.MakeStatefulSet(stsConf)
	// Creating Statefulset
	log.Println("Creating Statefulset")
	sts := stsClient.Create(ctx, stsTmpl)
	if sts.HasError() {
		return delFunc, sts.GetError()
	}
	sts = sts.Sync(ctx)
	if sts.HasError() {
		return delFunc, sts.GetError()
	}

	var pvNames []string
	podList, err := sts.GetPodList(ctx)
	if err != nil {
		return delFunc, err
	}
	g, _ := errgroup.WithContext(ctx)
	for _, pod := range podList.Items {
		pod := pod
		for _, volume := range pod.Spec.Volumes {
			volume := volume
			if volume.PersistentVolumeClaim != nil {
				g.Go(func() error {
					return vms.validateSTS(log, pvcClient, ctx, volume, err, pvNames, pvClient, stsConf, podClient, pod)
				})
			}
		}
	}

	delFunc = func(_ func() error) func() error {
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
		return delFunc, err
	}

	if vms.Flag {
		return delFunc, nil
	}

	log.Println("Deleting old Statefulset")
	deletionOrphan := metav1.DeletePropagationOrphan
	delSts := stsClient.DeleteWithOptions(ctx, sts.Set, metav1.DeleteOptions{PropagationPolicy: &deletionOrphan})
	if delSts.HasError() {
		return delFunc, delSts.GetError()
	}

	log.Println("Deleting pods")
	for _, pod := range podList.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				log.Println("Deleting PVC")
				pvc := pvcClient.Get(ctx, volume.PersistentVolumeClaim.ClaimName)
				if pvc.HasError() {
					return delFunc, pvc.GetError()
				}
				delPVC := pvcClient.Delete(ctx, pvc.Object)
				if delPVC.HasError() {
					return delFunc, delPVC.GetError()
				}
			}
		}
		pod := pod
		podClient.Delete(ctx, &pod)
	}

	newStsConf := testcore.VolumeMigrateStsConfig(vms.TargetSC, "1Gi", vms.VolumeNumber, int32(vms.PodNumber), "", vms.Image) // #nosec G115
	newStsTmpl := stsClient.MakeStatefulSet(newStsConf)
	// Creating new Statefulset
	log.Println("Creating new Statefulset")
	newSts := stsClient.Create(ctx, newStsTmpl)
	if newSts.HasError() {
		return delFunc, newSts.GetError()
	}
	newSts = newSts.Sync(ctx)
	if newSts.HasError() {
		return delFunc, newSts.GetError()
	}

	newPodList, err := newSts.GetPodList(ctx)
	if err != nil {
		return delFunc, err
	}

	for _, pod := range newPodList.Items {
		// Check if hash sum is correct
		sum := fmt.Sprintf("%s0/writer-%d.sha512", newStsConf.MountPath, 0)
		writer := bytes.NewBufferString("")
		log.Info("Checker pod: ", pod.Name)
		pod := pod
		if err := podClient.Exec(ctx, &pod, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
			return delFunc, err
		}
		if strings.Contains(writer.String(), "OK") {
			log.Info("Hashes match")
		} else {
			return delFunc, fmt.Errorf("hashes don't match")
		}
	}

	return delFunc, nil
}

func (vms *VolumeMigrateSuite) validateSTS(log *log.Entry, pvcClient *pvc.Client, ctx context.Context, volume v1.Volume,
	err error, pvNames []string, pvClient *pv.Client, stsConf *statefulset.Config, podClient *pod.Client, pod v1.Pod) error {

	log.Println("Getting PVC")
	pvcObj := pvcClient.Get(ctx, volume.PersistentVolumeClaim.ClaimName)
	if pvcObj.HasError() {
		return pvcObj.GetError()
	}
	err = pvcClient.WaitForAllToBeBound(ctx)
	if err != nil {
		return err
	}

	log.Println("Getting PV")
	pvName := pvcObj.Object.Spec.VolumeName
	pvNames = append(pvNames, pvName)
	pvObj := pvClient.Get(ctx, pvName)
	if pvObj.HasError() {
		return pvObj.GetError()
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

	pvObj.Object.Annotations["migration.storage.dell.com/migrate-to"] = vms.TargetSC
	log.Println("Updating PV")
	updatedPV := pvClient.Update(ctx, pvObj.Object)
	if updatedPV.HasError() {
		return updatedPV.GetError()
	}

	log.Println("Waiting PV to create")
	err = pvClient.WaitPV(ctx, pvName+"-to-"+vms.TargetSC)
	if err != nil {
		return err
	}
	log.Println("pvObj", pvName+"-to-"+vms.TargetSC, "seems good")
	return nil
}

// GetObservers returns all observers
func (*VolumeMigrateSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return common.GetAllObservers(obsType)
}

// GetClients creates and returns pvc, pv, sc, pod, statefulset, va, metrics clients
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

// GetNamespace returns volume migrate test suite namespace
func (*VolumeMigrateSuite) GetNamespace() string {
	return "migration-test"
}

// GetName returns volume migrate test suite name
func (vms *VolumeMigrateSuite) GetName() string {
	if vms.Description != "" {
		return vms.Description
	}
	return "VolumeMigrationSuite"
}

// Parameters returns formatted string of parameters
func (vms *VolumeMigrateSuite) Parameters() string {
	return fmt.Sprintf("{Target storageclass: %s, volumes: %d, pods: %d}", vms.TargetSC, vms.VolumeNumber, vms.PodNumber)
}
