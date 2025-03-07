/*
 *
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
 *
 */
package volumesnapshot_test

import (
	"context"
	"fmt"
	t1 "testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	snapv1 "github.com/dell/cert-csi/pkg/k8sclient/resources/volumesnapshot"
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned/typed/volumesnapshot/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// SnapshotClientTestSuite is a test suite for SnapshotClient
type SnapshotClientTestSuite struct {
	suite.Suite
	client        *snapv1.SnapshotClient
	mockInterface *MockVolumeSnapshotInterface
	kubeClient    *k8sclient.KubeClient
}

type MockVolumeSnapshotInterface struct {
	snapshotv1.VolumeSnapshotInterface
	Fake        *fake.Clientset
	Snapshots   []v1.VolumeSnapshot
	ListError   error
	DeleteError error
	GetError    error
	UpdateError error
	CreateError error
}

func (m *MockVolumeSnapshotInterface) List(_ context.Context, _ metav1.ListOptions) (*v1.VolumeSnapshotList, error) {
	if m.ListError != nil {
		return nil, m.ListError
	}
	return &v1.VolumeSnapshotList{Items: m.Snapshots}, nil
}

func (m *MockVolumeSnapshotInterface) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	if m.DeleteError != nil {
		return m.DeleteError
	}
	for i, snap := range m.Snapshots {
		if snap.Name == name {
			m.Snapshots = append(m.Snapshots[:i], m.Snapshots[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("blockvolsnapshot not found")
}

func (m *MockVolumeSnapshotInterface) Get(_ context.Context, name string, _ metav1.GetOptions) (*v1.VolumeSnapshot, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	for _, snap := range m.Snapshots {
		if snap.Name == name {
			return &snap, nil
		}
	}
	return nil, fmt.Errorf("blockvolsnapshot not found")
}

func (m *MockVolumeSnapshotInterface) Update(_ context.Context, snap *v1.VolumeSnapshot, _ metav1.UpdateOptions) (*v1.VolumeSnapshot, error) {
	if m.UpdateError != nil {
		return nil, m.UpdateError
	}
	for i, s := range m.Snapshots {
		if s.Name == snap.Name {
			m.Snapshots[i] = *snap
			return snap, nil
		}
	}
	return nil, fmt.Errorf("blockvolsnapshot not found")
}

func (m *MockVolumeSnapshotInterface) Create(_ context.Context, snap *v1.VolumeSnapshot, _ metav1.CreateOptions) (*v1.VolumeSnapshot, error) {
	if m.CreateError != nil {
		return nil, m.CreateError
	}
	m.Snapshots = append(m.Snapshots, *snap)
	return snap, nil
}

func (suite *SnapshotClientTestSuite) SetupTest() {
	suite.mockInterface = &MockVolumeSnapshotInterface{}
	suite.client = &snapv1.SnapshotClient{
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

func (suite *SnapshotClientTestSuite) TestDeleteAll_Success() {
	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{
		{ObjectMeta: metav1.ObjectMeta{Name: "snap1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "snap2"}},
	}

	err := suite.client.DeleteAll(context.Background())
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), suite.mockInterface.Snapshots)
}

func (suite *SnapshotClientTestSuite) TestDeleteAll_ListError() {
	suite.mockInterface.ListError = fmt.Errorf("list error")

	err := suite.client.DeleteAll(context.Background())
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "list error", err.Error())
}

func (suite *SnapshotClientTestSuite) TestDeleteAll_DeleteError() {
	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{
		{ObjectMeta: metav1.ObjectMeta{Name: "snap1"}},
	}
	suite.mockInterface.DeleteError = fmt.Errorf("delete error")

	err := suite.client.DeleteAll(context.Background())
	assert.NoError(suite.T(), err) // The function should not return an error even if one blockvolsnapshot fails to delete
}

func (suite *SnapshotClientTestSuite) TestWaitForAllToBeReady_Success() {
	ctx := context.TODO()
	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
			Status:     &v1.VolumeSnapshotStatus{ReadyToUse: boolPtr(true)},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "snap2"},
			Status:     &v1.VolumeSnapshotStatus{ReadyToUse: boolPtr(true)},
		},
	}

	err := suite.client.WaitForAllToBeReady(ctx)
	assert.NoError(suite.T(), err)
}

func (suite *SnapshotClientTestSuite) TestWaitForAllToBeReady_Timeout() {
	ctx := context.TODO()
	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
			Status:     &v1.VolumeSnapshotStatus{ReadyToUse: boolPtr(false)},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "snap2"},
			Status:     &v1.VolumeSnapshotStatus{ReadyToUse: boolPtr(false)},
		},
	}

	err := suite.client.WaitForAllToBeReady(ctx)
	assert.Error(suite.T(), err)
}

func (suite *SnapshotClientTestSuite) TestWaitForAllToBeReady_ListError() {
	ctx := context.TODO()
	suite.mockInterface.ListError = fmt.Errorf("list error")

	err := suite.client.WaitForAllToBeReady(ctx)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "list error", err.Error())
}

func (suite *SnapshotClientTestSuite) TestWaitForAllToBeReady_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := suite.client.WaitForAllToBeReady(ctx)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "stopped waiting to be ready", err.Error())
}

func (suite *SnapshotClientTestSuite) TestWaitUntilGone_GetError() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
	}
	snap := &snapv1.Snapshot{
		Client: suite.client,
		Object: snapshot,
	}

	suite.mockInterface.GetError = fmt.Errorf("get error")

	err := snap.WaitUntilGone(ctx)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "get error", err.Error())
}

func (suite *SnapshotClientTestSuite) TestWaitUntilGone_UpdateError() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
	}
	snap := &snapv1.Snapshot{
		Client: suite.client,
		Object: snapshot,
	}

	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{*snapshot}
	suite.mockInterface.UpdateError = fmt.Errorf("update error")

	err := snap.WaitUntilGone(ctx)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "update error", err.Error())
}

func boolPtr(b bool) *bool {
	return &b
}

func (suite *SnapshotClientTestSuite) TestCreate_Success() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snap1",
			Namespace: "default",
		},
	}

	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{*snapshot}

	result := suite.client.Create(ctx, snapshot)
	assert.NoError(suite.T(), result.GetError())
	assert.False(suite.T(), result.HasError())
	assert.Equal(suite.T(), snapshot, result.Object)
	assert.False(suite.T(), result.Deleted)
}

func (suite *SnapshotClientTestSuite) TestCreate_Error() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snap1",
			Namespace: "default",
		},
	}

	suite.mockInterface.CreateError = fmt.Errorf("create error")

	result := suite.client.Create(ctx, snapshot)
	assert.Error(suite.T(), result.GetError())
	assert.True(suite.T(), result.HasError())
	assert.Nil(suite.T(), result.Object)
	assert.False(suite.T(), result.Deleted)
}

func (suite *SnapshotClientTestSuite) TestDelete_Success() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snap1",
			Namespace: "default",
		},
	}

	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{*snapshot}

	result := suite.client.Delete(ctx, snapshot)
	assert.NoError(suite.T(), result.GetError())
	assert.False(suite.T(), result.HasError())
	assert.Equal(suite.T(), snapshot, result.Object)
	assert.True(suite.T(), result.Deleted)
}

func (suite *SnapshotClientTestSuite) TestDelete_Error() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snap1",
			Namespace: "default",
		},
	}

	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{*snapshot}
	suite.mockInterface.DeleteError = fmt.Errorf("delete error")

	result := suite.client.Delete(ctx, snapshot)
	assert.Error(suite.T(), result.GetError())
	assert.True(suite.T(), result.HasError())
	assert.Equal(suite.T(), snapshot, result.Object)
	assert.True(suite.T(), result.Deleted)
}

func (suite *SnapshotClientTestSuite) TestWaitForRunning_Success() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
		Status:     &v1.VolumeSnapshotStatus{ReadyToUse: boolPtr(true)},
	}
	snap := &snapv1.Snapshot{
		Client: suite.client,
		Object: snapshot,
	}

	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{*snapshot}

	err := snap.WaitForRunning(ctx)
	assert.NoError(suite.T(), err)
}

func (suite *SnapshotClientTestSuite) TestWaitForRunning_Timeout() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
		Status:     &v1.VolumeSnapshotStatus{ReadyToUse: boolPtr(false)},
	}
	snap := &snapv1.Snapshot{
		Client: suite.client,
		Object: snapshot,
	}

	suite.mockInterface.Snapshots = []v1.VolumeSnapshot{*snapshot}

	err := snap.WaitForRunning(ctx)
	assert.Error(suite.T(), err)
}

func (suite *SnapshotClientTestSuite) TestWaitForRunning_GetError() {
	ctx := context.TODO()
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
	}
	snap := &snapv1.Snapshot{
		Client: suite.client,
		Object: snapshot,
	}

	suite.mockInterface.GetError = fmt.Errorf("get error")

	err := snap.WaitForRunning(ctx)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "get error", err.Error())
}

func (suite *SnapshotClientTestSuite) TestWaitForRunning_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	snapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
	}
	snap := &snapv1.Snapshot{
		Client: suite.client,
		Object: snapshot,
	}

	cancel()

	err := snap.WaitForRunning(ctx)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "stopped waiting to be ready", err.Error())
}

func TestSnapshotClientTestSuite(t *t1.T) {
	suite.Run(t, new(SnapshotClientTestSuite))
}
