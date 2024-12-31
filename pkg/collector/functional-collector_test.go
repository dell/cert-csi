/*
 *
 * Copyright © 2024 Dell Inc. or its subsidiaries. All Rights Reserved.
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

type FunctionalMetricsCollectorTestSuite struct {
	suite.Suite
	db        store.Store
	collector *MetricsCollector
}

func (suite *FunctionalMetricsCollectorTestSuite) SetupSuite() {
	suite.db = store.NewSQLiteStore("file:test.db?cache=shared&mode=memory")
	suite.collector = NewMetricsCollector(suite.db)

	testRun := &store.TestRun{
		Name:           "Test Run",
		StartTimestamp: time.Now(),
		StorageClass:   "default",
		ClusterAddress: "localhost",
	}
	_ = suite.db.SaveTestRun(testRun)

	testCase1 := &store.TestCase{
		Name:           "test case 1",
		StartTimestamp: time.Now(),
		RunID:          testRun.ID,
	}
	testCase2 := &store.TestCase{
		Name:           "test case 2",
		StartTimestamp: time.Now(),
		RunID:          testRun.ID,
	}
	_ = suite.db.SaveTestCase(testCase1)
	_ = suite.db.SaveTestCase(testCase2)
}

func (suite *FunctionalMetricsCollectorTestSuite) TearDownSuite() {
	err := suite.db.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func (suite *FunctionalMetricsCollectorTestSuite) TestCollectFunctionalMetrics() {
	mc, err := suite.collector.CollectFunctionalMetrics()
	suite.Nil(err)
	suite.Equal(len(mc.TestCasesMetrics), 2)

	tc1 := mc.TestCasesMetrics[0]
	suite.Equal(tc1.TestCase.Name, "test case 1")

	tc2 := mc.TestCasesMetrics[1]
	suite.Equal(tc2.TestCase.Name, "test case 2")

	// Ensure metrics are cached
	mcCached, err := suite.collector.CollectFunctionalMetrics()
	suite.Nil(err)
	suite.Equal(mc, mcCached)
}

func TestFunctionalMetricsCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(FunctionalMetricsCollectorTestSuite))
}
