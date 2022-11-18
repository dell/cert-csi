package reporter

import (
	"cert-csi/pkg/collector"
	"cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
)

const TabularReport ReportType = "TABLE"
const XmlReport ReportType = "XML"

func GenerateFunctionalReport(db store.Store, reportTypes []ReportType) error {
	mc := collector.NewMetricsCollector(db)
	log.Infof("Started generating functional reports...")
	metricsCollection, err := mc.CollectFunctionalMetrics()
	if err != nil {
		return err
	}
	funcMap := map[ReportType]Reporter{
		TabularReport: &TabularReporter{},
		XmlReport:     &XmlReporter{},
	}
	for _, reportType := range reportTypes {
		if err := funcMap[reportType].Generate("", metricsCollection); err != nil {
			return err
		}
	}
	return nil
}
