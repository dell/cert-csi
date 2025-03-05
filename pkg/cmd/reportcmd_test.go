package cmd

import (
<<<<<<< HEAD
	"flag"
=======
>>>>>>> 4fd111769aa282546f68e2884c31dc2b2c79def3
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestGetReportCommand(t *testing.T) {
	// Test that the function returns a cli.Command
	reportCmd := GetReportCommand()
	assert.IsType(t, cli.Command{}, reportCmd)
<<<<<<< HEAD
	// Test that the command has the correct name
	assert.Equal(t, "report", reportCmd.Name)
	// Test that the command has the correct usage
	assert.Equal(t, "generate report from test run name", reportCmd.Usage)
	// Test that the command has the correct category
	assert.Equal(t, "main", reportCmd.Category)
	// Test that the command has the correct flags
	assert.Len(t, reportCmd.Flags, 6)
=======

	// Test that the command has the correct name
	assert.Equal(t, "report", reportCmd.Name)

	// Test that the command has the correct usage
	assert.Equal(t, "generate report from test run name", reportCmd.Usage)

	// Test that the command has the correct category
	assert.Equal(t, "main", reportCmd.Category)

	// Test that the command has the correct flags
	assert.Len(t, reportCmd.Flags, 6)

>>>>>>> 4fd111769aa282546f68e2884c31dc2b2c79def3
	assert.Contains(t, reportCmd.Flags, cli.StringFlag{
		Name:  "reportPath, path",
		Usage: "path to folder where reports will be created (if not specified `~/.cert-csi/` will be used)",
	})
	assert.Contains(t, reportCmd.Flags, cli.BoolFlag{
		Name:  "html",
		Usage: "specifies if html report should be generated",
	})
	assert.Contains(t, reportCmd.Flags, cli.BoolFlag{
		Name:  "txt",
		Usage: "specifies if txt report should be generated",
	})
	assert.Contains(t, reportCmd.Flags, cli.BoolFlag{
		Name:  "tabular",
		Usage: "specifies if tabular html report should be generated",
	})
	assert.Contains(t, reportCmd.Flags, cli.BoolFlag{
		Name:  "xml",
		Usage: "specifies if qTest xml report should be generated",
	})
<<<<<<< HEAD
	// Test that the command has the correct action function
	assert.NotNil(t, reportCmd.Action)
}
=======

	// Test that the command has the correct action function
	assert.NotNil(t, reportCmd.Action)
}

>>>>>>> 4fd111769aa282546f68e2884c31dc2b2c79def3
func Test_parseTestRun(t *testing.T) {
	tests := []struct {
		name     string
		tr       string
		wantDb   string
		wantName string
	}{
		{
			name:     "test with db and name",
			tr:       "file.db:testrun",
			wantDb:   "file.db",
			wantName: "testrun",
		},
		{
			name:     "test with only name",
			tr:       "testrun",
			wantDb:   "",
			wantName: "testrun",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDb, gotName := ParseTestRun(tt.tr)
			assert.Equal(t, tt.wantDb, gotDb)
			assert.Equal(t, tt.wantName, gotName)
		})
	}
}
<<<<<<< HEAD

func TestGetReportCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Bool("tabular", true, "specifies if tabular report should be generated")
	set.Bool("xml", true, "specifies if XML report should be generated")
	//set.String("db", "file:test1.db?cache=shared&mode=memory", "database name")
	//set.Func()
	var testRunNames *cli.StringSlice
	//mockTestRunNames := &cli.StringSlice{"file:test1.db?cache=shared&mode=memory"}
	testRunNamesFlag := cli.StringSliceFlag{
		Name:     "test-run-1",
		Usage:    "test run names from which reports will be generated (file.db:testrun)",
		Required: true,
		Value: &cli.StringSlice{
			"file.db?cache=shared&mode=memory:testrun",
		},
	}

	testRunNames = testRunNamesFlag.Value

	set.Var(testRunNames, "testrun", "test run names from which reports will be generated (file.db:testrun)")

	ctx := cli.NewContext(nil, set, nil)

	//store := &SimpleStore{}

	//store.NewSQLiteStore("file:test1.db?cache=shared&mode=memory").GetTestRuns(store.Conditions{}, "", 0) = func(conditions store.Conditions, orderBy string, limit int) ([]store.TestRun, error) {
	//	return []store.TestRun{{Name: "test-run-1"}}, nil
	//}
	command := GetReportCommand()

	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}
=======
>>>>>>> 4fd111769aa282546f68e2884c31dc2b2c79def3
