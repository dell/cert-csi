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
	"html/template"

	"github.com/dell/cert-csi/pkg/collector"
)

const htmlNameTemplate = "template.html"

// HTMLReporter is used to create and manage HTML report
type HTMLReporter struct{}

// Generate generates a HTML report
func (hr *HTMLReporter) Generate(runName string, mc *collector.MetricsCollection) error {
	var fm = template.FuncMap{
		"formatName":                      formatName,
		"inc":                             inc,
		"getResultStatus":                 hr.getResultStatus,
		"getColorResultStatus":            hr.getColorResultStatus,
		"shouldBeIncluded":                shouldBeIncluded,
		"getPlotStageMetricHistogramPath": getPlotStageMetricHistogramPath,
		"getPlotStageBoxPath":             getPlotStageBoxPath,
		"getPlotEntityOverTimePath":       getPlotEntityOverTimePath,
		"getMinMaxEntityOverTimePaths":    getMinMaxEntityOverTimePaths,
		"getDriverResourceUsage":          getDriverResourceUsage,
		"getAvgStageTimeOverIterations":   getAvgStageTimeOverIterations,
		"getIterationTimes":               getIterationTimes,
	}

	templateData, err := embedFS.ReadFile("templates/perf-template.html")
	if err != nil {
		return err
	}

	report, err := template.New(htmlNameTemplate).Funcs(fm).Parse(string(templateData))
	if err != nil {
		return err
	}

	htmlFile, _, err := getReportFile(runName, "html")
	if err != nil {
		return err
	}

	err = addPathToFile("report.path", "HTML_REPORT_PATH", htmlFile.Name())
	if err != nil {
		return err
	}

	if err := report.Execute(htmlFile, mc); err != nil {
		return err
	}

	return nil
}

func (hr *HTMLReporter) getResultStatus(result bool) string {
	if result {
		return "SUCCESS"
	}
	return "FAILURE"
}

func (hr *HTMLReporter) getColorResultStatus(result bool) string {
	if result {
		return "green"
	}
	return "red"
}
