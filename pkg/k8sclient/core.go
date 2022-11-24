package k8sclient

import (
	"cert-csi/pkg/k8sclient/resources/csistoragecapacity"
	"cert-csi/pkg/k8sclient/resources/metrics"
	"cert-csi/pkg/k8sclient/resources/node"
	"cert-csi/pkg/k8sclient/resources/pod"
	"cert-csi/pkg/k8sclient/resources/pv"
	"cert-csi/pkg/k8sclient/resources/pvc"
	rg "cert-csi/pkg/k8sclient/resources/replicationgroup"
	"cert-csi/pkg/k8sclient/resources/sc"
	"cert-csi/pkg/k8sclient/resources/statefulset"
	"cert-csi/pkg/k8sclient/resources/va"
	"cert-csi/pkg/k8sclient/resources/volumegroupsnapshot"
	snapv1 "cert-csi/pkg/k8sclient/resources/volumesnapshot/v1"
	snapbeta "cert-csi/pkg/k8sclient/resources/volumesnapshot/v1beta1"
	contentv1 "cert-csi/pkg/k8sclient/resources/volumesnapshotcontent/v1"
	contentbeta "cert-csi/pkg/k8sclient/resources/volumesnapshotcontent/v1beta1"
	"cert-csi/pkg/utils"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	vgsAlpha "github.com/dell/csi-volumegroup-snapshotter/api/v1"
	replalpha "github.com/dell/csm-replication/api/v1alpha1"
	"github.com/fatih/color"
	apiExtensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"

	snapclient "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

const (
	// NamespacePoll is a poll interval for Namespace tests
	NamespacePoll = 2 * time.Second
	// NamespaceTimeout is a timeout interval for Namespace operations
	NamespaceTimeout = 1800 * time.Second
)

type KubeClientInterface interface {
	CreateStatefulSetClient(namespace string) (*statefulset.Client, error)
	CreatePVCClient(namespace string) (*pvc.Client, error)
	CreatePodClient(namespace string) (*pod.Client, error)
	CreateVaClient(namespace string) (*va.Client, error)
	CreateMetricsClient(namespace string) (*metrics.Client, error)
	CreateNamespace(namespace string) (*v1.Namespace, error)
	CreateNamespaceWithSuffix(namespace string) (*v1.Namespace, error)
	DeleteNamespace(namespace string) error
	StorageClassExists(storageClass string) (bool, error)
	NamespaceExists(namespace string) (bool, error)
	CreateSCClient(namespace string) (*sc.Client, error)
	CreateRGClient(namespace string) (*rg.Client, error)
	CreateCSISCClient(namespace string) (*csistoragecapacity.Client, error)
	Timeout() int
}

// KubeClient is a central entity of framework, that gives access to kubernetes client handle
type KubeClient struct {
	ClientSet   kubernetes.Interface
	Config      *rest.Config
	VersionInfo *version.Info
	timeout     int
	mutex       sync.Mutex
	Minor       int
}

type Clients struct {
	PVCClient              *pvc.Client
	PodClient              *pod.Client
	StatefulSetClient      *statefulset.Client
	VaClient               *va.Client
	MetricsClient          *metrics.Client
	SnapClientGA           *snapv1.SnapshotClient
	SnapClientBeta         *snapbeta.SnapshotClient
	PersistentVolumeClient *pv.Client
	NodeClient             *node.Client
	SCClient               *sc.Client
	RgClient               *rg.Client
	VgsClient              *volumegroupsnapshot.Client
	KubeClient             *KubeClient
	CSISCClient            *csistoragecapacity.Client
}

// NewKubeClient is a KubeClient constructor, that creates new instance of KubeClient from provided config
func NewKubeClient(config *rest.Config, timeout int) (*KubeClient, error) {
	logrus.Debugf("Creating new KubeClient")
	if config == nil {
		return nil, fmt.Errorf("config can't be nil")
	}

	clientset, configErr := kubernetes.NewForConfig(config)
	if configErr != nil {
		return nil, configErr
	}

	var versionInfo *version.Info
	var minor int
	if config.Host != "" {
		var err error
		versionInfo, err = clientset.Discovery().ServerVersion()
		if err != nil {
			return nil, fmt.Errorf("kubernetes API server version is not available: %v", err)
		}
		re := regexp.MustCompile("[0-9]+")
		minor, err = strconv.Atoi(re.FindAllString(versionInfo.Minor, -1)[0])
		if err != nil {
			return nil, err
		}
	}

	kubeClient := &KubeClient{
		ClientSet:   clientset,
		Config:      config,
		VersionInfo: versionInfo,
		timeout:     timeout,
		Minor:       minor,
	}

	logrus.Info("Created new KubeClient")
	return kubeClient, nil
}

// NewRemoteKubeClient is a KubeClient constructor, that creates new instance of KubeClient from provided config
func NewRemoteKubeClient(config *rest.Config, timeout int) (*KubeClient, error) {
	logrus.Debugf("Creating new Remote KubeClient")
	if config == nil {
		return nil, fmt.Errorf("remote config can't be nil")
	}

	clientset, configErr := kubernetes.NewForConfig(config)
	if configErr != nil {
		return nil, configErr
	}

	NewkubeClient := &KubeClient{
		ClientSet: clientset,
		Config:    config,
		timeout:   timeout,
	}

	logrus.Info("Created new RemoteKubeClient")
	return NewkubeClient, nil
}

// CreateStatefulSetClient creates a new instance of StatefulSetClient in supplied namespace
func (c *KubeClient) CreateStatefulSetClient(namespace string) (*statefulset.Client, error) {
	stsclient := &statefulset.Client{
		Interface: c.ClientSet.AppsV1().StatefulSets(namespace),
		ClientSet: c.ClientSet,
		Namespace: namespace,
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created StatefulSet client in %s namespace", namespace)
	return stsclient, nil
}

// CreatePVCClient creates a new instance of Client in supplied namespace
func (c *KubeClient) CreatePVCClient(namespace string) (*pvc.Client, error) {
	pvcc := &pvc.Client{
		Interface: c.ClientSet.CoreV1().PersistentVolumeClaims(namespace),
		ClientSet: c.ClientSet,
		Namespace: namespace,
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created PersistentVolumeClaim client in %s namespace", namespace)
	return pvcc, nil
}

// CreateSCClient creates storage class client
func (c *KubeClient) CreateSCClient() (*sc.Client, error) {
	scc := &sc.Client{
		Interface: c.ClientSet.StorageV1().StorageClasses(),
		ClientSet: c.ClientSet,
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created Storageclass client")
	return scc, nil
}

func (c *KubeClient) CreatePVClient() (*pv.Client, error) {
	pvc := &pv.Client{
		Interface: c.ClientSet.CoreV1().PersistentVolumes(),
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created PersistentVolumeclient ")
	return pvc, nil
}

func (c *KubeClient) CreateNodeClient() (*node.Client, error) {
	node := &node.Client{
		Interface: c.ClientSet.CoreV1().Nodes(),
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created NodeClient ")
	return node, nil
}

func (c *KubeClient) CreatePodClient(namespace string) (*pod.Client, error) {
	podc := &pod.Client{
		Interface: c.ClientSet.CoreV1().Pods(namespace),
		ClientSet: c.ClientSet,
		Config:    c.Config,
		Namespace: namespace,
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created Pod client in %s namespace", namespace)
	return podc, nil
}

func (c *KubeClient) CreateVaClient(namespace string) (*va.Client, error) {
	vac := &va.Client{
		Interface: c.ClientSet.StorageV1().VolumeAttachments(),
		Namespace: namespace,
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created VA client in %s namespace", namespace)
	return vac, nil
}

func (c *KubeClient) CreateMetricsClient(namespace string) (*metrics.Client, error) {
	cset, err := metricsclientset.NewForConfig(c.Config)

	if err != nil {
		return nil, err
	}
	mc := &metrics.Client{
		Interface: cset,
		Namespace: namespace,
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created Metrics client in %s namespace", namespace)
	return mc, nil
}

func (c *KubeClient) CreateRGClient() (*rg.Client, error) {
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiExtensionsv1.AddToScheme(scheme))
	utilruntime.Must(replalpha.AddToScheme(scheme))

	k8sClient, err := client.New(c.Config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	rgc := &rg.Client{
		Interface: k8sClient,
		Timeout:   c.timeout,
	}

	logrus.Debugf("Created new replication group client")

	return rgc, nil
}

func (c *KubeClient) CreateVGSClient() (*volumegroupsnapshot.Client, error) {
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiExtensionsv1.AddToScheme(scheme))
	utilruntime.Must(vgsAlpha.AddToScheme(scheme))

	k8sClient, err := client.New(c.Config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	vgs := &volumegroupsnapshot.Client{
		Interface: k8sClient,
		Timeout:   c.timeout,
	}

	logrus.Debugf("Created new volume group snapshot client")

	return vgs, nil
}

func (c *KubeClient) CreateSnapshotGAClient(namespace string) (*snapv1.SnapshotClient, error) {
	cset, err := snapclient.NewForConfig(c.Config)

	if err != nil {
		return nil, err
	}

	sc := &snapv1.SnapshotClient{
		Interface: cset.SnapshotV1().VolumeSnapshots(namespace),
		Namespace: namespace,
		Timeout:   c.timeout,
	}

	logrus.Debugf("Created Alpha Snapshot client in %s namespace", namespace)
	return sc, nil
}

func (c *KubeClient) CreateSnapshotBetaClient(namespace string) (*snapbeta.SnapshotClient, error) {
	cset, err := snapclient.NewForConfig(c.Config)

	if err != nil {
		return nil, err
	}

	sc := &snapbeta.SnapshotClient{
		Interface: cset.SnapshotV1beta1().VolumeSnapshots(namespace),
		Namespace: namespace,
		Timeout:   c.timeout,
	}

	logrus.Debugf("Created Beta Snapshot client in %s namespace", namespace)
	return sc, nil
}

func (c *KubeClient) CreateSnapshotContentGAClient() (*contentv1.SnapshotContentClient, error) {
	cset, err := snapclient.NewForConfig(c.Config)

	if err != nil {
		return nil, err
	}
	sc := &contentv1.SnapshotContentClient{
		Interface: cset.SnapshotV1().VolumeSnapshotContents(),
		Timeout:   c.timeout,
	}

	logrus.Debugf("Created Alpha Snapshot Content client")
	return sc, nil
}
func (c *KubeClient) CreateSnapshotContentBetaClient() (*contentbeta.SnapshotContentClient, error) {
	cset, err := snapclient.NewForConfig(c.Config)

	if err != nil {
		return nil, err
	}
	sc := &contentbeta.SnapshotContentClient{
		Interface: cset.SnapshotV1beta1().VolumeSnapshotContents(),
		Timeout:   c.timeout,
	}

	logrus.Debugf("Created Beta Snapshot Content client")
	return sc, nil
}

func (c *KubeClient) CreateCSISCClient(namespace string) (*csistoragecapacity.Client, error) {
	csiscClient := &csistoragecapacity.Client{
		Interface: c.ClientSet.StorageV1().CSIStorageCapacities(namespace),
		Namespace: namespace,
		Timeout:   c.timeout,
	}
	logrus.Debugf("Created CSIStorageCapacity client in %s namespace", namespace)
	return csiscClient, nil
}

// CreateNamespace creates new namespace with provided name
func (c *KubeClient) CreateNamespace(ctx context.Context, namespace string) (*v1.Namespace, error) {
	log := utils.GetLoggerFromContext(ctx)
	ns, err := c.ClientSet.CoreV1().Namespaces().Create(ctx,
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespace,
				Namespace: "",
			},
		},
		metav1.CreateOptions{},
	)
	if err != nil {
		log.Errorf("Unexpected error: Can't create namespace")
		return nil, err
	}

	log.Infof("Successfully created namespace %s", color.CyanString(namespace))
	return ns, nil
}

// CreateNamespaceWithSuffix creates new namespace with provided name and appends random suffix
func (c *KubeClient) CreateNamespaceWithSuffix(ctx context.Context, namespace string) (*v1.Namespace, error) {
	suffix := RandomSuffix()
	ns, err := c.CreateNamespace(ctx, namespace+"-"+suffix)
	if err != nil {
		return nil, err
	}
	return ns, nil
}

// DeleteNamespace deletes all resources inside namespace, and waits for termination
func (c *KubeClient) DeleteNamespace(ctx context.Context, namespace string) error {
	log := utils.GetLoggerFromContext(ctx)
	startTime := time.Now()
	if err := c.ClientSet.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
		return err
	}
	timeout := NamespaceTimeout
	if c.Timeout() != 0 {
		timeout = time.Duration(c.Timeout()) * time.Second
	}

	pollErr := wait.PollImmediate(NamespacePoll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Namespace deletion interrupted")
				return true, fmt.Errorf("stopped waiting to delete ns")
			default:
				break
			}

			if _, err := c.ClientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); err != nil {
				if apierrs.IsNotFound(err) {
					return true, nil
				}
				log.Errorf("Error while waiting for namespace to be terminated: %v", err)
				return false, err
			}
			return false, nil
		})

	if pollErr != nil {
		select {
		case <-ctx.Done():
			return pollErr
		default:
			log.Errorf("Failed to delete namespace: %v", pollErr)
			err := c.ForceDeleteNamespace(ctx, namespace)
			if err != nil {
				return err
			}
		}
	}

	yellow := color.New(color.FgHiYellow)
	log.Infof("Namespace %s was deleted in %s", namespace, yellow.Sprint(time.Since(startTime)))
	return nil
}

func (c *KubeClient) ForceDeleteNamespace(ctx context.Context, namespace string) error {
	// Try to send delete request one more time, ignore errors
	_ = c.ClientSet.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})

	stsclient, err := c.CreateStatefulSetClient(namespace)
	if err != nil {
		return err
	}
	err = stsclient.DeleteAll(ctx)
	if err != nil {
		return err
	}
	logrus.Debugf("All StatefulSets are gone")

	podClient, err := c.CreatePodClient(namespace)
	if err != nil {
		return err
	}
	err = podClient.DeleteAll(ctx)
	if err != nil {
		return err
	}
	logrus.Debugf("All Pods are gone")

	if c.Minor >= 17 {
		logrus.Debug("Beta here")
		k8sbeta, err := c.CreateSnapshotBetaClient(namespace)
		if err != nil {
			return err
		}
		err = k8sbeta.DeleteAll(ctx)
		if err != nil {
			return err
		}
		logrus.Debugf("All VSs are gone")
		sncont, err := c.CreateSnapshotContentBetaClient()
		if err != nil {
			return err
		}
		err = sncont.DeleteAll(ctx)
		if err != nil {
			return err
		}
		logrus.Debugf("All VSConts are gone")
	} else {
		logrus.Debug("Alpha here")
		k8salpha, err := c.CreateSnapshotGAClient(namespace)
		if err != nil {
			return err
		}
		err = k8salpha.DeleteAll(ctx)
		if err != nil {
			return err
		}
		logrus.Debugf("All VS's are gone")
		sncont, err := c.CreateSnapshotContentGAClient()
		if err != nil {
			return err
		}
		err = sncont.DeleteAll(ctx)
		if err != nil {
			return err
		}
		logrus.Debugf("All VSConts are gone")
	}

	pvcClient, err := c.CreatePVCClient(namespace)
	if err != nil {
		return err
	}
	err = pvcClient.DeleteAll(ctx)
	if err != nil {
		return err
	}
	logrus.Debugf("All PVCs are gone")

	pvClient, err := c.CreatePVClient()
	if err != nil {
		return err
	}
	err = pvClient.DeleteAll(ctx)
	if err != nil {
		return err
	}
	logrus.Debugf("All PVs are gone")
	return nil
}

func (c *KubeClient) SnapshotClassExists(snapClass string) (bool, error) {
	cset, err := snapclient.NewForConfig(c.Config)

	if err != nil {
		return false, err
	}
	_, err = cset.SnapshotV1beta1().VolumeSnapshotClasses().Get(context.Background(), snapClass, metav1.GetOptions{})
	if err != nil {
		_, err = cset.SnapshotV1().VolumeSnapshotClasses().Get(context.Background(), snapClass, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (c *KubeClient) StorageClassExists(ctx context.Context, storageClass string) (bool, error) {
	var scExists bool
	storageClasses, err := c.ClientSet.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return scExists, err
	}

	for _, sc := range storageClasses.Items {
		if sc.Name == storageClass {
			scExists = true
			break
		}
	}

	return scExists, err
}

func (c *KubeClient) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	var nsExists bool
	nsList, err := c.ClientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, ns := range nsList.Items {
		if ns.Name == namespace {
			nsExists = true
			break
		}
	}

	return nsExists, nil
}

func (c *KubeClient) Timeout() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.timeout
}

func (c *KubeClient) SetTimeout(val int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.timeout = val

}

func GetConfig(configPath string) (*rest.Config, error) {
	configPath = strings.ReplaceAll(configPath, `"`, "")
	if len(configPath) != 0 {
		logrus.Infof("Using config from %s", configPath)
	} else {
		logrus.Infof("Using default config")
	}
	config, err := GetConfigFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("can't get config from specified file; %e", err)
	}
	logrus.Infof("Successfully loaded config. Host: %s", color.CyanString(config.Host))
	return config, nil
}

// GetConfigFromFile creates *rest.Config object from provided config path
func GetConfigFromFile(kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			return nil, fmt.Errorf("can not find config file in home directory, please explicitly specify it using flags")
		}
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Errorf("Can't load config at %q, error = %v", kubeconfig, err)
		return nil, err
	}

	return config, nil
}