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
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type CollectorTestSuit struct {
	suite.Suite
	db        store.Store
	collector *MetricsCollector
}

func (suite *CollectorTestSuit) SetupSuite() {
	suite.db = store.NewSQLiteStore("file:test.db?cache=shared&mode=memory")
	suite.collector = NewMetricsCollector(suite.db)

	testRun := &store.TestRun{
		Name:           "test run 1",
		StartTimestamp: time.Now(),
		StorageClass:   "default",
		ClusterAddress: "localhost",
	}
	_ = suite.db.SaveTestRun(testRun)

	testCase := &store.TestCase{
		Name:           "test case 1",
		StartTimestamp: time.Now(),
		RunID:          testRun.ID,
	}
	_ = suite.db.SaveTestCase(testCase)

	entityPVC1 := &store.Entity{
		Name:   "pvc1",
		K8sUID: "b0dac67e-c9a2-11e9-ad06-00505691819d",
		TcID:   testCase.ID,
		Type:   store.Pvc,
	}
	entityPVC2 := &store.Entity{
		Name:   "pvc2",
		K8sUID: "b0dac67e-c9a2-11e9-ad06-00505691765a",
		TcID:   testCase.ID,
		Type:   store.Pvc,
	}
	entityPod := &store.Entity{
		Name:   "pod1",
		K8sUID: "b0db734f-c9a2-11e9-ad06-00505691819d",
		TcID:   testCase.ID,
		Type:   store.Pod,
	}
	_ = suite.db.SaveEntities([]*store.Entity{entityPVC1, entityPVC2, entityPod})

	startTime := time.Now()

	events := []*store.Event{
		// PVC1 events
		{
			Name:      "added pvc 1",
			TcID:      testCase.ID,
			EntityID:  entityPVC1.ID,
			Type:      store.PvcAdded,
			Timestamp: startTime.Add(time.Second * 1),
		},
		{
			Name:      "bound pvc 1",
			TcID:      testCase.ID,
			EntityID:  entityPVC1.ID,
			Type:      store.PvcBound,
			Timestamp: startTime.Add(time.Second * 3),
		},
		{
			Name:      "attach started pvc 1",
			TcID:      testCase.ID,
			EntityID:  entityPVC1.ID,
			Type:      store.PvcAttachStarted,
			Timestamp: startTime.Add(time.Second * 4),
		},
		{
			Name:      "attach ended pvc 1",
			TcID:      testCase.ID,
			EntityID:  entityPVC1.ID,
			Type:      store.PvcAttachEnded,
			Timestamp: startTime.Add(time.Second * 6),
		},
		{
			Name:      "unattach started pvc 1",
			TcID:      testCase.ID,
			EntityID:  entityPVC1.ID,
			Type:      store.PvcUnattachStarted,
			Timestamp: startTime.Add(time.Second * 21),
		},
		{
			Name:      "unattach ended pvc 1",
			TcID:      testCase.ID,
			EntityID:  entityPVC1.ID,
			Type:      store.PvcUnattachEnded,
			Timestamp: startTime.Add(time.Second * 24),
		},
		{
			Name:      "deleting started pvc 1",
			TcID:      testCase.ID,
			EntityID:  entityPVC1.ID,
			Type:      store.PvcDeletingStarted,
			Timestamp: startTime.Add(time.Second * 20),
		},
		{
			Name:      "deleting ended pvc 1",
			TcID:      testCase.ID,
			EntityID:  entityPVC1.ID,
			Type:      store.PvcDeletingEnded,
			Timestamp: startTime.Add(time.Second * 24),
		},
		// PVC2 events
		{
			Name:      "added pvc 2",
			TcID:      testCase.ID,
			EntityID:  entityPVC2.ID,
			Type:      store.PvcAdded,
			Timestamp: startTime.Add(time.Second * 2),
		},
		{
			Name:      "bound pvc 2",
			TcID:      testCase.ID,
			EntityID:  entityPVC2.ID,
			Type:      store.PvcBound,
			Timestamp: startTime.Add(time.Second * 3),
		},
		{
			Name:      "attach started pvc 2",
			TcID:      testCase.ID,
			EntityID:  entityPVC2.ID,
			Type:      store.PvcAttachStarted,
			Timestamp: startTime.Add(time.Second * 5),
		},
		{
			Name:      "attach ended pvc 2",
			TcID:      testCase.ID,
			EntityID:  entityPVC2.ID,
			Type:      store.PvcAttachEnded,
			Timestamp: startTime.Add(time.Second * 7),
		},
		{
			Name:      "unattach started pvc 2",
			TcID:      testCase.ID,
			EntityID:  entityPVC2.ID,
			Type:      store.PvcUnattachStarted,
			Timestamp: startTime.Add(time.Second * 22),
		},
		{
			Name:      "unattach ended pvc 2",
			TcID:      testCase.ID,
			EntityID:  entityPVC2.ID,
			Type:      store.PvcUnattachEnded,
			Timestamp: startTime.Add(time.Second * 23),
		},
		{
			Name:      "deleting started pvc 2",
			TcID:      testCase.ID,
			EntityID:  entityPVC2.ID,
			Type:      store.PvcDeletingStarted,
			Timestamp: startTime.Add(time.Second * 19),
		},
		{
			Name:      "deleting ended pvc 2",
			TcID:      testCase.ID,
			EntityID:  entityPVC2.ID,
			Type:      store.PvcDeletingEnded,
			Timestamp: startTime.Add(time.Second * 24),
		},
		// Pod events
		{
			Name:      "added pod",
			TcID:      testCase.ID,
			EntityID:  entityPod.ID,
			Type:      store.PodAdded,
			Timestamp: startTime,
		},
		{
			Name:      "ready pod",
			TcID:      testCase.ID,
			EntityID:  entityPod.ID,
			Type:      store.PodReady,
			Timestamp: startTime.Add(time.Second * 7),
		},
		{
			Name:      "terminating pod",
			TcID:      testCase.ID,
			EntityID:  entityPod.ID,
			Type:      store.PodTerminating,
			Timestamp: startTime.Add(time.Second * 19),
		},
		{
			Name:      "deleted pod",
			TcID:      testCase.ID,
			EntityID:  entityPod.ID,
			Type:      store.PodDeleted,
			Timestamp: startTime.Add(time.Second * 25),
		},
	}
	_ = suite.db.SaveEvents(events)
}

func (suite *CollectorTestSuit) TearDownSuite() {
	err := suite.db.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func (suite *CollectorTestSuit) TestCollectMetrics() {
	mc, err := suite.collector.Collect("test run 1")
	suite.Nil(err)
	suite.Equal(len(mc.TestCasesMetrics), 1)

	tc := mc.TestCasesMetrics[0]
	suite.Equal(tc.TestCase.Name, "test case 1")

	suite.Equal(tc.StageMetrics[PVCBind].Max.Seconds(), float64(2))
	suite.Equal(tc.StageMetrics[PVCBind].Min.Seconds(), float64(1))
	suite.Equal(tc.StageMetrics[PVCBind].Avg.Seconds(), float64(1.5))

	suite.Equal(tc.StageMetrics[PVCDeletion].Max.Seconds(), float64(5))
	suite.Equal(tc.StageMetrics[PVCDeletion].Min.Seconds(), float64(4))
	suite.Equal(tc.StageMetrics[PVCDeletion].Avg.Seconds(), float64(4.5))

	suite.Equal(tc.StageMetrics[PVCUnattachment].Max.Seconds(), float64(3))
	suite.Equal(tc.StageMetrics[PVCUnattachment].Min.Seconds(), float64(1))
	suite.Equal(tc.StageMetrics[PVCUnattachment].Avg.Seconds(), float64(2))

	suite.Equal(tc.StageMetrics[PodCreation].Max.Seconds(), float64(7))
	suite.Equal(tc.StageMetrics[PodCreation].Min.Seconds(), float64(7))
	suite.Equal(tc.StageMetrics[PodCreation].Avg.Seconds(), float64(7))
}

func TestCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(CollectorTestSuit))
}
