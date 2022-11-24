package runner

import (
	"cert-csi/pkg/k8sclient"
	"cert-csi/pkg/observer"
	"cert-csi/pkg/store"
	"k8s.io/client-go/rest"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Runner struct {
	Config          *rest.Config
	DriverNamespace string
	KubeClient      *k8sclient.KubeClient
	Timeout         int
	NoCleanupOnFail bool
	SucceededSuites float64
	ObserverType    observer.Type

	noreport   bool
	noCleaning bool
	stop       bool
	allTime    time.Duration
	runTime    time.Duration
	delTime    time.Duration
	runNum     int

	sync.RWMutex
}

func getSuiteRunner(configPath, driverNs, observerType string, timeout int, noCleanup, noCleanupOnFail bool, noreport bool) *Runner {

	t := strings.ToUpper(observerType)
	correctType := (t == string(observer.EVENT)) || (t == string(observer.LIST))
	if !correctType {
		log.Fatal("Incorrect observer type")
	}

	obsType := observer.Type(t)
	log.Infof("Using %s observer type", obsType)

	// Loading config
	config, err := k8sclient.GetConfig(configPath)
	if err != nil {
		log.Error(err)
	}

	// Connecting to host and creating new Kubernetes Client
	kubeClient, kubeErr := k8sclient.NewKubeClient(config, timeout)
	if kubeErr != nil {
		log.Errorf("Couldn't create new kubernetes client. Error = %v", kubeErr)
	}

	return &Runner{
		Config:          config,
		DriverNamespace: driverNs,
		KubeClient:      kubeClient,
		Timeout:         timeout,
		NoCleanupOnFail: noCleanupOnFail,
		ObserverType:    obsType,
		noCleaning:      noCleanup,
		noreport:        noreport,
	}
}

func generateTestRunDetails(scDB *store.StorageClassDB, kubeClient *k8sclient.KubeClient, host string) {
	scDB.TestRun = store.TestRun{
		Name:           "test-run-" + k8sclient.RandomSuffix(),
		StartTimestamp: time.Now(),
		StorageClass:   scDB.StorageClass,
		ClusterAddress: host,
	}
}

func shouldClean(NoCleanupOnFail bool, suiteRes TestResult, noCleaning bool) (res bool) {
	if NoCleanupOnFail && suiteRes == FAILURE {
		res = false
	} else {
		res = !noCleaning
	}
	return res
}