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

package pvc_test

import (
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/k8sclient"
	pvc2 "github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/sc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	v2 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"os"
	t "testing"
	"time"
)

type PVCTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
}

func (suite *PVCTestSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()
	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)
}

type PersistentVolumeClaimSuite struct {
	suite.Suite
	pvcClient  *pvc2.PersistentVolumeClaim
	kubeClient *pvc2.Client
}

func (suite *PersistentVolumeClaimSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()
	suite.kubeClient = &pvc2.Client{
		ClientSet: client,
		Interface: client.CoreV1().PersistentVolumeClaims("default"),
	}
	suite.pvcClient = &pvc2.PersistentVolumeClaim{
		Client:  suite.kubeClient,
		Object:  &v1.PersistentVolumeClaim{},
		Deleted: false,
	}
}
func (suite *PersistentVolumeClaimSuite) TestWaitToBeBound() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	suite.pvcClient.Object = &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimPending, // Initially not bound
		},
	}

	// Run in a separate goroutine to simulate PVC being bound later
	go func() {
		time.Sleep(1 * time.Second) // Simulate some delay
		suite.pvcClient.Object.Status.Phase = v1.ClaimBound
	}()

	err := suite.pvcClient.WaitToBeBound(ctx)
	assert.NoError(suite.T(), err, "Expected no error when PVC gets bound")
}
func (suite *PersistentVolumeClaimSuite) TestWaitToBeBoundTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	suite.pvcClient.Object = &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimPending, // Never changes to bound
		},
	}

	err := suite.pvcClient.WaitToBeBound(ctx)
	assert.Error(suite.T(), err, "Expected an error due to timeout")
}
func (suite *PersistentVolumeClaimSuite) TestWaitUntilGone() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	suite.pvcClient.Object = &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
	}

	// Simulate PVC being deleted after some delay
	go func() {
		time.Sleep(1 * time.Second)
		suite.pvcClient.Object = nil // Simulate deletion
	}()

	err := suite.pvcClient.WaitUntilGone(ctx)
	assert.NoError(suite.T(), err, "Expected no error when PVC is deleted")
}

func (suite *PVCTestSuite) TestMakePVC() {
	blockMode := v1.PersistentVolumeBlock
	cfg := &pvc2.Config{
		Name:       "test-pvc",
		VolumeMode: &blockMode,
	}
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	pvc := pvcClient.MakePVC(cfg)
	suite.NotNil(pvc, "expected non-nil PVC")
	suite.Equal("test-pvc", pvc.Name, "expected PVC name 'test-pvc'")
}

func (suite *PVCTestSuite) TestCreatePVC() {
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	pvc := &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "test-pvc"}}
	createdPVC := pvcClient.Create(context.Background(), pvc)
	suite.NoError(createdPVC.GetError(), "expected no error for valid PVC creation")
	suite.Equal("test-pvc", createdPVC.Object.GetName(), "expected 'test-pvc' name")
	deletedPvc := pvcClient.Delete(context.Background(), pvc)
	suite.NoError(deletedPvc.GetError(), "expected no error for valid PVC deletion")
	suite.Equal("test-pvc", deletedPvc.Object.GetName(), "expected 'test-pvc' name")
}

func (suite *PVCTestSuite) TestGetPVC() {
	ctx := context.Background()

	// Ensure any existing PVC with the same name is deleted
	_ = suite.kubeClient.ClientSet.CoreV1().PersistentVolumeClaims("default").Delete(ctx, "test-pvc", metav1.DeleteOptions{})

	// Create a PVC with DryRun option
	pvc, err := suite.kubeClient.ClientSet.CoreV1().PersistentVolumeClaims("default").Create(ctx, &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
	}, metav1.CreateOptions{
		DryRun: []string{"All"},
	})
	suite.NoError(err, "expected no error during PVC creation")

	// Create PVC client
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err, "expected no error during PVC client creation")
	pvcClient.Timeout = 1

	// Test the Get method
	suite.Run("pvc get", func() {
		result := pvcClient.Get(ctx, pvc.Name)
		suite.NoError(result.GetError(), "expected no error for existing PVC")
		suite.Equal(pvc.Name, result.Object.GetName(), "expected PVC name to match")
	})
}

func (suite *PVCTestSuite) TestCreateMultiplePVCs() {
	ctx := context.Background()
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)

	suite.Run("create multiple PVCs with zero number", func() {
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pvc",
				Namespace: "default",
			},
		}
		err := pvcClient.CreateMultiple(ctx, pvc, 0, "5Gi")
		suite.Error(err, "expected error for zero number of PVCs")
		suite.EqualError(err, "number of pvcs can't be less or equal than zero")
	})

	suite.Run("create multiple PVCs with empty size", func() {
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pvc",
				Namespace: "default",
			},
		}
		err := pvcClient.CreateMultiple(ctx, pvc, 3, "")
		suite.Error(err, "expected error for empty PVC size")
		suite.EqualError(err, "volume size cannot be nulls")
	})
}

func (suite *PVCTestSuite) TestUpdate() {
	// Generate a unique PVC name to avoid conflicts
	pvcName := fmt.Sprintf("test-pvc-%d", time.Now().UnixNano())

	// Create PVC client
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.Require().NoError(err, "Expected no error while creating PVC client")
	suite.Require().NotNil(pvcClient, "PVC client should not be nil")

	// Create PVC
	pvc := &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: pvcName}}
	createdPVC := pvcClient.Create(context.Background(), pvc)
	suite.Require().NotNil(createdPVC, "Created PVC object should not be nil")
	suite.NoError(createdPVC.GetError(), "Expected no error for valid PVC creation")
	suite.Equal(pvcName, createdPVC.Object.GetName(), "Expected PVC name to match created name")

	// Update PVC
	updatedPVC := pvcClient.Update(context.Background(), pvc)
	suite.Require().NotNil(updatedPVC, "Updated PVC object should not be nil")
	suite.NoError(updatedPVC.GetError(), "Expected no error for valid PVC update")
	suite.Equal(pvcName, updatedPVC.Object.GetName(), "Expected PVC name to remain the same after update")

	// Delete PVC
	deletedPVC := pvcClient.Delete(context.Background(), pvc)
	suite.Require().NotNil(deletedPVC, "Deleted PVC object should not be nil")
	suite.NoError(deletedPVC.GetError(), "Expected no error for valid PVC deletion")
	suite.Equal(pvcName, deletedPVC.Object.GetName(), "Expected PVC name to match before deletion")
}

func (suite *PVCTestSuite) TestMakePVCFromYaml() {
	// Create a sample PVC YAML file
	pvcYaml := `
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
`
	// Write the sample PVC YAML to a temporary file
	tmpFile, err := os.CreateTemp("", "pvc-*.yaml")
	suite.NoError(err)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {

		}
	}(tmpFile.Name())

	_, err = tmpFile.Write([]byte(pvcYaml))
	suite.NoError(err)
	err = tmpFile.Close()
	if err != nil {
		return
	}

	// Create a context
	ctx := context.TODO()
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	// Call the MakePVCFromYaml function
	pvc, err := pvcClient.MakePVCFromYaml(ctx, tmpFile.Name())

	// Assertions
	suite.NoError(err)
	suite.NotNil(pvc, "expected non-nil PVC")
	suite.Equal("test-pvc", pvc.Name, "expected PVC name 'test-pvc'")
	suite.Equal("default", pvc.Namespace, "expected PVC namespace 'default'")
}

func (suite *PVCTestSuite) TestDeleteAll() {
	ctx := context.Background()

	// Create a list of PVCs to be returned by the fake client
	pvcList := &v1.PersistentVolumeClaimList{
		Items: []v1.PersistentVolumeClaim{
			{ObjectMeta: metav1.ObjectMeta{Name: "pvc-1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "pvc-2"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "pvc-3"}},
		},
	}

	// Mock the List function to return the PVC list
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "persistentvolumeclaims", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, pvcList, nil
	})

	// Mock the Delete function to simulate successful deletion
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("delete", "persistentvolumeclaims", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, nil
	})
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	// Call the DeleteAll function
	err = pvcClient.DeleteAll(ctx)
	suite.NoError(err)

	// Verify that the Delete function was called for each PVC
	for _, pvc := range pvcList.Items {
		suite.kubeClient.ClientSet.(*fake.Clientset).Actions()
		suite.Contains(suite.kubeClient.ClientSet.(*fake.Clientset).Actions(), testing.NewDeleteAction(v1.SchemeGroupVersion.WithResource("persistentvolumeclaims"), "default", pvc.Name))
	}
}

func (suite *PVCTestSuite) TestWaitForAllToBeBound() {
	ctx := context.Background()

	// Create a list of PVCs to be returned by the fake client
	pvcList := &v1.PersistentVolumeClaimList{
		Items: []v1.PersistentVolumeClaim{
			{ObjectMeta: metav1.ObjectMeta{Name: "pvc-1"}, Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimPending}},
			{ObjectMeta: metav1.ObjectMeta{Name: "pvc-2"}, Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimPending}},
			{ObjectMeta: metav1.ObjectMeta{Name: "pvc-3"}, Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimPending}},
		},
	}

	// Mock the List function to return the PVC list
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "persistentvolumeclaims", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, pvcList, nil
	})

	// Mock the List function to simulate PVCs becoming bound
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "persistentvolumeclaims", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		for i := range pvcList.Items {
			pvcList.Items[i].Status.Phase = v1.ClaimBound
		}
		return true, pvcList, nil
	})
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	// Call the WaitForAllToBeBound function
	err = pvcClient.WaitForAllToBeBound(ctx)
	suite.NoError(err)
}

func (suite *PVCTestSuite) TestWaitForAllToBeBound_Error() {
	ctx := context.Background()

	// Mock the List function to return an error
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "persistentvolumeclaims", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("failed to list PVCs")
	})
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	// Call the WaitForAllToBeBound function
	err = pvcClient.WaitForAllToBeBound(ctx)
	suite.Error(err)
	suite.EqualError(err, "failed to list PVCs")
}

func (suite *PVCTestSuite) TestCheckAnnotationsForVolumes_Error() {
	ctx := context.Background()

	// Create a mock StorageClass object
	scObject := &v2.StorageClass{
		Parameters: map[string]string{
			sc.RemoteClusterID:        "remote-cluster-id",
			sc.RemoteStorageClassName: "remote-storage-class-name",
		},
	}

	// Mock the List function to return an error
	suite.kubeClient.ClientSet.(*fake.Clientset).PrependReactor("list", "persistentvolumeclaims", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("failed to list PVCs")
	})
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	// Call the CheckAnnotationsForVolumes function
	err = pvcClient.CheckAnnotationsForVolumes(ctx, scObject)
	suite.Error(err)
	suite.EqualError(err, "failed to list PVCs")
}

func (suite *PVCTestSuite) TestCheckAnnotationsForVolumes_ContextCancelled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context immediately

	// Create a mock StorageClass object
	scObject := &v2.StorageClass{
		Parameters: map[string]string{
			sc.RemoteClusterID:        "remote-cluster-id",
			sc.RemoteStorageClassName: "remote-storage-class-name",
		},
	}
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	// Call the CheckAnnotationsForVolumes function
	err = pvcClient.CheckAnnotationsForVolumes(ctx, scObject)
	suite.Error(err)
	suite.EqualError(err, "stopped waiting for annotations and labels")
}

func (suite *PVCTestSuite) TestCreatePVCObject() {
	ctx := context.Background()

	// Create a mock PersistentVolume object
	remotePVObject := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-1",
			Annotations: map[string]string{
				"replication.storage.dell.com/remotePVC":              "pvc-1",
				"replication.storage.dell.com/resourceRequest":        `{"storage":"10Gi"}`,
				"replication.storage.dell.com/remoteClusterID":        "remote-cluster-id",
				"replication.storage.dell.com/remoteStorageClassName": "remote-storage-class-name",
			},
			Labels: map[string]string{
				"replication.storage.dell.com/some-label": "some-value",
			},
		},
		Spec: v1.PersistentVolumeSpec{
			StorageClassName: "standard",
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			VolumeMode:       func() *v1.PersistentVolumeMode { mode := v1.PersistentVolumeFilesystem; return &mode }(),
		},
	}

	remoteNamespace := "remote-namespace"
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	// Call the CreatePVCObject function
	pvcObject := pvcClient.CreatePVCObject(ctx, remotePVObject, remoteNamespace)

	// Verify the PVC object
	suite.Equal("pvc-1", pvcObject.Name)
	suite.Equal(remoteNamespace, pvcObject.Namespace)
	suite.Equal("remote-cluster-id", pvcObject.Annotations["replication.storage.dell.com/remoteClusterID"])
	suite.Equal("remote-storage-class-name", pvcObject.Annotations["replication.storage.dell.com/remoteStorageClassName"])
	suite.Equal("some-value", pvcObject.Labels["replication.storage.dell.com/some-label"])
	suite.Equal("standard", *pvcObject.Spec.StorageClassName)
	suite.Equal([]v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}, pvcObject.Spec.AccessModes)
	suite.Equal(v1.PersistentVolumeFilesystem, *pvcObject.Spec.VolumeMode)
	//suite.Equal("10Gi", pvcObject.Spec.Resources.Requests["storage"])
	suite.Equal("pv-1", pvcObject.Spec.VolumeName)
}
func (suite *PVCTestSuite) TestCreatePVCObject_Error() {
	ctx := context.Background()

	// Create a mock PersistentVolume object with invalid resourceRequest
	remotePVObject := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-1",
			Annotations: map[string]string{
				"replication.storage.dell.com/remotePVC":              "pvc-1",
				"replication.storage.dell.com/resourceRequest":        `invalid-json`,
				"replication.storage.dell.com/remoteClusterID":        "remote-cluster-id",
				"replication.storage.dell.com/remoteStorageClassName": "remote-storage-class-name",
			},
			Labels: map[string]string{
				"replication.storage.dell.com/some-label": "some-value",
			},
		},
		Spec: v1.PersistentVolumeSpec{
			StorageClassName: "standard",
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			VolumeMode:       func() *v1.PersistentVolumeMode { mode := v1.PersistentVolumeFilesystem; return &mode }(),
		},
	}

	remoteNamespace := "remote-namespace"
	pvcClient, err := suite.kubeClient.CreatePVCClient("default")
	suite.NoError(err)
	// Call the CreatePVCObject function
	pvcObject := pvcClient.CreatePVCObject(ctx, remotePVObject, remoteNamespace)

	// Verify the PVC object is empty due to the error
	suite.Empty(pvcObject.Name)
	suite.Empty(pvcObject.Namespace)
	suite.Empty(pvcObject.Annotations)
	suite.Empty(pvcObject.Labels)
	suite.Empty(pvcObject.Spec.StorageClassName)
	suite.Empty(pvcObject.Spec.AccessModes)
	suite.Empty(pvcObject.Spec.VolumeMode)
	suite.Empty(pvcObject.Spec.Resources.Requests)
	suite.Empty(pvcObject.Spec.VolumeName)
}

func TestPVCSuite(t *t.T) {
	suite.Run(t, new(PVCTestSuite))
	suite.Run(t, new(PersistentVolumeClaimSuite))
}
