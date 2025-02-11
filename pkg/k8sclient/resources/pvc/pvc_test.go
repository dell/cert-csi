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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"testing"
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

func (suite *PVCTestSuite) TestCreateMultiple() {
	// Create a PersistentVolumeClaim object
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
	}

	// Create a context
	ctx := context.TODO()

	// Test case: invalid pvcNum
	err := suite.client.CreateMultiple(ctx, pvc, 0, "10Gi")
	suite.Error(err)
	suite.EqualError(err, "number of pvcs can't be less or equal than zero")

	// Test case: invalid pvcSize
	err = suite.client.CreateMultiple(ctx, pvc, 1, "")
	suite.Error(err)
	suite.EqualError(err, "volume size cannot be nulls")

	// Test case: valid inputs
	err = suite.client.CreateMultiple(ctx, pvc, 3, "10Gi")
	suite.NoError(err)

	// Verify that 3 PVCs were created
	for i := 0; i < 3; i++ {
		createdPVC, err := suite.client.Interface.Get(ctx, pvc.Name, metav1.GetOptions{})
		suite.NoError(err)
		suite.NotNil(createdPVC)
	}

	// Check the log output (optional)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debugf("Created %d PVCs of size:%s", 3, "10Gi")
}
func TestPVCSuite(t *testing.T) {
	suite.Run(t, new(PVCTestSuite))
}
