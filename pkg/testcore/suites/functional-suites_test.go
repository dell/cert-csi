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
