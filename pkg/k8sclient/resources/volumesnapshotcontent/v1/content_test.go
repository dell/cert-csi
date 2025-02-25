/*
 * Copyright © 2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
 */
package v1_test

import (
	"context"
	"fmt"
	t1 "testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	contentv1 "github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshotcontent/v1"
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned/typed/volumesnapshot/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// SnapshotContentClientTestSuite is a test suite for SnapshotContentClient
type SnapshotContentClientTestSuite struct {
	suite.Suite
	client        *contentv1.SnapshotContentClient
	mockInterface *MockVolumeSnapshotContentInterface
	kubeClient    *k8sclient.KubeClient
}

type MockVolumeSnapshotContentInterface struct {
	snapshotv1.VolumeSnapshotContentInterface
	Fake             *fake.Clientset
	SnapshotContents []v1.VolumeSnapshotContent
	ListError        error
	DeleteError      error
	GetError         error
	UpdateError      error
	CreateError      error
}

func (m *MockVolumeSnapshotContentInterface) List(_ context.Context, _ metav1.ListOptions) (*v1.VolumeSnapshotContentList, error) {
	if m.ListError != nil {
		return nil, m.ListError
	}
	return &v1.VolumeSnapshotContentList{Items: m.SnapshotContents}, nil
}

func (m *MockVolumeSnapshotContentInterface) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	if m.DeleteError != nil {
		return m.DeleteError
	}
	for i, sc := range m.SnapshotContents {
		if sc.Name == name {
			m.SnapshotContents = append(m.SnapshotContents[:i], m.SnapshotContents[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("blockvolsnapshotcontent not found")
}

func (m *MockVolumeSnapshotContentInterface) Get(_ context.Context, name string, _ metav1.GetOptions) (*v1.VolumeSnapshotContent, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	for _, sc := range m.SnapshotContents {
		if sc.Name == name {
			return &sc, nil
		}
	}
	return nil, fmt.Errorf("blockvolsnapshotcontent not found")
}

func (m *MockVolumeSnapshotContentInterface) Update(_ context.Context, content *v1.VolumeSnapshotContent, _ metav1.UpdateOptions) (*v1.VolumeSnapshotContent, error) {
	if m.UpdateError != nil {
		return nil, m.UpdateError
	}
	for i, sc := range m.SnapshotContents {
		if sc.Name == content.Name {
			m.SnapshotContents[i] = *content
			return content, nil
		}
	}
	return nil, fmt.Errorf("blockvolsnapshotcontent not found")
}

func (suite *SnapshotContentClientTestSuite) SetupTest() {
	suite.mockInterface = &MockVolumeSnapshotContentInterface{}
	suite.client = &contentv1.SnapshotContentClient{
		Interface: suite.mockInterface,
		Namespace: "default",
		Timeout:   10, // Increase timeout for testing
	}

	fakeClient := fake.NewSimpleClientset()
	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   fakeClient,
		Config:      nil,
		VersionInfo: nil,
	}
}

func (suite *SnapshotContentClientTestSuite) TestDeleteAll_Success() {
	suite.mockInterface.SnapshotContents = []v1.VolumeSnapshotContent{
		{ObjectMeta: metav1.ObjectMeta{Name: "snapcontent1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "snapcontent2"}},
	}

	err := suite.client.DeleteAll(context.Background())
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), suite.mockInterface.SnapshotContents)
}

func (suite *SnapshotContentClientTestSuite) TestDeleteAll_ListError() {
	suite.mockInterface.ListError = fmt.Errorf("list error")

	err := suite.client.DeleteAll(context.Background())
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "list error", err.Error())
}

func (suite *SnapshotContentClientTestSuite) TestDeleteAll_DeleteError() {
	content := getSnapshotContent()
	suite.mockInterface.SnapshotContents = []v1.VolumeSnapshotContent{*content}
	suite.mockInterface.DeleteError = fmt.Errorf("delete error")

	err := suite.client.DeleteAll(context.Background())
	assert.NoError(suite.T(), err) // The function should not return an error even if one blockvolsnapshotcontent fails to delete
}

func (suite *SnapshotContentClientTestSuite) TestDelete_Success() {
	content := getSnapshotContent()
	suite.mockInterface.SnapshotContents = []v1.VolumeSnapshotContent{*content}

	result := suite.client.Delete(context.Background(), content)
	assert.NoError(suite.T(), result.GetError())
	assert.False(suite.T(), result.HasError())
	assert.Equal(suite.T(), content, result.Object)
	assert.True(suite.T(), result.Deleted)
}

func (suite *SnapshotContentClientTestSuite) TestDelete_Error() {
	content := getSnapshotContent()
	suite.mockInterface.SnapshotContents = []v1.VolumeSnapshotContent{*content}
	suite.mockInterface.DeleteError = fmt.Errorf("delete error")

	result := suite.client.Delete(context.Background(), content)
	assert.Error(suite.T(), result.GetError())
	assert.True(suite.T(), result.HasError())
	assert.Equal(suite.T(), content, result.Object)
	assert.True(suite.T(), result.Deleted)
}

func (suite *SnapshotContentClientTestSuite) TestWaitUntilGone_GetError() {
	content := getSnapshotContent()
	snapContent := &contentv1.SnapshotContent{
		Client: suite.client,
		Object: content,
	}
	suite.mockInterface.GetError = fmt.Errorf("get error")

	err := snapContent.WaitUntilGone(context.Background())
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "get error", err.Error())
}

func (suite *SnapshotContentClientTestSuite) TestWaitUntilGone_UpdateError() {
	content := getSnapshotContent()
	snapContent := &contentv1.SnapshotContent{
		Client: suite.client,
		Object: content,
	}
	suite.mockInterface.SnapshotContents = []v1.VolumeSnapshotContent{*content}
	suite.mockInterface.UpdateError = fmt.Errorf("update error")

	err := snapContent.WaitUntilGone(context.Background())
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "update error", err.Error())
}

func getSnapshotContent() *v1.VolumeSnapshotContent {
	return &v1.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{Name: "snapcontent1"},
	}
}

func getSnapshotContentWithStatus() *v1.VolumeSnapshotContent {
	return &v1.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{Name: "snapcontent1"},
		Status:     &v1.VolumeSnapshotContentStatus{ReadyToUse: boolPtr(true)},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func (suite *SnapshotContentClientTestSuite) TestWaitForRunning_Success() {
	content := getSnapshotContentWithStatus()
	snapContent := &contentv1.SnapshotContent{
		Client: suite.client,
		Object: content,
	}
	suite.mockInterface.SnapshotContents = []v1.VolumeSnapshotContent{*content}

	err := snapContent.WaitForRunning(context.Background())
	assert.NoError(suite.T(), err)
}

func (suite *SnapshotContentClientTestSuite) TestWaitForRunning_GetError() {
	content := getSnapshotContent()
	snapContent := &contentv1.SnapshotContent{
		Client: suite.client,
		Object: content,
	}
	suite.mockInterface.GetError = fmt.Errorf("get error")

	err := snapContent.WaitForRunning(context.Background())
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "get error", err.Error())
}

func (suite *SnapshotContentClientTestSuite) TestWaitForRunning_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	content := getSnapshotContent()
	snapContent := &contentv1.SnapshotContent{
		Client: suite.client,
		Object: content,
	}

	cancel()

	err := snapContent.WaitForRunning(ctx)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "stopped waiting to be ready", err.Error())
}

func TestSnapshotContentClientTestSuite(t *t1.T) {
	suite.Run(t, new(SnapshotContentClientTestSuite))
}
