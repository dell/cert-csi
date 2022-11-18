package store

import (
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Conditions map[string]interface{}

// Store is a generic interface for stores
type Store interface {
	SaveTestRun(tr *TestRun) error
	GetTestRuns(whereConditions Conditions, orderBy string, limit int) ([]TestRun, error)
	SaveEvents(events []*Event) error
	GetEvents(whereConditions Conditions, orderBy string, limit int) ([]Event, error)
	SaveTestCase(ts *TestCase) error
	GetTestCases(whereConditions Conditions, orderBy string, limit int) ([]TestCase, error)
	SuccessfulTestCase(ts *TestCase, endTimestamp time.Time) error
	FailedTestCase(ts *TestCase, endTimestamp time.Time, errMsg string) error
	SaveEntities(entities []*Entity) error
	GetEntities(whereConditions Conditions, orderBy string, limit int) ([]Entity, error)
	GetEntitiesWithEventsByTestCaseAndEntityType(tc *TestCase, eType EntityTypeEnum) (map[Entity][]Event, error)
	SaveNumberEntities(nEntities []*NumberEntities) error
	GetNumberEntities(whereConditions Conditions, orderBy string, limit int) ([]NumberEntities, error)
	SaveResourceUsage(resUsages []*ResourceUsage) error
	GetResourceUsage(whereConditions Conditions, orderBy string, limit int) ([]ResourceUsage, error)
	CreateEntitiesRelation(entity1, entity2 Entity) error
	GetEntityRelations(event Entity) ([]Entity, error)
	Close() error
}
