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
package collector

import (
	"testing"

	"github.com/dell/cert-csi/pkg/store"
	"github.com/stretchr/testify/assert"
)

func TestCollectFunctionalMetrics(t *testing.T) {
	t.Run("MetricsCacheAvailable", func(t *testing.T) {
		mc := &MetricsCollector{
			metricsCache: map[string]*MetricsCollection{
				"": {
					TestCasesMetrics: []TestCaseMetrics{
						{TestCase: store.TestCase{Name: "Cached Test Case"}},
					},
				},
			},
		}

		metrics, err := mc.CollectFunctionalMetrics()
		assert.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.Equal(t, "Cached Test Case", metrics.TestCasesMetrics[0].TestCase.Name)
	})

	t.Run("DatabaseNil", func(t *testing.T) {
		mc := &MetricsCollector{}

		metrics, err := mc.CollectFunctionalMetrics()
		assert.Error(t, err)
		assert.Nil(t, metrics)
		assert.Equal(t, "database can't be nil", err.Error())
	})
}
