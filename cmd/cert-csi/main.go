package main

import (
	"cert-csi/pkg/cmd"
	"os"

	"github.com/rifflock/lfshook"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func init() {
	terminal := &prefixed.TextFormatter{
		DisableColors:   false,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		ForceFormatting: true,
	}
	file := &log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	}

	log.SetFormatter(terminal)
	pathMap := lfshook.PathMap{
		log.InfoLevel:  "./info.log",
		log.ErrorLevel: "./error.log",
		log.FatalLevel: "./fatal.log",
	}
	log.AddHook(lfshook.NewHook(pathMap, file))
}

func main() {
	app := cli.NewApp()
	app.Name = "csi-cert"
	app.Version = "0.8.1"
	app.Usage = "unified method of benchmarking and certification of csi drivers"
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable debugging level of logs",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "enable quiet level of logs",
		},
		cli.StringFlag{
			Name:  "database, db",
			Usage: "provide db to use",
			Value: "default.db",
		},
	}

	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		if c.Bool("quiet") {
			log.SetLevel(log.PanicLevel)
		}
		return nil
	}

	app.Commands = []cli.Command{
		cmd.GetTestCommand(),
		cmd.GetFunctionalTestCommand(),
		cmd.GetReportCommand(),
		cmd.GetFunctionalReportCommand(),
		cmd.GetListCommand(),
		cmd.GetCleanupCommand(),
		cmd.GetCertifyCommand(),
	}
	if os.Args[len(os.Args)-1] != "--generate-bash-completion" {
		log.Infof("Starting cert-csi; ver. %v", app.Version)
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
