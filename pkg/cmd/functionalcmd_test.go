package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore/runner"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	"github.com/urfave/cli"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	fakeClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
)

type MockCleanupTestSuite struct {
	t *testing.T
}

func (m *MockCleanupTestSuite) GetName() string {
	return "mock"
}

func (m *MockCleanupTestSuite) Run(_ context.Context, _ string, _ *k8sclient.Clients) (delFunc func() error, e error) {
	m.t.Logf("mock test suite run")
	return nil, nil
}

func (m *MockCleanupTestSuite) GetObservers(_ observer.Type) []observer.Interface {
	return nil
}

func (m *MockCleanupTestSuite) GetClients(_ string, _ *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	return nil, nil
}

func (m *MockCleanupTestSuite) GetNamespace() string {
	return "mock-ns-prefix"
}

func (m *MockCleanupTestSuite) Parameters() string {
	return ""
}

type clientTestContext struct {
	testNamespace    string
	namespaceDeleted bool
	t                *testing.T
}

func TestCleanupAfterTest(t *testing.T) {
	// Create a temporary file that contains a simple kubernetes config
	// and set the environment variable to point to it
	confPath, err := createDummyKubeConfig(t.TempDir(), t)
	assert.NoError(t, err)

	s := []suites.Interface{
		&MockCleanupTestSuite{
			t: t,
		},
	}

	FuncNewClientSetOriginal := k8sclient.FuncNewClientSet
	defer func() {
		k8sclient.FuncNewClientSet = FuncNewClientSetOriginal
	}()

	clientCtx := &clientTestContext{t: t}

	k8sclient.FuncNewClientSet = func(_ *rest.Config) (kubernetes.Interface, error) {
		return createFakeKubeClient(clientCtx)
	}

	fset := flag.NewFlagSet("unit-test", flag.ContinueOnError)
	timeoutFlag := &cli.StringFlag{
		Name:  "timeout",
		Value: "1m",
	}
	timeoutFlag.Apply(fset)
	configFlag := &cli.StringFlag{
		Name:  "config",
		Value: confPath,
	}
	configFlag.Apply(fset)

	c := cli.NewContext(nil, fset, nil)

	tests := []struct {
		name          string
		noCleanup     bool
		expectCleanup bool
	}{
		{
			name:          "Functional test with no-cleanup = false",
			noCleanup:     false,
			expectCleanup: true,
		},
		{
			name:          "Functional test with no-cleanup = true",
			noCleanup:     true,
			expectCleanup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset callbacks state before each test
			clientCtx.testNamespace = ""
			clientCtx.namespaceDeleted = false

			sr := createFunctionalSuiteRunner(c, tt.noCleanup, tt.noCleanup)
			sr.RunFunctionalSuites(s)

			if clientCtx.namespaceDeleted != tt.expectCleanup {
				t.Errorf("Expected test namespace deletion %v, but got %v", tt.expectCleanup, clientCtx.namespaceDeleted)
			}
		})
	}
}

func createFakeKubeClient(ctx *clientTestContext) (kubernetes.Interface, error) {
	client := fakeClient.NewSimpleClientset()
	client.Discovery().(*fake.FakeDiscovery).FakedServerVersion = &version.Info{
		Major:      "1",
		Minor:      "32",
		GitVersion: "v1.32.0",
	}
	client.Fake.PrependReactor("create", "namespaces", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(clienttesting.CreateAction)
		namespace := createAction.GetObject().(*corev1.Namespace)
		if strings.HasPrefix(namespace.Name, "mock-ns-prefix-") {
			ctx.t.Logf("namespace %s creation called", namespace.Name)
			if ctx.testNamespace == "" {
				ctx.testNamespace = namespace.Name
			} else {
				return true, nil, fmt.Errorf("repeated test namespace creation call: was %s, now %s", ctx.testNamespace, namespace.Name)
			}
			return true, namespace, nil
		}
		return true, nil, fmt.Errorf("unexpected namespace creation %s", namespace.Name)
	})
	client.Fake.PrependReactor("delete", "namespaces", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		deleteAction := action.(clienttesting.DeleteAction)
		if ctx.testNamespace != "" && deleteAction.GetName() == ctx.testNamespace {
			ctx.t.Logf("namespace %s deletion called", deleteAction.GetName())
			ctx.namespaceDeleted = true
			return true, nil, nil
		}
		return true, nil, fmt.Errorf("unexpected namespace deletion %s", deleteAction.GetName())
	})
	return client, nil
}

func createDummyKubeConfig(tmpDir string, t *testing.T) (string, error) {
	// Define a simple kube client config
	kubeConfig := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://unit.test
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

	confPath := tmpDir + "/kube.config"

	err := os.WriteFile(confPath, []byte(kubeConfig), 0o600)
	if err != nil {
		return "", fmt.Errorf("failed to write dummy kube config file: %v", err)
	}

	// Set the environment variable to point to the temporary file
	t.Setenv("KUBECONFIG", confPath)

	// Print the path to the temporary file
	t.Logf("Created dummy kube config file at: %s", confPath)
	return confPath, nil
}

func TestGetFunctionalTestCommand(t *testing.T) {
	// Test the functionality of the GetFunctionalTestCommand function
	command := GetFunctionalTestCommand()

	// ctx := cli.NewContext(app, nil, nil)
	assert.Equal(t, "functional-test", command.Name)
	assert.Equal(t, "Test csi-driver functionality", command.Usage)
	assert.Equal(t, "main", command.Category)

	// Test the subcommands of the command
	assert.Equal(t, "list", command.Subcommands[0].Name)
	assert.Equal(t, "volume-deletion", command.Subcommands[1].Name)
	assert.Equal(t, "pod-deletion", command.Subcommands[2].Name)
	assert.Equal(t, "clone-volume-deletion", command.Subcommands[3].Name)
	assert.Equal(t, "volume-creation", command.Subcommands[4].Name)
	assert.Equal(t, "provisioning", command.Subcommands[5].Name)
	assert.Equal(t, "clone-volume", command.Subcommands[6].Name)
	assert.Equal(t, "snapshot", command.Subcommands[7].Name)
}

func TestGetFunctionalTestCommandAction(t *testing.T) {
	// Default context
	ctx := &cli.Context{}
	command := GetFunctionalTestCommand()
	// Call the action function
	action := command.Subcommands[0].Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestReadEphemeralConfig(t *testing.T) {
	// Test case for an empty filename
	config, err := readEphemeralConfig("")
	if err != nil {
		t.Errorf("readEphemeralConfig returned an error for an empty filename")
	}
	if len(config) != 0 {
		t.Errorf("readEphemeralConfig returned a non-empty config for an empty filename")
	}

	// Test case for a valid config file
	tempFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatalf("Could not create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	configData := `

key1=value1
key2=value2
`

	if _, err := tempFile.WriteString(configData); err != nil {
		t.Fatalf("Could not write to temporary file: %v", err)
	}

	config, err = readEphemeralConfig(tempFile.Name())
	if err != nil {
		t.Errorf("readEphemeralConfig returned an error for a valid config file: %v", err)
	}
	if len(config) != 2 {
		t.Errorf("readEphemeralConfig returned the wrong number of keys for a valid config file: %d", len(config))
	}
	if config["key1"] != "value1" || config["key2"] != "value2" {
		t.Errorf("readEphemeralConfig returned the wrong values for a valid config file: %v", config)
	}

	// Test case for a non-existent config file
	config, err = readEphemeralConfig("/path/to/non/existent/file")
	if err == nil {
		t.Errorf("readEphemeralConfig did not return an error for a non-existent config file")
	}
	if len(config) != 0 {
		t.Errorf("readEphemeralConfig returned a non-empty config for a non-existent config file")
	}
}


func TestGetVolumeDeletionCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("pvc-name", "test-pvc", "pvc name")
	set.String("pvc-namespace", "test-namespace", "pvc namespace")

	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeDeletionCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetPodDeletionCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("pod-name", "test-pod", "pod name")
	set.String("pod-namespace", "test-namespace", "pod namespace")

	ctx := cli.NewContext(nil, set, nil)
	command := getPodDeletionCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetCloneVolumeDeletionCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("clone-volume-name", "test-volume", "volume name")
	set.String("clone-pod-name", "test-pod", "pod name")
	set.String("resource-namespace", "test-namespace", "resource namespace")

	ctx := cli.NewContext(nil, set, nil)
	command := getCloneVolumeDeletionCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetFunctionalSnapDeletionCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("volume-snapshot-name", "test-vol-snapshot", "vol snapshot name")
	set.String("resource-namespace", "test-namespace", "resource namespace")

	ctx := cli.NewContext(nil, set, nil)
	command := getFunctionalSnapDeletionCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetFunctionalVolumeCreateCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.Int("number", 1, "Volume number")
	set.String("size", "2Gi", "volume size")
	set.String("custom-name", "test-name", "Volume custom name")
	set.String("access-mode", "ReadWriteMany", "Volume access mode")
	set.Bool("block,", false, "storge block volume")

	ctx := cli.NewContext(nil, set, nil)
	command := getFunctionalVolumeCreateCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetFunctionalCloneVolumeCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("pvc-name", "test-pvc", "pvc name")
	set.Int("volumeNumber", 1, "Volume number")
	set.Int("podNumber", 1, "Pod number")
	set.String("pod-name", "test-pod", "pod name")
	set.String("access-mode", "ReadWriteMany", "Volume access mode")
	set.String("image-config", "../k8sclient/testdata/empty_imageconfig.yaml", "test image path")

	ctx := cli.NewContext(nil, set, nil)
	command := getFunctionalCloneVolumeCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetFunctionalProvisioningCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.Int("volumeNumber", 1, "Volume number")
	set.Int("podNumber", 1, "Pod number")
	set.String("pod-name", "test-pod", "pod name")
	set.String("vol-access-mode", "ReadWriteMany", "Volume access mode")
	set.String("image-config", "../k8sclient/testdata/empty_imageconfig.yaml", "test image path")
	set.Bool("block,", false, "storge block volume")
	set.Bool("roFlag,", true, "Read only flag")

	ctx := cli.NewContext(nil, set, nil)
	command := getFunctionalProvisioningCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetFunctionalSnapCreationCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("volumeSnapshotName", "test-snap", "snap class  name")
	set.Int("snapshotAmount", 1, "snapshot Amount")
	set.String("size", "2Gi", "volume size")
	set.String("snap-name", "test-snap-name", "snapshot name")
	set.String("access-mode-original-volume", "ReadWriteMany", "access Mode Original ")
	set.String("access-mode-restored-volume", "ReadWriteMany", "access Mode Restored ")
	set.String("image-config", "../k8sclient/testdata/empty_imageconfig.yaml", "test image path")

	ctx := cli.NewContext(nil, set, nil)
	command := getFunctionalSnapCreationCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetFunctionalMultiAttachVolCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.Int("pods", 1, "pod number")
	set.Bool("block,", false, "storge block volume")
	set.String("access-mode", "ReadWriteMany", "Volume access mode")
	set.String("image-config", "../k8sclient/testdata/empty_imageconfig.yaml", "test image path")

	ctx := cli.NewContext(nil, set, nil)
	command := getFunctionalMultiAttachVolCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetFunctionalEphemeralCreationCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.Int("pods", 1, "pod number")
	set.String("driver", "powerflex", "driver name")
	set.String("fs-type", "ext4", "fs type")
	set.String("csi-attributes", "csi.storage.k8s.io/pod.name=test-pod", "csi attributes")
	set.String("image-config", "../k8sclient/testdata/empty_imageconfig.yaml", "test image path")
	set.Bool("no-cleanup", true, "no cleanup after test")
	set.Bool("no-cleanup-on-fail", true, "no cleanup after test on fail")

	ctx := cli.NewContext(nil, set, nil)
	command := getFunctionalEphemeralCreationCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetNodeDrainCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("node-name", "test-node", "node name")
	set.String("node-namespace", "test-namespace", "node namespace")
	set.Int("grace-period-seconds", 10, "Grace period for force deletion")

	ctx := cli.NewContext(nil, set, nil)
	command := getNodeDrainCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetNodeUnCordonCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("node-name", "test-node", "node name")

	ctx := cli.NewContext(nil, set, nil)
	command := getNodeUnCordonCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}

func TestGetCapacityTrackingCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("timeout", "10s", "specifies timeout for volume deletion")
	set.String("sc", "test-storage-class", "specifies storage class for volume deletion")
	set.String("description", "hello", "description for volume deletion")
	set.String("config", "../k8sclient/testdata/config", "kube config path")
	set.String("namespace", "test-namespace", "namespace name")
	set.Bool("no-reports", true, "specifies if reports should be generated")
	set.String("driverns", "test-namespace", "driver namespace")
	set.String("volSize", "2Gi", "volume size")
	set.String("image-config", "../k8sclient/testdata/empty_imageconfig.yaml", "test image path")
	set.Duration("poll-interval", 1*time.Minute, "poll interval set for external provisioner")

	ctx := cli.NewContext(nil, set, nil)
	command := getCapacityTrackingCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteSuite = func(sr *runner.FunctionalSuiteRunner, s []suites.Interface) {}
	actionFunc(ctx)
}