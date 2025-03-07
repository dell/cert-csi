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

package reporter

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dell/cert-csi/pkg/collector"
	"github.com/dell/cert-csi/pkg/plotter"
	"github.com/dell/cert-csi/pkg/store"

	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
)

// ReportType represent types of report to be generated
type ReportType string

const (
	// HTMLReport represents HTML report
	HTMLReport ReportType = "HTML"
	// TextReport represents Text report
	TextReport ReportType = "TEXT"
)

//go:embed templates/*
var embedFS embed.FS

// PathReport represent path of the report file
var PathReport string

// Reporter is an interface for generating reports
type Reporter interface {
	Generate(runName string, mc *collector.MetricsCollection) error
}

// MultiReporter is an interface for generating multiple type of reports
type MultiReporter interface {
	MultiGenerate(mc []*collector.MetricsCollection) error
}

// GenerateAllReports generates reports of types HTML and Text
func GenerateAllReports(dbs []*store.StorageClassDB) error {
	reportTypes := []ReportType{
		HTMLReport,
		TextReport,
	}
	return GenerateReports(reportTypes, dbs)
}

// GenerateReportsFromMultipleDBs generates reports from multiple DBs
func GenerateReportsFromMultipleDBs(reportTypes []ReportType, scDBs []*store.StorageClassDB) error {
	funcMap := map[ReportType]MultiReporter{
		TabularReport: &TabularReporter{},
		XMLReport:     &XMLReporter{},
	}

	log.Infof("Started generating reports...")

	var mcs []*collector.MetricsCollection
	for _, scDB := range scDBs {
		c := collector.NewMetricsCollector(scDB.DB)
		metricsCollection, err := c.Collect(scDB.TestRun.Name)
		if err != nil {
			return err
		}
		mcs = append(mcs, metricsCollection)
	}

	for _, reportType := range reportTypes {
		if err := funcMap[reportType].MultiGenerate(mcs); err != nil {
			return err
		}
	}

	return nil
}

// GenerateReports generates reports of type HTML and Text
func GenerateReports(reportTypes []ReportType, dbs []*store.StorageClassDB) error {
	funcMap := map[ReportType]Reporter{
		HTMLReport: &HTMLReporter{},
		TextReport: &TextReporter{},
	}
	log.Infof("Started generating reports...")

	invalidTestRuns := []string{}
	// Checking availability of all the test runs in every database
	for i := 0; i < len(dbs); i++ {
		mc := collector.NewMetricsCollector(dbs[i].DB)
		metricsCollection, err := mc.Collect(dbs[i].TestRun.Name)
		if err != nil {
			log.Warnf("unable to collect metrics for the test run: %s", dbs[i].TestRun.Name)
			invalidTestRuns = append(invalidTestRuns, dbs[i].TestRun.Name)
			continue
		}

		generatePlots(dbs[i].TestRun.Name, metricsCollection)

		for _, reportType := range reportTypes {
			if err := funcMap[reportType].Generate(dbs[i].TestRun.Name, metricsCollection); err != nil {
				return err
			}
		}
	}
	if len(invalidTestRuns) != 0 {
		return fmt.Errorf("test runs: %v not found", invalidTestRuns)
	}
	return nil
}

func generatePlots(runName string, mc *collector.MetricsCollection) {
	var bar *pb.ProgressBar
	if log.GetLevel() != log.PanicLevel {
		fmt.Println("Generating plots")
		bar = pb.Default.Start(len(mc.TestCasesMetrics))
	}
	for _, tcMetrics := range mc.TestCasesMetrics {
		if bar != nil {
			bar.Increment()
		}
		for stage, metrics := range tcMetrics.StageMetrics {
			if shouldBeIncluded(metrics) {
				_, err := plotter.PlotStageMetricHistogram(tcMetrics, stage, runName)
				if err != nil {
					log.Error("Unable to include metric")
				}
				_, err = plotter.PlotStageBoxPlot(tcMetrics, stage, runName)
				if err != nil {
					log.Error("Unable to include metric")
				}
			}
		}
		_, err := plotter.PlotEntityOverTime(tcMetrics, runName)
		if err != nil {
			log.Error(err)
		}
	}
	if bar != nil {
		bar.Finish()
	}
	err := plotter.PlotMinMaxEntityOverTime(mc.TestCasesMetrics, runName)
	if err != nil {
		log.Error(err)
	}
	err = plotter.PlotResourceUsageOverTime(mc.TestCasesMetrics, runName)
	if err != nil {
		log.Error(err)
	}
	err = plotter.PlotAvgStageTimeOverIterations(mc.TestCasesMetrics, runName)
	if err != nil {
		log.Error(err)
	}
	_, err = plotter.PlotIterationTimes(mc.TestCasesMetrics, runName)
	if err != nil {
		log.Error(err)
	}
}

// PlotPath holds report name and path
type PlotPath struct {
	Path       string
	ReportName string
}

// Txt formats the Text report path
func (pp *PlotPath) Txt() string {
	absolutePath := filepath.Join(PathReport, pp.Path)
	return strings.ReplaceAll(absolutePath, "\\", "/")
}

// HTML formats HTML report path
func (pp *PlotPath) HTML() template.URL {
	return template.URL(strings.ReplaceAll(pp.Path, "\\", "/")) // #nosec G203
}

func getPlotStageMetricHistogramPath(tc collector.TestCaseMetrics, stage interface{}, reportName string) *PlotPath {
	return &PlotPath{
		Path: filepath.Join(
			".",
			tc.TestCase.Name+strconv.Itoa(int(tc.TestCase.ID)),
			fmt.Sprintf("%s.png", stage),
		),
		ReportName: reportName,
	}
}

func getPlotStageBoxPath(tc collector.TestCaseMetrics, stage interface{}, reportName string) *PlotPath {
	var fileName string
	switch stage := stage.(type) {
	case collector.PVCStage:
		fileName = fmt.Sprintf("%s.png", stage+"_boxplot")
	default:
		fileName = fmt.Sprintf("%s.png", stage.(collector.PodStage)+"_boxplot")
	}
	return &PlotPath{
		Path:       filepath.Join(".", tc.TestCase.Name+strconv.Itoa(int(tc.TestCase.ID)), fileName),
		ReportName: reportName,
	}
}

func getPlotEntityOverTimePath(tc collector.TestCaseMetrics, reportName string) *PlotPath {
	return &PlotPath{
		Path: filepath.Join(
			".",
			tc.TestCase.Name+strconv.Itoa(int(tc.TestCase.ID)),
			"EntityNumberOverTime.png",
		),
		ReportName: reportName,
	}
}

func getIterationTimes(reportName string) *PlotPath {
	return &PlotPath{
		Path: filepath.Join(
			".",
			"IterationTimes.png",
		),
		ReportName: reportName,
	}
}

func getDriverResourceUsage(reportName string) []*PlotPath {
	var plotPath []*PlotPath
	cpuUsage := "CpuUsageOverTime.png"
	memUsage := "MemUsageOverTime.png"
	filePath := filepath.Dir(PathReport)
	if fileExists(fmt.Sprintf("%s/%s/%s", filePath, reportName, cpuUsage)) {
		plotPath = append(plotPath,
			&PlotPath{
				Path: filepath.Join(
					".",
					cpuUsage,
				),
				ReportName: reportName,
			})
	}

	if fileExists(fmt.Sprintf("%s/%s/%s", filePath, reportName, memUsage)) {
		plotPath = append(plotPath,
			&PlotPath{
				Path: filepath.Join(
					".",
					memUsage,
				),
				ReportName: reportName,
			})
	}
	return plotPath
}

func getAvgStageTimeOverIterations(reportName string) []*PlotPath {
	var plotPath []*PlotPath
	names := []string{
		"PodCreationOverIterations.png",
		"PodDeletionOverIterations.png",
		"PVCAttachmentOverIterations.png",
		"PVCBindOverIterations.png",
		"PVCCreationOverIterations.png",
		"PVCDeletionOverIterations.png",
		"PVCUnattachmentOverIterations.png",
	}

	filePath := filepath.Dir(PathReport)
	for _, name := range names {
		if fileExists(fmt.Sprintf("%s/%s/%s", filePath, reportName, name)) {
			plotPath = append(plotPath,
				&PlotPath{
					Path: filepath.Join(
						".",
						name,
					),
					ReportName: reportName,
				})
		}
	}
	return plotPath
}

func getMinMaxEntityOverTimePaths(reportName string) []*PlotPath {
	return []*PlotPath{
		{
			Path: filepath.Join(
				".",
				"PodsCreatingOverTime.png",
			),
			ReportName: reportName,
		},
		{
			Path: filepath.Join(
				".",
				"PodsReadyOverTime.png",
			),
			ReportName: reportName,
		},
		{
			Path: filepath.Join(
				".",
				"PodsTerminatingOverTime.png",
			),
			ReportName: reportName,
		},
		{
			Path: filepath.Join(
				".",
				"PvcsCreatingOverTime.png",
			),
			ReportName: reportName,
		},
		{
			Path: filepath.Join(
				".",
				"PvcsBoundOverTime.png",
			),
			ReportName: reportName,
		},
	}
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func addPathToFile(fileName string, exportVar string, path string) error {
	f, err := os.OpenFile(filepath.Clean(fileName), os.O_RDWR|os.O_CREATE, 0o600)
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return err
	}

	input, err := os.ReadFile(f.Name())
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(input), "\n")

	found := false
	for i, line := range lines {
		if strings.Contains(line, fmt.Sprintf("export %s=", exportVar)) {
			lines[i] = fmt.Sprintf("export %s=%s", exportVar, path)
			found = true
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("export %s=%s", exportVar, path))
	}

	output := strings.Join(lines, "\n")
	err = os.WriteFile(f.Name(), []byte(output), 0o600)
	if err != nil {
		panic(err)
	}

	return nil
}
