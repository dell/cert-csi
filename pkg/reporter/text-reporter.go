package reporter

import (
	"cert-csi/pkg/collector"
	"fmt"
	"os"
	"text/template"

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
