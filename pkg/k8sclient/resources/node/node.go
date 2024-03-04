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

// Client contains node client information
type Client struct {
	Interface v1core.NodeInterface
	ClientSet kubernetes.Interface
	Config    *restclient.Config
	Namespace string
	Timeout   int
	nodeInfos []*resource.Info
}

// NodeCordon cordons a node
func (c *Client) NodeCordon(ctx context.Context, nodename string) error {
	return c.cordonUnCordon(ctx, nodename, true)
}

// NodeUnCordon uncordons a node
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
