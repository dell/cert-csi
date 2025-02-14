package node

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
)

type fakeNodeInterface struct {
	v1core.NodeInterface
	patchError bool
}

func (f *fakeNodeInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.Node, error) {
	if f.patchError {
		return nil, errors.New("patch error")
	}
	return f.NodeInterface.Patch(ctx, name, pt, data, opts, subresources...)
}
func TestNodeCordon(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create a test node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	client.Interface.Create(context.Background(), node, metav1.CreateOptions{})

	err := client.NodeCordon(context.Background(), "test-node")
	if err != nil {
		t.Errorf("NodeCordon failed: %v", err)
	}

	// Verify the node is cordoned
	updatedNode, _ := client.Interface.Get(context.Background(), "test-node", metav1.GetOptions{})
	if !updatedNode.Spec.Unschedulable {
		t.Errorf("NodeCordon did not cordon the node")
	}
}

func TestNodeUnCordon(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create a test node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: v1.NodeSpec{
			Unschedulable: true,
		},
	}
	client.Interface.Create(context.Background(), node, metav1.CreateOptions{})

	err := client.NodeUnCordon(context.Background(), "test-node")
	if err != nil {
		t.Errorf("NodeUnCordon failed: %v", err)
	}

	// Verify the node is uncordoned
	updatedNode, _ := client.Interface.Get(context.Background(), "test-node", metav1.GetOptions{})
	if updatedNode.Spec.Unschedulable {
		t.Errorf("NodeUnCordon did not uncordon the node")
	}
}

func TestCordonUnCordonErrors(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Test error when node does not exist
	err := client.NodeCordon(context.Background(), "non-existent-node")
	if err == nil {
		t.Errorf("Expected error when cordoning non-existent node, got nil")
	}

	err = client.NodeUnCordon(context.Background(), "non-existent-node")
	if err == nil {
		t.Errorf("Expected error when uncordoning non-existent node, got nil")
	}
}

func TestCordonUnCordonPatchError(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create a test node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	client.Interface.Create(context.Background(), node, metav1.CreateOptions{})

	// Simulate a patch error by providing invalid patch data
	client.Interface = &fakeNodeInterface{
		NodeInterface: client.Interface,
		patchError:    true,
	}

	err := client.NodeCordon(context.Background(), "test-node")
	if err == nil {
		t.Errorf("Expected error when patching node, got nil")
	}
}

func TestNodeCordonAlreadyCordoned(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create a test node that is already cordoned
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: v1.NodeSpec{
			Unschedulable: true,
		},
	}
	client.Interface.Create(context.Background(), node, metav1.CreateOptions{})

	err := client.NodeCordon(context.Background(), "test-node")
	if err != nil {
		t.Errorf("NodeCordon failed: %v", err)
	}

	// Verify the node is still cordoned
	updatedNode, _ := client.Interface.Get(context.Background(), "test-node", metav1.GetOptions{})
	if !updatedNode.Spec.Unschedulable {
		t.Errorf("NodeCordon did not keep the node cordoned")
	}
}

func TestNodeUnCordonAlreadyUncordoned(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create a test node that is already uncordoned
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	client.Interface.Create(context.Background(), node, metav1.CreateOptions{})

	err := client.NodeUnCordon(context.Background(), "test-node")
	if err != nil {
		t.Errorf("NodeUnCordon failed: %v", err)
	}

	// Verify the node is still uncordoned
	updatedNode, _ := client.Interface.Get(context.Background(), "test-node", metav1.GetOptions{})
	if updatedNode.Spec.Unschedulable {
		t.Errorf("NodeUnCordon did not keep the node uncordoned")
	}
}

func TestCordonUnCordonJSONError(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create a test node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	client.Interface.Create(context.Background(), node, metav1.CreateOptions{})

	// Simulate a JSON marshalling error by providing invalid data
	client.Interface = &fakeNodeInterface{
		NodeInterface: client.Interface,
		patchError:    true,
	}

	err := client.cordonUnCordon(context.Background(), "test-node", true)
	if err == nil {
		t.Errorf("Expected error when marshalling node, got nil")
	}
}

func TestNodeCordonWithTimeout(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create a test node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	client.Interface.Create(context.Background(), node, metav1.CreateOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	err := client.NodeCordon(ctx, "test-node")
	if err == nil {
		t.Errorf("Expected timeout error when cordoning node, got nil")
	}
}

func TestNodeUnCordonWithTimeout(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create a test node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: v1.NodeSpec{
			Unschedulable: true,
		},
	}
	client.Interface.Create(context.Background(), node, metav1.CreateOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	err := client.NodeUnCordon(ctx, "test-node")
	if err == nil {
		t.Errorf("Expected timeout error when uncordoning node, got nil")
	}
}

func TestNodeCordonWithDifferentNodeNames(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create test nodes
	nodes := []string{"node1", "node2", "node3"}
	for _, nodeName := range nodes {
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
		}
		client.Interface.Create(context.Background(), node, metav1.CreateOptions{})
	}

	for _, nodeName := range nodes {
		err := client.NodeCordon(context.Background(), nodeName)
		if err != nil {
			t.Errorf("NodeCordon failed for %s: %v", nodeName, err)
		}

		// Verify the node is cordoned
		updatedNode, _ := client.Interface.Get(context.Background(), nodeName, metav1.GetOptions{})
		if !updatedNode.Spec.Unschedulable {
			t.Errorf("NodeCordon did not cordon the node %s", nodeName)
		}
	}
}

func TestNodeUnCordonWithDifferentNodeNames(t *testing.T) {
	client := &Client{
		Interface: fake.NewSimpleClientset().CoreV1().Nodes(),
	}

	// Create test nodes
	nodes := []string{"node1", "node2", "node3"}
	for _, nodeName := range nodes {
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
			Spec: v1.NodeSpec{
				Unschedulable: true,
			},
		}
		client.Interface.Create(context.Background(), node, metav1.CreateOptions{})
	}

	for _, nodeName := range nodes {
		err := client.NodeUnCordon(context.Background(), nodeName)
		if err != nil {
			t.Errorf("NodeUnCordon failed for %s: %v", nodeName, err)
		}

		// Verify the node is uncordoned
		updatedNode, _ := client.Interface.Get(context.Background(), nodeName, metav1.GetOptions{})
		if updatedNode.Spec.Unschedulable {
			t.Errorf("NodeUnCordon did not uncordon the node %s", nodeName)
		}
	}
}
