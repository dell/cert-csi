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

package reporter

import (
	"github.com/dell/cert-csi/pkg/collector"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
)

// TabularReport represents TABLE report type
const TabularReport ReportType = "TABLE"

// XMLReport represents XML report type
const XMLReport ReportType = "XML"

// GenerateFunctionalReport generates functions reports
func GenerateFunctionalReport(db store.Store, reportTypes []ReportType) error {
	mc := collector.NewMetricsCollector(db)
	log.Infof("Started generating functional reports...")
	metricsCollection, err := mc.CollectFunctionalMetrics()
	if err != nil {
		return err
	}
	funcMap := map[ReportType]Reporter{
		TabularReport: &TabularReporter{},
		XMLReport:     &XMLReporter{},
	}
	for _, reportType := range reportTypes {
		if err := funcMap[reportType].Generate("", metricsCollection); err != nil {
			return err
		}
	}
	return nil
}
