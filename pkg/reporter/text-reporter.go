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
	"fmt"
	"os"
	"text/template"

	"github.com/dell/cert-csi/pkg/collector"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
)

const txtNameTemplate = "template.txt"

// TextReporter is used to create and manage Text report
type TextReporter struct{}

// Generate text report for metrics collection
func (tr *TextReporter) Generate(runName string, mc *collector.MetricsCollection) error {
	fm := template.FuncMap{
		"formatName":                      formatName,
		"inc":                             inc,
		"getResultStatus":                 tr.getResultStatus,
		"shouldBeIncluded":                shouldBeIncluded,
		"colorYellow":                     colorYellow,
		"colorCyan":                       colorCyan,
		"getPlotStageMetricHistogramPath": getPlotStageMetricHistogramPath,
		"getPlotStageBoxPath":             getPlotStageBoxPath,
		"getPlotEntityOverTimePath":       getPlotEntityOverTimePath,
		"getMinMaxEntityOverTimePaths":    getMinMaxEntityOverTimePaths,
	}

	templateData, err := embedFS.ReadFile("templates/txt-template.txt")
	if err != nil {
		return err
	}

	report, err := template.New(txtNameTemplate).Funcs(fm).Parse(string(templateData))
	if err != nil {
		return err
	}
	// Write a report to Stdout
	if log.GetLevel() != log.PanicLevel {
		if err := report.Execute(os.Stdout, mc); err != nil {
			return err
		}
	}
	// Write a report to file
	txtFile, _, err := getReportFile(runName, "txt")
	if err != nil {
		return err
	}
	defer func() {
		if err := txtFile.Close(); err != nil {
			panic(err)
		}
	}()

	err = addPathToFile("report.path", "TXT_REPORT_PATH", txtFile.Name())
	if err != nil {
		return err
	}

	if err := report.Execute(txtFile, mc); err != nil {
		return err
	}

	return nil
}

func (tr *TextReporter) getResultStatus(result bool) string {
	if result {
		return color.GreenString("SUCCESS")
	}
	return color.RedString("FAILURE")
}

func colorYellow(colorable interface{}) string {
	return color.HiYellowString(fmt.Sprint(colorable))
}

func colorCyan(colorable interface{}) string {
	return color.CyanString(fmt.Sprint(colorable))
}
