package pv

import (
	"cert-csi/pkg/k8sclient/resources/commonparams"
	"cert-csi/pkg/k8sclient/resources/va"
	"cert-csi/pkg/utils"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	// Poll is a poll interval for PVC tests
	Poll = 2 * time.Second
	// Timeout is a timeout interval for PVC operations
	Timeout = 1800 * time.Second
)

// Client contains pvc interface and kubeclient
type Client struct {
	// KubeClient *core.KubeClient
	Interface tcorev1.PersistentVolumeInterface
	Timeout   int
}

// PersistentVolume conatins pv client and claim
type PersistentVolume struct {
	Client  *Client
	Object  *v1.PersistentVolume
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

// Delete deletes PersistentVolumeClaim from Kubernetes
func (c *Client) Delete(ctx context.Context, pv *v1.PersistentVolume) *PersistentVolume {
	var funcErr error
	err := c.Interface.Delete(ctx, pv.Name, metav1.DeleteOptions{})
	if err != nil {
		funcErr = err
	}
	log.Debugf("Deleted PV %s", pv.GetName())
	return &PersistentVolume{
		Client:  c,
		Object:  pv,
		Deleted: true,
		error:   funcErr,
	}
}

// DeleteAllPV deletes all pvs in timely manner on the basis of namespace
func (c *Client) DeleteAllPV(ctx context.Context, ns string, vaClient *va.Client) error {
	podList, podErr := c.Interface.List(ctx, metav1.ListOptions{})
	if podErr != nil {
		return podErr
	}
	log.Debugf("Deleting all PVs")
	for i, pv := range podList.Items {
		// try to delete only those PVs that has namespace matching with the suite's ns
		if val, ok := pv.Spec.PersistentVolumeSource.CSI.VolumeAttributes["csi.storage.k8s.io/pvc/namespace"]; ok && val == ns {
			log.Debugf("Deleting pv %s", pv.Name)
			err := vaClient.DeleteVaBasedOnPVName(ctx, pv.Name)
			if err != nil && !strings.Contains(err.Error(), "not found") {
				log.Errorf("Can't delete VolumeAttachment for %s; error=%v", pv.Name, err)
				continue
			}
			err = c.Delete(ctx, &podList.Items[i]).Sync(ctx).GetError()
			if err != nil {
				log.Errorf("Can't delete pv %s; error=%v", pv.Name, err)
			}
		}
	}
	return nil
}

// DeleteAll deletes all client pvs in timely manner
func (c *Client) DeleteAll(ctx context.Context) error {
	podList, podErr := c.Interface.List(ctx, metav1.ListOptions{})
	if podErr != nil {
		return podErr
	}
	log.Debugf("Deleting all PVs")
	for i, pvc := range podList.Items {
		log.Debugf("Deleting pv %s", pvc.Name)
		err := c.Delete(ctx, &podList.Items[i]).Sync(ctx).GetError()
		if err != nil {
			log.Errorf("Can't delete pv %s; error=%v", pvc.Name, err)
		}
	}
	return nil
}

// Get uses client interface to make API call for getting provided PersistentVolume
func (c *Client) Get(ctx context.Context, name string) *PersistentVolume {
	log := utils.GetLoggerFromContext(ctx)
	var funcErr error
	newPV, err := c.Interface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		funcErr = err
	}

	log.Debugf("Got PV %s", newPV.GetName())
	return &PersistentVolume{
		Client:  c,
		Object:  newPV,
		Deleted: false,
		error:   funcErr,
	}
}

// Update updates a PersistentVolume
func (c *Client) Update(ctx context.Context, pv *v1.PersistentVolume) *PersistentVolume {
	var funcErr error
	updatedPV, err := c.Interface.Update(ctx, pv, metav1.UpdateOptions{})
	if err != nil {
		funcErr = err
	}
	log.Debugf("Updated PV %s", pv.GetName())
	return &PersistentVolume{
		Client:  c,
		Object:  updatedPV,
		Deleted: false,
		error:   funcErr,
	}
}

// WaitPV waits when pv will be available
func (c *Client) WaitPV(ctx context.Context, newPVName string) error {
	log.Debugf("Waiting for PV %s to be %s", newPVName, color.GreenString("PRESENT"))
	startTime := time.Now()
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pv wait polling")
				return true, fmt.Errorf("stopped waiting to be bound")
			default:
				break
			}
			_, err := c.Interface.Get(ctx, newPVName, metav1.GetOptions{})
			if err != nil {
				log.Debugf("PV %s is still not present", newPVName)
				return false, nil
			}

			return true, nil
		})
	if pollErr != nil {
		return pollErr
	}

	yellow := color.New(color.FgHiYellow)
	log.Debugf("PV %s is present in %s", newPVName, yellow.Sprint(time.Since(startTime)))

	return nil
}

// WaitToBeBound waits for PVC to be in Bound state
func (pv *PersistentVolume) WaitToBeBound(ctx context.Context) error {
	log.Debugf("Waiting for  PV %s to be %s", pv.Object.Name, color.GreenString("PRESENT"))
	startTime := time.Now()
	timeout := Timeout
	if pv.Client.Timeout != 0 {
		timeout = time.Duration(pv.Client.Timeout) * time.Second
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pv wait polling")
				return true, fmt.Errorf("stopped waiting to be bound")
			default:
				break
			}
			_, err := pv.Client.Interface.Get(ctx, pv.Object.Name, metav1.GetOptions{})
			if err != nil {
				log.Debugf("PV %s is still not present", pv.Object.Name)
				return false, nil
			}

			return true, nil
		})
	if pollErr != nil {
		return pollErr
	}

	yellow := color.New(color.FgHiYellow)
	log.Debugf("All PVs in %s are presented in %s", pv.Object.Namespace, yellow.Sprint(time.Since(startTime)))
	return nil
}

// CheckReplicationAnnotationsForPV checks for replication related annotations and labels on PV
func (c *Client) CheckReplicationAnnotationsForPV(ctx context.Context, object *v1.PersistentVolume) error {
	pvName := object.Name
	log.Infof("Checking Replication Annotations and Labels on  PV %s", pvName)
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pv check polling")
				return true, fmt.Errorf("stopped checking pv Annotations")
			default:
				break
			}

			gotObj, err := c.Interface.Get(ctx, pvName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			found := true
			for _, v := range commonparams.LocalPVAnnotation {
				if gotObj.Annotations[v] == "" {
					log.Debugf("Annotations %s are not added for PV %s", v, pvName)
					found = false
				}
			}
			for _, v := range commonparams.LocalPVLabels {
				if gotObj.Labels[v] == "" {
					log.Debugf("Labels %s are not added for PV %s", v, pvName)
					found = false
				}
			}
			if !found {
				return false, nil
			}
			return true, nil
		})
	if pollErr != nil {
		return pollErr
	}
	return nil
}

// CheckReplicationAnnotationsForRemotePV checks for replication related annotations and labels for remote PV
func (c *Client) CheckReplicationAnnotationsForRemotePV(ctx context.Context, object *v1.PersistentVolume) error {
	pvName := object.Name
	log.Infof("Checking Replication Annotations and labels on  remote PV %s ", pvName)
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pvc wait polling")
				return true, fmt.Errorf("stopped waiting to be bound")
			default:
				break
			}
			gotObj, err := c.Interface.Get(ctx, pvName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			found := true
			for _, v := range commonparams.RemotePVAnnotations {
				if gotObj.Annotations[v] == "" {
					log.Debugf("Annotations %s are not added for remote PV %s", v, pvName)
					found = false
				}
			}
			for _, v := range commonparams.RemotePVLabels {
				if gotObj.Labels[v] == "" {
					log.Debugf("Labels %s are not added for remote PV %s ", v, pvName)
					found = false
				}
			}
			if !found {
				return false, nil
			}
			return true, nil
		})
	if pollErr != nil {
		return pollErr
	}

	return nil
}

func (pv *PersistentVolume) pollWait(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		log.Infof("Stopping pv wait polling")
		return true, fmt.Errorf("stopped waiting to be bound")
	default:
		break
	}
	if _, err := pv.Client.Interface.Get(ctx, pv.Object.Name, metav1.GetOptions{}); err != nil {
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		log.Errorf("Error while waiting for pv to be terminated: %v", err)
		return false, err
	}
	return false, nil
}

// WaitUntilGone stalls until said Pod no longer can be found in Kubernetes
func (pv *PersistentVolume) WaitUntilGone(ctx context.Context) error {
	startTime := time.Now()
	timeout := Timeout
	if pv.Client.Timeout != 0 {
		timeout = time.Duration(pv.Client.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout/100*95, func() (done bool, err error) {
		done, err = pv.pollWait(ctx)
		return done, err
	})
	if pollErr != nil {
		gotpv, err := pv.Client.Interface.Get(ctx, pv.Object.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		log.Errorf("Failed to delete pv: %v \n", pollErr)
		log.Info("Forcing finalizers cleanup")
		gotpv.SetFinalizers([]string{})
		_, er := pv.Client.Interface.Update(ctx, gotpv, metav1.UpdateOptions{})
		if er != nil {
			return er
		}
		pollErr = wait.PollImmediate(Poll, timeout/2, func() (done bool, err error) {
			done, err = pv.pollWait(ctx)
			return done, err
		})

		if pollErr != nil {
			log.Errorf("Failed to delete pv: %v \n", pollErr)
			log.Info("Forcing finalizers cleanup")
			return errors.New("failed to delete even with finalizers cleaned up")
		}
	}
	yellow := color.New(color.FgHiYellow)
	log.Debugf("Pv %s was deleted in %s", pv.Object.Name, yellow.Sprint(time.Since(startTime)))
	return nil

}

// Sync waits until PV is deleted or bound
func (pv *PersistentVolume) Sync(ctx context.Context) *PersistentVolume {
	if pv.Deleted {
		pv.error = pv.WaitUntilGone(ctx)
	} else {
		pv.error = pv.WaitToBeBound(ctx)
	}
	return pv
}

// HasError checks if PV contains error
func (pv *PersistentVolume) HasError() bool {
	if pv.error != nil {
		return true
	}
	return false
}

// GetError returns PV error
func (pv *PersistentVolume) GetError() error {
	return pv.error
}
