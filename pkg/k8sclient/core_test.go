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

package k8sclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

type CoreTestSuite struct {
	suite.Suite
	kubeClient KubeClient
}

func (suite *CoreTestSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()

	suite.kubeClient = KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
		timeout:     1,
	}
}

func (suite *CoreTestSuite) TestNewKubeClient() {
	suite.Run("nil config", func() {
		client, err := NewKubeClient(nil, 0)
		suite.Error(err)
		suite.Nil(client)
	})

	suite.Run("empty config", func() {
		client, err := NewKubeClient(&rest.Config{}, 0)
		suite.NoError(err)
		suite.NotNil(client)
	})

	suite.Run("incorrect config", func() {
		client, err := NewKubeClient(&rest.Config{Host: "localhost"}, 0)
		suite.Error(err)
		suite.Nil(client)
	})
}

func (suite *CoreTestSuite) TestCreateClients() {
	namespace := "test-namespace"
	suite.Run("pvc client", func() {
		pvcClient, err := suite.kubeClient.CreatePVCClient(namespace)
		suite.NoError(err)
		suite.NotNil(pvcClient)
		suite.Equal(namespace, pvcClient.Namespace)
	})
	suite.Run("pod client", func() {
		podClient, err := suite.kubeClient.CreatePodClient(namespace)
		suite.NoError(err)
		suite.NotNil(podClient)
		suite.Equal(namespace, podClient.Namespace)
	})

	suite.Run("va client", func() {
		vaClient, err := suite.kubeClient.CreateVaClient(namespace)
		suite.NoError(err)
		suite.NotNil(vaClient)
		suite.Equal(namespace, vaClient.Namespace)
	})

	suite.Run("sts client", func() {
		stsClient, err := suite.kubeClient.CreateStatefulSetClient(namespace)
		suite.NoError(err)
		suite.NotNil(stsClient)
		suite.Equal(namespace, stsClient.Namespace)
	})

	suite.Run("m client", func() {
		mClient, err := suite.kubeClient.CreateMetricsClient(namespace)
		suite.NoError(err)
		suite.NotNil(mClient)
		suite.Equal(namespace, mClient.Namespace)
	})
}

func (suite *CoreTestSuite) TestCreateNamespace() {
	type fields struct {
		ClientSet   kubernetes.Interface
		Config      *rest.Config
		VersionInfo *version.Info
		timeout     int
	}
	type args struct {
		namespace string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		assertFunc func(got *v1.Namespace)
	}{
		{
			name: "create empty",
			fields: fields{
				ClientSet:   suite.kubeClient.ClientSet,
				Config:      suite.kubeClient.Config,
				VersionInfo: suite.kubeClient.VersionInfo,
				timeout:     1,
			},
			args:    args{},
			wantErr: false,
			assertFunc: func(got *v1.Namespace) {
				want := &v1.Namespace{}
				suite.NotNil(got)
				suite.Equal(want.Name, got.Name, "different name was given")
			},
		},
		{
			name: "create test namespace",
			fields: fields{
				ClientSet:   suite.kubeClient.ClientSet,
				Config:      suite.kubeClient.Config,
				VersionInfo: suite.kubeClient.VersionInfo,
				timeout:     1,
			},
			args: args{
				namespace: "test-namespace",
			},
			wantErr: false,
			assertFunc: func(got *v1.Namespace) {
				want := &v1.Namespace{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
					Spec:   v1.NamespaceSpec{},
					Status: v1.NamespaceStatus{},
				}
				suite.NotNil(got)
				suite.Equal(want.Name, got.Name, "different name was given")
			},
		},
		{
			name: "create error",
			fields: fields{
				ClientSet:   suite.kubeClient.ClientSet,
				Config:      suite.kubeClient.Config,
				VersionInfo: suite.kubeClient.VersionInfo,
				timeout:     1,
			},
			args: args{
				namespace: "test-namespace",
			},
			wantErr: true,
			assertFunc: func(got *v1.Namespace) {
				suite.Nil(got)
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			c := &KubeClient{
				ClientSet:   tt.fields.ClientSet,
				Config:      tt.fields.Config,
				VersionInfo: tt.fields.VersionInfo,
				timeout:     tt.fields.timeout,
			}
			got, err := c.CreateNamespace(context.Background(), tt.args.namespace)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			tt.assertFunc(got)
		})
	}

	suite.Run("create with suffix", func() {
		c := &KubeClient{
			ClientSet:   suite.kubeClient.ClientSet,
			Config:      suite.kubeClient.Config,
			VersionInfo: suite.kubeClient.VersionInfo,
			timeout:     1,
		}
		got, err := c.CreateNamespaceWithSuffix(context.Background(), "test-namespace")
		suite.NoError(err)
		suite.NotNil(got)
		suite.Contains(got.Name, "test-namespace-")
	})
}

func (suite *CoreTestSuite) TestDeleteNamespace() {
	client := fake.NewSimpleClientset()

	kubeClient := KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
		timeout:     1,
	}

	name := "test-namespace"
	err := kubeClient.DeleteNamespace(context.Background(), name)
	suite.Error(err)

	ns, err := kubeClient.CreateNamespace(context.Background(), name)
	suite.NoError(err)

	err = kubeClient.DeleteNamespace(context.Background(), ns.Name)
	suite.NoError(err)
}

func (suite *CoreTestSuite) TestStorageExists() {
	storageClass, err := suite.kubeClient.ClientSet.StorageV1().StorageClasses().Create(context.Background(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "powerstore",
		},
	}, metav1.CreateOptions{})
	suite.NoError(err)
	exists, err := suite.kubeClient.StorageClassExists(context.Background(), storageClass.Name)
	suite.NoError(err)
	suite.Equal(true, exists)
}

func (suite *CoreTestSuite) TestNamespaceExists() {
	client := fake.NewSimpleClientset()

	kubeClient := KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
		timeout:     1,
	}

	name := "test-namespace"
	exists, err := kubeClient.NamespaceExists(context.Background(), name)
	suite.NoError(err)
	suite.Equal(false, exists)

	_, err = kubeClient.CreateNamespace(context.Background(), name)
	suite.NoError(err)

	exists, err = kubeClient.NamespaceExists(context.Background(), name)
	suite.NoError(err)
	suite.Equal(true, exists)
}

func (suite *CoreTestSuite) TestGetConfig() {
	conf, err := GetConfig("testdata/config")
	suite.NoError(err)
	suite.NotNil(conf)

	errConf, err := GetConfig("/non/existing/path")
	suite.Error(err)
	suite.Nil(errConf)
}

func TestCoreTestSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}

func TestNewRemoteKubeClient(t *testing.T) {
	t.Run("Testing with nil config", func(t *testing.T) {
		_, err := NewRemoteKubeClient(nil, 0)
		if err == nil {
			t.Error("Expected error, but got nil")
		}
	})

	t.Run("Testing with valid config", func(t *testing.T) {
		config := &rest.Config{
			Host: "https://example.com",
		}
		client, err := NewRemoteKubeClient(config, 0)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if client == nil {
			t.Error("Expected client, but got nil")
		}
	})
}

func TestCreateNodeClient(t *testing.T) {
	// Test case: Create node client successfully
	c := &KubeClient{
		ClientSet: &kubernetes.Clientset{},
		timeout:   10,
	}
	node, err := c.CreateNodeClient()
	if err != nil {
		t.Errorf("CreateNodeClient() failed: %v", err)
	}
	if node == nil {
		t.Error("CreateNodeClient() returned nil node")
	}
	if node.Interface == nil {
		t.Error("CreateNodeClient() returned nil node interface")
	}
	if node.Timeout != c.timeout {
		t.Errorf("CreateNodeClient() returned incorrect timeout: got %d, want %d", node.Timeout, c.timeout)
	}
}

func TestCreateRGClient(t *testing.T) {
	// Test case: Failed creation of replication group client
	t.Run("Create RG client successfully", func(t *testing.T) {
		kubeClient := &KubeClient{
			Config:    &rest.Config{},
			ClientSet: &kubernetes.Clientset{},
			timeout:   1,
		}

		rgClient, err := kubeClient.CreateRGClient()
		if err == nil {
			t.Errorf("Error not returned")
		}

		if rgClient != nil {
			t.Error("Expected nil RG client, got nil")
		}
	})
}
