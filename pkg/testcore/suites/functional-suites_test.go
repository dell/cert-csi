/*
 *
 * Copyright © 2024 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package suites

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/observer"
	"github.com/stretchr/testify/assert"
)

func TestGetTopologyCount(t *testing.T) {
	// Test case: Empty topology keys
	FindDriverLogs = func(_ []string) (string, error) {
		return "", nil
	}
	topologyCount, err := getTopologyCount([]string{})
	assert.NoError(t, err)
	assert.Equal(t, 0, topologyCount)

	// Test case: Non-empty topology keys

	FindDriverLogs = func(_ []string) (string, error) {
		keys := "Topology Keys: [csi-powerstore.dellemc.com/10.230.24.67-iscsi csi-powerstore.dellemc.com/10.230.24.67-nfs]"
		return keys, nil
	}
	topologyCount, err = getTopologyCount([]string{"csi-powerstore.dellemc.com/10.230.24.67-iscsi"})
	assert.NoError(t, err)
	assert.Equal(t, 1, topologyCount)

	// Test case: Error in FindDriverLogs
	FindDriverLogs = func(_ []string) (string, error) {
		return "", errors.New("error in FindDriverLogs")
	}
	topologyCount, err = getTopologyCount([]string{})
	assert.Error(t, err)
	assert.Equal(t, 0, topologyCount)
}

func TestVolumeDeletionSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		vds  *VolumeDeletionSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			vds: &VolumeDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Name:        "test-volume",
					Namespace:   "test-namespace",
					Description: "test-description",
				},
			},
			want: "test-description",
		},
		{
			name: "Testing GetName when Description is empty",
			vds: &VolumeDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Name:      "test-volume",
					Namespace: "test-namespace",
				},
			},
			want: "VolumeDeletionSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vds.GetName(); got != tt.want {
				t.Errorf("VolumeDeletionSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeDeletionSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		v    *VolumeDeletionSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			v:    &VolumeDeletionSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PvcObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			v:    &VolumeDeletionSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VolumeDeletionSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestVolumeDeletionSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		vds  *VolumeDeletionSuite
		want string
	}{
		{
			name: "Testing GetNamespace when Namespace is not empty",
			vds: &VolumeDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Namespace: "test-namespace",
				},
			},
			want: "test-namespace",
		},
		{
			name: "Testing GetNamespace when Namespace is empty",
			vds: &VolumeDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Namespace: "",
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vds.GetNamespace(); got != tt.want {
				t.Errorf("VolumeDeletionSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeDeletionSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		vds  *VolumeDeletionSuite
		want string
	}{
		{
			name: "Testing Parameters",
			vds:  &VolumeDeletionSuite{},
			want: "{}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vds.Parameters(); got != tt.want {
				t.Errorf("VolumeDeletionSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodDeletionSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		pds  *PodDeletionSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			pds: &PodDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Name:        "test-volume",
					Namespace:   "test-namespace",
					Description: "test-description",
				},
			},
			want: "test-description",
		},
		{
			name: "Testing GetName when Description is empty",
			pds: &PodDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Name:      "test-volume",
					Namespace: "test-namespace",
				},
			},
			want: "PodDeletionSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pds.GetName(); got != tt.want {
				t.Errorf("PodDeletionSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodDeletionSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		p    *PodDeletionSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			p:    &PodDeletionSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PodObserver{},
				&observer.PvcObserver{},
				&observer.VaObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			p:    &PodDeletionSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PodDeletionSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodDeletionSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		pds  *PodDeletionSuite
		want string
	}{
		{
			name: "Testing GetNamespace when Namespace is not empty",
			pds: &PodDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Namespace: "test-namespace",
				},
			},
			want: "test-namespace",
		},
		{
			name: "Testing GetNamespace when Namespace is empty",
			pds: &PodDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Namespace: "",
				},
			},
			want: "",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pds.GetNamespace(); got != tt.want {
				t.Errorf("PodDeletionSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodDeletionSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		pds  *PodDeletionSuite
		want string
	}{
		{
			name: "Testing Parameters",
			pds:  &PodDeletionSuite{},
			want: "{}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pds.Parameters(); got != tt.want {
				t.Errorf("PodDeletionSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClonedVolDeletionSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		pds  *ClonedVolDeletionSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			pds: &ClonedVolDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Name:        "test-volume",
					Namespace:   "test-namespace",
					Description: "test-description",
				},
			},
			want: "test-description",
		},
		{
			name: "Testing GetName when Description is empty",
			pds: &ClonedVolDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Name:      "test-volume",
					Namespace: "test-namespace",
				},
			},
			want: "ClonedVolumeDeletionSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pds.GetName(); got != tt.want {
				t.Errorf("ClonedVolDeletionSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClonedVolDeletionSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		c    *ClonedVolDeletionSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			c:    &ClonedVolDeletionSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PodObserver{},
				&observer.PvcObserver{},
				&observer.VaObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			c:    &ClonedVolDeletionSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClonedVolDeletionSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClonedVolDeletionSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		pds  *ClonedVolDeletionSuite
		want string
	}{
		{
			name: "Testing GetNamespace when Namespace is not empty",
			pds: &ClonedVolDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Namespace: "test-namespace",
				},
			},
			want: "test-namespace",
		},
		{
			name: "Testing GetNamespace when Namespace is empty",
			pds: &ClonedVolDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Namespace: "",
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pds.GetNamespace(); got != tt.want {
				t.Errorf("ClonedVolDeletionSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClonedVolDeletionSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		pds  *ClonedVolDeletionSuite
		want string
	}{
		{
			name: "Testing Parameters",
			pds:  &ClonedVolDeletionSuite{},
			want: "{}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pds.Parameters(); got != tt.want {
				t.Errorf("ClonedVolDeletionSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotDeletionSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		sds  *SnapshotDeletionSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			sds: &SnapshotDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Name:        "test-volume",
					Namespace:   "test-namespace",
					Description: "test-description",
				},
			},
			want: "test-description",
		},
		{
			name: "Testing GetName when Description is empty",
			sds: &SnapshotDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Name:      "test-volume",
					Namespace: "test-namespace",
				},
			},
			want: "SnapshotDeletionSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sds.GetName(); got != tt.want {
				t.Errorf("SnapshotDeletionSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotDeletionSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		s    *SnapshotDeletionSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			s:    &SnapshotDeletionSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PodObserver{},
				&observer.PvcObserver{},
				&observer.VaObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			s:    &SnapshotDeletionSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SnapshotDeletionSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotDeletionSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		sds  *SnapshotDeletionSuite
		want string
	}{
		{
			name: "Testing GetNamespace when Namespace is not empty",
			sds: &SnapshotDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Namespace: "test-namespace",
				},
			},
			want: "test-namespace",
		},
		{
			name: "Testing GetNamespace when Namespace is empty",
			sds: &SnapshotDeletionSuite{
				DeletionStruct: &DeletionStruct{
					Namespace: "",
				},
			},
			want: "",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sds.GetNamespace(); got != tt.want {
				t.Errorf("SnapshotDeletionSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotDeletionSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		sds  *SnapshotDeletionSuite
		want string
	}{
		{
			name: "Testing Parameters",
			sds:  &SnapshotDeletionSuite{},
			want: "{}",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sds.Parameters(); got != tt.want {
				t.Errorf("SnapshotDeletionSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEphemeralVolumeSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		e    *EphemeralVolumeSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			e:    &EphemeralVolumeSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PodObserver{},
				&observer.VaObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			e:    &EphemeralVolumeSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EphemeralVolumeSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEphemeralVolumeSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		e    *EphemeralVolumeSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			e:    &EphemeralVolumeSuite{},
			want: "functional-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.GetNamespace(); got != tt.want {
				t.Errorf("EphemeralVolumeSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEphemeralVolumeSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		ep   *EphemeralVolumeSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			ep: &EphemeralVolumeSuite{
				Description: "test-description",
			},
			want: "EphemeralVolumeSuite",
		},
		{
			name: "Testing GetName when Description is empty",
			ep: &EphemeralVolumeSuite{
				Description: "",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ep.GetName(); got == tt.want {
				t.Errorf("EphemeralVolumeSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEphemeralVolumeSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		ep   *EphemeralVolumeSuite
		want string
	}{
		{
			name: "Testing Parameters with driver, podNumber and volumeAttributes",
			ep: &EphemeralVolumeSuite{
				Driver:           "powerstore",
				PodNumber:        10,
				VolumeAttributes: map[string]string{"key1": "value1", "key2": "value2"},
			},
			want: "{driver: powerstore, podNumber: 10, volAttributes: map[key1:value1 key2:value2]}",
		},
		{
			name: "Testing Parameters with empty driver, podNumber and volumeAttributes",
			ep: &EphemeralVolumeSuite{
				Driver:           "",
				PodNumber:        0,
				VolumeAttributes: map[string]string{},
			},
			want: "{driver: , podNumber: 0, volAttributes: map[]}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ep.Parameters(); got != tt.want {
				t.Errorf("EphemeralVolumeSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeDrainSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		nds  *NodeDrainSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			nds: &NodeDrainSuite{
				Description: "test-description",
			},
			want: "NodeDrainSuite",
		},
		{
			name: "Testing GetName when Description is empty",
			nds: &NodeDrainSuite{
				Description: "",
			},
			want: "",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nds.GetName(); got == tt.want {
				t.Errorf("NodeDrainSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeDrainSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		n    *NodeDrainSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			n:    &NodeDrainSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PodObserver{},
				&observer.PvcObserver{},
				&observer.VaObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			n:    &NodeDrainSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeDrainSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeDrainSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		nds  *NodeDrainSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			nds: &NodeDrainSuite{
				Namespace: "test-namespace",
			},
			want: "test-namespace",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nds.GetNamespace(); got != tt.want {
				t.Errorf("NodeDrainSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeDrainSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		nds  *NodeDrainSuite
		want string
	}{
		{
			name: "Testing Parameters",
			nds:  &NodeDrainSuite{},
			want: "{}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nds.Parameters(); got != tt.want {
				t.Errorf("NodeDrainSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeUncordonSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		nds  *NodeUncordonSuite
		want string
	}{
		{
			name: "Testing GetName when Description is not empty",
			nds: &NodeUncordonSuite{
				Description: "test-description",
			},
			want: "NodeUncordonSuite",
		},
		{
			name: "Testing GetName when Description is empty",
			nds: &NodeUncordonSuite{
				Description: "",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nds.GetName(); got == tt.want {
				t.Errorf("NodeUncordonSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeUncordonSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		n    *NodeUncordonSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			n:    &NodeUncordonSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PodObserver{},
				&observer.PvcObserver{},
				&observer.VaObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			n:    &NodeUncordonSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeUncordonSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeUncordonSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		nds  *NodeUncordonSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			nds: &NodeUncordonSuite{
				Namespace: "test-namespace",
			},
			want: "test-namespace",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nds.GetNamespace(); got != tt.want {
				t.Errorf("NodeUncordonSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeUncordonSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		nds  *NodeUncordonSuite
		want string
	}{
		{
			name: "Testing Parameters",
			nds:  &NodeUncordonSuite{},
			want: "{}",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nds.Parameters(); got != tt.want {
				t.Errorf("NodeUncordonSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
