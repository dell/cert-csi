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
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

type HelmSuiteTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
}

func (suite *HelmSuiteTestSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()

	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)

	// suite.kubeClient.CreateNamespace(context.Background(), "test-namespace")
}

// func (suite *HelmSuiteTestSuite) TearDownSuite() {
// 	suite.kubeClient.DeleteNamespace(context.Background(), "test-namespace")
// }

func (suite *HelmSuiteTestSuite) TestPostgresqlSuite_Run() {

	// Create a test PostgresqlSuite
	ps := PostgresqlSuite{
		ConfigPath:        "/root/.kube/config",
		EnableReplication: true,
		SlaveReplicas:     2,
	}

	pvcClient, _ := suite.kubeClient.CreatePVCClient("test-namespace")
	podClient, _ := suite.kubeClient.CreatePodClient("test-namespace")

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
		wantErr        bool
		wantDelFunc    bool
		wantDelFuncErr bool
	}{
		{"invalid namespace", args{ctx, "test-storage-class", clients}, true, false, false},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			gotDelFunc, err := ps.Run(ctx, tt.args.storageClass, clients)
			if !tt.wantErr {
				suite.NoError(err)
			}
			suite.Equal((gotDelFunc != nil), tt.wantDelFunc)
			if gotDelFunc != nil && !tt.wantDelFuncErr {
				suite.NoError(gotDelFunc())
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

func (suite *HelmSuiteTestSuite) TestPostgresqlSuite_GetClients() {
	tests := []struct {
		name    string
		ps      *PostgresqlSuite
		ns      string
		wantErr bool
	}{
		{
			name:    "Testing GetClients",
			ps:      &PostgresqlSuite{ConfigPath: "test", VolumeSize: "1G", EnableReplication: true, Image: "test", SlaveReplicas: 1},
			ns:      "test-namespace",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			_, err := tt.ps.GetClients(tt.ns, suite.kubeClient)
			suite.NoError(err)
		})
	}
}

func TestHelmSuiteTestSuite(t *testing.T) {
	suite.Run(t, new(HelmSuiteTestSuite))
}
