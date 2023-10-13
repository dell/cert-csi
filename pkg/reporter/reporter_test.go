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

func TestReporterTestSuite(t *testing.T) {
	suite.Run(t, new(ReporterTestSuite))
}
