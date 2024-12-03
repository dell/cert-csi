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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	clientgotesting "k8s.io/client-go/testing"

	discoveryFake "k8s.io/client-go/discovery/fake"
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
		Name:           "test-pod",
		NamePrefix:     "pod-prov-test-",
		PvcNames:       []string{"pvc1", "pvc2", "pvc3"},
		VolumeName:     "vol",
		VolumeMode:     "Block",
		MountPath:      "/data",
		ContainerName:  "prov-test",
		ContainerImage: "quay.io/centos/centos:latest",
		Command:        []string{"/app/run.sh"},
	}

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

func (suite *PodTestSuite) TestMakePodFromYaml() {
	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)
	podTmpl := podClient.MakePodFromYaml("")
	suite.Equal(podTmpl.Name, "")
}

func (suite *PodTestSuite) TestCreatePod() {
	type fields struct {
		KubeClient *k8sclient.KubeClient
		Namespace  string
	}
	type args struct {
		pod *corev1.Pod
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
				pod: &corev1.Pod{},
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
				pod: &corev1.Pod{
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

func (suite *PodTestSuite) TestDelete() {
	podconf := &pod.Config{
		NamePrefix:     "pod-prov-test-",
		PvcNames:       []string{"pvc1", "pvc2", "pvc3"},
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "prov-test",
		ContainerImage: "quay.io/centos/centos:latest",
		Command:        []string{"/app/run.sh"},
	}

	client, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	podTmpl := client.MakePod(podconf)
	suite.Equal("test-namespace", podTmpl.Namespace)

	suite.Run("Delete pod test", func() {
		result := client.Delete(context.Background(), podTmpl)
		suite.NoError(result.GetError())
		suite.Equal(result.HasError(), false)

		result = client.Delete(context.Background(), podTmpl)
		suite.Error(result.GetError())
		suite.Equal(result.HasError(), true)
	})
}

func (suite *PodTestSuite) TestUpdate() {
	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)
	pod := &corev1.Pod{}
	podClient.Update(pod)
}

func (suite *PodTestSuite) TestDeleteAll() {
	client, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	suite.Run("Delete all pod test", func() {
		err := client.DeleteAll(context.Background())
		suite.NoError(err)
	})
}

func (suite *PodTestSuite) TestReadyPodsCount() {
	podconf := &pod.Config{
		NamePrefix:     "pod-prov-test-",
		PvcNames:       []string{"pvc1", "pvc2", "pvc3"},
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "prov-test",
		ContainerImage: "quay.io/centos/centos:latest",
		Command:        []string{"/app/run.sh"},
	}

	client, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	podTmpl := client.MakePod(podconf)
	suite.Equal("test-namespace", podTmpl.Namespace)

	suite.Run("Ready Pods Count test", func() {
		readyCount, err := client.ReadyPodsCount(context.Background())
		suite.Equal(readyCount, 0)
		suite.NoError(err)
	})
}

func (suite *PodTestSuite) TestWaitForAllToBeReady() {
	podconf := &pod.Config{
		NamePrefix:     "pod-prov-test-",
		PvcNames:       []string{"pvc1", "pvc2", "pvc3"},
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "prov-test",
		ContainerImage: "quay.io/centos/centos:latest",
		Command:        []string{"/app/run.sh"},
	}

	client, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	podTmpl := client.MakePod(podconf)
	suite.Equal("test-namespace", podTmpl.Namespace)

	suite.Run("waits for all Pods to be in Ready state test", func() {
		err := client.WaitForAllToBeReady(context.Background())
		suite.NoError(err)
	})
}

func (suite *PodTestSuite) TestWaitForRunning() {
	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	podObj := &pod.Pod{
		Client:  podClient,
		Object:  &corev1.Pod{},
		Deleted: false,
	}

	err = podObj.WaitForRunning(context.Background())
	suite.Error(err)
}

func (suite *PodTestSuite) TestMakeEphemeralPod() {
	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	podconf := &pod.Config{
		Name: "test-pod",
	}
	podClient.MakeEphemeralPod(podconf)

	suite.Equal(podconf.NamePrefix, "pod-")
	suite.Equal(podconf.MountPath, "/data")
	suite.Equal(podconf.ContainerName, "test-container")
	suite.Equal(podconf.ContainerImage, "quay.io/centos/centos:latest")
}

func (suite *PodTestSuite) TestDeleteOrEvictPods() {
	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	err = podClient.DeleteOrEvictPods(context.Background(), "", 10)
	suite.NoError(err)
}

func (suite *PodTestSuite) TestEvictPod() {
	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	pod1 := podClient.MakeEphemeralPod(&pod.Config{
		Name: "test-pod-1",
	})

	err = podClient.EvictPod(context.Background(), *pod1, "", 10)
	suite.Error(err)
}

func (suite *PodTestSuite) TestIsInPendingState() {
	namespace := "test-namespace"
	client := fake.NewSimpleClientset()
	kubeClient := k8sclient.KubeClient{
		ClientSet:   client,
		Config:      &rest.Config{},
		VersionInfo: nil,
	}
	kubeClient.SetTimeout(2)
	podClient, err := kubeClient.CreatePodClient(namespace)
	suite.NoError(err)

	podName := "test-pod"
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	}

	client.CoreV1().Pods(namespace).Create(context.Background(), testPod, metav1.CreateOptions{})

	podObj := &pod.Pod{
		Client:  podClient,
		Object:  testPod,
		Deleted: false,
	}

	suite.Run("Pod is in pending state", func() {
		err = podObj.IsInPendingState(context.Background())
		suite.NoError(err)
	})
}

func TestPodTestSuite(t *testing.T) {
	suite.Run(t, new(PodTestSuite))
}

func TestIsPodReady(t *testing.T) {
	podReady := &corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	podNotReady := &corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	t.Run("Pod is ready", func(t *testing.T) {
		assert.True(t, pod.IsPodReady(podReady), "Expected pod to be ready")
	})

	t.Run("Pod is not ready", func(t *testing.T) {
		assert.False(t, pod.IsPodReady(podNotReady), "Expected pod to not be ready")
	})
}

func TestIsPodReadyConditionTrue(t *testing.T) {
	podReady := corev1.PodStatus{
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
		},
	}

	podNotReady := corev1.PodStatus{
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionFalse,
			},
		},
	}

	t.Run("Pod is ready", func(t *testing.T) {
		assert.True(t, pod.IsPodReadyConditionTrue(podReady), "Expected pod to be ready")
	})

	t.Run("Pod is not ready", func(t *testing.T) {
		assert.False(t, pod.IsPodReadyConditionTrue(podNotReady), "Expected pod to not be ready")
	})
}

func TestGetPodConditionFromList(t *testing.T) {
	tests := []struct {
		name           string
		conditions     []corev1.PodCondition
		conditionType  corev1.PodConditionType
		expectedIndex  int
		expectedResult *corev1.PodCondition
	}{
		{
			name: "Condition exists",
			conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue},
			},
			conditionType: corev1.PodReady,
			expectedIndex: 0,
			expectedResult: &corev1.PodCondition{
				Type: corev1.PodReady, Status: corev1.ConditionTrue,
			},
		},
		{
			name: "Condition does not exist",
			conditions: []corev1.PodCondition{
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue},
			},
			conditionType:  corev1.PodReady,
			expectedIndex:  -1,
			expectedResult: nil,
		},
		{
			name:           "Empty conditions",
			conditions:     nil,
			conditionType:  corev1.PodReady,
			expectedIndex:  -1,
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, result := pod.GetPodConditionFromList(tt.conditions, tt.conditionType)
			if index != tt.expectedIndex {
				t.Errorf("expected index %d, got %d", tt.expectedIndex, index)
			}
			if result == nil && tt.expectedResult != nil || result != nil && tt.expectedResult == nil {
				t.Errorf("expected result %v, got %v", tt.expectedResult, result)
			} else if result != nil && tt.expectedResult != nil && *result != *tt.expectedResult {
				t.Errorf("expected result %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func (suite *PodTestSuite) TestEvictPods() {
	podClient, err := suite.kubeClient.CreatePodClient("test-namespace")
	suite.NoError(err)

	podList := &corev1.PodList{
		Items: []corev1.Pod{
			*podClient.MakeEphemeralPod(&pod.Config{
				Name: "test-pod-1",
			}),
			*podClient.MakeEphemeralPod(&pod.Config{
				Name: "test-pod-2",
			}),
		},
	}

	err = podClient.EvictPods(context.Background(), podList, "", 10)
	suite.Error(err)
}

func TestCheckEvictionSupport(t *testing.T) {
	clientSet := fake.NewSimpleClientset()
	discoveryClient := clientSet.Discovery().(*discoveryFake.FakeDiscovery)

	tests := []struct {
		name                 string
		serverGroups         []metav1.APIGroup
		serverResources      []*metav1.APIResourceList
		expectedGroupVersion string
		expectedError        error
	}{
		{
			name: "Eviction supported",
			serverGroups: []metav1.APIGroup{
				{
					Name: "policy",
					PreferredVersion: metav1.GroupVersionForDiscovery{
						GroupVersion: "v1",
					},
				},
			},
			serverResources: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{
							Name: "pods/eviction",
							Kind: "Eviction",
						},
					},
				},
			},
			expectedGroupVersion: "",
			expectedError:        nil,
		},
		{
			name:                 "Policy group not found",
			serverGroups:         []metav1.APIGroup{},
			serverResources:      nil,
			expectedGroupVersion: "",
			expectedError:        nil,
		},
		{
			name:                 "Eviction not supported",
			serverGroups:         []metav1.APIGroup{},
			serverResources:      []*metav1.APIResourceList{},
			expectedGroupVersion: "",
			expectedError:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discoveryClient.Resources = tt.serverResources
			// Simulate the server groups response
			discoveryClient.Fake.Resources = tt.serverResources
			discoveryClient.Fake.PrependReactor("get", "servergroups", func(_ clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &metav1.APIGroupList{Groups: tt.serverGroups}, nil
			})

			discoveryClient.Fake.PrependReactor("get", "serverresources", func(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
				getAction := action.(clientgotesting.GetAction)
				groupVersion := getAction.GetResource().GroupVersion().String()
				for _, resourceList := range tt.serverResources {
					if resourceList.GroupVersion == groupVersion {
						return true, resourceList, nil
					}
				}
				return true, nil, apierrs.NewNotFound(schema.GroupResource{Group: "policy", Resource: "resource"}, "")
			})

			groupVersion, err := pod.CheckEvictionSupport(clientSet)
			assert.Equal(t, tt.expectedGroupVersion, groupVersion)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}
