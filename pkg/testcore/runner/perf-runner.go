/*
 *
 * Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/reporter"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	"github.com/dell/cert-csi/pkg/utils"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// SuiteRunner contains configuration to run performance test suite
type SuiteRunner struct {
	*Runner
	CoolDownPeriod        int
	StartHookPath         string
	ReadyHookPath         string
	FinishHookPath        string
	DriverNSHealthMetrics string
	sequentialExecution   bool
	NoMetrics             bool
	NoReport              bool
	IterationNum          int
	Duration              time.Duration
	ScDBs                 []*store.StorageClassDB
}

// TestResult stores test result
type TestResult string

const (
	// SUCCESS represent success result
	SUCCESS TestResult = "SUCCESS"
	// FAILURE represents failure result
	FAILURE TestResult = "FAILURE"
	// Threshold represents threshold value
	Threshold = 0.9
)

func checkValidNamespace(driverNs string, runner *Runner) {
	// Check if driver namespace exists
	if driverNs != "" {
		nsEx, nsErr := runner.KubeClient.NamespaceExists(context.Background(), driverNs)
		if nsErr != nil {
			logrus.Errorf("Can't check existence of namespace; error=%v", nsErr)
		}
		if !nsEx {
			logrus.Fatalf("Can't find namespace %s", driverNs)
		}
	}
}

// NewSuiteRunner creates and returns SuiteRunner
func NewSuiteRunner(configPath, driverNs, startHook, readyHook, finishHook, observerType, longevity string, driverNSHealthMetrics string,
	timeout int, cooldown int, sequentialExecution, noCleanup, noCleanupOnFail, noMetrics bool, noReport bool, scDBs []*store.StorageClassDB,
) *SuiteRunner {
	runner := getSuiteRunner(
		configPath,
		driverNs,
		observerType,
		timeout,
		noCleanup,
		noCleanupOnFail,
		noReport,
		&K8sClient{},
	)
	for _, scDB := range scDBs {
		// Checking storage if storageClass exists
		scEx, scErr := runner.KubeClient.StorageClassExists(context.Background(), scDB.StorageClass)
		if scErr != nil {
			logrus.Errorf("Can't check existence of storageClass; error=%v", scErr)
		}
		if !scEx {
			logrus.Fatalf("Can't find storage class %s", scDB.StorageClass)
		}
		generateTestRunDetails(scDB, runner.KubeClient, runner.Config.Host)
	}

	// Parse the longevity
	iterNum := -1
	var duration time.Duration
	extendedDur, err := utils.ParseDuration(longevity)
	if err != nil {
		// Timeout wrongly formatted, try to convert to just an int
		if parseInt, err := strconv.Atoi(longevity); err == nil {
			iterNum = parseInt
			logrus.Infof("Running %d iteration(s)", iterNum)
		} else {
			logrus.Errorf("Can't launch %s iterations, using default 1", longevity)
			iterNum = 1
		}
	}
	if extendedDur != nil {
		duration = extendedDur.Duration()
		logrus.Infof("Running longevity for %d week(s) %d day(s) %d hour(s) %d minute(s) %d second(s)", extendedDur.Weeks, extendedDur.Days, extendedDur.Hours, extendedDur.Minutes, extendedDur.Seconds)
	}

	checkValidNamespace(driverNs, runner)
	checkValidNamespace(driverNSHealthMetrics, runner)

	return &SuiteRunner{
		&Runner{
			Config:          runner.Config,
			DriverNamespace: runner.DriverNamespace,
			KubeClient:      runner.KubeClient,
			Timeout:         runner.Timeout,
			NoCleanupOnFail: runner.NoCleanupOnFail,
			ObserverType:    runner.ObserverType,
			noCleaning:      runner.noCleaning,
			noreport:        runner.noreport,
		},
		cooldown,
		startHook,
		readyHook,
		finishHook,
		driverNSHealthMetrics,
		sequentialExecution,
		noMetrics,
		noReport,
		iterNum,
		duration,
		scDBs,
	}
}

// ExecuteSuite runs the test suite
func ExecuteSuite(iterCtx context.Context, num int, suites map[string][]suites.Interface, suite suites.Interface, sr *SuiteRunner, scDB *store.StorageClassDB, c chan os.Signal) {
	db := scDB.DB

	var logger *logrus.Entry
	if len(suites) > 1 {
		logger = logrus.WithFields(logrus.Fields{
			"name": suite.GetName(),
			"sc":   scDB.StorageClass,
			"num":  strconv.Itoa(num),
		})
	} else { // we don't need goroutine tracking for one suite at a time
		logger = logrus.NewEntry(logrus.StandardLogger())
	}

	ctx := context.WithValue(iterCtx, utils.LoggerContextKey, logger)
	log := utils.GetLoggerFromContext(ctx)

	// Create and save current test case
	testCase := &store.TestCase{
		Name:           suite.GetName(),
		Parameters:     suite.Parameters(),
		StartTimestamp: time.Now(),
		RunID:          scDB.TestRun.ID,
	}
	if dbErr := db.SaveTestCase(testCase); dbErr != nil {
		log.Errorf("Can't save test case to database; error=%v", dbErr)
	}

	log.Infof("Starting %s with %s storage class", color.CyanString(suite.GetName()), color.CyanString(scDB.StorageClass))
	startTime := time.Now()

	testResult, err := runSuite(ctx, suite, sr, testCase, db, scDB.StorageClass, c)
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
		if err == nil {
			err = fmt.Errorf("unknown error encountered")
		}
		if saveErr := db.FailedTestCase(testCase, time.Now(), err.Error()); saveErr != nil {
			log.Errorf("Can't save test case; error=%v", saveErr)
		}
		result = color.RedString(string(testResult))
	}
	elapsed := time.Since(startTime)

	log.Infof("%s: %s in %s", result,
		color.CyanString(suite.GetName()), color.HiYellowString(fmt.Sprint(elapsed)))

	if sr.IsStopped() {
		log.Debug("Suite range stopped")
	}

	if sr.CoolDownPeriod != 0 {
		log.Infof("sleeping %d seconds before next iteration", sr.CoolDownPeriod)
		sleepTimer := time.After(time.Duration(sr.CoolDownPeriod) * time.Second)
		sigCont := make(chan os.Signal, 1)
		signal.Notify(sigCont, os.Interrupt,
			syscall.SIGCONT,
		)
		select {
		case <-sleepTimer:
			log.Info("Continuing")
			signal.Stop(sigCont)
		case <-sigCont:
			log.Info("Skipping cooldown")
			signal.Stop(sigCont)
		}
	}
}

// RunSuites runs test suites
func (sr *SuiteRunner) RunSuites(suites map[string][]suites.Interface) {
	sr.SucceededSuites = 0.0
	defer func() {
		totalNumberOfSuites := 0
		for _, v := range suites {
			totalNumberOfSuites += len(v)
		}

		sr.SucceededSuites = sr.SucceededSuites / float64(totalNumberOfSuites*sr.IterationNum)
		sr.Close()
	}()

	for _, scDB := range sr.ScDBs {
		tempTestRun := scDB
		trErr := scDB.DB.SaveTestRun(&tempTestRun.TestRun)
		if trErr != nil {
			logrus.Errorf("Can't save test run; error=%v", trErr)
		}
	}
	if sr.Duration.Nanoseconds() > 0 {
		time.AfterFunc(sr.Duration, func() {
			sr.stop = true
		})
	}

	var iterCtx context.Context
	var c chan os.Signal
	iterCtx, c = sr.runFlowManagementGoroutine()
	iter := 1

	var charExecution byte
	if sr.sequentialExecution {
		charExecution = 's'
	} else {
		charExecution = 'p'
	}
	func() {
		for {
			select {
			case <-iterCtx.Done():
				if sr.stop {
					return
				}
				iterCtx, c = sr.runFlowManagementGoroutine()
				break
			default:
			}

			logrus.Infof(color.HiYellowString("\t*** ITERATION NUMBER %d ***\t"), iter)

			switch charExecution {
			case 'p':
				scErrs := errgroup.Group{}
				for _, scDB := range sr.ScDBs {
					scDB := scDB // https://golang.org/doc/faq#closures_and_goroutines
					scErrs.Go(func() error {
						suiteErr := errgroup.Group{}
						for i, suite := range suites[scDB.StorageClass] {
							num := i
							suite := suite
							suiteErr.Go(func() error {
								ExecuteSuite(iterCtx, num, suites, suite, sr, scDB, c)
								return nil
							})
						}
						err := suiteErr.Wait()
						if err != nil {
							return err
						}
						return nil
					})
				}
				err := scErrs.Wait()
				if err != nil {
					break
				}

			case 's':
				for _, scDB := range sr.ScDBs {
					scDB := scDB // https://golang.org/doc/faq#closures_and_goroutines

					for i, suite := range suites[scDB.StorageClass] {
						num := i
						suite := suite
						ExecuteSuite(iterCtx, num, suites, suite, sr, scDB, c)
					}
				}
			}

			if sr.IterationNum > 0 {
				if iter >= sr.IterationNum {
					break
				}
			}

			var kubeClient *k8sclient.KubeClient
			for {
				var kubeErr error
				logrus.Infof("Trying to connect to cluster...")
				kubeClient, kubeErr = k8sclient.NewKubeClient(sr.Config, sr.Timeout)
				if kubeErr != nil {
					logrus.Errorf("Couldn't create new kubernetes client. Error = %v", kubeErr)
					time.Sleep(10 * time.Second)
				} else {
					break
				}
			}
			sr.KubeClient = kubeClient
			iter++
		}
	}()
	sr.IterationNum = iter
}

func (sr *SuiteRunner) runFlowManagementGoroutine() (context.Context, chan os.Signal) {
	iterCtx, cancelIter := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
		syscall.SIGINT,  // Ctrl+C
	)
	go func(sr *SuiteRunner) {
		_, ok := <-c
		if !ok {
			return
		}
		logrus.Infof("Received termination signal, exiting asap")
		fmt.Println("Do you want to stop test run or current iteration? (R)un/(i)teration")
		readerStop := bufio.NewReader(os.Stdin)
		fmt.Print("-> ")
		charStop, _, err := readerStop.ReadRune()
		if err != nil {
			logrus.Error(err)
		}

		fmt.Println("Do you want cleanup namespace? (Y)es/(n)o")
		readerCleanup := bufio.NewReader(os.Stdin)
		fmt.Print("-> ")
		charCleanup, _, err := readerCleanup.ReadRune()
		if err != nil {
			logrus.Error(err)
		}
		switch charCleanup {
		case 'n', 'N':
			sr.NoCleaning()
			break
		}

		switch charStop {
		case 'i', 'I':
			cancelIter()
			break
		default:
			sr.Stop()
			cancelIter()
			signal.Stop(c)
			close(c)
		}
	}(sr)
	return iterCtx, c
}

func runSuite(ctx context.Context, suite suites.Interface, sr *SuiteRunner, testCase *store.TestCase, db *store.SQLiteStore, storageClass string, _ chan os.Signal) (res TestResult, resErr error) {
	log := utils.GetLoggerFromContext(ctx)

	startTime := time.Now()
	defer func() { sr.allTime += time.Since(startTime) }()

	sr.runNum++

	if err := runHook(sr.StartHookPath, "Start Hook"); err != nil {
		return FAILURE, fmt.Errorf("can't run start hook; error=%s", err.Error())
	}

	var delFunc func() error

	// Creating new namespace
	namespace, nsErr := sr.KubeClient.CreateNamespaceWithSuffix(ctx, suite.GetNamespace())
	if nsErr != nil {
		return FAILURE, fmt.Errorf("can't create namespace; error=%s", nsErr.Error())
	}

	// Get needed clients for the current suite
	clients, clientErr := suite.GetClients(namespace.Name, sr.KubeClient)
	if clientErr != nil {
		return FAILURE, fmt.Errorf("can't get suite's clients; error=%s", clientErr.Error())
	}

	var obs *observer.Runner
	if !sr.NoMetrics {
		// Create new observer runner, using list of important observers
		observers := suite.GetObservers(sr.ObserverType)
		obs = observer.NewObserverRunner(observers, clients, db, testCase, sr.DriverNamespace, sr.ShouldClean(SUCCESS))
		if obsErr := obs.Start(ctx); obsErr != nil {
			return FAILURE, fmt.Errorf("can't create observer; error=%s", obsErr.Error())
		}
	}

	defer func() {
		// So we don't lose log fields when first ctx cancelled
		ctx := context.WithValue(context.Background(), utils.LoggerContextKey, log)
		ctx, cancel := context.WithCancel(ctx)

		skipCh := make(chan os.Signal, 1)
		signal.Notify(skipCh, os.Interrupt,
			syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
			syscall.SIGINT,  // Ctrl+C
		)
		go func(_ *SuiteRunner) {
			_, ok := <-skipCh
			if !ok {
				return
			}
			log.Infof("Received termination signal, skipping ns deletion")
			cancel()
		}(sr)

		// Cleanup after test
		shouldClean := sr.ShouldClean(res)
		if shouldClean {
			log.Infof("Deleting all resources in namespace %s", namespace.Name)
			delTime := time.Now()
			if nsErr = sr.KubeClient.DeleteNamespace(ctx, namespace.Name); nsErr != nil {
				res = FAILURE
				resErr = fmt.Errorf("can't delete namespace; error=%v", nsErr.Error())
				sr.delTime += time.Since(delTime)
				return
			}
			if delFunc != nil {
				log.Info("Deletion callback function is not empty, calling it")
				err := delFunc()
				if err != nil {
					res = FAILURE
					resErr = fmt.Errorf("callback deletion function failed; error=%v", err.Error())
					sr.delTime += time.Since(delTime)
					return
				}
			}
			sr.delTime += time.Since(delTime)
		}
		if !sr.NoMetrics {
			obs.ShouldClean = shouldClean
			err := obs.Stop()
			if err != nil {
				log.Warnf("Won't be deleting %s namespace", namespace.Name)
				log.Errorf("Cancelling following iterations, stopping immediately")
				sr.stop = true // We should stop any further suites or iterations
				res = FAILURE
				resErr = fmt.Errorf("can't stop observers; error=%s", err.Error())
				return
			}
		}

		if err := runHook(sr.FinishHookPath, "Finish Hook"); err != nil {
			res = FAILURE
			resErr = fmt.Errorf("can't run finish hook; error=%s", err.Error())
			return
		}
	}()

	// Run the current suite
	runTime := time.Now()
	var err error
	delFunc, err = suite.Run(ctx, storageClass, clients)
	if err != nil {
		sr.runTime += time.Since(runTime)
		return FAILURE, fmt.Errorf("suite %s failed; error=%s", suite.GetName(), err.Error())
	}
	sr.runTime += time.Since(runTime)

	if err := runHook(sr.ReadyHookPath, "Ready Hook"); err != nil {
		return FAILURE, fmt.Errorf("can't run ready hook; error=%s", err.Error())
	}

	return SUCCESS, nil
}

func runHook(startHook, hookName string) error {
	if startHook == "" {
		return nil
	}
	cmdPath, err := filepath.Abs(startHook)
	if err != nil {
		logrus.Error(err)
		return err
	}
	ext := filepath.Ext(cmdPath)
	var cmd *exec.Cmd
	if ext == ".sh" {
		cmd = exec.Command("bash", cmdPath) // #nosec
	} else {
		cmd = exec.Command(cmdPath) // #nosec
	}
	c, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Infof("%s: %s", hookName, c)

	return nil
}

// NoCleaning sets noCleaning flag to true
func (sr *SuiteRunner) NoCleaning() {
	sr.noCleaning = true
}

// ShouldClean calls common clean function
func (sr *SuiteRunner) ShouldClean(suiteRes TestResult) (res bool) {
	// calling common clean function
	res = shouldClean(sr.NoCleanupOnFail, suiteRes, sr.noCleaning)
	return res
}

// IsStopped returns true if test suite run has stopped
func (sr *SuiteRunner) IsStopped() bool {
	return sr.stop
}

// Stop stops test suite run
func (sr *SuiteRunner) Stop() {
	sr.stop = true
}

// Close closes all databases
func (sr *SuiteRunner) Close() {
	// Closing all databases
	if !sr.noreport {
		err := reporter.GenerateReportsFromMultipleDBs([]reporter.ReportType{
			reporter.XMLReport,
			reporter.TabularReport,
		}, sr.ScDBs)
		if err != nil {
			logrus.Errorf("Can't generate reports; error=%v", err)
		}
	}

	for _, scDB := range sr.ScDBs {
		if !sr.NoMetrics && !sr.noreport {
			err := reporter.GenerateAllReports(sr.ScDBs)
			if err != nil {
				logrus.Errorf("Can't generate reports; error=%v", err)
			}
		}

		err := scDB.DB.Close()
		if err != nil {
			logrus.Errorf("Can't close database; error=%v", err)
		}
	}

	logrus.Infof("Avg time of a run:\t %.2fs", sr.runTime.Seconds()/float64(sr.runNum))
	logrus.Infof("Avg time of a del:\t %.2fs", sr.delTime.Seconds()/float64(sr.runNum))
	logrus.Infof("Avg time of all:\t %.2fs", sr.allTime.Seconds()/float64(sr.runNum))
	if sr.SucceededSuites > Threshold {
		logrus.Infof("During this run %.1f%% of suites succeeded", sr.SucceededSuites*100)
	} else {
		logrus.Errorf("During this run %.1f%% of suites succeeded", sr.SucceededSuites*100)
	}
}
