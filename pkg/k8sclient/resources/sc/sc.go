package sc

import (
	"cert-csi/pkg/utils"
	"context"

	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
)

const (
	IsReplicationEnabled   = "replication.storage.dell.com/isReplicationEnabled"
	RemoteClusterID        = "replication.storage.dell.com/remoteClusterID"
	RemoteStorageClassName = "replication.storage.dell.com/remoteStorageClassName"
)

// Client conatins sc interface and kubeclient
type Client struct {
	// KubeClient *core.KubeClient
	Interface tcorev1.StorageClassInterface
	ClientSet kubernetes.Interface
	Timeout   int
}

// StorageClass conatins pvc client and claim
type StorageClass struct {
	Client  *Client
	Object  *v1.StorageClass
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

// Get uses client interface to make API call for getting provided StorageClass
func (c *Client) Get(ctx context.Context, name string) *StorageClass {
	log := utils.GetLoggerFromContext(ctx)
	var funcErr error
	newSC, err := c.Interface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		funcErr = err
	}

	log.Debugf("Got Storage Class %s", newSC.GetName())
	return &StorageClass{
		Client:  c,
		Object:  newSC,
		Deleted: false,
		error:   funcErr,
	}
}

func (c *Client) Create(ctx context.Context, sc *v1.StorageClass) error {
	log := utils.GetLoggerFromContext(ctx)
	_, err := c.Interface.Create(ctx, sc, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Debugf("Created %s Storage Class ", sc.GetName())
	return nil
}

func (c *Client) Delete(ctx context.Context, name string) error {
	log := utils.GetLoggerFromContext(ctx)
	err := c.Interface.Delete(ctx, name, *metav1.NewDeleteOptions(0))
	if err != nil {
		return err
	}
	log.Debugf("Deleted %s Storage Class", name)
	return nil
}

func (c *Client) MakeStorageClass(name string, provisioner string) *v1.StorageClass {
	WaitForFirstConsumer := v1.VolumeBindingWaitForFirstConsumer
	return &v1.StorageClass{
		ObjectMeta:        metav1.ObjectMeta{Name: name},
		Provisioner:       provisioner,
		VolumeBindingMode: &WaitForFirstConsumer}

}

func (c *Client) DuplicateStorageClass(name string, sourceSc *v1.StorageClass) *v1.StorageClass {
	newSc := sourceSc.DeepCopy()
	newSc.ObjectMeta = metav1.ObjectMeta{Name: name}
	return newSc
}

func (sc *StorageClass) HasError() bool {
	if sc.error != nil {
		return true
	}
	return false
}

func (sc *StorageClass) GetError() error {
	return sc.error
}
