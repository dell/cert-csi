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
	"bufio"
	"fmt"
	"github.com/dell/cert-csi/pkg/testcore/suites/multivolattach"
	"github.com/dell/cert-csi/pkg/testcore/suites/podprovision"
	"github.com/dell/cert-csi/pkg/testcore/suites/volcreation"
	"github.com/dell/cert-csi/pkg/testcore/suites/volsnap"
	"github.com/dell/cert-csi/pkg/testcore/suites/volumeclone"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/testcore/runner"
	"github.com/dell/cert-csi/pkg/testcore/suites"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// GetFunctionalTestCommand returns a `functional-test` command with all prepared sub-commands
func GetFunctionalTestCommand() cli.Command {
	globalFlags := []cli.Flag{
		cli.StringFlag{
			Name:   "config, conf, c",
			Usage:  "config for connecting to kubernetes",
			EnvVar: "KUBECONFIG",
		},
		cli.StringFlag{
			Name:  "namespace, ns",
			Usage: "specify the driver namespace (used in driver resource usage)",
		},
		cli.StringFlag{
			Name:  "timeout, t",
			Usage: "set the timeout value for all of the resources (accepts format like 2h30m15s) default is 0s",
			Value: "0s",
		},
		cli.StringFlag{
			Name:  "description, de",
			Usage: "To provide test case description",
		},
		cli.StringFlag{
			Name:  "image-config",
			Usage: "path to images config file",
		},
	}

	listCmd := cli.Command{
		Name:     "list",
		Usage:    "lists all available test suites",
		Category: "functional-test",
		Action: func(_ *cli.Context) error {
			fmt.Println("volume-creation")
			fmt.Println("pod-creation")
			fmt.Println("volume-deletion")
			fmt.Println("pod-deletion")
			fmt.Println("cloned-volume-deletion")
			fmt.Println("node-drain")
			fmt.Println("node-uncordon")
			return nil
		},
	}

	funtestCmd := cli.Command{
		Name:     "functional-test",
		Usage:    "Test csi-driver functionality",
		Category: "main",
		Subcommands: []cli.Command{
			listCmd,
			getVolumeDeletionCommand(globalFlags),
			getPodDeletionCommand(globalFlags),
			getCloneVolumeDeletionCommand(globalFlags),
			getFunctionalVolumeCreateCommand(globalFlags),
			getFunctionalProvisioningCommand(globalFlags),
			getFunctionalCloneVolumeCommand(globalFlags),
			getFunctionalSnapCreationCommand(globalFlags),
			getFunctionalSnapDeletionCommand(globalFlags),
			getFunctionalMultiAttachVolCommand(globalFlags),
			getFunctionalEphemeralCreationCommand(globalFlags),
			getNodeDrainCommand(globalFlags),
			getNodeUnCordonCommand(globalFlags),
			getCapacityTrackingCommand(globalFlags),
		},
	}

	return funtestCmd
}

func createFunctionalSuiteRunner(c *cli.Context, noCleanup, noCleanupOnFail bool) *runner.FunctionalSuiteRunner {
	// Parse timeout
	timeout, err := time.ParseDuration(c.String("timeout"))
	if err != nil {
		log.Fatal("Timeout is wrong formatted")
	}
	timeOutInSeconds := int(timeout.Seconds())

	const dbName = "cert-csi-functional"
	pathToDb := fmt.Sprintf("file:%s.db", dbName)
	DB := store.NewSQLiteStore(pathToDb) // dbs should be closed in suite runner
	scDB := &store.StorageClassDB{
		StorageClass: c.String("sc"),
		DB:           DB,
	}
	description := c.String("description")
	if description == "" {
		description = "functional-test"
	}
	logDir := "logs"
	path := logDir + "/" + description + ".txt"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		createErr := os.Mkdir(logDir, 0o750)
		if createErr != nil {
			log.Errorf("Error creating log directory")
		}
	}
	// enable logging for each suite in a separate file.
	logFile, err := os.OpenFile(filepath.Clean(path), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		log.Errorf("Error creating log file: %v", err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	return runner.NewFunctionalSuiteRunner(
		c.String("config"),
		c.String("namespace"),
		timeOutInSeconds,
		noCleanup,
		noCleanupOnFail,
		c.Bool("no-reports"),
		scDB,
	)
}

func getVolumeDeletionCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "volume-deletion",
		Usage:    "deletes a specified volume by its name",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "pvc-name, pvc-n",
					Usage:    "PVC name",
					Required: true,
				},
				cli.StringFlag{
					Name:     "pvc-namespace, pvc-nm",
					Usage:    "PVC namespace",
					Required: true,
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			sr := createFunctionalSuiteRunner(c, true, true)
			pvcName := c.String("pvc-name")
			pvcNamespace := c.String("pvc-namespace")
			desc := c.String("description")
			s := []suites.Interface{
				&suites.VolumeDeletionSuite{
					DeletionStruct: &suites.DeletionStruct{
						Name:        pvcName,
						Namespace:   pvcNamespace,
						Description: desc,
					},
				},
			}
			sr.RunFunctionalSuites(s)
			return nil
		},
	}
}

func getPodDeletionCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "pod-deletion",
		Usage:    "deletes a specified pod by its name",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "pod-name, pod-n",
					Usage:    "Pod name",
					Required: true,
				},
				cli.StringFlag{
					Name:     "pod-namespace, pod-nm",
					Usage:    "Pod namespace",
					Required: true,
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			sr := createFunctionalSuiteRunner(c, true, true)
			podName := c.String("pod-name")
			podNamespace := c.String("pod-namespace")
			desc := c.String("description")
			s := []suites.Interface{
				&suites.PodDeletionSuite{
					DeletionStruct: &suites.DeletionStruct{
						Name:        podName,
						Namespace:   podNamespace,
						Description: desc,
					},
				},
			}
			sr.RunFunctionalSuites(s)
			return nil
		},
	}
}

func getCloneVolumeDeletionCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "clone-volume-deletion",
		Usage:    "deletes a specified cloned volume by its name",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "clone-volume-name, vol-n",
					Usage:    "Volume name",
					Required: true,
				},
				cli.StringFlag{
					Name:     "resource-namespace, res-nm",
					Usage:    "Resource namespace",
					Required: true,
				},
				cli.StringFlag{
					Name:     "clone-pod-name, pod-name",
					Usage:    "Pod namespace",
					Required: true,
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			sr := createFunctionalSuiteRunner(c, true, true)
			volName := c.String("clone-volume-name")
			podName := c.String("clone-pod-name")
			namespace := c.String("resource-namespace")
			desc := c.String("description")

			s := []suites.Interface{
				&suites.ClonedVolDeletionSuite{
					DeletionStruct: &suites.DeletionStruct{
						Name:        volName,
						Namespace:   namespace,
						Description: desc,
					},
					PodName: podName,
				},
			}
			sr.RunFunctionalSuites(s)
			return nil
		},
	}
}

func getFunctionalSnapDeletionCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "snap-deletion",
		Usage:    "deletes a specified volume snapshot by its name",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "volume-snapshot-name, snap-n",
					Usage:    "Volume Snapshot name",
					Required: true,
				},
				cli.StringFlag{
					Name:     "resource-namespace, res-nm",
					Usage:    "Resource namespace",
					Required: true,
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			sr := createFunctionalSuiteRunner(c, true, true)
			snapName := c.String("volume-snapshot-name")
			namespace := c.String("resource-namespace")
			desc := c.String("description")

			s := []suites.Interface{
				&suites.SnapshotDeletionSuite{
					DeletionStruct: &suites.DeletionStruct{
						Name:        snapName,
						Namespace:   namespace,
						Description: desc,
					},
				},
			}
			sr.RunFunctionalSuites(s)
			return nil
		},
	}
}

func getFunctionalVolumeCreateCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "volume-creation",
		Usage:    "creates a specified number of volumes",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "sc, storage, storageclass",
					Usage:    "storage csi",
					Required: true,
				},
				cli.IntFlag{
					Name:  "number, n",
					Usage: "number of volumes to create",
				},
				cli.StringFlag{
					Name:  "size, s",
					Usage: "volume size to be created",
				},
				cli.StringFlag{
					Name:  "custom-name, cn",
					Usage: "PVC name only when creating one PVC",
				},
				cli.StringFlag{
					Name:  "access-mode, am",
					Usage: "volume access mode",
				},
				cli.BoolFlag{
					Name:  "block, b",
					Usage: "use raw block volume",
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			volNum := c.Int("number")
			volSize := c.String("size")
			customName := c.String("custom-name")
			accessMode := c.String("access-mode")
			desc := c.String("description")
			blockVol := c.Bool("block")

			s := []suites.Interface{
				&volcreation.VolumeCreationSuite{
					VolumeNumber: volNum,
					VolumeSize:   volSize,
					CustomName:   customName,
					AccessMode:   accessMode,
					Description:  desc,
					RawBlock:     blockVol,
				},
			}
			sr := createFunctionalSuiteRunner(c, true, true)
			sr.RunFunctionalSuites(s)

			return nil
		},
	}
}

func getFunctionalCloneVolumeCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "clone-volume",
		Usage:    "creates a specific number of volumes, these volumes are then used as source to create more volumes",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "sc, storage, storageclass",
					Usage:    "storage csi",
					Required: true,
				},
				cli.IntFlag{
					Name:  "volumeNumber, volNum, vn, v",
					Usage: "number of volumes to attach to each pod",
				},
				cli.IntFlag{
					Name:  "podNumber, podNum, pn, p",
					Usage: "number of pod to create",
				},
				cli.StringFlag{
					Name:  "pvc-name, cn",
					Usage: "custom name for volume and cloned volume to create",
				},
				cli.StringFlag{
					Name:  "pod-name, pname",
					Usage: "custom name for pod and cloned volume pod to create",
				},
				cli.StringFlag{
					Name:  "access-mode, am",
					Usage: "volume access mode",
				},
			}, globalFlags...,
		),
		Action: func(c *cli.Context) error {
			volNum := c.Int("volumeNumber")
			podNum := c.Int("podNumber")
			desc := c.String("description")
			pvcName := c.String("pvc-name")
			podName := c.String("pod-name")
			accessMode := c.String("access-mode")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}

			s := []suites.Interface{
				&volumeclone.CloneVolumeSuite{
					VolumeNumber:  volNum,
					PodNumber:     podNum,
					Description:   desc,
					CustomPvcName: pvcName,
					CustomPodName: podName,
					AccessMode:    accessMode,
					Image:         testImage,
				},
			}

			sr := createFunctionalSuiteRunner(c, true, true)
			sr.RunFunctionalSuites(s)

			return nil
		},
	}
}

func getFunctionalProvisioningCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "provisioning",
		Usage:    "creates a specified number of volumes and pods",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "sc, storage, storageclass",
					Usage:    "storage csi",
					Required: true,
				},
				cli.IntFlag{
					Name:  "volumeNumber, volNum, vn, v",
					Usage: "number of volumes to attach to each pod",
				},
				cli.IntFlag{
					Name:  "podNumber, podNum, pn, p",
					Usage: "number of pod to create",
				},
				cli.StringFlag{
					Name:  "podName, pod-cn",
					Usage: "custom pod name while creating only 1 pod",
				},
				cli.BoolFlag{
					Name:  "block, b",
					Usage: "enable raw block volume testing",
				},
				cli.StringFlag{
					Name:  "vol-access-mode, v-am",
					Usage: "volume access mode to use",
				},
				cli.BoolFlag{
					Name:  "roFlag, mount-ro",
					Usage: "enable read only mount options",
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			volNum := c.Int("volumeNumber")
			podNum := c.Int("podNumber")
			customName := c.String("podName")
			blockVol := c.Bool("block")
			desc := c.String("description")
			volAccessMode := c.String("vol-access-mode")
			roFlag := c.Bool("roFlag")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&podprovision.ProvisioningSuite{
					VolumeNumber:  volNum,
					PodNumber:     podNum,
					PodCustomName: customName,
					Description:   desc,
					RawBlock:      blockVol,
					VolAccessMode: volAccessMode,
					ROFlag:        roFlag,
					Image:         testImage,
				},
			}

			sr := createFunctionalSuiteRunner(c, true, true)
			sr.RunFunctionalSuites(s)

			return nil
		},
	}
}

func getFunctionalSnapCreationCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:      "snapshot",
		ShortName: "snap",
		Usage:     "test full snapshot pipeline",
		Category:  "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "sc, storage, storageclass",
					Usage:    "storage csi",
					Required: true,
				},
				cli.StringFlag{
					Name:  "volumeSnapshotClass, vsc",
					Usage: "define your volumeSnapshotClass",
				},
				cli.IntFlag{
					Name:  "snapshotAmount, sa",
					Usage: "define the amount of snapshots to create",
					Value: 1,
				},
				cli.StringFlag{
					Name:  "size, s",
					Usage: "volume size to be created",
				},
				cli.StringFlag{
					Name:  "snap-name, sn",
					Usage: "custom name for snap that will be used to create vol and pod",
				},
				cli.StringFlag{
					Name:  "access-mode-original-volume, am",
					Usage: "volume access mode",
				},
				cli.StringFlag{
					Name:  "access-mode-restored-volume, sam",
					Usage: "volume from snap access mode",
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			snapClass := c.String("volumeSnapshotClass")
			snapshotAmount := c.Int("snapshotAmount")
			VolumeSize := c.String("size")
			accessModeOriginal := c.String("access-mode-original-volume")
			desc := c.String("description")
			snapName := c.String("snap-name")
			accessModeRestored := c.String("access-mode-restored-volume")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&volsnap.SnapSuite{
					SnapClass:          snapClass,
					SnapAmount:         snapshotAmount,
					VolumeSize:         VolumeSize,
					Description:        desc,
					CustomSnapName:     snapName,
					AccessModeOriginal: accessModeOriginal,
					AccessModeRestored: accessModeRestored,
					Image:              testImage,
				},
			}

			sr := createFunctionalSuiteRunner(c, true, true)
			sr.RunFunctionalSuites(s)

			return nil
		},
	}
}

func getFunctionalMultiAttachVolCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:      "multi-attach-vol",
		ShortName: "mul-attach-vol",
		Usage:     "test multi-pod creation using same vol",
		Category:  "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "sc, storage, storageclass",
					Usage:    "storage csi",
					Required: true,
				},
				cli.IntFlag{
					Name:  "pods, p",
					Usage: "number of pods to create",
				},
				cli.BoolFlag{
					Name:  "block, b",
					Usage: "use raw block for multi attach test",
				},
				cli.StringFlag{
					Name:  "access-mode, am",
					Usage: "volume access mode",
					Value: "ReadWriteMany",
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			pods := c.Int("pods")
			desc := c.String("description")
			isRawBlock := c.Bool("block")
			accessMode := c.String("access-mode")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&multivolattach.MultiAttachSuite{
					PodNumber:   pods,
					RawBlock:    isRawBlock,
					Description: desc,
					AccessMode:  accessMode,
					Image:       testImage,
				},
			}

			sr := createFunctionalSuiteRunner(c, true, true)
			sr.RunFunctionalSuites(s)

			return nil
		},
	}
}

func getFunctionalEphemeralCreationCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:      "ephemeral-volume",
		ShortName: "ephemeral",
		Usage:     "Ephemeral volume testing",
		Category:  "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "driver, dr",
					Usage:    "csi driver",
					Required: true,
				},
				cli.StringFlag{
					Name:  "fs-type, fs",
					Usage: "FS Type for ephemeral inline volume",
				},
				cli.StringFlag{
					Name:     "csi-attributes, attr",
					Usage:    "CSI attributes properties file for the ephemeral volume",
					Required: true,
				},
				cli.IntFlag{
					Name:  "pods, p",
					Usage: "number of pods to create",
				},
				cli.StringFlag{
					Name:  "pod-name, pname",
					Usage: "custom name for pod and cloned volume pod to create",
				},
				cli.BoolFlag{
					Name:  "no-cleanup, nc",
					Usage: "include this flag to disable cleanup after test",
				},
				cli.BoolFlag{
					Name:  "no-cleanup-on-fail, ncof",
					Usage: "include this flag to disable cleanup on fail",
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			pods := c.Int("pods")
			driver := c.String("driver")
			desc := c.String("description")
			fsType := c.String("fs-type")
			attributesFile := c.String("csi-attributes")
			podName := c.String("pod-name")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			// We will generate volumeAttributes by reading the properties file
			volAttributes, err := readEphemeralConfig(attributesFile)
			if err != nil {
				return err
			}
			log.Debugf("Volume Attributes:%+v", volAttributes)

			s := []suites.Interface{
				&suites.EphemeralVolumeSuite{
					PodNumber:        pods,
					Driver:           driver,
					FSType:           fsType,
					VolumeAttributes: volAttributes,
					Description:      desc,
					PodCustomName:    podName,
					Image:            testImage,
				},
			}

			sr := createFunctionalSuiteRunner(c, c.Bool("no-cleanup"), c.Bool("no-cleanup-on-fail"))
			sr.RunFunctionalSuites(s)

			return nil
		},
	}
}

func readEphemeralConfig(filename string) (map[string]string, error) {
	type Config map[string]string
	config := Config{}
	if len(filename) == 0 {
		return config, nil
	}
	file, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')

		// check if the line has = sign
		// and process the line. Ignore the rest.
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				// assign the config map
				config[key] = value
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func getNodeDrainCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "node-drain",
		Usage:    "drains a specified node by its name",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "node-name, name",
					Usage:    "Node name",
					Required: true,
				},
				cli.StringFlag{
					Name:     "node-namespace",
					Usage:    "Node namespace",
					Required: true,
				},
				cli.IntFlag{
					Name:     "grace-period-seconds, grace-period",
					Usage:    "Grace period for force deletion",
					Required: false,
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			sr := createFunctionalSuiteRunner(c, true, true)
			nodeName := c.String("node-name")
			nodeNamespace := c.String("node-namespace")
			gracePeriodSeconds := c.Int("grace-period-seconds")
			desc := c.String("description")

			s := []suites.Interface{
				&suites.NodeDrainSuite{
					Name:               nodeName,
					Namespace:          nodeNamespace,
					Description:        desc,
					GracePeriodSeconds: gracePeriodSeconds,
				},
			}
			sr.RunFunctionalSuites(s)
			return nil
		},
	}
}

func getNodeUnCordonCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "node-uncordon",
		Usage:    "UnCordon a specified node by its name",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "node-name, name",
					Usage:    "Node name",
					Required: true,
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			sr := createFunctionalSuiteRunner(c, true, true)
			nodeName := c.String("node-name")
			desc := c.String("description")

			s := []suites.Interface{
				&suites.NodeUncordonSuite{
					Name:        nodeName,
					Description: desc,
				},
			}
			sr.RunFunctionalSuites(s)
			return nil
		},
	}
}

func getCapacityTrackingCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "capacity-tracking",
		Usage:    "Storage capacity tracking feature test",
		Category: "functional-test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "driverns, drns",
					Usage:    "specify the driver namespace",
					Required: true,
				},
				cli.StringFlag{
					Name:   "sc, storage, storageclass",
					Usage:  "storage csi",
					EnvVar: "STORAGE_CLASS",
				},
				cli.StringFlag{
					Name:  "volSize, vs",
					Usage: "volume size to be created",
				},
				cli.DurationFlag{
					Name:  "poll-interval, pi",
					Usage: "poll interval set for external provisioner",
					Value: 5 * time.Minute,
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			driverns := c.String("driverns")
			storageClass := c.String("sc")
			volumeSize := c.String("volSize")
			pollInterval := c.Duration("poll-interval")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.CapacityTrackingSuite{
					DriverNamespace: driverns,
					StorageClass:    storageClass,
					VolumeSize:      volumeSize,
					PollInterval:    pollInterval,
					Image:           testImage,
				},
			}

			sr := createFunctionalSuiteRunner(c, true, true)
			sr.RunFunctionalSuites(s)

			return nil
		},
	}
}
