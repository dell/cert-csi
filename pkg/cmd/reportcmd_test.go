package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestGetReportCommand(t *testing.T) {
	// Test that the function returns a cli.Command
	reportCmd := GetReportCommand()
	assert.IsType(t, cli.Command{}, reportCmd)

	// Test that the command has the correct name
	assert.Equal(t, "report", reportCmd.Name)

	// Test that the command has the correct usage
	assert.Equal(t, "generate report from test run name", reportCmd.Usage)

	// Test that the command has the correct category
	assert.Equal(t, "main", reportCmd.Category)

	// Test that the command has the correct flags
	assert.Len(t, reportCmd.Flags, 6)

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

	// Test that the command has the correct action function
	assert.NotNil(t, reportCmd.Action)
}

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
