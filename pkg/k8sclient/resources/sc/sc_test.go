/*
 *
 * Copyright © 2024 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package sc

import (
	"context"
	"testing"

	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_Create_Get_Delete(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()
	scName := "test-sc"
	ctx := context.Background()

	// Create a new storage class
	storageClass := &v1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: scName,
		},
		Provisioner: "test-provisioner",
	}

	// Create a new client
	client := &Client{
		Interface: clientset.StorageV1().StorageClasses(),
		ClientSet: clientset,
		Timeout:   10,
	}

	tests := []struct {
		name          string
		getSCName     string
		deleteSCName  string
		wantCreateErr bool
		wantGetErr    bool
		wantDeleteErr bool
	}{
		{"normal case", scName, scName, false, false, false},
		{"wrong get storage class name", "wrong-name", scName, false, true, false},
		{"wrong delete storage class name", scName, "wrong-name", false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the storage class
			err := client.Create(ctx, storageClass)
			if (err != nil) != tt.wantCreateErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantCreateErr)
				return
			}

			// Get the storage class
			sc := client.Get(ctx, tt.getSCName)
			if (sc.error != nil) != tt.wantGetErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantGetErr)
			}

			// Check if the storage class is retrieved correctly
			if !tt.wantGetErr && sc.Object.Name != scName {
				t.Errorf("Expected storage class name to be '%s', got %s", scName, sc.Object.Name)
			}

			// Delete the storage class
			err = client.Delete(ctx, tt.deleteSCName)
			if (err != nil) != tt.wantDeleteErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantDeleteErr)
				return
			}
		})
	}
}
