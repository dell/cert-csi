/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package store

import (
	"time"

	// go-sqlite3 import
	_ "github.com/mattn/go-sqlite3"
)

// Conditions interface
type Conditions map[string]interface{}

// Store is a generic interface for stores
//
//go:generate mockgen -destination=mocks/storeinterface.go -package=mocks github.com/dell/cert-csi/pkg/store Store
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
