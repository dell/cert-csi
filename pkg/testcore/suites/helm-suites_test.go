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

package suites

import (
	"context"
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestPostgresqlSuite_Run(t *testing.T) {
	client := fake.NewSimpleClientset()

	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}
	kubeClient.SetTimeout(1)

	// Create a test PostgresqlSuite
	ps := PostgresqlSuite{
		ConfigPath:        "/root/.kube/config",
		VolumeSize:        "10Gi",
		EnableReplication: true,
		Image:             "docker.io/bitnami/postgresql:11.8.0-debian-10-r72",
		SlaveReplicas:     2,
	}

	pvcClient, _ := kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := kubeClient.CreatePodClient("test-namespace")

	// Create test clients
	clients := &k8sclient.Clients{
		PVCClient: pvcClient,
		PodClient: podClient,
	}

	// Create a test context
	ctx := context.Background()

	type args struct {
		ctx          context.Context
		storageClass string
		clients      *k8sclient.Clients
	}

	tests := []struct {
		name           string
		args           args
		wantDelFunc    bool
		wantErr        bool
		wantDelFuncErr bool
	}{
		{"invalid namespace run", args{ctx, "test-storage-class", clients}, false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDelFunc, err := ps.Run(ctx, "test-storage-class", clients)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (gotDelFunc != nil) != tt.wantDelFunc {
				t.Errorf("Run() delFunc error = %v, wantErr %v", err, tt.wantDelFunc)
				return
			}
			if (gotDelFunc != nil && gotDelFunc() != nil) != tt.wantDelFuncErr {
				t.Errorf("Run() delFunc run error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestPostgresqlSuite_GetObservers(t *testing.T) {
	type args struct {
		obsType observer.Type
	}
	tests := []struct {
		name string
		p    *PostgresqlSuite
		args args
		want []observer.Interface
	}{
		{
			name: "Testing GetObservers with event type",
			p:    &PostgresqlSuite{},
			args: args{
				obsType: observer.EVENT,
			},
			want: []observer.Interface{
				&observer.PvcObserver{},
				&observer.VaObserver{},
				&observer.PodObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name: "Testing GetObservers with list type",
			p:    &PostgresqlSuite{},
			args: args{
				obsType: observer.LIST,
			},
			want: []observer.Interface{
				&observer.PvcListObserver{},
				&observer.VaListObserver{},
				&observer.PodListObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.GetObservers(tt.args.obsType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PostgresqlSuite.GetObservers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostgresqlSuite_GetNamespace(t *testing.T) {
	tests := []struct {
		name string
		p    *PostgresqlSuite
		want string
	}{
		{
			name: "Testing GetNamespace",
			p:    &PostgresqlSuite{},
			want: "psql-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.GetNamespace(); got != tt.want {
				t.Errorf("PostgresqlSuite.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostgresqlSuite_GetName(t *testing.T) {
	tests := []struct {
		name string
		ps   *PostgresqlSuite
		want string
	}{
		{
			name: "Testing GetName",
			ps:   &PostgresqlSuite{},
			want: "PostgresqlSuite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.GetName(); got != tt.want {
				t.Errorf("PostgresqlSuite.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostgresqlSuite_Parameters(t *testing.T) {
	tests := []struct {
		name string
		ps   *PostgresqlSuite
		want string
	}{
		{
			name: "Testing Parameters",
			ps:   &PostgresqlSuite{ConfigPath: "test", VolumeSize: "1G", EnableReplication: true, Image: "test", SlaveReplicas: 1},
			want: "{replicas: 1, volumeSize: 1G, replication: true}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.Parameters(); got != tt.want {
				t.Errorf("PostgresqlSuite.Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
