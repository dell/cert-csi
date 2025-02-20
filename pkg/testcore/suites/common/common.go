package common

import (
	"context"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func ValidateCustomName(name string, volumes int) bool {
	// If no. of volumes is only 1 then we will take custom name else no.
	if volumes == 1 && len(name) != 0 {
		return true
	}
	// we will use custom name only if number of volumes is 1 else we will discard custom name
	return false
}
