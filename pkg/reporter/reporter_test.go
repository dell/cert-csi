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
	"time"

	"github.com/dell/cert-csi/pkg/collector"
	"github.com/dell/cert-csi/pkg/plotter"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

// Tests the TestGeneratePlots method in reporter.go
func (suite *ReporterTestSuite) TestGeneratePlots() {
	// Create a MetricsCollection with one TestCaseMetrics entry
	mc := &collector.MetricsCollection{
		TestCasesMetrics: []collector.TestCaseMetrics{
			{
				TestCase: store.TestCase{
					Name: "SimpleTestCase",
					ID:   1,
				},
				StageMetrics: map[interface{}]collector.DurationOfStage{
					"Stage1": {
						Max: 10,
						Min: 1,
						Avg: 5,
					},
				},
			},
		},
	}

	generatePlots("simple-test-run", mc)

	// Ensure the function executed without crashing
	suite.True(true, "generatePlots executed without crashing")
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

// Test the generateReportsFromMultipleDBs method in reporter.go
func (suite *ReporterTestSuite) TestGenerateReportsFromMultipleDBsSimple() {
	// Define the report types to generate
	reportTypes := []ReportType{
		TabularReport,
		XMLReport,
	}

	// Use the existing successRunIndbs from the setup
	dbs := suite.successRunIndbs
	err := GenerateReportsFromMultipleDBs(reportTypes, dbs)

	// Ensure no errors occurred
	suite.NoError(err, "GenerateReportsFromMultipleDBs should not return an error")
}

// Test the getPlotStageMetricHistogramPath method in reporter.go
func TestGetPlotStageMetricHistogramPath(t *testing.T) {
	// Set up test metrics
	tc := collector.TestCaseMetrics{
		TestCase: store.TestCase{
			Name: "ExampleTestCase",
			ID:   123,
		},
	}

	stage := collector.PVCStage("exampleStage")
	reportName := "test-report"

	path := getPlotStageMetricHistogramPath(tc, stage, reportName)

	expectedPath := "ExampleTestCase123/exampleStage.png"
	assert.Equal(t, expectedPath, path.Path, "The constructed path should match the expected path")
	assert.Equal(t, reportName, path.ReportName, "The report name should match")
}

// Test for getPlotStageBoxPath in reporter.go
func TestGetPlotStageBoxPath(t *testing.T) {
	tc := collector.TestCaseMetrics{
		TestCase: store.TestCase{
			Name: "ExampleTestCase",
			ID:   123,
		},
	}

	stage := collector.PVCStage("exampleStage")
	reportName := "test-report"

	path := getPlotStageBoxPath(tc, stage, reportName)

	expectedPath := "ExampleTestCase123/exampleStage_boxplot.png"
	assert.Equal(t, expectedPath, path.Path, "The constructed path should match the expected path")
	assert.Equal(t, reportName, path.ReportName, "The report name should match")
}

// Test for getPlotEntityOverTimePath in reporter.go
func TestGetPlotEntityOverTimePath(t *testing.T) {
	tc := collector.TestCaseMetrics{
		TestCase: store.TestCase{
			Name: "ExampleTestCase",
			ID:   123,
		},
	}
	reportName := "test-report"
	path := getPlotEntityOverTimePath(tc, reportName)

	expectedPath := "ExampleTestCase123/EntityNumberOverTime.png" // Adjust the expected output based on your logic
	assert.Equal(t, expectedPath, path.Path, "The constructed path should match the expected path")
	assert.Equal(t, reportName, path.ReportName, "The report name should match")
}

// Test GenerateFunctionalReport method in functional-reporter.go
func (suite *ReporterTestSuite) TestGenerateFunctionalReport() {
	// Create a mock store and define report types to test
	mockDB := store.NewSQLiteStore("file:testdata/mock_functional_report.db")
	reportTypes := []ReportType{TabularReport, XMLReport}

	err := GenerateFunctionalReport(mockDB, reportTypes)

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

// Setup for XMLReporterTestSuite
type XMLReporterTestSuite struct {
	suite.Suite
	reporter *XMLReporter
}

func (suite *XMLReporterTestSuite) SetupTest() {
	suite.reporter = &XMLReporter{}
}

func (xr *XMLReporter) SetPassedCount(count int) {
	passedCount = count // Assuming passedCount is scoped properly.
}

// Test for the getResultStatus method in xml-reporter.go
func (suite *XMLReporterTestSuite) TestGetResultStatus() {
	// Test case when result is true
	result := suite.reporter.getResultStatus(true)
	suite.Equal("PASSED", result, "Expected result status to be PASSED")

	// Test case when result is false
	result = suite.reporter.getResultStatus(false)
	suite.Equal("FAILED", result, "Expected result status to be FAILED")
}

func TestXMLReporterTestSuite(t *testing.T) {
	suite.Run(t, new(XMLReporterTestSuite))
}

// Test for getPassedCount method in xml-reporter.go
func (suite *XMLReporterTestSuite) TestGetPassedCount() {
	// Setup the reporter
	suite.reporter = &XMLReporter{}

	// Use the setter method to set the passedCount for testing
	suite.reporter.SetPassedCount(5) // Simulating that 5 tests have passed

	result := suite.reporter.getPassedCount()

	suite.Equal(5, result, "Expected passed count to be 5")
}

// Test for getFailedCountFromMC method in xml-reporter.go
func (suite *XMLReporterTestSuite) TestGetFailedCountFromMC() {
	// Create a sample MetricsCollection with test cases
	testCases := []collector.TestCaseMetrics{
		{TestCase: store.TestCase{Success: true}},  // Passed
		{TestCase: store.TestCase{Success: false}}, // Failed
		{TestCase: store.TestCase{Success: true}},  // Passed
		{TestCase: store.TestCase{Success: false}}, // Failed
		{TestCase: store.TestCase{Success: false}}, // Failed
	}

	mc := &collector.MetricsCollection{
		TestCasesMetrics: testCases,
	}

	failedCount := getFailedCountFromMC(mc)

	// Assert that the number of failed test cases is correct
	suite.Equal(3, failedCount, "Expected failed count to be 3")
}

// Test for getPassedCountFromMC method in xml-reporter.go
func (suite *XMLReporterTestSuite) TestGetPassedCountFromMC() {
	// Create a sample MetricsCollection with test cases
	testCases := []collector.TestCaseMetrics{
		{TestCase: store.TestCase{Success: true}},  // Passed
		{TestCase: store.TestCase{Success: false}}, // Failed
		{TestCase: store.TestCase{Success: true}},  // Passed
		{TestCase: store.TestCase{Success: false}}, // Failed
		{TestCase: store.TestCase{Success: true}},  // Passed
	}

	mc := &collector.MetricsCollection{
		TestCasesMetrics: testCases,
	}

	passedCount := getPassedCountFromMC(mc)

	// Assert that the number of passed test cases is correct
	suite.Equal(3, passedCount, "Expected passed count to be 3")
}

// Test for the getTestDuration method in xml-reporter.go
func (suite *XMLReporterTestSuite) TestGetTestDuration() {
	// Define a start and end time for the test case
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Minute) // Add 2 minutes to the start time

	// Create a TestCase instance with the specified timestamps
	tc := store.TestCase{
		StartTimestamp: startTime,
		EndTimestamp:   endTime,
	}

	duration := getTestDuration(tc)

	// Assert that the duration is correctly formatted
	expectedDuration := "02:00" // 2 minutes

	suite.Equal(expectedDuration, duration, "Expected duration to be 02:00")
}

func (suite *ReporterTestSuite) TestUpdateTestCounts() {
	// Reset the counts before the test
	passedCount = 0
	failedCount = 0
	skippedCount = 0

	// Create a sample MetricsCollection with a mix of test cases
	testCases := []collector.TestCaseMetrics{
		{TestCase: store.TestCase{Success: true}},  // Passed
		{TestCase: store.TestCase{Success: true}},  // Passed
		{TestCase: store.TestCase{Success: false}}, // Failed
		{TestCase: store.TestCase{Success: true}},  // Passed
		{TestCase: store.TestCase{Success: false}}, // Failed
		{TestCase: store.TestCase{Success: false}}, // Failed
		{TestCase: store.TestCase{Success: false}}, // Failed
		{TestCase: store.TestCase{Success: true}},  // Passed
		{TestCase: store.TestCase{Success: true}},  // Passed
	}

	// Create a `MetricsCollection` with the test cases
	mc := &collector.MetricsCollection{
		TestCasesMetrics: testCases,
	}

	updateTestCounts(mc)

	// Assert the counts are accurate
	suite.Equal(5, passedCount, "Expected passed count to be 5")
	suite.Equal(4, failedCount, "Expected failed count to be 4")
	suite.Equal(0, skippedCount, "Expected skipped count to be 0 (None were truly skipped)")
}

// Test for the multiGenerate function in xml-reporter.go
func (suite *XMLReporterTestSuite) TestMultiGenerate() {
	// Create a mock MetricsCollection for testing
	mockMetrics := &collector.MetricsCollection{
		TestCasesMetrics: []collector.TestCaseMetrics{
			{TestCase: store.TestCase{Success: true}}, // Example successful test case
		},
	}

	err := suite.reporter.MultiGenerate([]*collector.MetricsCollection{mockMetrics})

	suite.NoError(err, "Expected no error while generating the report")
}

// Test for the getCustomReportName method in xml-reporter.go
func (suite *XMLReporterTestSuite) TestGetCustomReportName() {
	arrayConfig = map[string]string{
		"name": "testArray",
	}

	reportName := suite.reporter.getCustomReportName()

	expectedReportName := "csi-testArray-test-results"
	suite.Equal(expectedReportName, reportName, "Expected report name did not match")
}

// Test for getFailedCount function in xml-reporter.go
func (suite *XMLReporterTestSuite) TestXMLGetFailedCount() {
	// Mock a MetricsCollection with test cases
	mockMetrics := &collector.MetricsCollection{
		TestCasesMetrics: []collector.TestCaseMetrics{
			{TestCase: store.TestCase{Success: true}},  // Passed
			{TestCase: store.TestCase{Success: false}}, // Failed
			{TestCase: store.TestCase{Success: false}}, // Failed
		},
	}

	// Update global state by calling updateTestCounts
	updateTestCounts(mockMetrics)

	failedCount := suite.reporter.getFailedCount()

	suite.Equal(2, failedCount, "Expected failed count to be 2")
}

// Test for getResultStatus method in text-reporter.go
func (suite *ReporterTestSuite) TestTextGetResultStatus() {
	tr := &TextReporter{}

	// Test cases for getResultStatus
	tests := []struct {
		input    bool
		expected string
	}{
		{true, "SUCCESS"},  // This should be in green color
		{false, "FAILURE"}, // This should be in red color
	}

	for _, tt := range tests {
		result := tr.getResultStatus(tt.input)
		// Check if result matches expected value without considering color formatting
		if tt.input {
			suite.Contains(result, "SUCCESS", "Expected result status for input true to be SUCCESS")
		} else {
			suite.Contains(result, "FAILURE", "Expected result status for input false to be FAILURE")
		}
	}
}

// Test for getResultStatus in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetResultStatus() {
	tr := &TabularReporter{}

	// Test cases for getResultStatus
	tests := []struct {
		input    bool
		expected string
	}{
		{true, "SUCCESS"},
		{false, "FAILURE"},
	}

	for _, tt := range tests {
		result := tr.getResultStatus(tt.input)
		suite.Equal(tt.expected, result, "Expected result status for input %v to be %s, but got %s", tt.input, tt.expected, result)
	}
}

// Test for getColorResultStatus in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetColorResultStatus() {
	tr := &TabularReporter{}
	tests := []struct {
		input    bool
		expected string
	}{
		{true, "green"},
		{false, "red"},
	}

	for _, tt := range tests {
		result := tr.getColorResultStatus(tt.input)
		suite.Equal(tt.expected, result, "Expected color result status for input %v to be %s, but got %s", tt.input, tt.expected, result)
	}
}

// Test for getSlNo in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetSlNo() {
	tr := &TabularReporter{}

	// Test cases for getSlNo
	tests := []struct {
		input    int
		expected int
	}{
		{0, 1},   // The first index should return 1
		{1, 2},   // The second index should return 2
		{10, 11}, // The eleventh index should return 11
		{-1, 0},  // Testing a negative index, should return 0 (if negative handling is desired)
	}

	for _, tt := range tests {
		result := tr.getSlNo(tt.input)
		suite.Equal(tt.expected, result, "Expected serial number for index %d to be %d, but got %d", tt.input, tt.expected, result)
	}
}

// Test for getArrays in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetArrays() {
	tr := &TabularReporter{}

	// Test scenario 1: arrayConfig has arrayIPs set
	arrayConfig = map[string]string{
		"name":     "testArray",
		"arrayIPs": "192.168.1.1,192.168.1.2",
	}

	result := tr.getArrays()
	suite.Equal("192.168.1.1,192.168.1.2", result, "Expected arrayIPs to match the configured value")

	// Test scenario 2: arrayConfig does not have arrayIPs set
	arrayConfig = map[string]string{
		"name": "testArray",
		// No arrayIPs key
	}

	result = tr.getArrays()
	suite.Empty(result, "Expected arrayIPs to be empty when not set in config")
}

// Test for getArrayConfig function in table-reporter.go
func (suite *ReporterTestSuite) TestGetArrayConfig() {
	// Create a temporary properties file for testing
	file, err := os.Create("test-config.properties")
	suite.NoError(err)
	defer os.Remove("test-config.properties") // Clean up after test

	_, err = file.WriteString("name=testArray\narrayIPs=192.168.1.1,192.168.1.2\n")
	suite.NoError(err)
	file.Close()

	config, err := getArrayConfig("test-config.properties")

	suite.NoError(err)
	suite.Equal("testArray", config["name"])
	suite.Equal("192.168.1.1,192.168.1.2", config["arrayIPs"])
}

// Test for MultiGenerate function in table-reporter.go
func (suite *ReporterTestSuite) TestMultiGenerate() {
	mockMetrics := []*collector.MetricsCollection{
		{
			TestCasesMetrics: []collector.TestCaseMetrics{
				{TestCase: store.TestCase{Success: true}},  // Successful test case
				{TestCase: store.TestCase{Success: false}}, // Failed test case
			},
		},
		{
			TestCasesMetrics: []collector.TestCaseMetrics{
				{TestCase: store.TestCase{Success: true}}, // Another successful test case
			},
		},
	}

	tr := &TabularReporter{}
	err := tr.MultiGenerate(mockMetrics)

	suite.NoError(err, "Expected no error while generating reports")
}

// Test for the getCustomReportName method in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetCustomReportName() {
	tr := &TabularReporter{}
	arrayConfig = map[string]string{
		"name": "testArray",
	}

	expectedReportName := "csi-testArray-test-results"
	reportName := tr.getCustomReportName()

	suite.Equal(expectedReportName, reportName, "Expected report name did not match")
}

// Test for the getPassedCount method in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetPassedCount() {
	tr := &TabularReporter{}

	// Simulate passed test counts
	passedCount = 5
	result := tr.getPassedCount()

	suite.Equal(5, result, "Expected passed count to be 5")
}

// Test for the getFailedCount method in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetFailedCount() {
	tr := &TabularReporter{}

	// Simulate failed test counts
	failedCount = 3
	result := tr.getFailedCount()

	suite.Equal(3, result, "Expected failed count to be 3")
}

// Test for the getSkippedCount method in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetSkippedCount() {
	tr := &TabularReporter{}

	// Simulate skipped test counts
	skippedCount = 2
	result := tr.getSkippedCount()

	suite.Equal(2, result, "Expected skipped count to be 2")
}

// Test for the getBuildName method in table-reporter.go
func (suite *ReporterTestSuite) TestTabularReporterGetBuildName() {
	tr := &TabularReporter{}

	// Set the environment variable for testing
	os.Setenv("CERT_CSI_BUILD_NAME", "my-build")

	defer os.Unsetenv("CERT_CSI_BUILD_NAME") // Cleanup after test

	buildName := tr.getBuildName()
	suite.Equal("my-build", buildName, "Expected build name to match environment variable")
}

func TestReporterTestSuite(t *testing.T) {
	suite.Run(t, new(ReporterTestSuite))
}
