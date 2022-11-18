package reporter

import (
	"cert-csi/pkg/collector"
	"cert-csi/pkg/store"
	"text/template"
	"time"
)

type XmlReporter struct{}

func (xr *XmlReporter) MultiGenerate(mcs []*collector.MetricsCollection) error {
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

func (xr *XmlReporter) Generate(runName string, mc *collector.MetricsCollection) error {
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

func (xr *XmlReporter) getResultStatus(result bool) string {
	if result {
		return "PASSED"
	}
	return "FAILED"
}

func (xr *XmlReporter) getCustomReportName() string {
	return "csi-" + arrayConfig["name"] + "-test-results"
}

func (xr *XmlReporter) getPassedCount() int {
	return passedCount
}

func (xr *XmlReporter) getFailedCount() int {
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

func (xr *XmlReporter) getSkippedCount() int {
	return skippedCount
}

func getTestDuration(tc store.TestCase) string {
	st := tc.StartTimestamp
	en := tc.EndTimestamp
	diff := en.Sub(st)
	out := time.Time{}.Add(diff)
	return out.Format("04:05")
}
