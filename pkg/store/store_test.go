package store

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreTestSuite struct {
	suite.Suite
	Stores map[string]Store
}

func (suite *StoreTestSuite) SetupSuite() {
	suite.Stores = make(map[string]Store)
	suite.Stores["SQLite"] = NewSQLiteStore("file:test.db?cache=shared&mode=memory")
}

func (suite *StoreTestSuite) TearDownSuite() {
	err := suite.Stores["SQLite"].Close()
	if err != nil {
		log.Fatal(err)
	}
}

func (suite *StoreTestSuite) TestAllStores() {
	for key, store := range suite.Stores {

		sourceTestRun := &TestRun{
			Name:           "test run 1",
			StartTimestamp: time.Now(),
			StorageClass:   "default",
			ClusterAddress: "localhost",
		}
		err := store.SaveTestRun(sourceTestRun)
		suite.NoError(err)

		sourceTestCase := &TestCase{
			Name:           "test case",
			Parameters:     "{size: 3GI}",
			StartTimestamp: time.Now(),
			RunID:          sourceTestRun.ID,
		}
		err = store.SaveTestCase(sourceTestCase)
		suite.NoError(err)

		sourceEntityPVC := &Entity{
			Name:   "pvc1",
			K8sUid: "b0dac67e-c9a2-11e9-ad06-00505691819d",
			TcID:   sourceTestCase.ID,
			Type:   PVC,
		}
		sourceEntityPod := &Entity{
			Name:   "pod1",
			K8sUid: "b0db734f-c9a2-11e9-ad06-00505691819d",
			TcID:   sourceTestCase.ID,
			Type:   POD,
		}
		err = store.SaveEntities([]*Entity{sourceEntityPVC, sourceEntityPod})
		suite.NoError(err)

		sourceEvents := []*Event{
			{
				Name:      "test event 1",
				TcID:      sourceTestCase.ID,
				EntityID:  sourceEntityPVC.ID,
				Type:      PVC_ADDED,
				Timestamp: time.Now(),
			},
			{
				Name:      "test event 2",
				TcID:      sourceTestCase.ID,
				EntityID:  sourceEntityPVC.ID,
				Type:      PVC_BOUND,
				Timestamp: time.Now(),
			},
			{
				Name:      "test event 3",
				TcID:      sourceTestCase.ID,
				EntityID:  sourceEntityPod.ID,
				Type:      POD_ADDED,
				Timestamp: time.Now(),
			},
		}

		err = store.SaveEvents(sourceEvents)
		suite.Nil(err, fmt.Sprintf("able to create a new events %v using %s store", sourceEvents, store))

		err = store.SaveEntities([]*Entity{
			{
				Name:   "pod2",
				K8sUid: "b0db734f-c9a2-11e9-ad06-00505691819d",
				TcID:   sourceTestCase.ID,
				Type:   POD,
			},
		})
		suite.EqualError(err, "UNIQUE constraint failed: entities.k8s_uid")

		events, err := store.GetEvents(Conditions{"name": "test event 1"}, "", 0)
		suite.Nil(err, "able to get event by name")
		suite.Equal(len(events), 1, fmt.Sprintf("able to get event by name using %s store", key))
		suite.Equal(events[0].Name, "test event 1", fmt.Sprintf("able to get event name using %s store", key))

		events, err = store.GetEvents(Conditions{}, "name DESC", 1)
		suite.Nil(err, "able to get event in descending order")
		suite.Equal(len(events), 1, fmt.Sprintf("able to get events in descending order using %s store", key))
		suite.Equal(events[0].Name, "test event 3", fmt.Sprintf("able to get event name using %s store", key))

		events, err = store.GetEvents(Conditions{"tc_id": sourceTestCase.ID}, "", 0)
		suite.Nil(err, "able to get events by test case id")
		suite.Equal(len(events), 3, fmt.Sprintf("able to get events by test case id using %s store", key))

		events, err = store.GetEvents(Conditions{"entity_id": sourceEntityPod.ID}, "", 0)
		suite.Nil(err, "able to get events by entity id")
		suite.Equal(len(events), 1, fmt.Sprintf("able to get events by entity id using %s store", key))

		tcs, err := store.GetTestCases(Conditions{"name": "test case"}, "", 0)
		suite.Nil(err, "able to get test case by uid")
		suite.Equal(len(tcs), 1, fmt.Sprintf("able to get test case by uid using %s store", key))
		tc := tcs[0]
		suite.Equal(tc.Name, "test case", fmt.Sprintf("able to get test name case using %s store", key))

		suite.False(tc.Success, "success status for test case must be false")
		err = store.SuccessfulTestCase(&tc, time.Now())
		suite.Nil(err, "able to set success status to test case")
		suite.True(tc.Success, "success status for test case must be true")

		tcs, err = store.GetTestCases(Conditions{"success": true}, "", 0)
		suite.Nil(err, "able to get test case by uid")
		suite.Equal(len(tcs), 1, fmt.Sprintf("able to get test case by uid using %s store", key))

		err = store.CreateEntitiesRelation(*sourceEntityPod, *sourceEntityPVC)
		suite.Nil(err, "able to create entities relations %s store", store)

		entities, err := store.GetEntityRelations(*sourceEntityPod)
		suite.Nil(err, "able to get entities relations %s store", store)
		suite.Equal(len(entities), 1, fmt.Sprintf("able to get entity relations using %s store", key))
		suite.Equal(entities[0].Name, "pvc1", fmt.Sprintf("able to get entity name from relation using %s store", key))

		entities, err = store.GetEntities(Conditions{"type": PVC}, "", 0)
		suite.Equal(len(entities), 1)

		nEntities := []*NumberEntities{
			{TcID: sourceTestCase.ID,
				Timestamp:       time.Now(),
				PodsCreating:    2,
				PodsReady:       0,
				PodsTerminating: 0,
				PvcCreating:     2,
				PvcBound:        1,
				PvcTerminating:  0,
			},
			{TcID: sourceTestCase.ID,
				Timestamp:       time.Now().Add(time.Second * 5),
				PodsCreating:    1,
				PodsReady:       2,
				PodsTerminating: 0,
				PvcCreating:     1,
				PvcBound:        2,
				PvcTerminating:  0,
			},
		}
		err = store.SaveNumberEntities(nEntities)
		suite.NoError(err)

		ne, err := store.GetNumberEntities(Conditions{}, "timestamp DESC", 1)
		suite.Nil(err, "able to get number of entities %s store", store)
		suite.Equal(len(events), 1)
		suite.Equal(ne[0].PodsReady, 2)

		podWithEvents, err := store.GetEntitiesWithEventsByTestCaseAndEntityType(&tc, POD)
		suite.NoError(err)
		suite.Equal(len(podWithEvents), 1)
		suite.Equal(podWithEvents[*sourceEntityPod][0].Name, "test event 3")
	}
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}
