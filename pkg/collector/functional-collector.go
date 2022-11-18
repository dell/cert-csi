package collector

import (
	"cert-csi/pkg/store"
	"errors"
	"fmt"

	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
)

// CollectFunctionalMetrics gets metrics required for tabular reporting
func (mc *MetricsCollector) CollectFunctionalMetrics() (*MetricsCollection, error) {
	if mc.metricsCache != nil {
		metrics, ok := mc.metricsCache[""]
		if ok {
			return metrics, nil
		}
	}
	if mc.db == nil {
		return nil, errors.New("database can't be nil")
	}

	var testRun store.TestRun
	testRun.Name = "Test Run"
	testCases, err := mc.db.GetTestCases(store.Conditions{}, "", 0)
	if err != nil {
		log.Error("Couldn't get test cases")
		return nil, err
	}

	var testCasesMetrics []TestCaseMetrics
	var bar *pb.ProgressBar
	if log.GetLevel() != log.PanicLevel {
		fmt.Println("Collecting functional metrics")
		bar = pb.Default.Start(len(testCases))
	}
	for _, tc := range testCases {
		if bar != nil {
			bar.Increment()
		}

		testCaseMetrics := TestCaseMetrics{
			TestCase: tc,
		}
		testCasesMetrics = append(testCasesMetrics, testCaseMetrics)
	}
	if bar != nil {
		bar.Finish()
	}
	mc.metricsCache[""] = &MetricsCollection{testRun, testCasesMetrics}

	return mc.metricsCache[""], nil
}
