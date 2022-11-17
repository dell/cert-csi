package node

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
)

type Client struct {
	Interface v1core.NodeInterface
	ClientSet kubernetes.Interface
	Config    *restclient.Config
	Namespace string
	Timeout   int
	nodeInfos []*resource.Info
}

func (c *Client) NodeCordon(ctx context.Context, nodename string) error {
	return c.cordonUnCordon(ctx, nodename, true)
}
func (c *Client) NodeUnCordon(ctx context.Context, nodename string) error {
	return c.cordonUnCordon(ctx, nodename, false)
}
func (c *Client) cordonUnCordon(ctx context.Context, nodename string, cordon bool) error {

	node, err := c.Interface.Get(ctx, nodename, metav1.GetOptions{})
	if err != nil {
		return err
	}

	oldData, err := json.Marshal(node)
	if err != nil {
		return err
	}

	node.Spec.Unschedulable = cordon

	newData, err := json.Marshal(node)
	if err != nil {
		return err
	}

	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, node)
	if patchErr == nil {
		patchOptions := metav1.PatchOptions{}
		_, err = c.Interface.Patch(ctx, node.Name, types.StrategicMergePatchType, patchBytes, patchOptions)
	} else {
		updateOptions := metav1.UpdateOptions{}
		_, err = c.Interface.Update(ctx, node, updateOptions)
	}
	return err
}
