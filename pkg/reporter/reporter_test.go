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

package reporter

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/dell/cert-csi/pkg/collector"
	"github.com/dell/cert-csi/pkg/plotter"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type StdoutCapture struct {
	oldStdout *os.File
	readPipe  *os.File
}

func (sc *StdoutCapture) StartCapture() {
	sc.oldStdout = os.Stdout
	sc.readPipe, os.Stdout, _ = os.Pipe()
}

func (sc *StdoutCapture) StopCapture() (string, error) {
	if sc.oldStdout == nil || sc.readPipe == nil {
		return "", errors.New("StartCapture not called before StopCapture")
	}
	_ = os.Stdout.Close()
	os.Stdout = sc.oldStdout
	bytes, err := io.ReadAll(sc.readPipe)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type ReporterTestSuite struct {
	suite.Suite
	db                   store.Store
	runName              string
	filepath             string
	noRunIndbs           []*store.StorageClassDB
	successRunIndbs      []*store.StorageClassDB
	unsuccessfulRunIndbs []*store.StorageClassDB
}

func (suite *ReporterTestSuite) SetupSuite() {
	plotter.FolderPath = "/.cert-csi/tmp/report-tests/"
	suite.db = store.NewSQLiteStore("file:testdata/reporter_test.db")

	// When the test run is not present in the database
	noRunIndbsdbs := &store.StorageClassDB{DB: store.NewSQLiteStore("file:testdata/reporter_test.db")}

	// When a test run is present in the database
	successRunIndbs := &store.StorageClassDB{
		DB: store.NewSQLiteStore("file:testdata/reporter_test.db"),
		TestRun: store.TestRun{
			Name: "test-run-d6d1f7c8",
		},
	}

	// When unsuccessful test run is found in the database
	unsuccessfulRunIndbs := &store.StorageClassDB{
		DB: store.NewSQLiteStore("file:testdata/reporter_test.db"),
		TestRun: store.TestRun{
			Name: "unsuccessful-test-run",
		},
	}

	suite.noRunIndbs = []*store.StorageClassDB{noRunIndbsdbs}
	suite.successRunIndbs = []*store.StorageClassDB{successRunIndbs}
	suite.unsuccessfulRunIndbs = []*store.StorageClassDB{unsuccessfulRunIndbs}
	suite.runName = "test-run-d6d1f7c8"
	curUser, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}
	curUser = curUser + plotter.FolderPath
	curUserPath, err := filepath.Abs(curUser)
	if err != nil {
		log.Panic(err)
	}

	suite.filepath = curUserPath
}

func (suite *ReporterTestSuite) TearDownSuite() {
	err := suite.db.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = os.RemoveAll(suite.filepath)
	if err != nil {
		log.Fatal(err)
	}
}

func (suite *ReporterTestSuite) TestGenerateTextReporter() {
	capture := StdoutCapture{}
	capture.StartCapture()
	mc := collector.NewMetricsCollector(suite.db)
	metrics, err := mc.Collect(suite.runName)
	if err != nil {
		suite.Failf("error generating report", "Error generating report %v", err)
	}
	textReporter := &TextReporter{}
	err = textReporter.Generate(suite.runName, metrics)
	if err != nil {
		suite.Failf("error generating report", "Error generating report %v", err)
	}
	output, err := capture.StopCapture()
	suite.NoError(err, "Got an error trying to capture stdout and stderr!")
	suite.NotEmpty(output, "output content must not be empty")
	suite.Contains(output, fmt.Sprintf("Name: %s", suite.runName))
	suite.Contains(output, "StorageClass: powerstore")
	fmt.Println(output)
}

func (suite *ReporterTestSuite) TestGenerateAllReports() {
	type args struct {
		dbs []*store.StorageClassDB
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "nil dbs",
			args: args{
				dbs: nil,
			},
			wantErr: false,
		},
		{
			name: "no run in dbs",
			args: args{
				dbs: suite.noRunIndbs,
			},
			wantErr: true,
		},
		{
			name: "successful test run",
			args: args{
				dbs: suite.successRunIndbs,
			},
			wantErr: false,
		},
		{
			name: "unsuccessful test run",
			args: args{
				dbs: suite.unsuccessfulRunIndbs,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := GenerateAllReports(tt.args.dbs)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				for _, run := range tt.args.dbs {
					name := fmt.Sprintf("report-%s", run.TestRun.Name)
					htmlPath := fmt.Sprintf(suite.filepath+"/reports/%s/%s.html", run.TestRun.Name, name)
					txtPath := fmt.Sprintf(suite.filepath+"/reports/%s/%s.txt", run.TestRun.Name, name)
					suite.FileExists(htmlPath)
					suite.FileExists(txtPath)
				}
			}
		})
	}
}

// Test GenerateFunctionalReport method in functional-reporter.go
func (suite *ReporterTestSuite) TestGenerateFunctionalReport() {
	// Create a mock store
	mockDB := store.NewSQLiteStore("file:testdata/mock_functional_report.db")
	// Define report types to test
	reportTypes := []ReportType{TabularReport, XMLReport}

	// Execute the GenerateFunctionalReport function
	err := GenerateFunctionalReport(mockDB, reportTypes)

	// Verify the results
	suite.NoError(err, "Expected no error while generating functional reports")
}

// Test GetRestultStatus method in html-reporter.go
func (suite *ReporterTestSuite) TestGetResultStatus() {
	hr := &HTMLReporter{}

	// Test case: result is true
	result := hr.getResultStatus(true)
	suite.Equal("SUCCESS", result, "Expected result status to be SUCCESS")

	// Test case: result is false
	result = hr.getResultStatus(false)
	suite.Equal("FAILURE", result, "Expected result status to be FAILURE")
}

// Test GetColorResultStatus method in html-reporter.go
func (suite *ReporterTestSuite) TestGetColorResultStatus() {
	hr := &HTMLReporter{}

	// Test case: result is true
	result := hr.getColorResultStatus(true)
	suite.Equal("green", result, "Expected color result status to be green")

	// Test case: result is false
	result = hr.getColorResultStatus(false)
	suite.Equal("red", result, "Expected color result status to be red")
}

// Test the inc function in tools.go
func (suite *ReporterTestSuite) TestInc() {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "Increment positive number",
			input:    5,
			expected: 6,
		},
		{
			name:     "Increment zero",
			input:    0,
			expected: 1,
		},
		{
			name:     "Increment negative number",
			input:    -3,
			expected: -2,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := inc(tt.input)
			suite.Equal(tt.expected, result, "Expected %d but got %d", tt.expected, result)
		})
	}
}

// Tests the shouldBeIncluded function in tools.go
func (suite *ReporterTestSuite) TestShouldBeIncluded() {
	tests := []struct {
		name     string
		metric   collector.DurationOfStage
		expected bool
	}{
		{
			name: "All metrics are valid",
			metric: collector.DurationOfStage{
				Max: 10,
				Min: 1,
				Avg: 5,
			},
			expected: true,
		},
		{
			name: "Max is negative",
			metric: collector.DurationOfStage{
				Max: -1,
				Min: 1,
				Avg: 5,
			},
			expected: false,
		},
		{
			name: "Min is negative",
			metric: collector.DurationOfStage{
				Max: 10,
				Min: -1,
				Avg: 5,
			},
			expected: false,
		},
		{
			name: "Avg is negative",
			metric: collector.DurationOfStage{
				Max: 10,
				Min: 1,
				Avg: -5,
			},
			expected: false,
		},
		{
			name: "All metrics are zero",
			metric: collector.DurationOfStage{
				Max: 0,
				Min: 0,
				Avg: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := shouldBeIncluded(tt.metric)
			suite.Equal(tt.expected, result, "Expected %v but got %v", tt.expected, result)
		})
	}
}

func TestReporterTestSuite(t *testing.T) {
	suite.Run(t, new(ReporterTestSuite))
}
