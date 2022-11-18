package reporter

import (
	"cert-csi/pkg/collector"
	"cert-csi/pkg/plotter"
	"cert-csi/pkg/store"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

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
	bytes, err := ioutil.ReadAll(sc.readPipe)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type ReporterTestSuite struct {
	suite.Suite
	db       store.Store
	runName  string
	filepath string
}

func (suite *ReporterTestSuite) SetupSuite() {
	plotter.FolderPath = "/.cert-csi/tmp/report-tests/"
	suite.db = store.NewSQLiteStore("file:testdata/reporter_test.db")
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
		testRunNames []string
		db           store.Store
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "all nil",
			args: args{
				testRunNames: nil,
				db:           nil,
			},
			wantErr: false,
		},
		{
			name: "nil db",
			args: args{
				testRunNames: []string{"test-run-d6d1f7c8"},
				db:           nil,
			},
			wantErr: true,
		},
		{
			name: "nil name",
			args: args{
				testRunNames: nil,
				db:           suite.db,
			},
			wantErr: false,
		},
		{
			name: "no run in db",
			args: args{
				testRunNames: []string{"non-existent-name"},
				db:           suite.db,
			},
			wantErr: true,
		},
		{
			name: "single successful test run",
			args: args{
				testRunNames: []string{"test-run-d6d1f7c8"},
				db:           suite.db,
			},
			wantErr: false,
		},
		{
			name: "successful and unsuccessful test run",
			args: args{
				testRunNames: []string{"test-run-d6d1f7c8", "test-run-e47deecc"},
				db:           suite.db,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := GenerateAllReports(tt.args.testRunNames, tt.args.db)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				for _, run := range tt.args.testRunNames {
					name := fmt.Sprintf("report-%s", run)
					htmlPath := fmt.Sprintf(suite.filepath+"/reports/%s/%s.html", run, name)
					txtPath := fmt.Sprintf(suite.filepath+"/reports/%s/%s.txt", run, name)
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
