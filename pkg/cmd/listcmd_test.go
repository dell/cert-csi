package cmd

import (
	"flag"
	"testing"

	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/store/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"go.uber.org/mock/gomock"
)

func TestGetListCommand(t *testing.T) {
	// Test that the function returns a cli.Command
	listCmd := GetListCommand()
	assert.IsType(t, cli.Command{}, listCmd)
	// Test that the command has the correct name
	assert.Equal(t, "list", listCmd.Name)
	// Test that the command has the correct usage
	assert.Equal(t, "lists different data", listCmd.Usage)
	// Test that the command has the correct category
	assert.Equal(t, "main", listCmd.Category)
	// Test that the command has the correct subcommand
	assert.Len(t, listCmd.Subcommands, 1)
	assert.Equal(t, "test-runs", listCmd.Subcommands[0].Name)
	assert.Equal(t, "tr", listCmd.Subcommands[0].ShortName)
	assert.Equal(t, "list", listCmd.Subcommands[0].Category)
}

func TestGetTestrunsCmd(t *testing.T) {
	// Test that the function returns a cli.Command
	testrunsCmd := GetTestrunsCmd()
	assert.IsType(t, cli.Command{}, testrunsCmd)
	// Test that the command has the correct name
	assert.Equal(t, "test-runs", testrunsCmd.Name)
	// Test that the command has the correct short name
	assert.Equal(t, "tr", testrunsCmd.ShortName)
	// Test that the command has the correct category
	assert.Equal(t, "list", testrunsCmd.Category)
}

func TestGetListCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Bool("tabular", true, "specifies if tabular report should be generated")
	set.Bool("xml", true, "specifies if XML report should be generated")
	// ss := store.NewSQLiteStore()
	set.String("db", "test1.db?cache=shared&mode=memory", "database name")
	// set.Func()
	ctx := cli.NewContext(nil, set, nil)

	// store := &SimpleStore{}

	//store.NewSQLiteStore("file:test1.db?cache=shared&mode=memory").GetTestRuns(store.Conditions{}, "", 0) = func(conditions store.Conditions, orderBy string, limit int) ([]store.TestRun, error) {
	//	return []store.TestRun{{Name: "test-run-1"}}, nil
	//}

	mockStore := mocks.NewMockStore(gomock.NewController(t))
	mockStore.EXPECT().GetTestRuns(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]store.TestRun{{Name: "test-run-1"}}, nil)
	mockStore.EXPECT().GetTestCases(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]store.TestCase{{Name: "test-case-1"}}, nil)
	mockStore.EXPECT().Close().Times(1)

	GetDatabase = func(c *cli.Context) store.Store {
		return mockStore
	}

	command := GetTestrunsCmd()
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}
