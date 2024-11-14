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
package suites

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTopologyCount(t *testing.T) {
	// Test case: Empty topology keys
	FindDriverLogs = func(_ []string) (string, error) {
		return "", nil
	}
	topologyCount, err := getTopologyCount([]string{})
	assert.NoError(t, err)
	assert.Equal(t, 0, topologyCount)

	// Test case: Non-empty topology keys

	FindDriverLogs = func(_ []string) (string, error) {
		var keys = "Topology Keys: [csi-powerstore.dellemc.com/10.230.24.67-iscsi csi-powerstore.dellemc.com/10.230.24.67-nfs]"
		return keys, nil
	}
	topologyCount, err = getTopologyCount([]string{"csi-powerstore.dellemc.com/10.230.24.67-iscsi"})
	assert.NoError(t, err)
	assert.Equal(t, 1, topologyCount)

	// Test case: Error in FindDriverLogs
	FindDriverLogs = func(_ []string) (string, error) {
		return "", errors.New("error in FindDriverLogs")
	}
	type FindDriverLogs func(url string) string
	topologyCount, err = getTopologyCount([]string{})
	assert.Error(t, err)
	assert.Equal(t, 0, topologyCount)
}
