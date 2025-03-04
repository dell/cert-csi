package cmd

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestGetK8sEndToEndCommand(t *testing.T) {
	// Test that the function returns a cli.Command
	e2eCmd := GetK8sEndToEndCommand()
	assert.IsType(t, cli.Command{}, e2eCmd)

	// Test that the command has the correct name
	assert.Equal(t, "k8s-e2e", e2eCmd.Name)

	// Test that the command has the correct usage
	assert.Equal(t, "k8s-e2e command to execute the kubernetes end-to-end testcases", e2eCmd.Usage)

	// Test that the command has the correct category
	assert.Equal(t, "main", e2eCmd.Category)

	// Test that the command has the correct flags
	assert.Len(t, e2eCmd.Flags, 10)

	// Test that the command has the correct action function
	assert.NotNil(t, e2eCmd.Action)
}

func TestGetK8sEndToEndCommandAction(t *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("driver-config", "../k8sclient/testdata/config", "path to test driver config file")
	set.String("reportPath", "../testdata/", "path to folder where reports will be created (if not specified `$HOME/reports/execution_[storage-class].xml` will be used")
	set.String("focus", "\"External.Storage.*\"", "what your k8s e2e tests should foucs on")
	set.String("focus-file", "testsuitename.go", "what your k8s e2e tests should focus on")
	set.String("config", "../testdata/config", "config file")
	set.String("skip", "", "what your k8s e2e tests should skip")
	set.String("skip-file", "- \"skip this test\"", "skip unsupported files give the file path")
	set.String("skip-tests", "- \"skip this test\"", "skip unsupported tests give the file path")
	set.String("timeout", "1m", "ime out for kubernetes e2e")
	set.String("version", "1.29", "Kubernetes version")
	ctx := cli.NewContext(nil, set, nil)
	command := GetK8sEndToEndCommand()
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetK8sEndToEndCommandBefore(t *testing.T) {
	// Default context
	set1 := flag.NewFlagSet("test", 0)
	set1.String("driver-config", "../k8sclient/testdata/config", "path to test driver config file")

	//empty path
	set2 := flag.NewFlagSet("test2", 0)
	set2.String("driver-config", "", "path to test driver config file")

	ctx1 := cli.NewContext(nil, set1, nil)
	ctx2 := cli.NewContext(nil, set2, nil)
	command := GetK8sEndToEndCommand()
	// Call the action function
	before := command.Before
	before(ctx1)
	before(ctx2)
}
