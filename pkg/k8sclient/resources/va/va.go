package va

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	v12 "k8s.io/client-go/kubernetes/typed/storage/v1"
)

const (
	// Poll is a poll interval for VA
	Poll = 2 * time.Second
	// Timeout is a timeout for Va operations
	Timeout = 1800 * time.Second
)

// Client is a VA client for managing VolumeAttachments
type Client struct {
	// KubeClient *core.KubeClient
	Interface v12.VolumeAttachmentInterface
	Namespace string
	Timeout   int

	CustomTimeout time.Duration
}

// WaitUntilNoneLeft waits for all volume attachments to be deleted in a namespace
func (c *Client) WaitUntilNoneLeft(ctx context.Context) error {
	log.Infof("Waiting until no Volume Attachments left in %s", c.Namespace)
	timeout := Timeout
	if c.CustomTimeout == 0 {
		if c.Timeout != 0 {
			timeout = time.Duration(c.Timeout) * time.Second
		}
	} else {
		timeout = c.CustomTimeout
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			vaList, err := c.Interface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}

			if len(vaList.Items) != 0 {
				return false, nil
			}

			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}
	log.Infof("All VolumeAttachments deleted in %s namespace", c.Namespace)
	return nil
}

// WaitUntilVaGone waits until volume attachments for a PV are deleted
func (c *Client) WaitUntilVaGone(ctx context.Context, pvName string) error {
	log.Infof("Waiting until no Volume Attachments with PV %s left", pvName)
	timeout := Timeout
	if c.CustomTimeout == 0 {
		if c.Timeout != 0 {
			timeout = time.Duration(c.Timeout) * time.Second
		}
	} else {
		timeout = c.CustomTimeout
	}

	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			vaList, err := c.Interface.List(ctx, metav1.ListOptions{
				FieldSelector: "",
			})
			if err != nil {
				return false, err
			}

			for _, va := range vaList.Items {
				if *va.Spec.Source.PersistentVolumeName == pvName {
					log.Debugf("Waiting for the volume-attachment to be deleted for :%s", pvName)
					return false, nil
				}
			}

			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}
	log.Infof("VolumeAttachment deleted")
	return nil

}
