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
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient/resources/va"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clientgotesting "k8s.io/client-go/testing" // Aliased import
)


func TestWaitUntilNoneLeft(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(client *fake.Clientset)
		customTimeout time.Duration
		expectedError string
	}{
		{
			name: "Success",
			setup: func(client *fake.Clientset) {
				// Ensure no VolumeAttachments exist
			},
			customTimeout: 0,
			expectedError: "",
		},
		{
			name: "Error",
			setup: func(client *fake.Clientset) {
				// Simulate an error
				client.PrependReactor("list", "volumeattachments", func(action clientgotesting.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("simulated error")
				})
			},
			customTimeout: 0,
			expectedError: "simulated error",
		},
		{
			name: "NonEmptyList",
			setup: func(client *fake.Clientset) {
				// Create a VolumeAttachment to simulate non-empty list
				va := &storagev1.VolumeAttachment{
					ObjectMeta: v1.ObjectMeta{
						Name: "test-va",
					},
				}
				client.StorageV1().VolumeAttachments().Create(context.TODO(), va, v1.CreateOptions{})
			},
			customTimeout: 0,
			expectedError: "timed out waiting for the condition",
		},
		{
			name: "CustomTimeout",
			setup: func(client *fake.Clientset) {
				// Ensure no VolumeAttachments exist
			},
			customTimeout: 5 * time.Second,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			vaClient := &va.Client{
				Interface:     client.StorageV1().VolumeAttachments(),
				Namespace:     "default",
				Timeout:       10,
				CustomTimeout: tt.customTimeout,
			}

			ctx := context.TODO()
			tt.setup(client)

			err := vaClient.WaitUntilNoneLeft(ctx)
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestWaitUntilVaGone(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(client *fake.Clientset)
		pvName        string
		customTimeout time.Duration
		expectedError string
	}{
		{
			name: "Success",
			setup: func(client *fake.Clientset) {
				// Create a VolumeAttachment to be deleted
				va := &storagev1.VolumeAttachment{
					ObjectMeta: v1.ObjectMeta{
						Name: "test-va",
					},
					Spec: storagev1.VolumeAttachmentSpec{
						Source: storagev1.VolumeAttachmentSource{
							PersistentVolumeName: func() *string { s := "test-pv"; return &s }(),
						},
					},
				}
				client.StorageV1().VolumeAttachments().Create(context.TODO(), va, v1.CreateOptions{})

				// Delete the VolumeAttachment
				client.StorageV1().VolumeAttachments().Delete(context.TODO(), "test-va", v1.DeleteOptions{})
			},
			pvName:        "test-pv",
			customTimeout: 0,
			expectedError: "",
		},
		{
			name: "Error",
			setup: func(client *fake.Clientset) {
				// Simulate an error
				client.PrependReactor("list", "volumeattachments", func(action clientgotesting.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("simulated error")
				})
			},
			pvName:        "test-pv",
			customTimeout: 0,
			expectedError: "simulated error",
		},
		{
			name: "MatchingPVName",
			setup: func(client *fake.Clientset) {
				// Create a VolumeAttachment with a matching PersistentVolumeName
				va := &storagev1.VolumeAttachment{
					ObjectMeta: v1.ObjectMeta{
						Name: "test-va",
					},
					Spec: storagev1.VolumeAttachmentSpec{
						Source: storagev1.VolumeAttachmentSource{
							PersistentVolumeName: func() *string { s := "test-pv"; return &s }(),
						},
					},
				}
				client.StorageV1().VolumeAttachments().Create(context.TODO(), va, v1.CreateOptions{})

				// Simulate a delay before deleting the VolumeAttachment
				go func() {
					time.Sleep(2 * time.Second)
					client.StorageV1().VolumeAttachments().Delete(context.TODO(), "test-va", v1.DeleteOptions{})
				}()
			},
			pvName:        "test-pv",
			customTimeout: 0,
			expectedError: "",
		},
		{
			name: "CustomTimeout",
			setup: func(client *fake.Clientset) {
				// Ensure no VolumeAttachments exist
			},
			pvName:        "non-existent-pv",
			customTimeout: 5 * time.Second,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up a buffer to capture log output
			var logBuffer bytes.Buffer
			logrus.SetOutput(&logBuffer)
			logrus.SetLevel(logrus.DebugLevel)

			client := fake.NewSimpleClientset()
			vaClient := &va.Client{
				Interface:     client.StorageV1().VolumeAttachments(),
				Namespace:     "default",
				Timeout:       10,
				CustomTimeout: tt.customTimeout,
			}

			ctx := context.TODO()
			tt.setup(client)

			err := vaClient.WaitUntilVaGone(ctx, tt.pvName)
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}

			// Verify the log message for MatchingPVName test case
			if tt.name == "MatchingPVName" {
				logOutput := logBuffer.String()
				assert.Contains(t, logOutput, "Waiting for the volume-attachment to be deleted for :test-pv")
			}
		})
	}
}

func TestDeleteVaBasedOnPVName(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(client *fake.Clientset)
		pvName        string
		expectedError string
	}{
		{
			name: "Success",
			setup: func(client *fake.Clientset) {
				// Create a VolumeAttachment to be deleted
				va := &storagev1.VolumeAttachment{
					ObjectMeta: v1.ObjectMeta{
						Name: "test-va",
					},
					Spec: storagev1.VolumeAttachmentSpec{
						Source: storagev1.VolumeAttachmentSource{
							PersistentVolumeName: func() *string { s := "test-pv"; return &s }(),
						},
					},
				}
				client.StorageV1().VolumeAttachments().Create(context.TODO(), va, v1.CreateOptions{})
			},
			pvName:        "test-pv",
			expectedError: "",
		},
		{
			name: "ListError",
			setup: func(client *fake.Clientset) {
				// Simulate an error for the List operation
				client.PrependReactor("list", "volumeattachments", func(action clientgotesting.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("simulated list error")
				})
			},
			pvName:        "test-pv",
			expectedError: "simulated list error",
		},
		{
			name: "DeleteError",
			setup: func(client *fake.Clientset) {
				// Create a VolumeAttachment to be deleted
				va := &storagev1.VolumeAttachment{
					ObjectMeta: v1.ObjectMeta{
						Name: "test-va",
					},
					Spec: storagev1.VolumeAttachmentSpec{
						Source: storagev1.VolumeAttachmentSource{
							PersistentVolumeName: func() *string { s := "test-pv"; return &s }(),
						},
					},
				}
				client.StorageV1().VolumeAttachments().Create(context.TODO(), va, v1.CreateOptions{})

				// Simulate an error for the Delete operation
				client.PrependReactor("delete", "volumeattachments", func(action clientgotesting.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("simulated delete error")
				})
			},
			pvName:        "test-pv",
			expectedError: "simulated delete error",
		},
		{
			name: "NotFound",
			setup: func(client *fake.Clientset) {
				// No setup needed for not found case
			},
			pvName:        "non-existent-pv",
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			vaClient := &va.Client{
				Interface: client.StorageV1().VolumeAttachments(),
				Namespace: "default",
				Timeout:   10,
			}

			ctx := context.TODO()
			tt.setup(client)

			err := vaClient.DeleteVaBasedOnPVName(ctx, tt.pvName)
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}

			// Verify the VolumeAttachment is deleted for Success test case
			if tt.name == "Success" {
				_, err = client.StorageV1().VolumeAttachments().Get(ctx, "test-va", v1.GetOptions{})
				assert.Error(t, err)
			}
		})
	}
}