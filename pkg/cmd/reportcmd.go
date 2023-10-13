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
	"fmt"
	"strings"

	"github.com/dell/cert-csi/pkg/plotter"
	"github.com/dell/cert-csi/pkg/reporter"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli"
)

// GetReportCommand returns a `report` command with all prepared sub-commands
func GetReportCommand() cli.Command {
	reportTypeFlags := []cli.Flag{
		cli.BoolFlag{
			Name:  "html",
			Usage: "specifies if html report should be generated",
		},
		cli.BoolFlag{
			Name:  "txt",
			Usage: "specifies if txt report should be generated",
		},
		cli.BoolFlag{
			Name:  "tabular",
			Usage: "specifies if tabular html report should be generated",
		},
		cli.BoolFlag{
			Name:  "xml",
			Usage: "specifies if qTest xml report should be generated",
		},
		cli.StringFlag{
			Name:  "reportPath, path",
			Usage: "path to folder where reports will be created (if not specified `~/.cert-csi/` will be used)",
		},
	}

	var testRunNames cli.StringSlice

	testRunNamesFlag := cli.StringSliceFlag{
		Name:     "testrun, tr",
		Usage:    "test run names from which reports will be generated (file.db:testrun)",
		EnvVar:   "TESTRUNNAMES",
		Required: true,
		Value:    &testRunNames,
	}

	reportCmd := cli.Command{
		Name:     "report",
		Usage:    "generate report from test run name",
		Category: "main",
		Flags:    append([]cli.Flag{testRunNamesFlag}, reportTypeFlags...),
		Action: func(c *cli.Context) error {
			var scDBs []*store.StorageClassDB
			for _, testRun := range testRunNames {
				db, name := parseTestRun(testRun)
				if db == "" {
					db = c.GlobalString("db")
				}
				log.Info(db, ":", name)

				pathToDb := fmt.Sprintf("file:%s", db)
				DB := store.NewSQLiteStore(pathToDb)
				scDBs = append(scDBs, &store.StorageClassDB{
					DB: DB,
					TestRun: store.TestRun{
						Name: name,
					},
				})
			}

			defer func() {
				for _, scDB := range scDBs {
					_ = scDB.DB.Close()
				}
			}()

			if c.String("path") != "" {
				plotter.UserPath = c.String("path")
				plotter.FolderPath = ""
			}

			var multiTypes []reporter.ReportType
			if c.Bool("xml") {
				multiTypes = append(multiTypes, reporter.XMLReport)
			}

			if c.Bool("tabular") {
				multiTypes = append(multiTypes, reporter.TabularReport)
			}

			if len(multiTypes) != 0 {
				err := reporter.GenerateReportsFromMultipleDBs(multiTypes, scDBs)
				if err != nil {
					return err
				}
			}

			var types []reporter.ReportType
			if c.Bool("html") {
				types = append(types, reporter.HTMLReport)
			}
			if c.Bool("txt") {
				types = append(types, reporter.TextReport)
			}

			var err error
			if len(types) == 0 {
				err = reporter.GenerateAllReports(scDBs)
			} else {
				err = reporter.GenerateReports(types, scDBs)
			}

			if err != nil {
				log.Errorf("Can't generate reports; error=%v", err)
				return err
			}
			return nil
		},
	}

	return reportCmd
}

func parseTestRun(tr string) (db string, name string) {
	s := strings.Split(tr, ":")
	if len(s) == 1 {
		name = s[0]
		db = ""
	} else {
		name = s[1]
		db = s[0]
	}
	return
}
