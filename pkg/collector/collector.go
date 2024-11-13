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

package collector

import (
	"errors"
	"fmt"
	"time"

	"github.com/dell/cert-csi/pkg/store"

	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
)

// Stage step
type Stage string

// PVCStage step
type PVCStage Stage

// PodStage step
type PodStage Stage

const (
	// PVCBind stage
	PVCBind PVCStage = "PVCBind"
	// PVCAttachment stage
	PVCAttachment PVCStage = "PVCAttachment"
	// PVCCreation stage
	PVCCreation PVCStage = "PVCCreation"
	// PVCDeletion stage
	PVCDeletion PVCStage = "PVCDeletion"
	// PVCUnattachment stage
	PVCUnattachment PVCStage = "PVCUnattachment"

	// PodCreation stage
	PodCreation PodStage = "PodCreation"
	// PodDeletion stage
	PodDeletion PodStage = "PodDeletion"
)

// DurationOfStage represents staging time
type DurationOfStage struct {
	Min time.Duration
	Max time.Duration
	Avg time.Duration
}

// PVCMetrics contains PVC and corresponding metrics
type PVCMetrics struct {
	PVC     store.Entity
	Metrics map[PVCStage]time.Duration
}

// PodMetrics contains Pod and corresponding metrics
type PodMetrics struct {
	Pod     store.Entity
	Metrics map[PodStage]time.Duration
}

// TestCaseMetrics contains metrics for each testcase
type TestCaseMetrics struct {
	TestCase     store.TestCase
	Pods         []PodMetrics
	PVCs         []PVCMetrics
	StageMetrics map[interface{}]DurationOfStage

	EntityNumberMetrics  []store.NumberEntities
	ResourceUsageMetrics []store.ResourceUsage
}

// MetricsCollection contains collection of TestCaseMetrics
type MetricsCollection struct {
	Run              store.TestRun
	TestCasesMetrics []TestCaseMetrics
}

// MetricsCollector contains db store and metrics collection
type MetricsCollector struct {
	db           store.Store
	metricsCache map[string]*MetricsCollection
}

// NewMetricsCollector creates a MetricsCollector
func NewMetricsCollector(db store.Store) *MetricsCollector {
	mc := make(map[string]*MetricsCollection)
	return &MetricsCollector{db, mc}
}

// Collect consolidates the metrics and returns MetricsCollection
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
	tc *store.TestCase,
) ([]PodMetrics, map[interface{}]DurationOfStage, error) {
	var podMetrics []PodMetrics
	stageMetrics := make(map[interface{}][]time.Duration)

	entitiesWithEvents, err := mc.db.GetEntitiesWithEventsByTestCaseAndEntityType(tc, store.Pod)
	if err != nil {
		return podMetrics, make(map[interface{}]DurationOfStage), err
	}

	for pod, events := range entitiesWithEvents {
		timestamps := make(map[store.EventTypeEnum]time.Time)

		for _, e := range events {
			timestamps[e.Type] = e.Timestamp
		}
		metrics := make(map[PodStage]time.Duration)

		metrics[PodCreation] = timestamps[store.PodReady].Sub(timestamps[store.PodAdded])
		metrics[PodDeletion] = timestamps[store.PodDeleted].Sub(timestamps[store.PodTerminating])

		stageMetrics[PodCreation] = append(stageMetrics[PodCreation], metrics[PodCreation])
		stageMetrics[PodDeletion] = append(stageMetrics[PodDeletion], metrics[PodDeletion])

		podMetrics = append(podMetrics, PodMetrics{pod, metrics})
	}
	return podMetrics, calculateMetricsOfStages(stageMetrics), nil
}

func (mc *MetricsCollector) getPVCsMetrics(
	tc *store.TestCase,
) ([]PVCMetrics, map[interface{}]DurationOfStage, error) {
	var pvcMetrics []PVCMetrics
	stageMetrics := make(map[interface{}][]time.Duration)

	entitiesWithEvents, err := mc.db.GetEntitiesWithEventsByTestCaseAndEntityType(tc, store.Pvc)
	if err != nil {
		return pvcMetrics, make(map[interface{}]DurationOfStage), err
	}

	for pvc, events := range entitiesWithEvents {
		timestamps := make(map[store.EventTypeEnum]time.Time)

		for _, e := range events {
			timestamps[e.Type] = e.Timestamp
		}
		metrics := make(map[PVCStage]time.Duration)

		metrics[PVCBind] = timestamps[store.PvcBound].Sub(timestamps[store.PvcAdded])
		metrics[PVCAttachment] = timestamps[store.PvcAttachEnded].Sub(timestamps[store.PvcAttachStarted])
		metrics[PVCCreation] = timestamps[store.PvcAttachEnded].Sub(timestamps[store.PvcAdded])
		metrics[PVCDeletion] = timestamps[store.PvcDeletingEnded].Sub(timestamps[store.PvcDeletingStarted])
		metrics[PVCUnattachment] = timestamps[store.PvcUnattachEnded].Sub(timestamps[store.PvcUnattachStarted])

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
		minMetric, maxMetric := findMaxAndMin(v)
		avg := findAvg(v)
		calculatedMetrics[k] = DurationOfStage{minMetric, maxMetric, avg}
	}
	return calculatedMetrics
}

func findMaxAndMin(metrics []time.Duration) (minMetric, maxMetric time.Duration) {
	minMetric = metrics[0]
	maxMetric = metrics[0]
	for _, m := range metrics[1:] {
		if m > maxMetric {
			maxMetric = m
		}
		if m < minMetric {
			minMetric = m
		}
	}
	return minMetric, maxMetric
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
