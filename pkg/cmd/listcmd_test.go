/*
 *
 * Copyright © 2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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

	"github.com/dell/cert-csi/pkg/store"
	mocks "github.com/dell/cert-csi/pkg/testmocks"
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
	set.String("db", "test1.db?cache=shared&mode=memory", "database name")
	ctx := cli.NewContext(nil, set, nil)

	mockStore := mocks.NewMockStore(gomock.NewController(t))
	mockStore.EXPECT().GetTestRuns(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]store.TestRun{{Name: "test-run-1"}}, nil)
	mockStore.EXPECT().GetTestCases(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]store.TestCase{{Name: "test-case-1"}}, nil)
	mockStore.EXPECT().Close().Times(1)

	GetDatabase = func(_ *cli.Context) store.Store {
		return mockStore
	}

	command := GetTestrunsCmd()
	// Call the action function
	action := command.Action
	actionFunc := action.(func(_ *cli.Context) error)
	actionFunc(ctx)
}
