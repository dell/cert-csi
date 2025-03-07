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

package reporter

import (
	"reflect"
	"testing"
)

func TestGetArrayConfig(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     map[string]string
		wantErr  bool
	}{
		{
			name:     "file not found",
			filename: "nonexistent-file.properties",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "invalid file format",
			filename: "",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getArrayConfig(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("getArrayConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getArrayConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
