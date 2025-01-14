package cmd

import (
	"context"
	"flag"
	"fmt"
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
	"strings"
	"testing"
)

type MockCleanupTestSuite struct {
	t *testing.T
}

func (m *MockCleanupTestSuite) GetName() string {
	return "mock"
}

func (m *MockCleanupTestSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	m.t.Logf("mock test suite run")
	return nil, nil
}

func (m *MockCleanupTestSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return nil
}

func (m *MockCleanupTestSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
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

	k8sclient.FuncNewClientSet = func(config *rest.Config) (kubernetes.Interface, error) {
		return createFakeKubeClient(clientCtx)
	}

	fset := flag.NewFlagSet("unit-test", flag.ContinueOnError)
	timeoutFlag := &cli.StringFlag{
		Name:  "timeout",
		Value: "1m",
	}
	timeoutFlag.Apply(fset)
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
