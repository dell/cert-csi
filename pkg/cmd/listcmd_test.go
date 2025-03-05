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

func TestGetListCommand(t *testing.T) {
	// Test that the function returns a cli.Command
	listCmd := GetListCommand()
	assert.IsType(t, cli.Command{}, listCmd)
<<<<<<< HEAD
	// Test that the command has the correct name
	assert.Equal(t, "list", listCmd.Name)
	// Test that the command has the correct usage
	assert.Equal(t, "lists different data", listCmd.Usage)
	// Test that the command has the correct category
	assert.Equal(t, "main", listCmd.Category)
=======

	// Test that the command has the correct name
	assert.Equal(t, "list", listCmd.Name)

	// Test that the command has the correct usage
	assert.Equal(t, "lists different data", listCmd.Usage)

	// Test that the command has the correct category
	assert.Equal(t, "main", listCmd.Category)

>>>>>>> 4fd111769aa282546f68e2884c31dc2b2c79def3
	// Test that the command has the correct subcommand
	assert.Len(t, listCmd.Subcommands, 1)
	assert.Equal(t, "test-runs", listCmd.Subcommands[0].Name)
	assert.Equal(t, "tr", listCmd.Subcommands[0].ShortName)
	assert.Equal(t, "list", listCmd.Subcommands[0].Category)
}
<<<<<<< HEAD
=======

>>>>>>> 4fd111769aa282546f68e2884c31dc2b2c79def3
func TestGetTestrunsCmd(t *testing.T) {
	// Test that the function returns a cli.Command
	testrunsCmd := GetTestrunsCmd()
	assert.IsType(t, cli.Command{}, testrunsCmd)
<<<<<<< HEAD
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
	//ss := store.NewSQLiteStore()
	set.String("db", "test1.db?cache=shared&mode=memory", "database name")
	//set.Func()
	ctx := cli.NewContext(nil, set, nil)

	//store := &SimpleStore{}

	//store.NewSQLiteStore("file:test1.db?cache=shared&mode=memory").GetTestRuns(store.Conditions{}, "", 0) = func(conditions store.Conditions, orderBy string, limit int) ([]store.TestRun, error) {
	//	return []store.TestRun{{Name: "test-run-1"}}, nil
	//}
	command := GetTestrunsCmd()
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
=======

	// Test that the command has the correct name
	assert.Equal(t, "test-runs", testrunsCmd.Name)

	// Test that the command has the correct short name
	assert.Equal(t, "tr", testrunsCmd.ShortName)

	// Test that the command has the correct category
	assert.Equal(t, "list", testrunsCmd.Category)
>>>>>>> 4fd111769aa282546f68e2884c31dc2b2c79def3
}
