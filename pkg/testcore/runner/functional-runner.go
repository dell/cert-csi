/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package runner

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/testcore/suites"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
)

// FunctionalSuiteRunner contains configuration for running functional test suite
type FunctionalSuiteRunner struct {
	*Runner
	noreport bool
	ScDB     *store.StorageClassDB
}

// NewFunctionalSuiteRunner creates functional suite runner instance
func NewFunctionalSuiteRunner(configPath, namespace string, timeout int, noCleanup, noCleanupOnFail bool, noreport bool,
	scDB *store.StorageClassDB, k8s K8sClientInterface,
) *FunctionalSuiteRunner {
	const observerType = "event"
	r := getSuiteRunner(
		configPath,
		namespace,
		observerType,
		timeout,
		noCleanup,
		noCleanupOnFail,
		noreport,
		k8s,
	)
	generateTestRunDetails(scDB, r.KubeClient, r.Config.Host)

	return &FunctionalSuiteRunner{
		&Runner{
			Config:          r.Config,
			DriverNamespace: r.DriverNamespace,
			KubeClient:      r.KubeClient,
			Timeout:         r.Timeout,
			NoCleanupOnFail: r.NoCleanupOnFail,
			ObserverType:    r.ObserverType,
			noCleaning:      r.noCleaning,
			noreport:        r.noreport,
		},
		noreport,
		scDB,
	}
}

// RunFunctionalSuites runs functional test suites
func (sr *FunctionalSuiteRunner) RunFunctionalSuites(suites []suites.Interface) {
	sr.SucceededSuites = 0.0
	defer sr.Close()

	trErr := sr.ScDB.DB.SaveTestRun(&sr.ScDB.TestRun)
	if trErr != nil {
		log.Errorf("Can't save test run; error=%v", trErr)
	}

	db := sr.ScDB.DB
	for _, suite := range suites {
		// Create and save current test case
		testCase := &store.TestCase{
			Name:           suite.GetName(),
			StartTimestamp: time.Now(),
			RunID:          sr.ScDB.TestRun.ID,
		}
		if dbErr := db.SaveTestCase(testCase); dbErr != nil {
			log.Errorf("Can't save test case to database; error=%v", dbErr)
		}

		startTime := time.Now()

		testResult, err := runFunctionalSuite(suite, sr, testCase, db, sr.ScDB.StorageClass)
		if err != nil {
			log.Error(err)
		}
		var result string

		if testResult == SUCCESS {
			sr.SucceededSuites++
			if saveErr := db.SuccessfulTestCase(testCase, time.Now()); saveErr != nil {
				log.Errorf("Can't save test case; error=%v", saveErr)
			}
			result = color.HiGreenString(string(testResult))
		} else {
			if saveErr := db.FailedTestCase(testCase, time.Now(), "TODO: some error"); saveErr != nil {
				log.Errorf("Can't save test case; error=%v", saveErr)
			}
			result = color.RedString(string(testResult))
		}
		elapsed := time.Since(startTime)

		log.Infof("%s: %s in %s", result,
			color.CyanString(suite.GetName()), color.HiYellowString(fmt.Sprint(elapsed)))

		if sr.IsStopped() { // Don't run next suite if stopped
			log.Debugf("Suite range stopped")
			break
		}
	}

	var kubeClient k8sclient.KubeClientInterface
	for {
		var kubeErr error
		log.Infof("Trying to connect to cluster...")
		kubeClient, kubeErr = k8sclient.NewKubeClient(sr.Config, sr.Timeout)
		if kubeErr != nil {
			log.Errorf("Couldn't create new kubernetes client. Error = %v", kubeErr)
			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}
	sr.KubeClient = kubeClient
	sr.SucceededSuites = sr.SucceededSuites / float64(len(suites))
}

func runFunctionalSuite(suite suites.Interface, sr *FunctionalSuiteRunner, testCase *store.TestCase, db store.Store, storageClass string) (res TestResult, resErr error) {
	iterCtx, cancelIter := context.WithCancel(context.Background())
	startTime := time.Now()
	defer func() {
		sr.allTime += time.Since(startTime)
	}()

	sr.runNum++

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt,
		syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
		syscall.SIGINT,  // Ctrl+C
	)
	// Go routine to listen for termination signal
	go func(sr *FunctionalSuiteRunner) {
		_, ok := <-c
		if !ok {
			return
		}
		log.Infof("Received termination signal, exiting asap")
		fmt.Printf("Do you want to cleanup namespace? (Y/n)\n")
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("-> ")
		char, _, err := reader.ReadRune()
		if err != nil {
			log.Error(err)
		}

		if char == 'n' || char == 'N' {
			sr.NoCleaning()
		}

		sr.Stop()
		cancelIter()
	}(sr)

	// Creating new namespace with unique name
	namespace, err := sr.KubeClient.CreateNamespaceWithSuffix(iterCtx, suite.GetNamespace())
	if err != nil {
		log.Errorf("can't create namespace %s: %v", namespace, err)
		return FAILURE, fmt.Errorf("can't create namespace; error=%s", err.Error())
	}

	// Cleanup namespace after test
	defer func() {
		if sr.ShouldClean(res) {
			log.Infof("Deleting all resources in namespace %s", namespace.Name)
			delTime := time.Now()
			if err := sr.KubeClient.DeleteNamespace(context.Background(), namespace.Name); err != nil {
				log.Errorf("Can't delete namespace: %v", err)
				res = FAILURE
			}
			sr.delTime += time.Since(delTime)
		}
	}()

	// Get needed clients for the current suite
	clients, clientErr := suite.GetClients(namespace.Name, sr.KubeClient)
	if clientErr != nil {
		log.Errorf("Can't get suite's clients; error=%v", clientErr)
		return FAILURE, fmt.Errorf("can't get suite's clients; error=%s", clientErr.Error())
	}

	var obs *observer.Runner
	// Create new observer runner, using list of important observers
	observers := suite.GetObservers(sr.ObserverType)
	obs = observer.NewObserverRunner(observers, clients, db, testCase, sr.DriverNamespace, false)
	if obsErr := obs.Start(iterCtx); obsErr != nil {
		log.Errorf("Error creating observer; error=%v", obsErr)
		return FAILURE, fmt.Errorf("can't create observer; error=%s", obsErr.Error())
	}

	defer func() {
		signal.Stop(c)
		close(c)

		err := obs.Stop()
		if err != nil {
			log.Errorf("Can't stop observers, error=%v", err)
			log.Errorf("Cancelling following suites, stopping immediately")
			sr.stop = true // We should stop any further suites or iterations
			res = FAILURE
			return
		}
	}()

	// Run the current suite
	runTime := time.Now()
	if _, err := suite.Run(iterCtx, storageClass, clients); err != nil {
		sr.runTime += time.Since(runTime)
		log.Errorf("Suite %s failed; error=%v", suite.GetName(), err)
		return FAILURE, fmt.Errorf("suite %s failed; error=%s", suite.GetName(), err.Error())
	}
	sr.runTime += time.Since(runTime)

	return SUCCESS, nil
}

// IsStopped returns true if test suite run has stopped
func (sr *FunctionalSuiteRunner) IsStopped() bool {
	return sr.stop
}

// Stop stops test suite run
func (sr *FunctionalSuiteRunner) Stop() {
	sr.stop = true
}

// Close logs the status of test suite run
func (sr *FunctionalSuiteRunner) Close() {
	if sr.SucceededSuites > Threshold {
		log.Infof("During this run %.1f%% of suites succeeded", sr.SucceededSuites*100)
	} else {
		log.Errorf("During this run %.1f%% of suites succeeded", sr.SucceededSuites*100)
	}
}

// NoCleaning sets noCleaning flag to true
func (sr *FunctionalSuiteRunner) NoCleaning() {
	sr.noCleaning = true
}

// ShouldClean calls common clean function
func (sr *FunctionalSuiteRunner) ShouldClean(suiteRes TestResult) (res bool) {
	// calling common clean function
	res = shouldClean(sr.NoCleanupOnFail, suiteRes, sr.noCleaning)
	return res
}
