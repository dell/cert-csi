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
	"errors"

	"github.com/dell/cert-csi/pkg/reporter"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// GetFunctionalReportCommand returns a `report` command with all prepared sub-commands for functinoal reporting
func GetFunctionalReportCommand() cli.Command {
	reportTypeFlags := []cli.Flag{
		cli.BoolFlag{
			Name:  "tabular",
			Usage: "specifies if tabular report should be generated",
		},
		cli.BoolFlag{
			Name:  "xml",
			Usage: "specifies if XML report should be generated",
		},
	}

	functionalReportCmd := cli.Command{
		Name:     "functional-report",
		Usage:    "To generate functional tests report",
		Category: "main",
		Flags:    reportTypeFlags,

		Action: func(c *cli.Context) error {
			var types []reporter.ReportType

			databaseName := c.GlobalString("db")
			if c.Bool("tabular") {
				types = append(types, reporter.TabularReport)
			}
			if c.Bool("xml") {
				types = append(types, reporter.XMLReport)
			}
			if len(types) == 0 {
				return errors.New("one type of report type is required")
			}
			// forcing user to give always DB name by overriding global default value
			if databaseName == "" || databaseName == "default.db" {
				log.Fatal("Error no database is given please add -db <database_name> to generate report!")
			}
			db := store.NewSQLiteStore("file:" + databaseName)
			defer db.Close()

			err := reporter.GenerateFunctionalReport(db, types)
			if err != nil {
				log.Errorf("Can't generate reports; error=%v", err)
				return err
			}
			return nil
		},
	}

	return functionalReportCmd
}
