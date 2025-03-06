package statefulset_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	t "testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/statefulset"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v12 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

type StatefulSetTestSuite struct {
	suite.Suite
	stsClient  *statefulset.Client
	kubeClient *k8sclient.KubeClient
}

func (suite *StatefulSetTestSuite) SetupSuite() {
	client := fake.NewSimpleClientset()
	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)
}

func (suite *StatefulSetTestSuite) SetupTest() {
	suite.kubeClient.ClientSet = fake.NewSimpleClientset()
	stsClient, err := suite.kubeClient.CreateStatefulSetClient("default")
	suite.NoError(err)
	suite.stsClient = stsClient

	sts := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-sts",
		},
	}

	// Attempt to delete the STS *before* creating it.  Ignore errors if it doesn't exist.
	delStatus := stsClient.Delete(context.Background(), sts)
	if delStatus.GetError() != nil && !apierrs.IsNotFound(delStatus.GetError()) {
		suite.NoError(delStatus.GetError())
	}
}

func (suite *StatefulSetTestSuite) TestMakeStatefulSet() {
	tests := []struct {
		name     string
		config   *statefulset.Config
		validate func(*v1.StatefulSet)
	}{
		{
			name:   "Default Values",
			config: &statefulset.Config{},
			validate: func(sts *v1.StatefulSet) {
				suite.NotNil(sts)
				suite.Equal(int32(1), *sts.Spec.Replicas)
				suite.Equal("vol0", sts.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
				// suite.Equal("3Gi", sts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[v1.ResourceStorage].String())
				suite.Equal("Parallel", string(sts.Spec.PodManagementPolicy))
			},
		},
		{
			name: "Multiple Volumes",
			config: &statefulset.Config{
				VolumeNumber: 3,
			},
			validate: func(sts *v1.StatefulSet) {
				suite.NotNil(sts)
				suite.Equal(3, len(sts.Spec.Template.Spec.Containers[0].VolumeMounts))
				suite.Equal(3, len(sts.Spec.VolumeClaimTemplates))
			},
		},
		{
			name: "Empty Command and Args",
			config: &statefulset.Config{
				Command: []string{},
				Args:    []string{},
			},
			validate: func(sts *v1.StatefulSet) {
				suite.NotNil(sts)
				suite.Equal([]string{"/bin/bash"}, sts.Spec.Template.Spec.Containers[0].Command)
				suite.Empty(sts.Spec.Template.Spec.Containers[0].Args)
			},
		},
		{
			name: "Custom Pod Management Policy",
			config: &statefulset.Config{
				PodManagementPolicy: "OrderedReady",
			},
			validate: func(sts *v1.StatefulSet) {
				suite.NotNil(sts)
				suite.Equal("OrderedReady", string(sts.Spec.PodManagementPolicy))
			},
		},
		{
			name: "Existing Scenario",
			config: &statefulset.Config{
				VolumeNumber:     1,
				Replicas:         1,
				VolumeName:       "vol",
				MountPath:        "/data",
				StorageClassName: "standard",
				ContainerName:    "test-container",
				ContainerImage:   "quay.io/centos/centos:latest",
				NamePrefix:       "sts-",
				Command:          []string{"/bin/bash"},
				Args:             []string{"-c", "echo Hello World"},
				Annotations:      map[string]string{"annotation-key": "annotation-value"},
				Labels:           map[string]string{"label-key": "label-value"},
			},
			validate: func(sts *v1.StatefulSet) {
				suite.NotNil(sts)
				suite.Equal(int32(1), *sts.Spec.Replicas)
				suite.Equal("test-container", sts.Spec.Template.Spec.Containers[0].Name)
				suite.Equal("quay.io/centos/centos:latest", sts.Spec.Template.Spec.Containers[0].Image)
				suite.Equal([]string{"/bin/bash"}, sts.Spec.Template.Spec.Containers[0].Command)
				suite.Equal([]string{"-c", "echo Hello World"}, sts.Spec.Template.Spec.Containers[0].Args)
				suite.Equal(map[string]string{"annotation-key": "annotation-value"}, sts.Annotations)
				suite.Equal(map[string]string{"label-key": "label-value"}, sts.Spec.Template.Labels)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			stsClient, _ := suite.kubeClient.CreateStatefulSetClient("default")
			sts := stsClient.MakeStatefulSet(tt.config)
			tt.validate(sts)
		})
	}
}

func (suite *StatefulSetTestSuite) TestMakeStatefulSetFromYaml() {
	stsYaml := `
apiVersion: apps/v1
kind: StatefulSet
metadata:
 name: test-sts
spec:
 selector:
   matchLabels:
     app: test
 serviceName: "test"
 replicas: 1
 template:
   metadata:
     labels:
       app: test
   spec:
     containers:
     - name: test-container
       image: quay.io/centos/centos:latest
       ports:
       - containerPort: 80
 volumeClaimTemplates:
 - metadata:
     name: test-pvc
   spec:
     accessModes: [ "ReadWriteOnce" ]
     resources:
       requests:
         storage: 1Gi
`

	tmpFile, err := os.CreateTemp("", "sts-*.yaml")
	suite.NoError(err)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Error(err)
		}
	}(tmpFile.Name())

	_, err = tmpFile.Write([]byte(stsYaml))
	suite.NoError(err)
	err = tmpFile.Close()
	if err != nil {
		return
	}

	suite.NoError(err)
	sts := suite.stsClient.MakeStatefulSetFromYaml(tmpFile.Name())
	suite.NotNil(sts)
	suite.Equal("test-sts", sts.Name)
	suite.Equal(int32(1), *sts.Spec.Replicas)
	suite.Equal("test-container", sts.Spec.Template.Spec.Containers[0].Name)
	suite.Equal("quay.io/centos/centos:latest", sts.Spec.Template.Spec.Containers[0].Image)
}

func (suite *StatefulSetTestSuite) TestCreate() {
	stsClient, _ := suite.kubeClient.CreateStatefulSetClient("default")

	sts := &v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "test-sts"}}
	result := stsClient.Create(context.Background(), sts)
	suite.NoError(result.GetError())
	suite.False(result.Deleted)
	suite.Equal(sts.Name, result.Set.Name)
}

func (suite *StatefulSetTestSuite) TestDelete() {
	stsClient, _ := suite.kubeClient.CreateStatefulSetClient("default")

	sts := &v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "test-sts1"}}
	createdSts := stsClient.Create(context.Background(), sts)
	suite.NoError(createdSts.GetError())

	deletedSts := stsClient.Delete(context.Background(), createdSts.Set)
	suite.NoError(deletedSts.GetError())
	suite.True(deletedSts.Deleted)
}

func (suite *StatefulSetTestSuite) TestDeleteWithOptions() {
	tests := []struct {
		name          string
		stsStatus     string
		expectDeleted bool
		expectedError string
	}{
		{
			name:          "Successful deletion",
			stsStatus:     "",
			expectDeleted: true,
			expectedError: "",
		},
		{
			name:          "StatefulSet in Terminating status",
			stsStatus:     "Terminating",
			expectDeleted: true,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			fakeClient := fake.NewSimpleClientset()

			// Create the StatefulSet object with the given status
			sts := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sts",
					Namespace: "default",
				},
				Status: v1.StatefulSetStatus{
					Conditions: []v1.StatefulSetCondition{
						{
							Type:   v1.StatefulSetConditionType(tt.stsStatus),
							Status: v12.ConditionTrue,
						},
					},
				},
			}
			_, err := fakeClient.AppsV1().StatefulSets("default").Create(context.TODO(), sts, metav1.CreateOptions{})
			suite.NoError(err)

			// Initialize the StatefulSet client
			stsClient := &statefulset.Client{
				Interface: fakeClient.AppsV1().StatefulSets("default"),
			}

			// Define delete options
			deleteOptions := metav1.DeleteOptions{}

			// Call DeleteWithOptions
			deletedSts := stsClient.DeleteWithOptions(context.TODO(), sts, deleteOptions)

			// Validate results
			suite.Equal(tt.expectDeleted, deletedSts.Deleted, "expected deleted status does not match")
			if tt.expectedError == "" {
				suite.NoError(deletedSts.GetError())
			} else {
				suite.Contains(deletedSts.GetError().Error(), tt.expectedError)
			}
		})
	}
}

func (suite *StatefulSetTestSuite) TestDeleteAll() {
	suite.kubeClient.ClientSet = fake.NewSimpleClientset()
	stsClient, _ := suite.kubeClient.CreateStatefulSetClient("default")

	sts := &v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "test-sts123"}}
	createdSts := stsClient.Create(context.Background(), sts)
	suite.NoError(createdSts.GetError())

	err := stsClient.DeleteAll(context.Background())
	suite.NoError(err)
}

func (suite *StatefulSetTestSuite) TestDeleteAll_ListFails() {
	suite.kubeClient.ClientSet = fake.NewSimpleClientset()
	suite.stsClient, _ = suite.kubeClient.CreateStatefulSetClient("default")

	ctx := context.Background()

	// Mock List function to return an error
	suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("list", "statefulsets",
		func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("mock list error")
		})

	err := suite.stsClient.DeleteAll(ctx)
	suite.Error(err)
	suite.Contains(err.Error(), "mock list error")
}

func (suite *StatefulSetTestSuite) TestDeleteAll_DeleteFails() {
	suite.kubeClient.ClientSet = fake.NewSimpleClientset()
	suite.stsClient, _ = suite.kubeClient.CreateStatefulSetClient("default")

	ctx := context.Background()

	// Create a test StatefulSet
	sts1 := &v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "-test-sts1", Namespace: "default"}}
	_ = suite.stsClient.Create(ctx, sts1)

	stsList := &v1.StatefulSetList{Items: []v1.StatefulSet{*sts1}}
	suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("list", "statefulsets",
		func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, stsList, nil
		})

	// Mock Delete function to fail
	suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("delete", "statefulsets",
		func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("mock delete error")
		})

	err := suite.stsClient.DeleteAll(ctx)
	suite.NoError(err, "DeleteAll should not return an error when deletion fails")
}

func (suite *StatefulSetTestSuite) TestScale() {
	tests := []struct {
		name            string
		initialReplicas int32
		scaleTo         int32
		updateErrorStep int // Indicates at which step to simulate the update error (0 for no error)
		expectedError   bool
	}{
		{
			name:            "Scale Up",
			initialReplicas: 1,
			scaleTo:         3,
			updateErrorStep: 0,
			expectedError:   false,
		},
		{
			name:            "Scale Down",
			initialReplicas: 3,
			scaleTo:         1,
			updateErrorStep: 0,
			expectedError:   false,
		},
		{
			name:            "Update Error on Retry",
			initialReplicas: 2,
			scaleTo:         4,
			updateErrorStep: 2, // Error on the second UpdateScale invocation
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ctx := context.TODO()
			fakeClient := fake.NewSimpleClientset()

			// Local variables to track replicas and update invocation count
			replicas := tt.initialReplicas
			updateInvocationCount := 0

			// Add a reactor for "get" action on Scale subresource
			fakeClient.Fake.PrependReactor("get", "statefulsets/scale", func(action testing.Action) (bool, runtime.Object, error) {
				getAction := action.(testing.GetAction)
				return true, &autoscalingv1.Scale{
					ObjectMeta: metav1.ObjectMeta{
						Name:      getAction.GetName(),
						Namespace: "default",
					},
					Spec: autoscalingv1.ScaleSpec{
						Replicas: replicas,
					},
					Status: autoscalingv1.ScaleStatus{
						Replicas: replicas,
					},
				}, nil
			})

			// Add a reactor for "update" action on Scale subresource
			fakeClient.Fake.PrependReactor("update", "statefulsets/scale", func(action testing.Action) (bool, runtime.Object, error) {
				updateInvocationCount++
				updateAction := action.(testing.UpdateAction)
				scale := updateAction.GetObject().(*autoscalingv1.Scale)

				if updateInvocationCount == 1 {
					// First UpdateScale call: Simulate a mismatch in replicas
					replicas = tt.initialReplicas // Keep replicas unchanged
					return true, scale, nil
				}

				if updateInvocationCount == tt.updateErrorStep {
					// Simulate an error during the specified step
					return true, nil, fmt.Errorf("update error simulated on step %d", updateInvocationCount)
				}

				// Successful update
				replicas = scale.Spec.Replicas
				return true, scale, nil
			})

			// Initialize StatefulSet client wrapper
			stsClient := &statefulset.Client{
				Interface: fakeClient.AppsV1().StatefulSets("default"),
			}

			// Create a StatefulSet resource
			sts := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sts",
					Namespace: "default",
				},
				Spec: v1.StatefulSetSpec{
					Replicas: &tt.initialReplicas,
				},
			}
			_, err := fakeClient.AppsV1().StatefulSets("default").Create(ctx, sts, metav1.CreateOptions{})
			suite.NoError(err)

			statefulSetResult := stsClient.Scale(ctx, sts, tt.scaleTo)

			if tt.expectedError {
				suite.NotNil(statefulSetResult.GetError(), "expected an error but got none")
				suite.Contains(statefulSetResult.GetError().Error(), "update error simulated", "unexpected error message")
			} else {
				suite.Nil(statefulSetResult.GetError(), "did not expect an error but got one")

				// Validate the replicas count
				scale, err := stsClient.Interface.GetScale(ctx, "test-sts", metav1.GetOptions{})
				suite.NoError(err, "failed to get scale")
				suite.Equal(tt.scaleTo, scale.Spec.Replicas, "replica count mismatch")
			}
		})
	}
}

func (suite *StatefulSetTestSuite) TestScaleAPIError() {
	ctx := context.TODO()
	tests := []struct {
		name             string
		getScaleError    bool
		updateScaleError bool
		expectedError    string
	}{
		{
			name:          "GetScale API Error",
			getScaleError: true,
			expectedError: "failed to get scale",
		},
		{
			name:             "UpdateScale API Error",
			getScaleError:    false,
			updateScaleError: true,
			expectedError:    "failed to update scale",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			fakeClient := fake.NewSimpleClientset()

			// Add a reactor for "get" action on Scale subresource to simulate GetScale error
			if tt.getScaleError {
				fakeClient.Fake.PrependReactor("get", "statefulsets/scale", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("failed to get scale")
				})
			} else {
				// Successful GetScale response
				fakeClient.Fake.PrependReactor("get", "statefulsets/scale", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &autoscalingv1.Scale{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-sts",
							Namespace: "default",
						},
						Spec: autoscalingv1.ScaleSpec{
							Replicas: 2,
						},
						Status: autoscalingv1.ScaleStatus{
							Replicas: 2,
						},
					}, nil
				})
			}

			// Add a reactor for "update" action on Scale subresource to simulate UpdateScale error
			if tt.updateScaleError {
				fakeClient.Fake.PrependReactor("update", "statefulsets/scale", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("failed to update scale")
				})
			} else {
				// Successful UpdateScale response
				fakeClient.Fake.PrependReactor("update", "statefulsets/scale", func(action testing.Action) (bool, runtime.Object, error) {
					updateAction := action.(testing.UpdateAction)
					scale := updateAction.GetObject().(*autoscalingv1.Scale)
					return true, scale, nil
				})
			}

			// Initialize StatefulSet client wrapper
			stsClient := &statefulset.Client{
				Interface: fakeClient.AppsV1().StatefulSets("default"),
			}

			// Create a StatefulSet resource
			sts := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sts",
					Namespace: "default",
				},
				Spec: v1.StatefulSetSpec{
					Replicas: new(int32), // Initialize with a default replica count
				},
			}
			_, err := fakeClient.AppsV1().StatefulSets("default").Create(ctx, sts, metav1.CreateOptions{})
			suite.NoError(err)

			// Call the Scale function
			statefulSetResult := stsClient.Scale(ctx, sts, 3)

			// Assert the error
			suite.NotNil(statefulSetResult.GetError(), "expected an error but got none")
			suite.Contains(statefulSetResult.GetError().Error(), tt.expectedError, "unexpected error message")
		})
	}
}

func (suite *StatefulSetTestSuite) TestScaleContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	fakeClient := fake.NewSimpleClientset()

	// Add a reactor for "get" action on Scale subresource
	fakeClient.Fake.PrependReactor("get", "statefulsets/scale", func(_ testing.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sts",
				Namespace: "default",
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: 2,
			},
			Status: autoscalingv1.ScaleStatus{
				Replicas: 2,
			},
		}, nil
	})

	// Initialize StatefulSet client wrapper
	stsClient := &statefulset.Client{
		Interface: fakeClient.AppsV1().StatefulSets("default"),
	}

	// Create a StatefulSet resource
	sts := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sts",
			Namespace: "default",
		},
		Spec: v1.StatefulSetSpec{
			Replicas: new(int32),
		},
	}
	_, err := fakeClient.AppsV1().StatefulSets("default").Create(ctx, sts, metav1.CreateOptions{})
	suite.NoError(err)

	// Cancel the context
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Call the Scale function
	statefulSetResult := stsClient.Scale(ctx, sts, 4)

	// Assert the error
	suite.NotNil(statefulSetResult.GetError(), "expected an error but got none")
	suite.Contains(statefulSetResult.GetError().Error(), "stopping waiting for sts to scale", "unexpected error message")
}

func (suite *StatefulSetTestSuite) TestGetPodList() {
	testCases := []struct {
		name        string
		sts         *v1.StatefulSet
		mockPodList *v12.PodList
		mockError   error
		expectError bool
		expectEmpty bool
	}{
		{
			name: "Successful scenario (matching pods)",
			sts: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "test-sts"},
				Spec: v1.StatefulSetSpec{
					Replicas: intPtr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: v12.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
					},
				},
			},
			mockPodList: &v12.PodList{
				Items: []v12.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-pod-1",
							Labels:    map[string]string{"app": "test"},
							Namespace: "default",
						},
						Status: v12.PodStatus{Phase: v12.PodRunning},
					},
				},
			},
			expectError: false,
			expectEmpty: false,
		},
		{
			name: "No matching pods",
			sts: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "test-sts"},
				Spec: v1.StatefulSetSpec{
					Replicas: intPtr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: v12.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
					},
				},
			},
			mockPodList: &v12.PodList{Items: []v12.Pod{}}, // Empty pod list
			expectError: false,
			expectEmpty: true,
		},
		{
			name: "Error listing pods",
			sts: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "test-sts"},
				Spec: v1.StatefulSetSpec{
					Replicas: intPtr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: v12.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
					},
				},
			},
			mockPodList: &v12.PodList{},
			mockError:   errors.New("failed to list pods"),
			expectError: true,
			expectEmpty: false, // Doesn't matter in error case
		},
		{
			name: "Invalid selector",
			sts: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "test-sts"},
				Spec: v1.StatefulSetSpec{
					Replicas: intPtr(1),
					// Invalid selector - missing matchLabels
					Selector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "invalid-key",
								Operator: "InvalidOperator", // This will cause an error
								Values:   []string{"value"},
							},
						},
					},
					Template: v12.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
					},
				},
			},
			expectError: true,
			expectEmpty: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			testSts := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sts",
				},
			}
			delStatus := suite.stsClient.Delete(context.Background(), testSts)
			if delStatus.GetError() != nil && !apierrs.IsNotFound(delStatus.GetError()) {
				suite.NoError(delStatus.GetError())
			}

			createdSts := suite.stsClient.Create(ctx, tc.sts)
			suite.NoError(createdSts.GetError())

			suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor(
				"list",
				"pods",
				func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
					return true, tc.mockPodList, tc.mockError
				},
			)

			podList, err := createdSts.GetPodList(ctx)

			if tc.expectError {
				suite.Error(err)
				if tc.mockError != nil { // Check for specific error message if provided
					suite.Contains(err.Error(), tc.mockError.Error())
				}
			} else {
				suite.NoError(err)
				if tc.expectEmpty {
					suite.Empty(podList.Items)
				} else {
					suite.NotEmpty(podList.Items)
					for _, pod := range podList.Items {
						suite.Equal(tc.sts.Spec.Selector.MatchLabels, pod.Labels)
					}
				}
			}
		})
	}
}

func (suite *StatefulSetTestSuite) TestWaitForRunningAndReady() {
	testCases := []struct {
		name             string
		initialReplicas  int32
		numPodsRunning   int32
		mockPodList      *v12.PodList
		mockListError    error
		mockGetError     error // Mock for pollWait case
		ctxTimeout       bool  // Simulate context timeout
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:            "Successful scenario",
			initialReplicas: 2,
			numPodsRunning:  2,
			mockPodList:     createMockPodList(2, true),
			expectedError:   false,
		},
		{
			name:             "Context timeout in WaitForRunningAndReady",
			initialReplicas:  2,
			numPodsRunning:   2,
			ctxTimeout:       true,
			expectedError:    true,
			expectedErrorMsg: "stopped waiting to be ready",
		},
		{
			name:             "GetPodList error",
			initialReplicas:  2,
			numPodsRunning:   2,
			mockPodList:      &v12.PodList{},
			mockListError:    errors.New("get pod list error"),
			expectedError:    true,
			expectedErrorMsg: "get pod list error",
		},
		{
			name:             "Fewer pods than expected",
			initialReplicas:  2,
			numPodsRunning:   2,
			mockPodList:      createMockPodList(1, true),
			expectedError:    true,
			expectedErrorMsg: "timed out",
		},
		{
			name:             "More pods than expected",
			initialReplicas:  2,
			numPodsRunning:   2,
			mockPodList:      createMockPodList(3, true),
			expectedError:    true,
			expectedErrorMsg: "timed out",
		},
		{
			name:             "Pods are running but not ready",
			initialReplicas:  2,
			numPodsRunning:   2,
			mockPodList:      createMockPodList(2, false),
			expectedError:    true,
			expectedErrorMsg: "timed out",
		},
		{
			name:            "numPodsRunning is 0",
			initialReplicas: 0,
			numPodsRunning:  0,
			mockPodList:     &v12.PodList{Items: []v12.Pod{}},
			expectedError:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if tc.ctxTimeout {
				cancel() // Simulate context timeout
			} else {
				defer cancel()
			}

			// Cleanup
			sts := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sts",
				},
			}
			delStatus := suite.stsClient.Delete(context.Background(), sts)
			if delStatus.GetError() != nil && !apierrs.IsNotFound(delStatus.GetError()) {
				suite.NoError(delStatus.GetError())
			}

			initialReplicas := tc.initialReplicas
			numPodsRunning := tc.numPodsRunning

			createdSts := suite.stsClient.Create(ctx, sts)
			suite.NoError(createdSts.GetError())

			if tc.mockPodList != nil || tc.mockListError != nil {
				suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("list", "pods", func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
					return true, tc.mockPodList, tc.mockListError
				})
			}

			if tc.mockGetError != nil {
				suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("get", "statefulsets", func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, tc.mockGetError
				})
			}

			err := createdSts.WaitForRunningAndReady(ctx, numPodsRunning, initialReplicas)

			if tc.expectedError {
				suite.Error(err)
				if tc.expectedErrorMsg != "" {
					suite.Contains(err.Error(), tc.expectedErrorMsg)
				}
			} else {
				suite.NoError(err)
				if tc.numPodsRunning > 0 {
					podList, err := createdSts.GetPodList(ctx)
					suite.NoError(err)
					suite.Len(podList.Items, int(tc.numPodsRunning))
					for _, p := range podList.Items {
						suite.True(pod.IsPodReady(&p))
						suite.Equal(v12.PodRunning, p.Status.Phase)
					}
				}
			}
		})
	}
}

func (suite *StatefulSetTestSuite) TestWaitUntilGone() {
	testCases := []struct {
		name              string
		mockGetReactor    func(action testing.Action) (handled bool, ret runtime.Object, err error)
		mockDeleteReactor func(action testing.Action) (handled bool, ret runtime.Object, err error)
		expectError       bool
		expectedErrorMsg  string
	}{
		{
			name: "StatefulSet successfully deleted",
			mockDeleteReactor: func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, nil
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Add reactors for get and delete
			if tc.mockGetReactor != nil {
				suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("get", "statefulsets", tc.mockGetReactor)
			}
			if tc.mockDeleteReactor != nil {
				suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("delete", "statefulsets", tc.mockDeleteReactor)
			}

			// Create a dummy StatefulSet to delete
			initialReplicas := int32(1)
			sts := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sts",
				},
				Spec: v1.StatefulSetSpec{
					Replicas: &initialReplicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
			}
			_ = suite.stsClient.Delete(context.Background(), sts)

			createdSts := suite.stsClient.Create(ctx, sts)
			suite.NoError(createdSts.GetError())

			// Call WaitUntilGone and verify the result
			err := createdSts.WaitUntilGone(ctx)
			if tc.expectError {
				suite.Error(err)
			}
		})
	}
}

func (suite *StatefulSetTestSuite) TestUpdate() {
	c := suite.stsClient
	sts := &v1.StatefulSet{}
	updatedSts := c.Update(sts)
	suite.NotNil(updatedSts)
}

func (suite *StatefulSetTestSuite) TestSync() {
	ctx := context.Background()
	initialReplicas := int32(1)

	sts := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sts1"},
		Spec: v1.StatefulSetSpec{
			Replicas: &initialReplicas,
		},
	}
	createdSts1 := suite.stsClient.Create(context.Background(), sts)
	mockPodList1 := createMockPodList(1, true) // Create a mock pod list
	suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("list", "pods", func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, mockPodList1, nil
	})
	suite.kubeClient.ClientSet.(*fake.Clientset).Fake.AddReactor("get", "pods", func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, &mockPodList1.Items[0], nil
	})

	syncedSts1 := createdSts1.Sync(ctx)
	suite.NoError(syncedSts1.GetError())
	suite.False(syncedSts1.HasError(), "expected no error, but HasError() returned true")

	sts2 := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sts2"},
		Spec: v1.StatefulSetSpec{
			Replicas: &initialReplicas,
		},
	}
	createdSts2 := suite.stsClient.Create(context.Background(), sts2)

	syncedSts2 := createdSts2.Sync(ctx)
	suite.NoError(syncedSts2.GetError()) // WaitUntilGone should not return an error in this fake client test

	sts3 := &v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "test-sts3"}, Spec: v1.StatefulSetSpec{
		Replicas: &initialReplicas,
	}}
	createdSts3 := suite.stsClient.Create(context.Background(), sts3)
	mockPodList3 := createMockPodList(0, true) // Create a mock pod list (0 pods)
	suite.kubeClient.ClientSet.(*fake.Clientset).Fake.PrependReactor("list", "pods", func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, mockPodList3, nil
	})

	syncedSts3 := createdSts3.Sync(ctx)
	suite.Error(syncedSts3.GetError())
}

func createMockPodList(numPods int, ready bool) *v12.PodList {
	items := make([]v12.Pod, 0, numPods)
	for i := 0; i < numPods; i++ {
		pod := &v12.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-sts-%d", i),
				Labels:    map[string]string{"app": "test"},
				Namespace: "default",
			},
			Status: v12.PodStatus{
				Phase: v12.PodRunning,
			},
		}
		if ready {
			pod.Status.Conditions = []v12.PodCondition{{Type: v12.PodReady, Status: v12.ConditionTrue}}
		}
		items = append(items, *pod)
	}
	return &v12.PodList{Items: items}
}

// Helper function to create int pointers
func intPtr(i int32) *int32 {
	return &i
}

func TestStatefulSetSuite(t *t.T) {
	suite.Run(t, new(StatefulSetTestSuite))
}
