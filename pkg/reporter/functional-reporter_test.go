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
	"testing"

	"github.com/dell/cert-csi/pkg/store"
)

func TestGenerateFunctionalReport(t *testing.T) {
	tests := []struct {
		name        string
		db          store.Store
		reportTypes []ReportType
		wantErr     bool
	}{
		{
			name: "error during report generation",
			db:   nil,
			reportTypes: []ReportType{
				TabularReport,
				XMLReport,
			},
			wantErr: true,
		},
		// add more test cases with different scenarios
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateFunctionalReport(tt.db, tt.reportTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateFunctionalReport() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
