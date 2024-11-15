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

package pod_test

import (
	"context"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type PodTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
}

func (suite *PodTestSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()

	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)
}

func (suite *PodTestSuite) TestMakePod() {
	podconf := &pod.Config{
		NamePrefix:     "pod-prov-test-",
		PvcNames:       []string{"pvc1", "pvc2", "pvc3"},
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "prov-test",
		ContainerImage: "quay.io/centos/centos:latest",
		Command:        []string{"/app/run.sh"},
	}

	podconf.VolumeMode = "Block"

	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)
	podTmpl := podClient.MakePod(podconf)
	suite.NoError(err)
	suite.Equal("test-namespace", podTmpl.Namespace)
	suite.Equal([]string{"vol0", "vol1", "vol2"}, []string{
		podTmpl.Spec.Volumes[0].Name,
		podTmpl.Spec.Volumes[1].Name,
		podTmpl.Spec.Volumes[2].Name,
	})
	suite.Equal([]string{"pvc1", "pvc2", "pvc3"}, []string{
		podTmpl.Spec.Volumes[0].PersistentVolumeClaim.ClaimName,
		podTmpl.Spec.Volumes[1].PersistentVolumeClaim.ClaimName,
		podTmpl.Spec.Volumes[2].PersistentVolumeClaim.ClaimName,
	})
	suite.Equal(podconf.ContainerName, podTmpl.Spec.Containers[0].Name)
	suite.Equal(podconf.ContainerImage, podTmpl.Spec.Containers[0].Image)
	suite.Equal(podconf.Command, podTmpl.Spec.Containers[0].Command)
}

func (suite *PodTestSuite) TestMakePod_default() {
	podconf := &pod.Config{}
	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)
	podTmpl := podClient.MakePod(podconf)
	suite.NoError(err)
	suite.Equal("test-namespace", podTmpl.Namespace)
	suite.Equal(podconf.ContainerImage, "quay.io/centos/centos:latest")
	suite.Equal(podconf.Command, []string{"/bin/bash"})
}

func (suite *PodTestSuite) TestCreatePod() {
	type fields struct {
		KubeClient *k8sclient.KubeClient
		Namespace  string
	}
	type args struct {
		pod *v1.Pod
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		assertFunc func(pod *pod.Pod)
	}{
		{
			name: "nil pod object",
			fields: fields{
				KubeClient: suite.kubeClient,
				Namespace:  "test-namespace",
			},
			args: args{
				pod: nil,
			},
			wantErr: true,
			assertFunc: func(pod *pod.Pod) {
				suite.Nil(pod.Object)
			},
		},
		{
			name: "empty pod object",
			fields: fields{
				KubeClient: suite.kubeClient,
				Namespace:  "test-namespace",
			},
			args: args{
				pod: &v1.Pod{},
			},
			wantErr: false,
			assertFunc: func(pod *pod.Pod) {
				suite.NotNil(pod.Object)
				suite.Equal("", pod.Object.Name)
				suite.Equal("test-namespace", pod.Object.Namespace)
			},
		},
		{
			name: "simple pod object",
			fields: fields{
				KubeClient: suite.kubeClient,
				Namespace:  "test-namespace",
			},
			args: args{
				pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "simple-pod",
					},
				},
			},
			wantErr: false,
			assertFunc: func(pod *pod.Pod) {
				suite.NotNil(pod.Object)
				suite.Equal("simple-pod", pod.Object.Name)
				suite.Equal("test-namespace", pod.Object.Namespace)
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			client, err := suite.kubeClient.CreatePodClient("test-namespace")
			suite.NoError(err)

			pod := client.Create(context.Background(), tt.args.pod)
			if tt.wantErr {
				suite.Error(pod.GetError())
			} else {
				suite.NoError(pod.GetError())
			}
			tt.assertFunc(pod)
		})
	}
}

func TestPodTestSuite(t *testing.T) {
	suite.Run(t, new(PodTestSuite))
}
