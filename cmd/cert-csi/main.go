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

package main

import (
	"os"

	"github.com/dell/cert-csi/pkg/cmd"

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
	app.Name = "cert-csi"
	app.Version = "1.3.1"
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
		cmd.GetK8sEndToEndCommand(),
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
