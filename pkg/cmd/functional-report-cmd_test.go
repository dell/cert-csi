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

package cmd

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestGetFunctionalReportCommand(t *testing.T) {
	// Test that the function returns a cli.Command
	reportCmd := GetFunctionalReportCommand()

	// Test that the command has the correct name
	if reportCmd.Name != "functional-report" {
		t.Errorf("Expected GetFunctionalReportCommand to return a command with name 'functional-report', got '%s'", reportCmd.Name)
	}

	// Test that the command has the correct usage
	if reportCmd.Usage != "To generate functional tests report" {
		t.Errorf("Expected GetFunctionalReportCommand to return a command with usage 'To generate functional tests report', got '%s'", reportCmd.Usage)
	}

	// Test that the command has the correct category
	if reportCmd.Category != "main" {
		t.Errorf("Expected GetFunctionalReportCommand to return a command with category 'main', got '%s'", reportCmd.Category)
	}

	// Test that the command has the correct flags
	expectedFlags := []cli.Flag{
		cli.BoolFlag{
			Name:  "tabular",
			Usage: "specifies if tabular report should be generated",
		},
		cli.BoolFlag{
			Name:  "xml",
			Usage: "specifies if XML report should be generated",
		},
	}
	if len(reportCmd.Flags) != len(expectedFlags) {
		t.Errorf("Expected GetFunctionalReportCommand to return a command with %d flags, got %d", len(expectedFlags), len(reportCmd.Flags))
	}
	for _, flag := range expectedFlags {
		found := false
		for _, cmdFlag := range reportCmd.Flags {
			if flag == cmdFlag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected GetFunctionalReportCommand to return a command with flag '%s', but it was not found", flag.GetName())
		}
	}

	// Test that the command has the correct action function
	if reportCmd.Action == nil {
		t.Error("Expected GetFunctionalReportCommand to return a command with a non-nil Action function")
	}
}

func TestGetFunctionalReportCommandAction(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Bool("tabular", true, "specifies if tabular report should be generated")
	set.Bool("xml", true, "specifies if XML report should be generated")
	set.String("db", "postgres", "database name")
	ctx := cli.NewContext(nil, set, nil)

	set2 := flag.NewFlagSet("test", 0)
	set2.Bool("tabular", true, "specifies if tabular report should be generated")
	set2.Bool("xml", true, "specifies if XML report should be generated")
	ctx2 := cli.NewContext(nil, set2, nil)

	command := GetFunctionalReportCommand()
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
	err := actionFunc(ctx2)
	assert.Error(t, err)
}
