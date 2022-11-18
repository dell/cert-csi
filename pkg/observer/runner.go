package observer

import (
	"cert-csi/pkg/k8sclient"
	"cert-csi/pkg/store"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
	"time"
)

type Type string

const (
	// BatchSize sets the number of events to be added at once
	BatchSize = 10
	// WatchTimeout to override timeout set in kubeapi-server settings, Default: 24 hours
	WatchTimeout int64 = 60 * 60 * 24

	LIST  Type = "LIST"
	EVENT Type = "EVENT"
)

type Runner struct {
	WaitGroup       sync.WaitGroup
	Observers       []Interface
	Clients         *k8sclient.Clients
	Database        store.Store
	TestCase        *store.TestCase
	PvcShare        sync.Map
	DriverNamespace string
	ShouldClean     bool
}

func NewObserverRunner(observers []Interface, clients *k8sclient.Clients,
	db store.Store, testCase *store.TestCase, driverNs string, shouldClean bool) *Runner {
	return &Runner{
		Observers:       observers,
		Clients:         clients,
		Database:        db,
		TestCase:        testCase,
		DriverNamespace: driverNs,
		ShouldClean:     shouldClean,
	}
}

func (runner *Runner) Start(ctx context.Context) error {
	for _, obs := range runner.Observers {
		runner.WaitGroup.Add(1)
		obs.MakeChannel()
		go obs.StartWatching(ctx, runner)
	}
	return nil
}

func (runner *Runner) Stop() error {
	for _, obs := range runner.Observers {
		obs.StopWatching()
	}

	// Erase map
	defer runner.PvcShare.Range(func(key interface{}, value interface{}) bool {
		runner.PvcShare.Delete(key)
		return true
	})

	// Wait for all of observers to complete
	if runner.waitTimeout(2 * time.Minute) {
		// If we are here, then some of observers haven't received ending events
		// Check if any volumeattachments remain
		mismatch := false
		pvList, err := runner.Clients.PVCClient.ClientSet.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return err
		}

		runner.PvcShare.Range(func(key interface{}, value interface{}) bool {
			pvName, ok := key.(string)
			if !ok {
				return false
			}

			for _, pv := range pvList.Items {
				if pv.Name == pvName {
					mismatch = true
					return false
				}
			}

			return true
		})

		if mismatch {
			logrus.Warn("Some pvs are still left in cluster; ")
			if runner.waitTimeout(time.Duration(len(pvList.Items)) * 10 * time.Second) {
				return fmt.Errorf("pvs are in hanging state, something's wrong")
			}
		}
	}

	return nil
}

func (runner *Runner) waitTimeout(timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		runner.WaitGroup.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

type Interface interface {
	StartWatching(context.Context, *Runner)
	StopWatching()
	GetName() string
	MakeChannel()
}
