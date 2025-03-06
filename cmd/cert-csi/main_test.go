package main

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func TestBefore(t *testing.T) {
	// Test case: Enable debugging level of logs
	app := cli.NewApp()
	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "debug, d",
		},
	}
	app.Run([]string{"app", "--debug"})
	if log.GetLevel() != log.DebugLevel {
		t.Errorf("Log level was not set to debug")
	}

	// Test case: Enable quiet level of logs
	app.Before = func(c *cli.Context) error {
		if c.Bool("quiet") {
			log.SetLevel(log.PanicLevel)
		}
		return nil
	}
	app.Run([]string{"app", "--quiet"})
}

func TestStartApp(t *testing.T) {
	setUp := func() {
		log.SetLevel(log.PanicLevel)
		tearDown := func() {
			log.SetLevel(log.InfoLevel)
		}
		defer tearDown()
	}
	setUp()

	// mock args for testing
	if len(os.Args) >= 1 {
		os.Args = []string{"app", "--debug"}
	}
	main()

	t.Run("quiet mode", func(_ *testing.T) {
		// mock args for testing
		if len(os.Args) >= 1 {
			os.Args = []string{"app", "--quiet"}
		}
		main()
	})
}
