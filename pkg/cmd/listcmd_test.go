package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
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
