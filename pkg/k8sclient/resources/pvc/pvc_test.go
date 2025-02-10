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
	"github.com/dell/cert-csi/pkg/k8sclient"
	pvc2 "github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
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

func TestPVCSuite(t *testing.T) {
	suite.Run(t, new(PVCTestSuite))
}
