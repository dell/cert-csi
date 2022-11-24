package reporter

import (
	"cert-csi/pkg/collector"
	"cert-csi/pkg/plotter"
	"cert-csi/pkg/store"
	"embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
)

type ReportType string

const (
	HtmlReport ReportType = "HTML"
	TextReport ReportType = "TEXT"
)

//go:embed templates/*
var embedFS embed.FS

var (
	PathReport string
)

type Reporter interface {
	Generate(runName string, mc *collector.MetricsCollection) error
}

type MultiReporter interface {
	MultiGenerate(mc []*collector.MetricsCollection) error
}

func GenerateAllReports(testRunNames []string, db store.Store) error {
	reportTypes := []ReportType{
		HtmlReport,
		TextReport,
	}
	return GenerateReports(testRunNames, reportTypes, db)
}

func GenerateReportsFromMultipleDBs(reportTypes []ReportType, scDBs []*store.StorageClassDB) error {
	funcMap := map[ReportType]MultiReporter{
		TabularReport: &TabularReporter{},
		XmlReport:     &XmlReporter{},
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

func GenerateReports(testRunNames []string, reportTypes []ReportType, db store.Store) error {
	mc := collector.NewMetricsCollector(db)
	funcMap := map[ReportType]Reporter{
		HtmlReport: &HtmlReporter{},
		TextReport: &TextReporter{},
	}
	log.Infof("Started generating reports...")

	for _, testRun := range testRunNames {
		metricsCollection, err := mc.Collect(testRun)
		if err != nil {
			return err
		}

		generatePlots(testRun, metricsCollection)

		for _, reportType := range reportTypes {
			if err := funcMap[reportType].Generate(testRun, metricsCollection); err != nil {
				return err
			}
		}
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

type PlotPath struct {
	Path       string
	ReportName string
}

func (pp *PlotPath) Txt() string {
	absolutePath := filepath.Join(PathReport, pp.Path)
	return strings.ReplaceAll(absolutePath, "\\", "/")
}

func (pp *PlotPath) Html() template.URL {
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
	switch stage.(type) {
	case collector.PVCStage:
		fileName = fmt.Sprintf("%s.png", stage.(collector.PVCStage)+"_boxplot")
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
	f, err := os.OpenFile(filepath.Clean(fileName), os.O_RDWR|os.O_CREATE, 0600)
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return err
	}

	input, err := ioutil.ReadFile(f.Name())
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
	err = ioutil.WriteFile(f.Name(), []byte(output), 0600)
	if err != nil {
		panic(err)
	}

	return nil
}