package suites

import (
	"cert-csi/pkg/k8sclient"
	"cert-csi/pkg/observer"
	"context"
)

type Interface interface {
	Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error)
	GetName() string
	GetObservers(obsType observer.Type) []observer.Interface
	GetClients(string, *k8sclient.KubeClient) (*k8sclient.Clients, error)
	GetNamespace() string
	Parameters() string
}
