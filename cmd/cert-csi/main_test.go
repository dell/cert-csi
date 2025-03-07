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
