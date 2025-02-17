package common

import (
	"context"

	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kfake "k8s.io/client-go/kubernetes/fake"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
)

func GetAllObservers(obsType observer.Type) []observer.Interface {
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

func ShouldWaitForFirstConsumer(ctx context.Context, storageClass string, pvcClient *pvc.Client) (bool, error) {
	s, err := pvcClient.ClientSet.StorageV1().StorageClasses().Get(ctx, storageClass, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return *s.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer, nil
}

type FakeExtendedCoreV1 struct {
	typedcorev1.CoreV1Interface
	restClient rest.Interface
}

func (c *FakeExtendedCoreV1) RESTClient() rest.Interface {
	if c.restClient == nil {
		c.restClient = &restfake.RESTClient{}
	}
	return c.restClient
}

type FakeExtendedClientset struct {
	*kfake.Clientset
}

func (f *FakeExtendedClientset) CoreV1() typedcorev1.CoreV1Interface {
	return &FakeExtendedCoreV1{f.Clientset.CoreV1(), nil}
}

func NewFakeClientsetWithRestClient(objs ...runtime.Object) *FakeExtendedClientset {
	return &FakeExtendedClientset{kfake.NewSimpleClientset(objs...)}
}
