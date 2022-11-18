package collector

import (
	"cert-csi/pkg/store"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
	"time"
)

type Stage string
type PVCStage Stage
type PodStage Stage

const (
	PVCBind         PVCStage = "PVCBind"
	PVCAttachment   PVCStage = "PVCAttachment"
	PVCCreation     PVCStage = "PVCCreation"
	PVCDeletion     PVCStage = "PVCDeletion"
	PVCUnattachment PVCStage = "PVCUnattachment"

	PodCreation PodStage = "PodCreation"
	PodDeletion PodStage = "PodDeletion"
)

type DurationOfStage struct {
	Min time.Duration
	Max time.Duration
	Avg time.Duration
}

type PVCMetrics struct {
	PVC     store.Entity
	Metrics map[PVCStage]time.Duration
}

type PodMetrics struct {
	Pod     store.Entity
	Metrics map[PodStage]time.Duration
}

type TestCaseMetrics struct {
	TestCase     store.TestCase
	Pods         []PodMetrics
	PVCs         []PVCMetrics
	StageMetrics map[interface{}]DurationOfStage

	EntityNumberMetrics  []store.NumberEntities
	ResourceUsageMetrics []store.ResourceUsage
}

type MetricsCollection struct {
	Run              store.TestRun
	TestCasesMetrics []TestCaseMetrics
}

type MetricsCollector struct {
	db           store.Store
	metricsCache map[string]*MetricsCollection
}

func NewMetricsCollector(db store.Store) *MetricsCollector {
	mc := make(map[string]*MetricsCollection)
	return &MetricsCollector{db, mc}
}

func (mc *MetricsCollector) Collect(runName string) (*MetricsCollection, error) {
	if mc.metricsCache != nil {
		metrics, ok := mc.metricsCache[runName]
		if ok {
			return metrics, nil
		}
	}
	if mc.db == nil {
		return nil, errors.New("database can't be nil")
	}
	runs, err := mc.db.GetTestRuns(store.Conditions{"name": runName}, "", 1)
	if err != nil {
		log.Errorf("Couldn't get test run by name %s", runName)
		return nil, err
	}
	if len(runs) == 0 {
		return nil, fmt.Errorf("test run with name %s not found", runName)
	}

	testCases, err := mc.db.GetTestCases(store.Conditions{"run_id": runs[0].ID}, "", 0)
	if err != nil {
		log.Errorf("Couldn't get test cases for test run with name %s", runName)
		return nil, err
	}

	var testCasesMetrics []TestCaseMetrics
	var bar *pb.ProgressBar
	if log.GetLevel() != log.PanicLevel {
		fmt.Println("Collecting metrics")
		bar = pb.Default.Start(len(testCases))
	}
	for i, tc := range testCases {
		if bar != nil {
			bar.Increment()
		}
		var (
			tcPodsMetrics                          []PodMetrics
			tcPVCsMetrics                          []PVCMetrics
			tcPodsStageMetrics, tcPVCSStageMetrics map[interface{}]DurationOfStage
		)

		tcPodsMetrics, tcPodsStageMetrics, err := mc.getPodsMetrics(&testCases[i])
		if err != nil {
			log.Errorf("Can't get pods with events for test case %d", tc.ID)
		}

		tcPVCsMetrics, tcPVCSStageMetrics, err = mc.getPVCsMetrics(&testCases[i])
		if err != nil {
			log.Errorf("Can't get pvcs with events for test case %d", tc.ID)
		}

		tcNumber, err := mc.db.GetNumberEntities(store.Conditions{"tc_id": tc.ID}, "", 0)
		if err != nil {
			log.Errorf("Failed to get Number Entities for test case with name %s", tc.Name)
		}

		resUsage, err := mc.db.GetResourceUsage(store.Conditions{"tc_id": tc.ID}, "", 0)
		if err != nil {
			log.Errorf("Failed to get Number Entities for test case with name %s", tc.Name)
		}

		stageMetrics := make(map[interface{}]DurationOfStage)
		mergeStageMetrics(stageMetrics, tcPodsStageMetrics)
		mergeStageMetrics(stageMetrics, tcPVCSStageMetrics)

		testCaseMetrics := TestCaseMetrics{
			TestCase:             tc,
			Pods:                 tcPodsMetrics,
			PVCs:                 tcPVCsMetrics,
			StageMetrics:         stageMetrics,
			EntityNumberMetrics:  tcNumber,
			ResourceUsageMetrics: resUsage,
		}
		testCasesMetrics = append(testCasesMetrics, testCaseMetrics)
	}
	if bar != nil {
		bar.Finish()
	}
	mc.metricsCache[runName] = &MetricsCollection{runs[0], testCasesMetrics}
	return mc.metricsCache[runName], nil
}

func (mc *MetricsCollector) getPodsMetrics(
	tc *store.TestCase) ([]PodMetrics, map[interface{}]DurationOfStage, error) {

	var podMetrics []PodMetrics
	stageMetrics := make(map[interface{}][]time.Duration)

	entitiesWithEvents, err := mc.db.GetEntitiesWithEventsByTestCaseAndEntityType(tc, store.POD)
	if err != nil {
		return podMetrics, make(map[interface{}]DurationOfStage), err
	}

	for pod, events := range entitiesWithEvents {
		timestamps := make(map[store.EventTypeEnum]time.Time)

		for _, e := range events {
			timestamps[e.Type] = e.Timestamp
		}
		metrics := make(map[PodStage]time.Duration)

		metrics[PodCreation] = timestamps[store.POD_READY].Sub(timestamps[store.POD_ADDED])
		metrics[PodDeletion] = timestamps[store.POD_DELETED].Sub(timestamps[store.POD_TERMINATING])

		stageMetrics[PodCreation] = append(stageMetrics[PodCreation], metrics[PodCreation])
		stageMetrics[PodDeletion] = append(stageMetrics[PodDeletion], metrics[PodDeletion])

		podMetrics = append(podMetrics, PodMetrics{pod, metrics})
	}
	return podMetrics, calculateMetricsOfStages(stageMetrics), nil
}

func (mc *MetricsCollector) getPVCsMetrics(
	tc *store.TestCase) ([]PVCMetrics, map[interface{}]DurationOfStage, error) {

	var pvcMetrics []PVCMetrics
	stageMetrics := make(map[interface{}][]time.Duration)

	entitiesWithEvents, err := mc.db.GetEntitiesWithEventsByTestCaseAndEntityType(tc, store.PVC)
	if err != nil {
		return pvcMetrics, make(map[interface{}]DurationOfStage), err
	}

	for pvc, events := range entitiesWithEvents {
		timestamps := make(map[store.EventTypeEnum]time.Time)

		for _, e := range events {
			timestamps[e.Type] = e.Timestamp
		}
		metrics := make(map[PVCStage]time.Duration)

		metrics[PVCBind] = timestamps[store.PVC_BOUND].Sub(timestamps[store.PVC_ADDED])
		metrics[PVCAttachment] = timestamps[store.PVC_ATTACH_ENDED].Sub(timestamps[store.PVC_ATTACH_STARTED])
		metrics[PVCCreation] = timestamps[store.PVC_ATTACH_ENDED].Sub(timestamps[store.PVC_ADDED])
		metrics[PVCDeletion] = timestamps[store.PVC_DELETING_ENDED].Sub(timestamps[store.PVC_DELETING_STARTED])
		metrics[PVCUnattachment] = timestamps[store.PVC_UNATTACH_ENDED].Sub(timestamps[store.PVC_UNATTACH_STARTED])

		stageMetrics[PVCBind] = append(stageMetrics[PVCBind], metrics[PVCBind])
		stageMetrics[PVCAttachment] = append(stageMetrics[PVCAttachment], metrics[PVCAttachment])
		stageMetrics[PVCCreation] = append(stageMetrics[PVCCreation], metrics[PVCCreation])
		stageMetrics[PVCDeletion] = append(stageMetrics[PVCDeletion], metrics[PVCDeletion])
		stageMetrics[PVCUnattachment] = append(stageMetrics[PVCUnattachment], metrics[PVCUnattachment])

		pvcMetrics = append(pvcMetrics, PVCMetrics{pvc, metrics})
	}

	return pvcMetrics, calculateMetricsOfStages(stageMetrics), nil
}

func calculateMetricsOfStages(stageMetrics map[interface{}][]time.Duration) map[interface{}]DurationOfStage {
	calculatedMetrics := make(map[interface{}]DurationOfStage)
	for k, v := range stageMetrics {
		min, max := findMaxAndMin(v)
		avg := findAvg(v)
		calculatedMetrics[k] = DurationOfStage{min, max, avg}
	}
	return calculatedMetrics
}

func findMaxAndMin(metrics []time.Duration) (min, max time.Duration) {
	min = metrics[0]
	max = metrics[0]
	for _, m := range metrics[1:] {
		if m > max {
			max = m
		}
		if m < min {
			min = m
		}
	}
	return min, max
}

func findAvg(metrics []time.Duration) time.Duration {
	var total time.Duration
	for _, v := range metrics {
		total += v
	}
	return time.Duration(float64(total) / float64(len(metrics)))
}

func mergeStageMetrics(map1, map2 map[interface{}]DurationOfStage) {
	for k, v := range map2 {
		map1[k] = v
	}
}
