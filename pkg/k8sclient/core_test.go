/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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

	vs "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapclient "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
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

func (suite *CoreTestSuite) TestNewRemoteKubeClient() {
	suite.Run("nil config", func() {
		client, err := NewRemoteKubeClient(nil, 0)
		suite.Error(err)
		suite.Nil(client)
	})

	suite.Run("empty config", func() {
		client, err := NewRemoteKubeClient(&rest.Config{}, 0)
		suite.NoError(err)
		suite.NotNil(client)
	})
}

func (suite *CoreTestSuite) TestCreateClients() {
	namespace := "test-namespace"
	suite.kubeClient.Config = &rest.Config{}

	suite.Run("pvc client", func() {
		pvcClient, err := suite.kubeClient.CreatePVCClient(namespace)
		suite.NoError(err)
		suite.NotNil(pvcClient)
		suite.Equal(namespace, pvcClient.Namespace)
	})

	suite.Run("pvc client error case", func() {
		podClient, err := suite.kubeClient.CreatePVCClient("")
		suite.Error(err)
		suite.Nil(podClient)
	})

	suite.Run("sc client", func() {
		scClient, err := suite.kubeClient.CreateSCClient()
		suite.NoError(err)
		suite.NotNil(scClient)
	})

	suite.Run("pod client", func() {
		podClient, err := suite.kubeClient.CreatePodClient(namespace)
		suite.NoError(err)
		suite.NotNil(podClient)
		suite.Equal(namespace, podClient.Namespace)
	})

	suite.Run("pod client error case", func() {
		podClient, err := suite.kubeClient.CreatePodClient("")
		suite.Error(err)
		suite.Nil(podClient)
	})

	suite.Run("va client", func() {
		vaClient, err := suite.kubeClient.CreateVaClient(namespace)
		suite.NoError(err)
		suite.NotNil(vaClient)
		suite.Equal(namespace, vaClient.Namespace)
	})

	suite.Run("va client error case", func() {
		vaClient, err := suite.kubeClient.CreateVaClient("")
		suite.Error(err)
		suite.Nil(vaClient)
	})

	suite.Run("sts client", func() {
		stsClient, err := suite.kubeClient.CreateStatefulSetClient(namespace)
		suite.NoError(err)
		suite.NotNil(stsClient)
		suite.Equal(namespace, stsClient.Namespace)
	})

	suite.Run("sts client error case", func() {
		mClient, err := suite.kubeClient.CreateStatefulSetClient("")
		suite.Error(err)
		suite.Nil(mClient)
	})

	suite.Run("m client error case", func() {
		mClient, err := suite.kubeClient.CreateMetricsClient("")
		suite.Error(err)
		suite.Nil(mClient)
	})

	suite.Run("m client", func() {
		mClient, err := suite.kubeClient.CreateMetricsClient(namespace)
		suite.NoError(err)
		suite.NotNil(mClient)
		suite.Equal(namespace, mClient.Namespace)
	})

	suite.Run("node client", func() {
		nodeClient, err := suite.kubeClient.CreateNodeClient()
		suite.NoError(err)
		suite.NotNil(nodeClient)
	})

	suite.Run("snapshot GA client", func() {
		client, err := suite.kubeClient.CreateSnapshotGAClient(namespace)
		suite.NoError(err)
		suite.NotNil(client)
	})

	suite.Run("snapshot GA client error case", func() {
		mClient, err := suite.kubeClient.CreateSnapshotGAClient("")
		suite.Error(err)
		suite.Nil(mClient)
	})

	suite.Run("snapshot content ga client", func() {
		client, err := suite.kubeClient.CreateSnapshotContentGAClient()
		suite.NoError(err)
		suite.NotNil(client)
	})

	suite.Run("csi client", func() {
		client, err := suite.kubeClient.CreateCSISCClient(namespace)
		suite.NoError(err)
		suite.NotNil(client)
	})

	suite.Run("csi client error case", func() {
		mClient, err := suite.kubeClient.CreateCSISCClient("")
		suite.Error(err)
		suite.Nil(mClient)
	})

	suite.kubeClient.Config.QPS = 1
	suite.kubeClient.Config.Burst = 0
	suite.Run("snapshot GA client", func() {
		_, err := suite.kubeClient.CreateSnapshotGAClient(namespace)
		suite.Error(err)
	})

	suite.Run("snapshot content ga client", func() {
		_, err := suite.kubeClient.CreateSnapshotContentGAClient()
		suite.Error(err)
	})

	// Rg client and vgs client creation returns error becuase of connection errors
	suite.Run("rg client", func() {
		_, err := suite.kubeClient.CreateRGClient()
		suite.Error(err)
	})

	suite.Run("vgs client", func() {
		_, err := suite.kubeClient.CreateVGSClient()
		suite.Error(err)
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

	suite.Run("create with suffix error case", func() {
		c := &KubeClient{
			ClientSet:   suite.kubeClient.ClientSet,
			Config:      suite.kubeClient.Config,
			VersionInfo: suite.kubeClient.VersionInfo,
			timeout:     1,
		}
		got, err := c.CreateNamespaceWithSuffix(context.Background(), "")
		suite.Error(err)
		suite.Nil(got)
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

	_, err = kubeClient.CreateNamespace(context.Background(), name)
	suite.NoError(err)

	err = kubeClient.DeleteNamespace(context.Background(), name)
	suite.NoError(err)

	err = kubeClient.DeleteNamespace(context.Background(), "fake-namespace")
	suite.Error(err)
}

func (suite *CoreTestSuite) TestForceDeleteNamespace() {
	client := fake.NewSimpleClientset()

	kubeClient := KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
		timeout:     1,
	}

	name := "test-namespace"

	ns, err := kubeClient.CreateNamespace(context.Background(), name)
	suite.NoError(err)

	err = kubeClient.ForceDeleteNamespace(context.Background(), ns.Name)
	suite.NoError(err)

	kubeClient.Minor = 17

	err = kubeClient.ForceDeleteNamespace(context.Background(), ns.Name)
	suite.NoError(err)

	err = kubeClient.ForceDeleteNamespace(context.Background(), "")
	suite.Error(err)
}

func (suite *CoreTestSuite) TestSnapshotClassExists() {
	suite.kubeClient.Config = &rest.Config{}
	cset, _ := snapclient.NewForConfig(suite.kubeClient.Config)
	volumeSnapshotClass := &vs.VolumeSnapshotClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-snapshot-class",
		},
	}
	cset.SnapshotV1().VolumeSnapshotClasses().Create(context.Background(), volumeSnapshotClass, metav1.CreateOptions{})

	exists, err := suite.kubeClient.SnapshotClassExists("test-snapshot-class")
	suite.Error(err)
	suite.Equal(false, exists)

	// Test case: Snapshot class doesn't exist
	exists, err = suite.kubeClient.SnapshotClassExists("non-existent-snapshot-class")
	suite.Error(err)
	suite.False(exists)

	exists, err = suite.kubeClient.SnapshotClassExists("")
	suite.Error(err)
	suite.False(exists)
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
	exists, err = suite.kubeClient.StorageClassExists(context.Background(), "")
	suite.Error(err)
	suite.Equal(false, exists)
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
	suite.Error(err)
	suite.Equal(false, exists)

	_, err = kubeClient.CreateNamespace(context.Background(), name)
	suite.NoError(err)

	exists, err = kubeClient.NamespaceExists(context.Background(), name)
	suite.NoError(err)
	suite.Equal(true, exists)

	exists, err = kubeClient.NamespaceExists(context.Background(), "namespace-does-not-exist")
	suite.Error(err)
	suite.Equal(false, exists)
}

func (suite *CoreTestSuite) TestSetTimeout() {
	kubeClient := &KubeClient{
		timeout: 0,
	}

	kubeClient.SetTimeout(10)
	suite.Equal(kubeClient.timeout, 10)

	kubeClient.SetTimeout(-1)
	suite.Equal(kubeClient.timeout, -1)

	kubeClient.SetTimeout(0)
	suite.Equal(kubeClient.timeout, 0)
}

func (suite *CoreTestSuite) TestGetConfig() {
	conf, err := GetConfig("testdata/config")
	suite.NoError(err)
	suite.NotNil(conf)

	errConf, err := GetConfig("/non/existing/path")
	suite.Error(err)
	suite.Nil(errConf)

	errConf, err = GetConfig("testdata/empty_config.yaml")
	suite.Error(err)
	suite.Nil(errConf)

	_, err = GetConfig("")
	suite.Error(err)
}

func TestCoreTestSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}
