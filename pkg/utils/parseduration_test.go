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

package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	type parseTest struct {
		n        string
		expected time.Duration
		wantErr  bool
		errMsg   string
	}

	parseTests := []parseTest{
		{"2w3d4m14h", time.Duration(1519440000000000), false, ""},
		{"1w", time.Duration(604800000000000), false, ""},
		{"1H3w2M5s", time.Duration(1818125000000000), false, ""},
		{"343", time.Duration(0), true, "duration string is not in the correct format"},
	}

	for _, tt := range parseTests {
		var actual time.Duration
		ext, err := ParseDuration(tt.n)
		if ext != nil {
			actual = ext.Duration()
		}
		if tt.wantErr {
			assert.Error(t, err)
			assert.Equal(t, tt.errMsg, err.Error())
			continue
		}
		assert.NoError(t, err)

		if actual != tt.expected {
			t.Errorf("Parse (%s): expected %d, actual %d", tt.n, tt.expected, actual)
		}
	}
}
