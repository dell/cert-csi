package reporter

import (
	"cert-csi/pkg/collector"
	"cert-csi/pkg/store"

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
