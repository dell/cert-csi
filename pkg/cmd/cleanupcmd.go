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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dell/cert-csi/pkg/k8sclient"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCleanupCommand returns cleanup CLI command
func GetCleanupCommand() cli.Command {
	globalFlags := []cli.Flag{
		cli.StringFlag{
			Name:   "config, conf, c",
			Usage:  "config for connecting to kubernetes",
			EnvVar: "KUBECONFIG",
		},
		cli.StringFlag{
			Name:  "timeout, t",
			Usage: "set the timeout value for all of the resources (accepts format like 2h30m15s) default is 30s",
			Value: "30s",
		},
		cli.BoolFlag{
			Name:  "yes, y",
			Usage: "include this flag to auto approve cleanup cmd. Could be useful if you are running cert-csi from non-interactive environment",
		},
	}
	cleanupCmd := cli.Command{
		Name:     "cleanup",
		Usage:    "cleanups all existing namespaces and resources, that can be created by cert-csi",
		Category: "main",
		Flags:    globalFlags,
		Action: func(c *cli.Context) error {
			fmt.Println("*** THIS WILL DELETE ALL NAMESPACES AND RESOURCES THAT HAVE A \"-test-\" or \"-suite-\" IN THEIR NAMES ***")
			fmt.Println("Are you sure (y/N)")
			if !c.Bool("yes") {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("-> ")
				char, _, err := reader.ReadRune()
				if err != nil {
					log.Error(err)
				}

				if !(char == 'y' || char == 'Y') {
					fmt.Println("Exiting...")
					return nil
				}
			}
			// Loading config
			config, err := k8sclient.GetConfig(c.String("config"))
			if err != nil {
				return err
			}
			// Connecting to host and creating new Kubernetes Client
			tout, err := time.ParseDuration(c.String("timeout"))
			if err != nil {
				log.Fatal("Timeout is wrong formatted")
			}
			toutsec := int(tout.Seconds())
			kubeClient, kubeErr := k8sclient.NewKubeClient(config, toutsec)
			if kubeErr != nil {
				log.Errorf("Couldn't create new kubernetes client. Error = %v", kubeErr)
				return kubeErr
			}

			nsList, nsErr := kubeClient.GetClientSet().CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
			if nsErr != nil {
				return nsErr
			}

			for _, ns := range nsList.Items {
				if strings.Contains(ns.Name, "-test-") || strings.Contains(ns.Name, "-suite-") {
					log.Infof("Deleting namespace %s", ns.Name)
					// kubeClient.SetTimeout(1)
					err := kubeClient.DeleteNamespace(context.Background(), ns.Name)
					if err != nil {
						log.Errorf("Can't delete namespace %s; error=%v", ns.Name, err)
					}
				}
			}
			log.Infof("No suitable namespaces left")
			return nil
		},
	}

	return cleanupCmd
}
