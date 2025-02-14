package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
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
	// by asserting the expected command and flags.
	command := GetFunctionalTestCommand()
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

// Add more test cases for other functions in functionalcmd.go
