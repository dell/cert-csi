/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dell/cert-csi/pkg/store"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

// GetListCommand returns list CLI command
func GetListCommand() cli.Command {
	listCmd := cli.Command{
		Name:     "list",
		Usage:    "lists different data",
		Category: "main",
		Subcommands: []cli.Command{
			GetTestrunsCmd(),
		},
	}

	return listCmd
}

var GetDatabase = func(c *cli.Context) store.Store {
	return store.NewSQLiteStore("file:" + c.GlobalString("db"))
}

func GetTestrunsCmd() cli.Command {
	const padding = 3
	return cli.Command{
		Name:      "test-runs",
		ShortName: "tr",
		Category:  "list",
		Action: func(c *cli.Context) error {
			db := GetDatabase(c)
			defer db.Close()

			runs, err := db.GetTestRuns(store.Conditions{}, "", 0)
			if err != nil {
				return err
			}
			if len(runs) == 0 {
				return fmt.Errorf("can't list test runs")
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.TabIndent)
			fmt.Println("Test Runs:")

			for _, run := range runs {
				testCases, err := db.GetTestCases(store.Conditions{"run_id": run.ID}, "", 0)
				if err != nil {
					return err
				}
				if len(runs) == 0 {
					return fmt.Errorf("can't list test cases")
				}
				nameTypes := make(map[string]bool)
				var suites strings.Builder

				success := true
				for _, tc := range testCases {
					if !nameTypes[tc.Name] {
						nameTypes[tc.Name] = true
						suites.WriteString(tc.Name + " ") // #nosec
					}
					success = success && tc.Success
				}
				var result string
				if success {
					result = color.HiGreenString("SUCCESS") + " "
				} else {
					result = color.HiRedString("FAILURE") + " "
				}

				_, _ = fmt.Fprintf(w, "%v\t%s\t%s\t%d\t%s\t%s\t\n",
					color.HiBlackString(fmt.Sprintf("[%s]", run.StartTimestamp.Truncate(time.Second).Local().String())),
					color.YellowString(run.Name),
					run.StorageClass,
					len(testCases),
					suites.String(),
					result,
				)
			}

			err = w.Flush()
			if err != nil {
				return err
			}
			return nil
		},
	}
}
