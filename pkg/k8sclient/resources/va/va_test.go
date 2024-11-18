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

package va_test

import (
	"context"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/stretchr/testify/suite"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type VaTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
}

func (suite *VaTestSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()

	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)
}

func (suite *VaTestSuite) TestVaClient_WaitUntilNoneLeft() {
	va, err := suite.kubeClient.ClientSet.StorageV1().VolumeAttachments().Create(context.Background(), &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
	}, metav1.CreateOptions{})
	suite.NoError(err)

	vaClient, err := suite.kubeClient.CreateVaClient("test-namespace")
	vaClient.CustomTimeout = time.Second
	suite.NoError(err)

	suite.Run("timeout error", func() {
		err := vaClient.WaitUntilNoneLeft(context.Background())
		suite.Error(err)
	})

	vaClient, err = suite.kubeClient.CreateVaClient("test-namespace")
	suite.NoError(err)
	err = suite.kubeClient.ClientSet.StorageV1().VolumeAttachments().Delete(context.Background(), va.Name, metav1.DeleteOptions{})
	suite.NoError(err)

	suite.Run("all deleted", func() {
		err := vaClient.WaitUntilNoneLeft(context.Background())
		suite.NoError(err)
	})
}

func (suite *VaTestSuite) TestWaitUntilVaGone() {
	pvName := "test-pv"
	va, err := suite.kubeClient.ClientSet.StorageV1().VolumeAttachments().Create(context.Background(), &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: &pvName,
			},
		},
	}, metav1.CreateOptions{})
	suite.NoError(err)

	vaClient, err := suite.kubeClient.CreateVaClient("test-namespace")
	suite.NoError(err)

	err = suite.kubeClient.ClientSet.StorageV1().VolumeAttachments().Delete(context.Background(), va.Name, metav1.DeleteOptions{})
	suite.NoError(err)

	suite.Run("wait until VA is gone", func() {
		err = vaClient.WaitUntilVaGone(context.Background(), *va.Spec.Source.PersistentVolumeName)
		suite.NoError(err)
	})

	suite.Run("no VA found", func() {
		nonExistentPvName := "non-existent-pv"
		err = vaClient.WaitUntilVaGone(context.Background(), nonExistentPvName)
		suite.NoError(err)
	})
}

func TestVaTestSuite(t *testing.T) {
	suite.Run(t, new(VaTestSuite))
}
