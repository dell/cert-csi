//go:build !testcoverage
// +build !testcoverage

package mockutils

import (
	"fmt"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned/typed/volumesnapshot/v1"
	"github.com/stretchr/testify/mock"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kfake "k8s.io/client-go/kubernetes/fake"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
)

type FakeRemoteExecutor struct{}

func (f FakeRemoteExecutor) Execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout,
	stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	return nil
}

type FakeExtendedCoreV1 struct {
	typedcorev1.CoreV1Interface
	restClient rest.Interface
}

func (c *FakeExtendedCoreV1) RESTClient() rest.Interface {
	if c.restClient == nil {
		c.restClient = &restfake.RESTClient{}
	}
	return c.restClient
}

type FakeExtendedClientset struct {
	*kfake.Clientset
}

func (f *FakeExtendedClientset) CoreV1() typedcorev1.CoreV1Interface {
	return &FakeExtendedCoreV1{f.Clientset.CoreV1(), nil}
}

func NewFakeClientsetWithRestClient(objs ...runtime.Object) *FakeExtendedClientset {
	return &FakeExtendedClientset{kfake.NewSimpleClientset(objs...)}
}
func MockClientSetPodFunctions(clientset interface{}) interface{} {
	// Set up a reactor to simulate Pods becoming Ready
	createPodFunc := func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		pod := createAction.GetObject().(*v1.Pod)
		// Set pod phase to Running
		pod.Status.Phase = v1.PodRunning
		// Simulate the Ready condition
		pod.Status.Conditions = append(pod.Status.Conditions, v1.PodCondition{
			Type:   v1.PodReady,
			Status: v1.ConditionTrue,
		})
		return false, nil, nil // Allow normal processing to continue
	}

	// Also, when getting pods, return the pod with Running status and Ready condition
	getPodFunc := func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		podName := getAction.GetName()
		// Create a pod object with the expected name and Ready status
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "test-namespace",
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		}
		return true, pod, nil
	}

	switch v := clientset.(type) {
	case *kfake.Clientset:
		v.Fake.PrependReactor("create", "pods", createPodFunc)
		v.Fake.PrependReactor("get", "pods", getPodFunc)
		return clientset
	case *FakeExtendedClientset:
		v.Fake.PrependReactor("create", "pods", createPodFunc)
		v.Fake.PrependReactor("get", "pods", getPodFunc)
		return clientset
	default:
		panic(fmt.Sprintf("unexpected type %T", clientset))
	}
}

type FakeVolumeSnapshotInterface struct {
	snapshotv1.VolumeSnapshotInterface
}
type RESTMapping struct {
	mock.Mock
}

func (m *RESTMapping) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	args := m.Called(resource)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (m *RESTMapping) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	args := m.Called(resource)
	return args.Get(0).([]schema.GroupVersionKind), args.Error(1)
}

func (m *RESTMapping) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	args := m.Called(input)
	return args.Get(0).(schema.GroupVersionResource), args.Error(1)
}

func (m *RESTMapping) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	args := m.Called(input)
	return args.Get(0).([]schema.GroupVersionResource), args.Error(1)
}

func createRESTMapping() *meta.RESTMapping {
	// Create a GroupVersionResource
	gvr := schema.GroupVersionResource{
		Group:    "group",
		Version:  "version",
		Resource: "resource",
	}

	// Create a RESTMapping
	restMapping := &meta.RESTMapping{
		Resource:         gvr,
		GroupVersionKind: schema.GroupVersionKind{},
		Scope:            meta.RESTScopeNamespace,
	}

	return restMapping
}

func (m *RESTMapping) RESTMapping(_ schema.GroupKind, _ ...string) (*meta.RESTMapping, error) {
	return createRESTMapping(), nil
}

func (m *RESTMapping) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	args := m.Called(gk, versions)
	return args.Get(0).([]*meta.RESTMapping), args.Error(1)
}

func (m *RESTMapping) ResourceSingularizer(resource string) (string, error) {
	args := m.Called(resource)
	return args.String(0), args.Error(1)
}
