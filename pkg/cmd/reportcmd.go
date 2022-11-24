package cmd

import (
	"cert-csi/pkg/plotter"
	"cert-csi/pkg/reporter"
	"cert-csi/pkg/store"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"

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
			var convertedNames []string
			for _, testRun := range testRunNames {
				db, name := parseTestRun(testRun)
				if db == "" {
					db = c.GlobalString("db")
				}
				log.Info(db, ":", name)

				convertedNames = append(convertedNames, name)

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
				multiTypes = append(multiTypes, reporter.XmlReport)
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
				types = append(types, reporter.HtmlReport)
			}
			if c.Bool("txt") {
				types = append(types, reporter.TextReport)
			}

			var err error
			if len(types) == 0 {
				err = reporter.GenerateAllReports(convertedNames, scDBs[0].DB)
			} else {
				err = reporter.GenerateReports(convertedNames, types, scDBs[0].DB)
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