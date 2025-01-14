/*
 *
 * Copyright © 2023-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package utils

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v3"
)

var (
	executionLogFile = "e2e_execution" // executionLogFile is suffix for log file
	// Home home environmental variable
	Home              = os.Getenv("HOME")
	defaultDir        = Home + "/reports/"     // defaultDir reports directory
	defaultKubeConfig = Home + "/.kube/config" // defaultKubeConfig default KUBECONFIG
)

// Constants to support e2e tests
const (
	// KubeConfigEnv Environmental Variable
	KubeConfigEnv  = "KUBECONFIG"
	k8sBinary      = "kubernetes/test/bin/e2e.test"        // k8sBinary is the actual e2e binary that we need to execute
	kubeconfigKey  = "-kubeconfig"                         // kubeconfigKey  is to provide kubeconfig
	focusStringKey = "--ginkgo.focus"                      // focusStringKey  is to provide focus tests
	skipStringKey  = "--ginkgo.skip"                       // skipStringKey  is to provide skip tests
	focusFileKey   = "--ginkgo.focus-file"                 // focusFileKey  is to provide focus test suite
	skipFileKey    = "--ginkgo.skip-file"                  // focusStringKey  is to provide focus tests
	testdriverKey  = "-storage.testdriver"                 // testdriverKey is to provide the test driver config file
	junitReportKey = "--ginkgo.junit-report"               // junitReportKey to generate the e2e report
	timeoutKey     = "--ginkgo.timeout"                    // timeoutKey is to provide the final
	XML            = ".xml"                                // XML is an extension for .xml
	IgnoreFile     = "pkg/utils/ignore.yaml"               // IgnoreFile contains the tests to be skipped
	BinaryPrefix   = "https://dl.k8s.io/"                  // BinaryPrefix Binary url prefix
	BinarySuffix   = "/kubernetes-test-linux-amd64.tar.gz" // BinarySuffix  binary url suffix
	BinaryFile     = "kubernetes-test-linux-amd64.tar.gz"  // BinaryFile downloaded binary file
)

// DownloadBinary will download the binary from the kubernetes artifactory based the version
func DownloadBinary(version string) error {
	url, err := GetURL(version)
	if err != nil {
		return err
	}
	log.Infof("Downloading tar file....")
	client := http.Client{
		Timeout: 20 * time.Second,
	}
	resp, err := client.Get(url)
	if resp.StatusCode != http.StatusOK {
		return errors.New("Unable to download tar file with return code :" + strconv.Itoa(resp.StatusCode))
	}
	if err != nil {
		log.Errorf("Unable to Dowonload tar file :%s with error: %s", url, err.Error())
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("Error closing HTTP response: %s", err.Error())
		}
	}()

	// Create the file
	out, err := os.Create(BinaryFile)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			log.Errorf("Unable to close binaryfile with error: %s", err.Error())
		}
	}(out)

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Errorf("%s", err.Error())
	}

	return err
}

// UnTarBinary will untar the kubernetes tar.gz file to get the e2e binaries
func UnTarBinary() error {
	log.Infof("Untaring...: %s", BinaryFile)
	stdout, err := exec.Command("tar", "-xvf", BinaryFile).Output()
	if err != nil {
		log.Errorf("Unable to untar binaryfile with error: %s", err.Error())
		return err
	}
	log.Infof("%s", string(stdout))
	return nil
}

// CheckKubeConfigEnv will check the environment variable KUBECONFIG and returns default KUBECONFIG if not set
func CheckKubeConfigEnv() string {
	kubeConfig := os.Getenv(KubeConfigEnv)
	if kubeConfig == "" {
		log.Infof("KUBECONFIG env varibale is not set switching to default kube config '~/.kube/config'")
		return filepath.Clean(defaultKubeConfig)
	}
	return kubeConfig
}

// FileExists will check the file existence and return true if it exists otherwise return false
func FileExists(filename string) bool {
	log.Infof("Checking file %s", filename)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// Prechecks function will check some prechecks like driver config file existence
func Prechecks(c *cli.Context) bool {
	exist := FileExists(c.String("driver-config"))
	if !exist {
		log.Errorf("Driver config file not found: %s ", c.String("driver-config"))
	}
	return exist
}

// GetURL will return the URL by including the given version
func GetURL(version string) (string, error) {
	if len(version) == 0 {
		return "", errors.New("unable to build URL with the given empty version")
	}
	return BinaryPrefix + version + BinarySuffix, nil
}

// CheckIfBinaryExists will check the existing binary version and return true if version match else return false
func CheckIfBinaryExists(version string) bool {
	c, b := exec.Command(k8sBinary, "-version"), new(strings.Builder)
	c.Stdout = b
	err := c.Run()
	if err != nil {
		log.Infof("Error wile executing version command: %s", err.Error())
		return false
	}
	if version == strings.TrimSpace(b.String()) {
		return true
	}
	return false
}

// Prerequisites function will full-fill all prerequisites
func Prerequisites(version string) error {
	err := os.MkdirAll(defaultDir, os.ModeDir)
	if err != nil {
		return err
	}
	log.Infof("Creating report directory: %s", defaultDir)
	if !CheckIfBinaryExists(version) {
		log.Infof("Kuberners e2e Binary with version:%s not found ", version)
		err = DownloadBinary(version)
		if err != nil {
			return err
		}
		err = UnTarBinary()
		if err != nil {
			return err
		}
	} else {
		log.Infof("Found the Kuberners e2e Binary with version:%s ", version)
	}

	return nil
}

// DriverConfig is used to get the storage-class name
type DriverConfig struct {
	StorageClass struct {
		Class string `yaml:"FromExistingClassName"`
	} `yaml:"StorageClass"`
}

// ReadTestDriverConfig will read the driver config
func ReadTestDriverConfig(driverconfig string) string {
	file, _ := os.ReadFile(filepath.Clean(driverconfig))
	var DriverConfig DriverConfig
	err := yaml.Unmarshal(file, &DriverConfig)
	if err != nil {
		log.Errorf("Unable to read the driver config:%s", err.Error())
		return ""
	}
	return DriverConfig.StorageClass.Class
}

// SkipTests will skip the tests mentioned in the ignore.yaml
func SkipTests(skipFile string) (string, error) {
	skippedTests := ""
	type SkippedTests struct {
		Skip []string
	}
	if skipFile == "" {
		skipFile = IgnoreFile
	}
	file, _ := os.ReadFile(filepath.Clean(skipFile))
	var tests SkippedTests
	err := yaml.Unmarshal(file, &tests)
	if err != nil {
		log.Errorf("YAML-Encoded error :%s", err)
		return "", err
	}
	if len(tests.Skip) == 0 {
		return "", nil
	}
	for i := 0; i < len(tests.Skip); i++ {
		skippedTests = skippedTests + tests.Skip[i] + "|"
	}
	skippedTests = strings.TrimSpace(skippedTests)
	skippedTests = strings.TrimSuffix(skippedTests, "|")

	return skippedTests, nil
}

// BuildE2eCommand will build the command args
func BuildE2eCommand(ctx *cli.Context) ([]string, error) {
	var args []string
	var (
		driverConfig  = ctx.String("driver-config")
		reportDir     = ctx.String("reportPath")
		skipString    = ctx.String("skip")
		focusString   = ctx.String("focus")
		focusFile     = ctx.String("focus-file")
		skipFile      = ctx.String("skip-file")
		skipTestsFile = ctx.String("skip-tests")
		timeout       = ctx.String("timeout")
	)
	kubeconfig := ctx.String("config")
	if kubeconfig == "" {
		kubeconfig = CheckKubeConfigEnv()
	}
	args = append(args, kubeconfigKey)
	args = append(args, kubeconfig)
	args = append(args, testdriverKey)
	if FileExists(driverConfig) {
		args = append(args, driverConfig)
	} else {
		log.Errorf("Driver config file not found: %s", driverConfig)
		return args, errors.New("driver config file not found")
	}
	if focusString != "" {
		args = append(args, focusStringKey)
		args = append(args, focusString)
	}
	multiSkip, _ := SkipTests(skipTestsFile)
	if multiSkip != "" {
		args = append(args, skipStringKey)
		args = append(args, multiSkip)
	}
	if skipString != "" {
		args = append(args, skipStringKey)
		args = append(args, skipString)
	}

	if focusFile != "" {
		args = append(args, focusFileKey)
		args = append(args, focusFile)
	}
	if skipFile != "" {
		args = append(args, skipFileKey)
		args = append(args, skipFile)
	}
	if timeout != "" {
		args = append(args, timeoutKey)
		args = append(args, timeout)
	}
	if reportDir == "" {
		reportDir = defaultDir + "execution" + "_" + ReadTestDriverConfig(driverConfig) + XML
	}
	if FileExists(reportDir) {
		err := os.Remove(filepath.Clean(reportDir))
		if err != nil {
			log.Errorf("Unable to remove old report: %s err: %s", reportDir, err.Error())
		} else {
			log.Infof("Removed old reports: %s", reportDir)
		}
	}
	args = append(args, junitReportKey)
	args = append(args, reportDir)
	return args, nil
}

// GenerateReport will call parser function and generates the report
func GenerateReport(report string) {
	_, _ = E2eReportParser(report)
}

// ExecuteE2ECommand will execute the ./e2e.test command and generates the reports
func ExecuteE2ECommand(args []string, ch chan os.Signal) error {
	log.Info("e2e command arguments are: ", args)
	cmd := exec.Command(k8sBinary, args...)
	go func() {
		_, ok := <-ch
		if !ok {
			return
		}
		log.Info("Received termination signal,")
		log.Info("Do you really want to terminate the process ? (Y/n)")
		reader := bufio.NewReader(os.Stdin)
		log.Info("-> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Error(err)
		}
		input = strings.TrimSuffix(input, "\n")
		if input == "n" || input == "N" || input == "No" || input == "no" {
			return
		}
		err = cmd.Process.Signal(syscall.SIGINT)
		if err != nil {
			log.Errorf("Unable to terminate the process, Termination failed with -->%s", err.Error())
			return
		}
		log.Infof("Teriminated kubernetes e2e process")
	}()

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw
	// Execute the command
	cmdErr := cmd.Run()
	executionLogFile := defaultDir + executionLogFile + "_" + ReadTestDriverConfig(args[3]) + ".log"
	_ = os.Remove(filepath.Clean(executionLogFile))
	f, err := os.Create(filepath.Clean(executionLogFile))
	if err != nil {
		log.Errorf("Error in file operation: %s", err.Error())
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Errorf("Error in file operation: %s", err.Error())
		}
	}(f)

	_, err2 := f.WriteString(stdBuffer.String() + "\n")

	if err2 != nil {
		log.Errorf("Error in file operation: %s", err.Error())
	}
	if cmdErr != nil {
		return cmdErr
	}
	return nil
}
