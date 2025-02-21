package node

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
)

type fakeNodeInterface struct {
	v1core.NodeInterface
	getFunc    func(ctx context.Context, name string, options metav1.GetOptions) (*v1.Node, error)
	patchFunc  func(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*v1.Node, error)
	updateFunc func(ctx context.Context, node *v1.Node, options metav1.UpdateOptions) (*v1.Node, error)
}

func (f *fakeNodeInterface) Get(ctx context.Context, name string, options metav1.GetOptions) (*v1.Node, error) {
	return f.getFunc(ctx, name, options)
}

func (f *fakeNodeInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*v1.Node, error) {
	return f.patchFunc(ctx, name, pt, data, options, subresources...)
}

func (f *fakeNodeInterface) Update(ctx context.Context, node *v1.Node, options metav1.UpdateOptions) (*v1.Node, error) {
	return f.updateFunc(ctx, node, options)
}

func TestCordonUnCordon(t *testing.T) {
	tests := []struct {
		name       string
		nodename   string
		cordon     bool
		getFunc    func(ctx context.Context, name string, options metav1.GetOptions) (*v1.Node, error)
		patchFunc  func(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*v1.Node, error)
		updateFunc func(ctx context.Context, node *v1.Node, options metav1.UpdateOptions) (*v1.Node, error)
	}{
		{
			name:     "cordon node successfully",
			nodename: "node1",
			cordon:   true,
			getFunc: func(ctx context.Context, name string, options metav1.GetOptions) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
			patchFunc: func(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
			updateFunc: func(ctx context.Context, node *v1.Node, options metav1.UpdateOptions) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
		},
		{
			name:     "get node error",
			nodename: "node1",
			cordon:   true,
			getFunc: func(ctx context.Context, name string, options metav1.GetOptions) (*v1.Node, error) {
				return nil, errors.New("get node error")
			},
			patchFunc: func(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
			updateFunc: func(ctx context.Context, node *v1.Node, options metav1.UpdateOptions) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
		},
		{
			name:     "create two way merge patch error, update successfully",
			nodename: "node1",
			cordon:   true,
			getFunc: func(ctx context.Context, name string, options metav1.GetOptions) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
			patchFunc: func(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*v1.Node, error) {
				return &v1.Node{}, errors.New("create two way merge patch error")
			},
			updateFunc: func(ctx context.Context, node *v1.Node, options metav1.UpdateOptions) (*v1.Node, error) {
				return &v1.Node{}, nil // Return success here
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				Interface: &fakeNodeInterface{
					getFunc:    tt.getFunc,
					patchFunc:  tt.patchFunc,
					updateFunc: tt.updateFunc,
				},
			}

			err := client.cordonUnCordon(context.Background(), tt.nodename, tt.cordon)
			if err != nil {
				if tt.name == "get node error" {
					assert.Error(t, err)
				} else if tt.name == "create two way merge patch error, update successfully" {
					assert.NoError(t, err)
				} else {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNodeCordon(t *testing.T) {
	client := &Client{
		Interface: &fakeNodeInterface{
			getFunc: func(ctx context.Context, name string, options metav1.GetOptions) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
			patchFunc: func(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
			updateFunc: func(ctx context.Context, node *v1.Node, options metav1.UpdateOptions) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
		},
	}

	err := client.NodeCordon(context.Background(), "node1")
	assert.NoError(t, err)
}

func TestNodeUnCordon(t *testing.T) {
	client := &Client{
		Interface: &fakeNodeInterface{
			getFunc: func(ctx context.Context, name string, options metav1.GetOptions) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
			patchFunc: func(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
			updateFunc: func(ctx context.Context, node *v1.Node, options metav1.UpdateOptions) (*v1.Node, error) {
				return &v1.Node{}, nil
			},
		},
	}

	err := client.NodeUnCordon(context.Background(), "node1")
	assert.NoError(t, err)
}
