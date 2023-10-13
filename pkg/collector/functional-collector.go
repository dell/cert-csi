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

package collector

import (
	"errors"
	"fmt"

	"github.com/dell/cert-csi/pkg/store"

	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
)

// CollectFunctionalMetrics gets metrics required for tabular reporting
func (mc *MetricsCollector) CollectFunctionalMetrics() (*MetricsCollection, error) {
	if mc.metricsCache != nil {
		metrics, ok := mc.metricsCache[""]
		if ok {
			return metrics, nil
		}
	}
	if mc.db == nil {
		return nil, errors.New("database can't be nil")
	}

	var testRun store.TestRun
	testRun.Name = "Test Run"
	testCases, err := mc.db.GetTestCases(store.Conditions{}, "", 0)
	if err != nil {
		log.Error("Couldn't get test cases")
		return nil, err
	}

	var testCasesMetrics []TestCaseMetrics
	var bar *pb.ProgressBar
	if log.GetLevel() != log.PanicLevel {
		fmt.Println("Collecting functional metrics")
		bar = pb.Default.Start(len(testCases))
	}
	for _, tc := range testCases {
		if bar != nil {
			bar.Increment()
		}

		testCaseMetrics := TestCaseMetrics{
			TestCase: tc,
		}
		testCasesMetrics = append(testCasesMetrics, testCaseMetrics)
	}
	if bar != nil {
		bar.Finish()
	}
	mc.metricsCache[""] = &MetricsCollection{testRun, testCasesMetrics}

	return mc.metricsCache[""], nil
}
