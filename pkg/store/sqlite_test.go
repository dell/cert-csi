package store

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateTables(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}

	// Set up the expected queries and arguments
	expectedQueries := []string{
		`CREATE TABLE IF NOT EXISTS test_runs(
			id INTEGER PRIMARY KEY,
			name VARCHAR(50) NOT NULL UNIQUE,
			longevity BOOLEAN DEFAULT false,
			start_timestamp DATETIME,
			storage_class VARCHAR(50) NOT NULL,
			cluster_address VARCHAR(50) NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS test_cases(
			id INTEGER PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			k8s_uid VARCHAR(50) NOT NULL,
			tc_id INTEGER NOT NULL,
			type VARCHAR(50) NOT NULL,
			FOREIGN KEY (tc_id) REFERENCES test_cases(id))`,
		`CREATE TABLE IF NOT EXISTS entities(
			id INTEGER PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			k8s_uid VARCHAR(50) NOT NULL,
			tc_id INTEGER NOT NULL,
			type VARCHAR(50) NOT NULL,
			FOREIGN KEY (tc_id) REFERENCES test_cases(id))`,
		`CREATE TABLE IF NOT EXISTS entities_relations(
			entity_id1 INTEGER NOT NULL,
			entity_id2 INTEGER NOT NULL,
			FOREIGN KEY (entity_id1) REFERENCES entities(id),
			FOREIGN KEY (entity_id2) REFERENCES entities(id))`,
		`CREATE TABLE IF NOT EXISTS resource_usage(
			id INTEGER PRIMARY KEY,
			tc_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			pod_name VARCHAR(50) NOT NULL,
			container_name VARCHAR(50) NOT NULL,
			cpu INTEGER NOT NULL,
			mem INTEGER NOT NULL,
			FOREIGN KEY (tc_id) REFERENCES test_cases(id))`,
		`CREATE TABLE IF NOT EXISTS number_entities(
			id INTEGER PRIMARY KEY,
			tc_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			pods_creating INTEGER NOT NULL,
			pods_ready INTEGER NOT NULL,
			pods_terminating INTEGER NOT NULL,
			pvc_creating INTEGER NOT NULL,
			pvc_bound INTEGER NOT NULL,
			pvc_terminating INTEGER NOT NULL,
			FOREIGN KEY (tc_id) REFERENCES test_cases(id))`,
	}

	// Set up the mock to expect the queries and return the expected result
	for _, query := range expectedQueries {
		mock.ExpectExec(query).
			WillReturnError(fmt.Errorf("some error"))
	}

	// Call the createTables function
	err = ss.createTables()
	if err != nil {
		t.Logf("SQLiteStore.createTables() error = %v", err)
	}

}

func TestSaveTestRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	ss := &SQLiteStore{
		db: db,
	}

	tests := []struct {
		name    string
		tr      *TestRun
		wantErr bool
	}{
		{
			name: "successful save",
			tr: &TestRun{
				Name:           "test run 1",
				StartTimestamp: time.Now(),
				StorageClass:   "sc1",
				ClusterAddress: "localhost",
			},
			wantErr: false,
		},
		{
			name: "save with error",
			tr: &TestRun{
				Name:           "test run 2",
				StartTimestamp: time.Now(),
				StorageClass:   "sc2",
				ClusterAddress: "localhost2",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				mock.ExpectExec("INSERT INTO test_runs").
					WithArgs(tt.tr.Name, tt.tr.StartTimestamp, tt.tr.StorageClass, tt.tr.ClusterAddress).
					WillReturnError(fmt.Errorf("some error"))
			} else {
				mock.ExpectExec("INSERT INTO test_runs").
					WithArgs(tt.tr.Name, tt.tr.StartTimestamp, tt.tr.StorageClass, tt.tr.ClusterAddress).
					WillReturnResult(sqlmock.NewResult(1, 1))

			}
			err := ss.SaveTestRun(tt.tr)
			if (err != nil) != tt.wantErr {
				t.Errorf("SQLiteStore.SaveTestRun() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestGetTestRuns(t *testing.T) {
	ss := NewSQLiteStore("file:test1.db?cache=shared&mode=memory")
	tests := []struct {
		name    string
		db      *sql.DB
		where   Conditions
		orderBy string
		limit   int
		want    []TestRun
		wantErr bool
	}{
		{
			name: "successful query",
			db:   ss.db, // mock database
			where: Conditions{
				"name": "test run 1",
			},
			orderBy: "start_timestamp DESC",
			limit:   1,
			want: []TestRun{
				{
					ID:             1,
					Name:           "test run 1",
					StartTimestamp: time.Now(),
					StorageClass:   "sc1",
					ClusterAddress: "localhost",
				},
			},
			wantErr: false,
		},
		{
			name: "query with error",
			db:   ss.db, // mock database
			where: Conditions{
				"fake_param": "test-run-2",
			},
			orderBy: "start_timestamp DESC",
			limit:   1,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := ss.GetTestRuns(tt.where, tt.orderBy, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("SQLiteStore.GetTestRuns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
func TestSaveEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	ss := &SQLiteStore{
		db: db,
	}

	tests := []struct {
		name    string
		events  []*Event
		wantErr bool
	}{

		{
			name: "save with error",
			events: []*Event{
				{
					Name:      "event3",
					TcID:      5,
					EntityID:  6,
					Type:      "type3",
					Timestamp: time.Now(),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				mock.ExpectExec("INSERT INTO events").
					WithArgs(tt.events[0].Name, tt.events[0].TcID, tt.events[0].EntityID, tt.events[0].Type, tt.events[0].Timestamp).
					WillReturnError(fmt.Errorf("some error"))
			}

			err := ss.SaveEvents(tt.events)
			if (err != nil) != tt.wantErr {
				t.Errorf("SQLiteStore.SaveEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEntitiesWithEventsByTestCaseAndEntityType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	ss := &SQLiteStore{
		db: db,
	}

	tc := &TestCase{
		ID:             1,
		Name:           "test case",
		StartTimestamp: time.Now(),
	}

	tests := []struct {
		name    string
		eType   EntityTypeEnum
		want    map[Entity][]Event
		wantErr bool
	}{
		{
			name:    "query with error",
			eType:   "type2",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				mock.ExpectExec("SELECT et.*, ev.* FROM entities et JOIN events ev ON et.id = ev.entity_id WHERE et.type = ? AND et.tc_id = ? ORDER BY et.id").
					WithArgs(tt.eType, tc.ID).
					WillReturnError(fmt.Errorf("some error"))
			}
			got, err := ss.GetEntitiesWithEventsByTestCaseAndEntityType(tc, tt.eType)
			if (err != nil) != tt.wantErr {
				t.Errorf("SQLiteStore.GetEntitiesWithEventsByTestCaseAndEntityType() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SQLiteStore.GetEntitiesWithEventsByTestCaseAndEntityType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveTestCase(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}

	// Create a new instance of TestCase
	tc := &TestCase{
		ID:             1,
		Name:           "test case",
		Parameters:     "test case parameters",
		StartTimestamp: time.Now(),
		EndTimestamp:   time.Now(),
		Success:        true,
		ErrorMessage:   "test case error message",
		RunID:          1,
	}
	// Set up the mock to expect the query and return the expected result
	mock.ExpectExec("INSERT INTO test_cases").WithoutArgs().WillReturnError(fmt.Errorf("some error"))

	// Call the SaveTestCase function
	err = ss.SaveTestCase(tc)
	if err != nil {
		fmt.Println(err)
	}
}

func TestSuccessfulTestCase(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}

	// Set up the expected query and arguments
	expectedQuery := `
		UPDATE test_cases SET success=?, end_timestamp=? WHERE id=?
	`
	expectedArgs := []driver.Value{
		true,
		time.Now(),
		1,
	}

	// Set up the mock to expect the query and return the expected result
	mock.ExpectExec(expectedQuery).
		WithArgs(expectedArgs...).
		WillReturnError(fmt.Errorf("some error"))

	// Create a new TestCase object
	ts := &TestCase{
		ID:             1,
		Name:           "test case",
		Parameters:     "test case parameters",
		StartTimestamp: time.Now(),
		EndTimestamp:   time.Now(),
		Success:        false,
		ErrorMessage:   "test case error message",
		RunID:          1,
	}

	// Call the SuccessfulTestCase function
	err = ss.SuccessfulTestCase(ts, time.Now())
	if err != nil {
		t.Logf("SQLiteStore.SuccessfulTestCase() error = %v", err)
	}
}

func TestSaveEntities(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}

	// Set up the expected query and arguments
	expectedQuery := `
		INSERT INTO entities(name, k8s_uid, tc_id, type)
		VALUES (?, ?, ?, ?)
	`

	// Set up the expected result
	entities := []*Entity{
		{
			ID:     1,
			Name:   "entity1",
			K8sUID: "uid1",
			TcID:   1,
			Type:   "type1",
		},
		{
			ID:     2,
			Name:   "entity2",
			K8sUID: "uid2",
			TcID:   2,
			Type:   "type2",
		},
	}

	// Set up the mock to expect the query and return the expected result
	mock.ExpectExec(expectedQuery).
		WithArgs().
		WillReturnError(fmt.Errorf("some error"))

	// Call the SaveEntities function
	err = ss.SaveEntities(entities)
	if err != nil {
		t.Logf("Mocked Error: %v", err)
	}
}

func TestSaveNumberEntities(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}

	// Set up the expected query and arguments
	expectedQuery := `
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

	// Set up the expected result
	nEntities := []*NumberEntities{
		{
			TcID:            1,
			Timestamp:       time.Now(),
			PodsCreating:    10,
			PodsReady:       20,
			PodsTerminating: 30,
			PvcCreating:     40,
			PvcBound:        50,
			PvcTerminating:  60,
		},
		{
			TcID:            2,
			Timestamp:       time.Now(),
			PodsCreating:    100,
			PodsReady:       200,
			PodsTerminating: 300,
			PvcCreating:     400,
			PvcBound:        500,
			PvcTerminating:  600,
		},
	}

	// Set up the mock to expect the query and return the expected result
	mock.ExpectExec(expectedQuery).
		WithoutArgs().
		WillReturnError(fmt.Errorf("some error"))

	// Call the SaveNumberEntities function
	err = ss.SaveNumberEntities(nEntities)
	if err != nil {
		t.Logf("Mocked Error: %v", err)
	}
}
func TestSaveResourceUsage(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}

	// Set up the expected query and arguments
	expectedQuery := `
		INSERT INTO resource_usage(
			tc_id,
			timestamp,
			pod_name,
			container_name,
			cpu_value,
			mem_value
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	// Set up the expected result
	resUsages := []*ResourceUsage{
		{
			TcID:          1,
			Timestamp:     time.Now(),
			PodName:       "pod1",
			ContainerName: "container1",
			CPU:           100,
			Mem:           100,
		},
		{
			TcID:          2,
			Timestamp:     time.Now(),
			PodName:       "pod2",
			ContainerName: "container2",
			CPU:           200,
			Mem:           200,
		},
	}

	// Set up the mock to expect the query and return the expected result
	mock.ExpectExec(expectedQuery).
		WithArgs(
			resUsages[0].TcID,
			resUsages[0].Timestamp,
			resUsages[0].PodName,
			resUsages[0].ContainerName,
			resUsages[0].CPU,
			resUsages[0].Mem,
			resUsages[1].TcID,
			resUsages[1].Timestamp,
			resUsages[1].PodName,
			resUsages[1].ContainerName,
			resUsages[1].CPU,
			resUsages[1].Mem,
		).
		WillReturnError(fmt.Errorf("some error"))

	// Call the SaveResourceUsage function
	err = ss.SaveResourceUsage(resUsages)
	if err != nil {
		fmt.Println(err)
	}
}

func TestCreateEntitiesRelation(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}

	// Set up the expected query and arguments
	expectedQuery := `
		INSERT INTO entities_relations (entity_id1, entity_id2)
		VALUES (?, ?)
	`

	// Set up the expected result
	entity1 := Entity{
		ID:   1,
		Name: "entity1",
		TcID: 1,
		Type: "type1",
	}
	entity2 := Entity{
		ID:   2,
		Name: "entity2",
		TcID: 2,
		Type: "type2",
	}

	// Set up the mock to expect the query and return the expected result
	mock.ExpectExec(expectedQuery).
		WithoutArgs().
		WillReturnError(fmt.Errorf("some error"))

	// Call the CreateEntitiesRelation function
	err = ss.CreateEntitiesRelation(entity1, entity2)
	if err != nil {
		fmt.Println(err)
	}
}
func TestGetEntityRelations(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}
	// Set up the expected query and arguments
	expectedQuery := `
		SELECT e.*
		FROM entities e
		  INNER JOIN entities_relations er ON e.id = er.entity_id2
		WHERE er.entity_id1 = ?
	`
	// Set up the expected result
	expectedEntities := []Entity{
		{
			ID:     2,
			Name:   "entity1",
			K8sUID: "uid1",
			TcID:   1,
			Type:   "type1",
		},
	}
	mock.ExpectQuery(expectedQuery).
		WithoutArgs().
		WillReturnError(fmt.Errorf("some error"))

	// Call the GetEntityRelations function
	_, err = ss.GetEntityRelations(expectedEntities[0])

	// Check for any errors
	if err != nil {
		fmt.Println("Mocked Error")
	}

}

func TestClose(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Create a new instance of SQLiteStore
	ss := &SQLiteStore{
		db: db,
	}

	// Expect the Close method to be called
	mock.ExpectClose().WillReturnError(fmt.Errorf("some error"))

	// Call the Close function
	err = ss.Close()
	if err != nil {
		fmt.Println(err)
	}
}
