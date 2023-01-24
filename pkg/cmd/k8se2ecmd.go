/*
 *
 * Copyright © 2023 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package cmd

import (
	"cert-csi/pkg/utils"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// GetK8sEndToEndCommand returns k8s-e2e CLI command by executing kubernetes tests
func GetK8sEndToEndCommand() cli.Command {
	e2eCmd := cli.Command{
		Name:     "k8s-e2e",
		Usage:    "k8s-e2e command to execute the kubernetes end-to-end testcases",
		Category: "main",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "driver-config",
					Usage:    "path to test driver config file",
					Required: true,
				},
				cli.StringFlag{
					Name:  "reportPath, path",
					Usage: "path to folder where reports will be created (if not specified `$HOME/reports/execution_[storage-class].xml` will be used)",
				},
				cli.StringFlag{
					Name:  "focus",
					Usage: "focus string(regx) what your k8s e2e tests should foucs on Ex: \"External.Storage.*\"",
				},
				cli.StringFlag{
					Name:  "focus-file",
					Usage: "focus file(regx) what your k8s e2e tests should focus on Ex: 'testsuitename.go'",
				},
				cli.StringFlag{
					Name:   "config, conf, c",
					Usage:  "config for connecting to kubernetes",
					EnvVar: "KUBECONFIG",
				},
				cli.StringFlag{
					Name:  "skip",
					Usage: "skip string(regx) what your k8s e2e tests should skip Ex: '\\[Feature:|\\[Disruptive\\]'",
				},
				cli.StringFlag{
					Name:  "skip-file",
					Usage: "skip file(regx) what your k8s e2e tests should skip Ex: 'testsuitename.go'",
				},
				cli.StringFlag{
					Name: "skip-tests",
					Usage: "skip unsupported tests give the file path Ex:  /root/tests/skip.yaml" +
						"ignore:\n" +
						"  - \"skip this test\"",
				},
				cli.StringFlag{
					Name:  "timeout",
					Usage: "time out for kubernetes e2e command to exit default will be 1h Ex: 2h",
				},
				cli.StringFlag{
					Name:     "version",
					Usage:    "Kubernetes version that you want to test end-to-end tests",
					Required: true,
				},
			},
		),
		Before: func(c *cli.Context) error {
			checks := utils.Prechecks(c)
			if !checks {
				return errors.New("pre-checks not met Exiting the command")
			}
			pre := utils.Prerequisites(c.String("version"))
			if pre != nil {
				return pre
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			CmdArgs, err := utils.BuildE2eCommand(ctx)
			if err != nil {
				log.Errorf("Unable to Build e2e command %s", err.Error())
				return err
			}

			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt,
				syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
				syscall.SIGINT,  // Ctrl+C
			)
			err = utils.ExecuteE2ECommand(CmdArgs, ch)
			if err != nil {
				log.Errorf("Error while executing the k8s-e2e command %s", err.Error())
				log.Errorf("NOTE: This might be because of failed tests")

			}
			fmt.Println("Generating Report:")
			utils.GenerateReport(CmdArgs[len(CmdArgs)-1])
			return nil
		},
	}
	return e2eCmd

}
