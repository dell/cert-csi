package reporter

import (
	"cert-csi/pkg/collector"
	"html/template"
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
