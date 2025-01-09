/*
 *
 * Copyright © 2023 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package utils

import (
	"reflect"
	"testing"
)

func TestE2eReportParser(t *testing.T) {
	var preResult []map[string]string
	preResult = append(preResult, map[string]string{
		"TestSuiteName":      "Kubernetes e2e suite",
		"TotalTestsExecuted": "47",
		"TotalTestsPassed":   "47",
		"TotalTestsFailed":   "0",
	})
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    []map[string]string
		wantErr bool
	}{
		{"Check generated config", args{"testdata/execution_powerstore-nfs.xml"}, preResult, false},
		{"Check generated config negative", args{"testdata/execution_powerstore-nfs-dummy.xml"}, preResult, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := E2eReportParser(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("E2eReportParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) && !tt.wantErr {
				t.Errorf("E2eReportParser() = %v, want %v", got, tt.want)
			}
		})
	}
}
