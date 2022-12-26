package cmd

import (
	"cert-csi/pkg/store"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

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
			getTestrunsCmd(),
		},
	}

	return listCmd
}

func getTestrunsCmd() cli.Command {
	const padding = 3
	return cli.Command{
		Name:      "test-runs",
		ShortName: "tr",
		Category:  "list",
		Action: func(c *cli.Context) error {
			db := store.NewSQLiteStore("file:" + c.GlobalString("db"))
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
				var nameTypes map[string]bool
				nameTypes = make(map[string]bool)
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
