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
	"cert-csi/pkg/collector"
	"cert-csi/pkg/store"
	"text/template"
	"time"
)

// XMLReporter is used to create and manage XML report
type XMLReporter struct{}

// MultiGenerate generates report from multiple metrics collection
func (xr *XMLReporter) MultiGenerate(mcs []*collector.MetricsCollection) error {
	var fm = template.FuncMap{
		"formatName":           formatName,
		"getResultStatus":      xr.getResultStatus,
		"getCustomReportName":  xr.getCustomReportName,
		"getFailedCountFromMC": getFailedCountFromMC,
		"getTestDuration":      getTestDuration,
	}

	templateData, err := embedFS.ReadFile("templates/multi-xml-template.xml")
	if err != nil {
		return err
	}

	report, err := template.New("multi-xml-template").Funcs(fm).Parse(string(templateData))
	if err != nil {
		return err
	}

	xmlFile, _, err := getReportFile("junit", "xml")
	if err != nil {
		return err
	}

	err = addPathToFile("report.path", "JUNIT_REPORT_PATH", xmlFile.Name())
	if err != nil {
		return err
	}

	defer func() {
		if err := xmlFile.Close(); err != nil {
			panic(err)
		}
	}()

	if err := report.Execute(xmlFile, mcs); err != nil {
		return err
	}

	return nil
}

// Generate generates report from metrics collection
func (xr *XMLReporter) Generate(runName string, mc *collector.MetricsCollection) error {
	var fm = template.FuncMap{
		"formatName":          formatName,
		"getResultStatus":     xr.getResultStatus,
		"getCustomReportName": xr.getCustomReportName,
		"getPassedCount":      xr.getPassedCount,
		"getSkippedCount":     xr.getSkippedCount,
		"getFailedCount":      xr.getFailedCount,
		"getTestDuration":     getTestDuration,
	}

	// Fixing the file name to constant as it is not going to change/taken from user every time.
	arrayConfig, _ = getArrayConfig("./array-config.properties")

	updateTestCounts(mc)

	templateData, err := embedFS.ReadFile("templates/func-xml-template.xml")
	if err != nil {
		return err
	}

	report, err := template.New("func-xml-template").Funcs(fm).Parse(string(templateData))
	if err != nil {
		return err
	}

	// Write a report to file
	currentTime := time.Now()

	xmlFile, _, err := getReportFile("xml-report-"+currentTime.Format("2006-01-02_15_04_05"), "xml")
	if err != nil {
		return err
	}
	defer func() {
		if err := xmlFile.Close(); err != nil {
			panic(err)
		}
	}()

	if err := report.Execute(xmlFile, mc); err != nil {
		return err
	}

	return nil
}

func (xr *XMLReporter) getResultStatus(result bool) string {
	if result {
		return "PASSED"
	}
	return "FAILED"
}

func (xr *XMLReporter) getCustomReportName() string {
	return "csi-" + arrayConfig["name"] + "-test-results"
}

func (xr *XMLReporter) getPassedCount() int {
	return passedCount
}

func (xr *XMLReporter) getFailedCount() int {
	return failedCount
}

func getFailedCountFromMC(mc *collector.MetricsCollection) int {
	failed := 0
	for i := 0; i < len(mc.TestCasesMetrics); i++ {
		if !mc.TestCasesMetrics[i].TestCase.Success {
			failed++
		}
	}
	return failed
}

func getPassedCountFromMC(mc *collector.MetricsCollection) int {
	passed := 0
	for i := 0; i < len(mc.TestCasesMetrics); i++ {
		if mc.TestCasesMetrics[i].TestCase.Success {
			passed++
		}
	}
	return passed
}

func (xr *XMLReporter) getSkippedCount() int {
	return skippedCount
}

func getTestDuration(tc store.TestCase) string {
	st := tc.StartTimestamp
	en := tc.EndTimestamp
	diff := en.Sub(st)
	out := time.Time{}.Add(diff)
	return out.Format("04:05")
}
