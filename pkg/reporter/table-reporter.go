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
	"bufio"
	"errors"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dell/cert-csi/pkg/collector"

	log "github.com/sirupsen/logrus"
)

// TabularReporter is used to create and manage report in a tabular form
type TabularReporter struct{}

var arrayConfig map[string]string
var passedCount int
var failedCount int
var skippedCount int

// MultiGenerate generates reports for multiple metrics collections
func (tr *TabularReporter) MultiGenerate(mcs []*collector.MetricsCollection) error {
	var fm = template.FuncMap{
		"formatName":           formatName,
		"getResultStatus":      tr.getResultStatus,
		"getColorResultStatus": tr.getColorResultStatus,
		"getCurrentDate":       tr.getCurrentDate,
		"getSlNo":              tr.getSlNo,
		"getCustomReportName":  tr.getCustomReportName,
		"getPassedCount":       tr.getPassedCount,
		"getFailedCount":       tr.getFailedCount,
		"getSkippedCount":      tr.getSkippedCount,
		"getBuildName":         tr.getBuildName,
		"getArrays":            tr.getArrays,
		"inc":                  inc,
		"getTestDuration":      getTestDuration,
		"getFailedCountFromMC": getFailedCountFromMC,
		"getPassedCountFromMC": getPassedCountFromMC,
	}

	templateData, err := embedFS.ReadFile("templates/multi-tabular-html-template.html")
	if err != nil {
		return err
	}

	report, err := template.New("multi-tabular-html-template").Funcs(fm).Parse(string(templateData))
	if err != nil {
		return err
	}

	htmlFile, _, err := getReportFile("tabular", "html")
	if err != nil {
		return err
	}

	err = addPathToFile("report.path", "TABULAR_REPORT_PATH", htmlFile.Name())
	if err != nil {
		return err
	}

	defer func() {
		if err := htmlFile.Close(); err != nil {
			panic(err)
		}
	}()

	if err := report.Execute(htmlFile, mcs); err != nil {
		return err
	}

	return nil
}

// Generate generates report for a metrics collection
func (tr *TabularReporter) Generate(runName string, mc *collector.MetricsCollection) error {
	fm := template.FuncMap{
		"formatName":           formatName,
		"getResultStatus":      tr.getResultStatus,
		"getColorResultStatus": tr.getColorResultStatus,
		"getCurrentDate":       tr.getCurrentDate,
		"getSlNo":              tr.getSlNo,
		"getCustomReportName":  tr.getCustomReportName,
		"getPassedCount":       tr.getPassedCount,
		"getFailedCount":       tr.getFailedCount,
		"getSkippedCount":      tr.getSkippedCount,
		"getBuildName":         tr.getBuildName,
		"getArrays":            tr.getArrays,
	}

	// Fixing the file name to constant as it is not going to change/taken from user every time.

	arrayConfig, _ = getArrayConfig("./array-config.properties")

	updateTestCounts(mc)

	templateData, err := embedFS.ReadFile("templates/tabular-html-template.html")
	if err != nil {
		return err
	}

	report, err := template.New("tabular-html-template").Funcs(fm).Parse(string(templateData))
	if err != nil {
		return err
	}

	// Write a report to file
	currentTime := time.Now()

	htmFile, _, err := getReportFile("tabular-report-"+currentTime.Format("2006-01-02_15_04_05"), "html")
	if err != nil {
		return err
	}
	defer func() {
		if err := htmFile.Close(); err != nil {
			panic(err)
		}
	}()

	if err := report.Execute(htmFile, mc); err != nil {
		return err
	}

	return nil
}

func updateTestCounts(mc *collector.MetricsCollection) {
	for i := 0; i < len(mc.TestCasesMetrics); i++ {
		if mc.TestCasesMetrics[i].TestCase.Success {
			passedCount++
		} else if !mc.TestCasesMetrics[i].TestCase.Success {
			failedCount++
		} else {
			skippedCount++
		}
	}
}

func (tr *TabularReporter) getResultStatus(result bool) string {
	if result {
		return "SUCCESS"
	}
	return "FAILURE"
}

func (tr *TabularReporter) getColorResultStatus(result bool) string {
	if result {
		return "green"
	}
	return "red"
}

func (tr *TabularReporter) getSlNo(index int) int {
	return index + 1
}

func (tr *TabularReporter) getCurrentDate() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
func (tr *TabularReporter) getCustomReportName() string {
	return "csi-" + arrayConfig["name"] + "-test-results"
}

func (tr *TabularReporter) getPassedCount() int {
	return passedCount
}

func (tr *TabularReporter) getFailedCount() int {
	return failedCount
}

func (tr *TabularReporter) getSkippedCount() int {
	return skippedCount
}

func (tr *TabularReporter) getBuildName() string {
	return os.Getenv("CERT_CSI_BUILD_NAME")
}

func (tr *TabularReporter) getArrays() string {
	val, ok := arrayConfig["arrayIPs"]
	if !ok {
		return ""
	}
	return val
}

// ReadConfigFile reads any .properties file with = as delimiter
func getArrayConfig(filename string) (map[string]string, error) {
	// init with some bogus data
	configPropertiesMap := map[string]string{}
	if len(filename) == 0 {
		return nil, errors.New("Error reading properties file " + filename)
	}
	file, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Errorf("Error closing file: %s\n", err)
		}
	}()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')

		// check if the line has = sign
		// and process the line. Ignore the rest.
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				// assign the config map
				configPropertiesMap[key] = value
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return configPropertiesMap, nil
}
