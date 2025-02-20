package mockutils_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/dell/cert-csi/pkg/testcore/suites/mockutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
)


func TestFakeRemoteExecutor_Execute(t *testing.T) {
	executor := mockutils.FakeRemoteExecutor{}
	err := executor.Execute("POST", &url.URL{}, &rest.Config{}, nil, nil, nil, false, nil)
	assert.NoError(t, err)
}

func TestFakeExtendedCoreV1_RESTClient(t *testing.T) {
	coreV1 := mockutils.FakeExtendedCoreV1{}
	client := coreV1.RESTClient()
	assert.NotNil(t, client)
	assert.IsType(t, &restfake.RESTClient{}, client)
}

func TestFakeExtendedClientset_CoreV1(t *testing.T) {
	clientset := &mockutils.FakeExtendedClientset{Clientset: fake.NewSimpleClientset()}
	coreV1 := clientset.CoreV1()
	assert.NotNil(t, coreV1)
}


func TestMockClientSetPodFunctions(t *testing.T) {
	// Create a FakeExtendedClientset
	clientset := mockutils.NewFakeClientsetWithRestClient()

	// Mock the clientset functions
	mockedClientset := mockutils.MockClientSetPodFunctions(clientset).(*mockutils.FakeExtendedClientset)

	// Test creating a pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
	}
	createdPod, err := mockedClientset.CoreV1().Pods("test-namespace").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err)
	assert.Equal(t, v1.PodRunning, createdPod.Status.Phase)
	assert.Len(t, createdPod.Status.Conditions, 1)
	assert.Equal(t, v1.PodReady, createdPod.Status.Conditions[0].Type)
	assert.Equal(t, v1.ConditionTrue, createdPod.Status.Conditions[0].Status)

	// Test getting a pod
	retrievedPod, err := mockedClientset.CoreV1().Pods("test-namespace").Get(context.TODO(), "test-pod", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "test-pod", retrievedPod.Name)
	assert.Equal(t, v1.PodRunning, retrievedPod.Status.Phase)
	assert.Len(t, retrievedPod.Status.Conditions, 1)
	assert.Equal(t, v1.PodReady, retrievedPod.Status.Conditions[0].Type)
	assert.Equal(t, v1.ConditionTrue, retrievedPod.Status.Conditions[0].Status)
}

func TestMockClientSetPodFunctionsWithKfakeClientset(t *testing.T) {
	// Create a kfake.Clientset
	clientset := fake.NewSimpleClientset()

	// Mock the clientset functions
	mockedClientset := mockutils.MockClientSetPodFunctions(clientset).(*fake.Clientset)

	// Test creating a pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
	}
	createdPod, err := mockedClientset.CoreV1().Pods("test-namespace").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err)
	assert.Equal(t, v1.PodRunning, createdPod.Status.Phase)
	assert.Len(t, createdPod.Status.Conditions, 1)
	assert.Equal(t, v1.PodReady, createdPod.Status.Conditions[0].Type)
	assert.Equal(t, v1.ConditionTrue, createdPod.Status.Conditions[0].Status)

	// Test getting a pod
	retrievedPod, err := mockedClientset.CoreV1().Pods("test-namespace").Get(context.TODO(), "test-pod", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "test-pod", retrievedPod.Name)
	assert.Equal(t, v1.PodRunning, retrievedPod.Status.Phase)
	assert.Len(t, retrievedPod.Status.Conditions, 1)
	assert.Equal(t, v1.PodReady, retrievedPod.Status.Conditions[0].Type)
	assert.Equal(t, v1.ConditionTrue, retrievedPod.Status.Conditions[0].Status)
}

func TestMockClientSetPodFunctionsWithUnexpectedType(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, r, "unexpected type")
		} else {
			t.Errorf("Expected panic, but did not get one")
		}
	}()

	// Pass an unexpected type to the function
	mockutils.MockClientSetPodFunctions("unexpected-type")
}


func TestNewFakeClientsetWithRestClient(t *testing.T) {
	// Create a fake Pod object
	pod := &v1.Pod{}

	// Call the function
	clientset := mockutils.NewFakeClientsetWithRestClient(pod)

	// Verify the result
	assert.NotNil(t, clientset)
	assert.IsType(t, &mockutils.FakeExtendedClientset{}, clientset)
	assert.IsType(t, &fake.Clientset{}, clientset.Clientset)
}

func TestRESTMapping_KindFor(t *testing.T) {
	mapping := &mockutils.RESTMapping{}
	mapping.On("KindFor", mock.Anything).Return(schema.GroupVersionKind{}, nil)

	kind, err := mapping.KindFor(schema.GroupVersionResource{})
	assert.NoError(t, err)
	assert.Equal(t, schema.GroupVersionKind{}, kind)
}

func TestRESTMapping_KindsFor(t *testing.T) {
	mapping := &mockutils.RESTMapping{}
	mapping.On("KindsFor", mock.Anything).Return([]schema.GroupVersionKind{}, nil)

	kinds, err := mapping.KindsFor(schema.GroupVersionResource{})
	assert.NoError(t, err)
	assert.Equal(t, []schema.GroupVersionKind{}, kinds)
}

func TestRESTMapping_ResourceFor(t *testing.T) {
	mapping := &mockutils.RESTMapping{}
	mapping.On("ResourceFor", mock.Anything).Return(schema.GroupVersionResource{}, nil)

	resource, err := mapping.ResourceFor(schema.GroupVersionResource{})
	assert.NoError(t, err)
	assert.Equal(t, schema.GroupVersionResource{}, resource)
}

func TestRESTMapping_ResourcesFor(t *testing.T) {
	mapping := &mockutils.RESTMapping{}
	mapping.On("ResourcesFor", mock.Anything).Return([]schema.GroupVersionResource{}, nil)

	resources, err := mapping.ResourcesFor(schema.GroupVersionResource{})
	assert.NoError(t, err)
	assert.Equal(t, []schema.GroupVersionResource{}, resources)
}

func TestRESTMapping_RESTMapping(t *testing.T) {
	mapping := &mockutils.RESTMapping{}
	result, err := mapping.RESTMapping(schema.GroupKind{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRESTMapping_RESTMappings(t *testing.T) {
	mapping := &mockutils.RESTMapping{}
	mapping.On("RESTMappings", mock.Anything, mock.Anything).Return([]*meta.RESTMapping{}, nil)

	mappings, err := mapping.RESTMappings(schema.GroupKind{})
	assert.NoError(t, err)
	assert.Equal(t, []*meta.RESTMapping{}, mappings)
}

func TestRESTMapping_ResourceSingularizer(t *testing.T) {
	mapping := &mockutils.RESTMapping{}
	mapping.On("ResourceSingularizer", mock.Anything).Return("", nil)

	singular, err := mapping.ResourceSingularizer("resource")
	assert.NoError(t, err)
	assert.Equal(t, "", singular)
}