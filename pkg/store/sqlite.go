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
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// StorageClassDB represents storage class db
type StorageClassDB struct {
	StorageClass string
	DB           Store
	TestRun      TestRun
}

// SQLiteStore implements the Store interface, used for storing objects to sqlite database
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLiteStore
func NewSQLiteStore(dsn string) *SQLiteStore {
	store := &SQLiteStore{}

	var err error

	store.db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		panic(err)
	}
	// Disable connections pool
	store.db.SetMaxOpenConns(1)

	err = store.createTables()
	if err != nil {
		panic(err)
	}
	logrus.Info("Created tables")
	return store
}

func (ss *SQLiteStore) createTables() error {
	_, err := ss.db.Exec(`
	CREATE TABLE IF NOT EXISTS test_runs(
		id INTEGER PRIMARY KEY,
		name VARCHAR(50) NOT NULL UNIQUE,
		longevity BOOLEAN DEFAULT false,
		start_timestamp DATETIME,
		storage_class VARCHAR(50) NOT NULL,
		cluster_address VARCHAR(50) NOT NULL)
		`)
	if err != nil {
		return err
	}
	logrus.Info("Created test_runs table")

	_, err = ss.db.Exec(`
	CREATE TABLE IF NOT EXISTS test_cases(
		id INTEGER PRIMARY KEY,
		name VARCHAR(30) NOT NULL,
 		parameters VARCHAR(100),
		start_timestamp DATETIME,
		end_timestamp DATETIME,
		success BOOLEAN,
 		error_msg VARCHAR(250),
		run_id INTEGER NOT NULL,
		FOREIGN KEY(run_id) REFERENCES test_runs(id))
		`)
	if err != nil {
		return err
	}
	logrus.Info("Created test_cases table")

	_, err = ss.db.Exec(`
	CREATE TABLE IF NOT EXISTS events(
		id INTEGER PRIMARY KEY,
		name VARCHAR(30) NOT NULL,
		tc_id INTEGER NOT NULL,
		entity_id INTEGER NOT NULL,
		type VARCHAR(50) NOT NULL,
		timestamp DATETIME NOT NULL,
		FOREIGN KEY(tc_id) REFERENCES test_cases(id),
		FOREIGN KEY(entity_id) REFERENCES entities(id))
		`)
	if err != nil {
		return err
	}
	logrus.Info("Created events table")

	_, err = ss.db.Exec(`
	CREATE TABLE IF NOT EXISTS entities(
		id INTEGER PRIMARY KEY,
		name VARCHAR(50) NOT NULL,
		k8s_uid VARCHAR(40) NOT NULL UNIQUE,
		tc_id INTEGER NOT NULL,
		type VARCHAR(15) NOT NULL DEFAULT 'UNKNOWN',
		FOREIGN KEY(tc_id) REFERENCES test_cases(id))
		`)
	if err != nil {
		return err
	}
	logrus.Info("Created entities table")

	_, err = ss.db.Exec(`
	CREATE TABLE IF NOT EXISTS number_entities(
		id INTEGER PRIMARY KEY,
		tc_id INTEGER NOT NULL,
		timestamp DATETIME NOT NULL,
		pods_creating INTEGER,
		pods_ready INTEGER,
		pods_terminating INTEGER,
		pvc_creating INTEGER,
		pvc_bound INTEGER,
		pvc_terminating INTEGER,
		FOREIGN KEY(tc_id) REFERENCES test_cases(id))
		`)
	if err != nil {
		return err
	}
	logrus.Info("Created number_entities table")

	_, err = ss.db.Exec(`
	CREATE TABLE IF NOT EXISTS resource_usage(
		id INTEGER PRIMARY KEY,
		tc_id INTEGER NOT NULL,
		timestamp DATETIME NOT NULL,
		pod_name VARCHAR,
		container_name VARCHAR,
		cpu_value INTEGER,
		mem_value INTEGER,
		FOREIGN KEY(tc_id) REFERENCES test_cases(id))
		`)
	if err != nil {
		return err
	}
	logrus.Info("Created resource_usage table")

	_, err = ss.db.Exec(`
	CREATE TABLE IF NOT EXISTS entities_relations(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entity_id1 INTEGER NOT NULL,
		entity_id2 INTEGER NOT NULL,
		FOREIGN KEY(entity_id1) REFERENCES entities(id),
		FOREIGN KEY(entity_id2) REFERENCES entities(id))
		`)
	if err != nil {
		return err
	}
	logrus.Info("Created entities_relations table")

	return nil
}

// SaveTestRun saves test run result in db
func (ss *SQLiteStore) SaveTestRun(tr *TestRun) error {
	result, err := ss.db.Exec(`
	INSERT INTO test_runs(
		name, start_timestamp, storage_class, cluster_address
	)VALUES ($1, $2, $3, $4)`,
		tr.Name, tr.StartTimestamp, tr.StorageClass, tr.ClusterAddress)
	if err != nil {
		return err
	}
	tr.ID, err = result.LastInsertId()
	if err != nil {
		logrus.Error("Couldn't get last insert ID")
	}
	return nil
}

// GetTestRuns queries test run information from db
func (ss *SQLiteStore) GetTestRuns(whereConditions Conditions, orderBy string, limit int) ([]TestRun, error) {
	sqlStmt := ss.prepareSQLSelectStmt(whereConditions, orderBy, limit, "test_runs")
	rows, err := ss.db.Query(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testRuns []TestRun

	for rows.Next() {
		tr := TestRun{}
		if err = rows.Scan(
			&tr.ID, &tr.Name, &tr.Longevity, &tr.StartTimestamp, &tr.StorageClass, &tr.ClusterAddress); err == nil {
			testRuns = append(testRuns, tr)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return testRuns, nil
}

// SaveEvents saves events into db
func (ss *SQLiteStore) SaveEvents(events []*Event) error {
	sqlAddEvent := `
	INSERT INTO events(
		name, tc_id, entity_id, type, timestamp
	) VALUES (?, ?, ?, ?, ?)
	`

	stmt, err := ss.db.Prepare(sqlAddEvent)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		result, err := stmt.Exec(e.Name, e.TcID, e.EntityID, e.Type, e.Timestamp)
		if err != nil {
			return err
		}
		e.ID, err = result.LastInsertId()
		if err != nil {
			logrus.Error("Couldn't get last insert ID")
		}
	}

	return nil
}

func (ss *SQLiteStore) prepareSQLSelectStmt(
	whereConditions Conditions,
	orderBy string,
	limit int,
	tableName string,
) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("SELECT * FROM %s", tableName)) // #nosec

	if len(whereConditions) > 0 {
		b.WriteString(" WHERE") // #nosec
		for k, v := range whereConditions {
			switch v.(type) {
			case string, EntityTypeEnum, EventTypeEnum:
				b.WriteString(fmt.Sprintf(" %s='%s' AND", k, v)) // #nosec
			case int64:
				b.WriteString(fmt.Sprintf(" %s=%d AND", k, v)) // #nosec
			case bool:
				b.WriteString(fmt.Sprintf(" %s=%t AND", k, v)) // #nosec
			default:
				continue
			}
		}
		b.WriteString(" 1=1") // #nosec
	}

	if orderBy != "" {
		b.WriteString(fmt.Sprintf(" ORDER BY %s", orderBy)) // #nosec
	}

	if limit > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", limit)) // #nosec
	}

	return b.String()
}

// GetEvents queries events from db
func (ss *SQLiteStore) GetEvents(whereConditions Conditions, orderBy string, limit int) ([]Event, error) {
	sqlStmt := ss.prepareSQLSelectStmt(whereConditions, orderBy, limit, "events")
	rows, err := ss.db.Query(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event

	for rows.Next() {
		event := Event{}
		if err = rows.Scan(
			&event.ID, &event.Name, &event.TcID, &event.EntityID, &event.Type, &event.Timestamp); err == nil {
			events = append(events, event)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// GetEntitiesWithEventsByTestCaseAndEntityType queries entities with events and
// returns the results sorted by testcase and entity
func (ss *SQLiteStore) GetEntitiesWithEventsByTestCaseAndEntityType(
	tc *TestCase,
	eType EntityTypeEnum,
) (map[Entity][]Event, error) {
	sqlStmt := `
		SELECT et.*, ev.*
		FROM entities et JOIN events ev
			ON et.id = ev.entity_id
		WHERE et.type = ? AND et.tc_id= ?
		ORDER BY et.id`

	stmt, err := ss.db.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(eType, tc.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ewe := make(map[Entity][]Event)
	for rows.Next() {
		en := Entity{}
		ev := Event{}
		if err = rows.Scan(
			&en.ID, &en.Name, &en.K8sUID, &en.TcID, &en.Type,
			&ev.ID, &ev.Name, &ev.TcID, &ev.EntityID, &ev.Type, &ev.Timestamp); err == nil {
			if events, ok := ewe[en]; ok {
				ewe[en] = append(events, ev)
			} else {
				ewe[en] = []Event{ev}
			}
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ewe, nil
}

// SaveTestCase saves testcases in db
func (ss *SQLiteStore) SaveTestCase(ts *TestCase) error {
	sqlStmt := `
	INSERT INTO test_cases(name, parameters, start_timestamp, end_timestamp, success, error_msg, run_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	stmt, err := ss.db.Prepare(sqlStmt)
	if err != nil {
		return err
	}
	defer stmt.Close()

	result, err := stmt.Exec(ts.Name, ts.Parameters, ts.StartTimestamp, ts.EndTimestamp, ts.Success, ts.ErrorMessage, ts.RunID)
	if err != nil {
		return err
	}
	ts.ID, err = result.LastInsertId()
	if err != nil {
		logrus.Error("Couldn't get last insert ID")
	}
	return nil
}

// GetTestCases queries testcases from db
func (ss *SQLiteStore) GetTestCases(whereConditions Conditions, orderBy string, limit int) ([]TestCase, error) {
	sqlStmt := ss.prepareSQLSelectStmt(whereConditions, orderBy, limit, "test_cases")
	rows, err := ss.db.Query(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []TestCase

	for rows.Next() {
		tc := TestCase{}
		if err = rows.Scan(
			&tc.ID, &tc.Name, &tc.Parameters, &tc.StartTimestamp, &tc.EndTimestamp, &tc.Success, &tc.ErrorMessage, &tc.RunID); err == nil {
			testCases = append(testCases, tc)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return testCases, nil
}

func (ss *SQLiteStore) updateStatusTestCase(ts *TestCase) error {
	_, err := ss.db.Exec(
		"UPDATE test_cases SET success=?, end_timestamp=?, error_msg=? WHERE id=?",
		ts.Success, ts.EndTimestamp, ts.ErrorMessage, ts.ID)
	if err != nil {
		return err
	}
	return nil
}

// SuccessfulTestCase updates testcase status as success
func (ss *SQLiteStore) SuccessfulTestCase(ts *TestCase, endTimestamp time.Time) error {
	ts.Success = true
	ts.EndTimestamp = endTimestamp
	if err := ss.updateStatusTestCase(ts); err != nil {
		return err
	}
	return nil
}

// FailedTestCase updates testcase status as failure
func (ss *SQLiteStore) FailedTestCase(ts *TestCase, endTimestamp time.Time, errMsg string) error {
	ts.Success = false
	ts.EndTimestamp = endTimestamp
	ts.ErrorMessage = errMsg
	if err := ss.updateStatusTestCase(ts); err != nil {
		return err
	}
	return nil
}

// SaveEntities saves entities in db
func (ss *SQLiteStore) SaveEntities(entities []*Entity) error {
	sqlAddEvent := `
	INSERT INTO entities(name, k8s_uid, tc_id, type
	) VALUES (?, ?, ?, ?)
	`

	stmt, err := ss.db.Prepare(sqlAddEvent)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entities {
		result, err := stmt.Exec(e.Name, e.K8sUID, e.TcID, e.Type)
		if err != nil {
			return err
		}
		e.ID, err = result.LastInsertId()
		if err != nil {
			logrus.Error("Couldn't get last insert ID")
		}
	}

	return nil
}

// GetEntities queries entities from db
func (ss *SQLiteStore) GetEntities(whereConditions Conditions, orderBy string, limit int) ([]Entity, error) {
	sqlStmt := ss.prepareSQLSelectStmt(whereConditions, orderBy, limit, "entities")
	rows, err := ss.db.Query(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity

	for rows.Next() {
		entity := Entity{}
		if err = rows.Scan(
			&entity.ID, &entity.Name, &entity.K8sUID, &entity.TcID, &entity.Type); err == nil {
			entities = append(entities, entity)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return entities, nil
}

// SaveNumberEntities saves NumberEntities in db
func (ss *SQLiteStore) SaveNumberEntities(nEntities []*NumberEntities) error {
	sqlAdd := `
	INSERT INTO number_entities(
		tc_id,
		timestamp,
		pods_creating,
		pods_ready,
		pods_terminating,
		pvc_creating,
		pvc_bound,
		pvc_terminating
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := ss.db.Prepare(sqlAdd)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range nEntities {
		result, err := stmt.Exec(
			e.TcID,
			e.Timestamp,
			e.PodsCreating,
			e.PodsReady,
			e.PodsTerminating,
			e.PvcCreating,
			e.PvcBound,
			e.PvcTerminating)
		if err != nil {
			return err
		}
		e.ID, err = result.LastInsertId()
		if err != nil {
			logrus.Error("Couldn't get last insert ID")
		}
	}

	return nil
}

// GetNumberEntities queries NumberEntities from db
func (ss *SQLiteStore) GetNumberEntities(
	whereConditions Conditions,
	orderBy string,
	limit int,
) ([]NumberEntities, error) {
	sqlStmt := ss.prepareSQLSelectStmt(whereConditions, orderBy, limit, "number_entities")
	rows, err := ss.db.Query(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nEntities []NumberEntities

	for rows.Next() {
		e := NumberEntities{}
		if err = rows.Scan(
			&e.ID,
			&e.TcID,
			&e.Timestamp,
			&e.PodsCreating,
			&e.PodsReady,
			&e.PodsTerminating,
			&e.PvcCreating,
			&e.PvcBound,
			&e.PvcTerminating); err == nil {
			nEntities = append(nEntities, e)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return nEntities, nil
}

// SaveResourceUsage saves resource usage in db
func (ss *SQLiteStore) SaveResourceUsage(resUsages []*ResourceUsage) error {
	sqlAdd := `
	INSERT INTO resource_usage(
		tc_id,
		timestamp,
		pod_name,
		container_name,
		cpu_value,
		mem_value
	) VALUES (?, ?, ?, ?, ?, ?)
	`

	stmt, err := ss.db.Prepare(sqlAdd)
	if err != nil {
		logrus.Errorf("Can't prepare statement")
		return err
	}
	defer stmt.Close()

	for _, e := range resUsages {
		result, err := stmt.Exec(
			e.TcID,
			e.Timestamp,
			e.PodName,
			e.ContainerName,
			e.CPU,
			e.Mem,
		)
		if err != nil {
			logrus.Errorf("Can't execute statement")
			return err
		}
		e.ID, err = result.LastInsertId()
		if err != nil {
			logrus.Error("Couldn't get last insert ID")
		}
	}

	return nil
}

// GetResourceUsage queries resource usage from db
func (ss *SQLiteStore) GetResourceUsage(
	whereConditions Conditions,
	orderBy string,
	limit int,
) ([]ResourceUsage, error) {
	sqlStmt := ss.prepareSQLSelectStmt(whereConditions, orderBy, limit, "resource_usage")
	rows, err := ss.db.Query(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resUsage []ResourceUsage

	for rows.Next() {
		e := ResourceUsage{}
		if err = rows.Scan(
			&e.ID,
			&e.TcID,
			&e.Timestamp,
			&e.PodName,
			&e.ContainerName,
			&e.CPU,
			&e.Mem); err == nil {
			resUsage = append(resUsage, e)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return resUsage, nil
}

// CreateEntitiesRelation adds EntitiesRelation to db
func (ss *SQLiteStore) CreateEntitiesRelation(entity1, entity2 Entity) error {
	_, err := ss.db.Exec(
		"INSERT INTO entities_relations (entity_id1, entity_id2) VALUES ($1, $2)",
		entity1.ID, entity2.ID)
	if err != nil {
		return err
	}
	return nil
}

// GetEntityRelations queries EntitiesRelation from db
func (ss *SQLiteStore) GetEntityRelations(entity Entity) ([]Entity, error) {
	sqlStmt := `
	SELECT e.*
	FROM entities e
	  INNER JOIN entities_relations er ON e.id = er.entity_id2
	WHERE er.entity_id1 = ?
	`

	stmt, err := ss.db.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(entity.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity

	for rows.Next() {
		entity := Entity{}
		if err = rows.Scan(&entity.ID, &entity.Name, &entity.K8sUID, &entity.TcID, &entity.Type); err == nil {
			entities = append(entities, entity)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return entities, nil
}

// Close closes db handle
func (ss *SQLiteStore) Close() error {
	if err := ss.db.Close(); err != nil {
		return err
	}
	return nil
}

func NewSQLiteStoreWithDB(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}
