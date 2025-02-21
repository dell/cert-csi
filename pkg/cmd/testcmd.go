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
	"fmt"
	"time"

	"github.com/dell/cert-csi/pkg/plotter"
	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/testcore"
	"github.com/dell/cert-csi/pkg/testcore/runner"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// GetTestCommand returns a `test` command with all prepared sub-commands
func GetTestCommand() cli.Command {
	globalFlags := []cli.Flag{
		cli.StringFlag{
			Name:   "config, conf, c",
			Usage:  "config for connecting to kubernetes",
			EnvVar: "KUBECONFIG",
		},
		cli.StringSliceFlag{
			Name:     "sc, storage, storageclass",
			Usage:    "storage csi",
			EnvVar:   "STORAGE_CLASS",
			Required: true,
		},
		cli.StringFlag{
			Name:  "namespace, ns",
			Usage: "specify the driver namespace (used in driver resource usage)",
		},
		cli.StringFlag{
			Name: "longevity, long, l",
			Usage: "launch in longevity mode with provided number of iterations. Accepted formats are:" +
				" number of iterations (ex. 10) or time (ex. 3d.2h30m15s)",
			Value: "1",
		},
		cli.StringFlag{
			Name:  "timeout, t",
			Usage: "set the timeout value for all of the resources (accepts format like 2h30m15s) default is 0s",
			Value: "0s",
		},
		cli.BoolFlag{
			Name:  "no-cleanup, nc",
			Usage: "include this flag do disable cleanup between iterations",
		},
		cli.BoolFlag{
			Name:  "no-cleanup-on-fail, ncof",
			Usage: "include this flag do disable cleanup on fail",
		},
		cli.StringFlag{
			Name:  "start-hook, sh",
			Usage: "specify the path to the start-hook",
		},
		cli.StringFlag{
			Name:  "ready-hook, rh",
			Usage: "specify the path to the ready-hook",
		},
		cli.StringFlag{
			Name:  "finish-hook, fh",
			Usage: "specify the path to the finish-hook",
		},
		cli.BoolFlag{
			Name:  "no-metrics, nm",
			Usage: "include this flag to disable event-based performance metrics, thus creating only-load scenario",
		},
		cli.BoolFlag{
			Name:  "no-reports, nr",
			Usage: "include this flag to skip report generating stage",
		},
		cli.StringFlag{
			Name:  "observer-type, ot",
			Usage: "set the observer type to use [event] or [list]",
			Value: "event",
		},
		cli.StringFlag{
			Name:  "reportPath, path",
			Usage: "path to folder where reports will be created (if not specified `~/.cert-csi/` will be used)",
		},
		cli.StringFlag{
			Name:  "cooldown, cd",
			Usage: "set to add cooldown time between iterations, format is time (ex. 3d.2h30m15s)",
			Value: "0s",
		},
		cli.StringFlag{
			Name:  "driver-namespace, driver-ns",
			Usage: "specify the driver namespace to find the driver resources for the volume health metrics suite",
		},
		cli.StringFlag{
			Name:  "image-config",
			Usage: "path to images config file",
		},
	}

	testCmd := cli.Command{
		Name:     "test",
		Usage:    "test csi-driver",
		Category: "main",
		Subcommands: []cli.Command{
			getVolumeCreationCommand(globalFlags),
			getProvisioningCommand(globalFlags),
			getScalingCommand(globalFlags),
			getVolumeIoCommand(globalFlags),
			getSnapCommand(globalFlags),
			getVolumeGroupSnapCommand(globalFlags),
			getReplicationCommand(globalFlags),
			getCloneVolumeCommand(globalFlags),
			getMultiAttachVolCommand(globalFlags),
			getVolumeExpansionCommand(globalFlags),
			getVolumeHealthMetricsCommand(globalFlags),
			getBlockSnapCommand(globalFlags),
			getPostgresCommand(globalFlags),
			getRemoteReplicationProvisioningCommand(globalFlags),
			getVolumeMigrateCommand(globalFlags),
			getEphemeralCreationCommand(globalFlags),
		},
	}

	return testCmd
}

func createSuiteRunner(c *cli.Context, s []suites.Interface) (*runner.SuiteRunner, map[string][]suites.Interface) {
	// Parse timeout
	timeout, err := time.ParseDuration(c.String("timeout"))
	if err != nil {
		log.Fatal("Timeout is wrong formatted")
	}
	timeOutInSeconds := int(timeout.Seconds())

	// Parse cooldown time
	cooldown, err := time.ParseDuration(c.String("cooldown"))
	if err != nil {
		log.Fatal("Cooldown is wrong formatted")
	}
	cooldownInSeconds := int(cooldown.Seconds())

	var scDBs []*store.StorageClassDB
	ss := make(map[string][]suites.Interface)
	for _, sc := range c.StringSlice("sc") {
		pathToDb := fmt.Sprintf("file:%s.db", sc)
		DB := store.NewSQLiteStore(pathToDb) // dbs should be closed in suite runner
		scDBs = append(scDBs, &store.StorageClassDB{
			StorageClass: sc,
			DB:           DB,
		})
		ss[sc] = s
	}
	return runner.NewSuiteRunner(
		c.String("config"),
		c.String("namespace"),
		c.String("start-hook"),
		c.String("ready-hook"),
		c.String("finish-hook"),
		c.String("observer-type"),
		c.String("longevity"),
		c.String("driver-namespace"),
		timeOutInSeconds,
		cooldownInSeconds,
		c.Bool("sequential"),
		c.Bool("no-cleanup"),
		c.Bool("no-cleanup-on-fail"),
		c.Bool("no-metrics"),
		c.Bool("no-reports"),
		scDBs,
	), ss
}

func updatePath(c *cli.Context) error {
	if c.String("path") != "" {
		plotter.UserPath = c.String("path")
		plotter.FolderPath = ""
	}
	return nil
}

func readImageConfig(configFilePath string) (testcore.Images, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFilePath)

	err := viper.ReadInConfig()
	if err != nil {
		return testcore.Images{}, fmt.Errorf("can't find image config file: %w", err)
	}

	var imageConfig testcore.Images
	err = viper.Unmarshal(&imageConfig)
	if err != nil {
		return testcore.Images{}, fmt.Errorf("unable to decode image Config: %s", err)
	}

	return imageConfig, nil
}

func getTestImage(imageConfigPath string) (string, error) {
	var testImage string
	if imageConfigPath != "" {
		img, err := readImageConfig(imageConfigPath)
		if err != nil {
			return "", fmt.Errorf("failed to find image Config: %s", err)
		}
		for _, img := range img.Images {
			testImage = img.Test
		}
	}
	return testImage, nil
}

// *************
// Test commands
// *************
func getVolumeCreationCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "volume-creation",
		Usage:    "creates a specified number of volumes",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "number, n",
					Usage: "number of volumes to create",
				},
				cli.StringFlag{
					Name:  "size, s",
					Usage: "volume size to be created",
				},
				cli.StringFlag{
					Name:  "access-mode, am",
					Usage: "volume access mode",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			volNum := c.Int("number")
			volSize := c.String("size")
			accessMode := c.String("access-mode")

			s := []suites.Interface{
				&suites.VolumeCreationSuite{
					VolumeNumber: volNum,
					VolumeSize:   volSize,
					AccessMode:   accessMode,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getVolumeMigrateCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "volume-migrate",
		Usage:    "migrates volumes",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:     "target-sc",
					Usage:    "storageclass to which we want to migrate volumes",
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
					Name:  "short",
					Usage: "provide this flag if you want short version of the test without deleting old sts and creating new sts",
				},
			}, globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			targetSC := c.String("target-sc")
			volNum := c.Int("volumeNumber")
			podNum := c.Int("podNumber")
			flag := c.Bool("short")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.VolumeMigrateSuite{
					TargetSC:     targetSC,
					VolumeNumber: volNum,
					PodNumber:    podNum,
					Flag:         flag,
					Image:        testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getRemoteReplicationProvisioningCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "replication-provisioning",
		Usage:    "creates a specified number of RemoteReplication volumes and pods",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "volumeNumber, volNum, vn, v",
					Usage: "number of volumes to attach to each pod",
				},
				cli.StringFlag{
					Name:     "remote-config-path, rcp",
					Usage:    "Config file path for remote cluster",
					Required: true,
				},
				cli.BoolFlag{
					Name:  "no-failover",
					Usage: "set to `true` if you don't want to execute failover/reprotect actions",
				},
				cli.StringFlag{
					Name:  "volumeSize, volSize",
					Usage: "volume size to be created",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			volNum := c.Int("volumeNumber")
			remoteConfigPath := c.String("remote-config-path")
			noFailover := c.Bool("no-failover")
			volSize := c.String("volumeSize")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.RemoteReplicationProvisioningSuite{
					VolumeNumber:     volNum,
					RemoteConfigPath: remoteConfigPath,
					NoFailover:       noFailover,
					VolumeSize:       volSize,
					Image:            testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getReplicationCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "replication",
		Usage:    "creates a specific number of volumes and snapshots from them, the snapshots are then used to create more volumes",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "volumeNumber, volNum, vn, v",
					Usage: "number of volumes to attach to each pod",
				},
				cli.IntFlag{
					Name:  "podNumber, podNum, pn, p",
					Usage: "number of pod to create",
				},
				cli.StringFlag{
					Name:  "volumeSnapshotClass, vsc",
					Usage: "define your volumeSnapshotClass",
				},
				cli.StringFlag{
					Name:  "volumeSize, volSize",
					Usage: "volume size to be created",
				},
			}, globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			volNum := c.Int("volumeNumber")
			podNum := c.Int("podNumber")
			snapClass := c.String("volumeSnapshotClass")
			volSize := c.String("volumeSize")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.ReplicationSuite{
					VolumeNumber: volNum,
					PodNumber:    podNum,
					VolumeSize:   volSize,
					SnapClass:    snapClass,
					Image:        testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)
			return nil
		},
	}
}

func getCloneVolumeCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "clone-volume",
		Usage:    "creates a specific number of volumes, these volumes are then used as source to create more volumes",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "volumeNumber, volNum, vn, v",
					Usage: "number of volumes to attach to each pod",
				},
				cli.IntFlag{
					Name:  "podNumber, podNum, pn, p",
					Usage: "number of pod to create",
				},
			}, globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			volNum := c.Int("volumeNumber")
			podNum := c.Int("podNumber")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.CloneVolumeSuite{
					VolumeNumber: volNum,
					PodNumber:    podNum,
					Image:        testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getMultiAttachVolCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "multi-attach-vol",
		Usage:    "creates a volume, mounts it to different pods, writes and checks hashes",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "podNumber, podNum, pn, p",
					Usage: "number of pod to create",
					Value: 2,
				},
				cli.BoolFlag{
					Name:  "block, b",
					Usage: "use raw block vol for multi attach",
				},
				cli.StringFlag{
					Name:  "access-mode, am",
					Usage: "volume access mode",
					Value: "ReadWriteMany",
				},
			}, globalFlags...,
		),
		Action: func(c *cli.Context) error {
			podNum := c.Int("podNumber")
			rawBlock := c.Bool("block")
			accessMode := c.String("access-mode")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.MultiAttachSuite{
					PodNumber:  podNum,
					RawBlock:   rawBlock,
					AccessMode: accessMode,
					Image:      testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getVolumeExpansionCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "expansion",
		Usage:    "creates a specific number of volumes on the specific number of pods and then expands the volumes",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "volumeNumber, volNum, vn, v",
					Usage: "number of volumes to attach to each pod",
				},
				cli.IntFlag{
					Name:  "podNumber, podNum, pn, p",
					Usage: "number of pod to create",
				},
				cli.StringFlag{
					Name:  "intialSize, iSize",
					Usage: "Initial size of the volumes to be created",
					Value: "3Gi",
				},
				cli.StringFlag{
					Name:  "expandedSize, expSize",
					Usage: "Size to expand the volumes to",
					Value: "6Gi",
				},
				cli.BoolFlag{
					Name:  "block, b",
					Usage: "create a block device",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			volNum := c.Int("volumeNumber")
			podNum := c.Int("podNumber")
			isBlock := c.Bool("block")
			initialSize := c.String("intialSize")
			expandedSize := c.String("expandedSize")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}

			s := []suites.Interface{
				&suites.VolumeExpansionSuite{
					VolumeNumber: volNum,
					PodNumber:    podNum,
					IsBlock:      isBlock,
					InitialSize:  initialSize,
					ExpandedSize: expandedSize,
					Image:        testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getVolumeHealthMetricsCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "volumehealthmetrics",
		Usage:    "test suites executed for the volume health metric changes",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "volumeNumber, volNum, vn, v",
					Usage: "number of volumes to attach to each pod",
				},
				cli.IntFlag{
					Name:  "podNumber, podNum, pn, p",
					Usage: "number of pod to create",
				},
				cli.StringFlag{
					Name:  "volumeSize, volSize",
					Usage: "Size of the volumes to be created",
					Value: "3Gi",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			volNum := c.Int("volumeNumber")
			podNum := c.Int("podNumber")
			volSize := c.String("volumeSize")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.VolumeHealthMetricsSuite{
					VolumeNumber: volNum,
					PodNumber:    podNum,
					VolumeSize:   volSize,
					Namespace:    c.String("driver-namespace"),
					Image:        testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getProvisioningCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "provisioning",
		Usage:    "creates a specified number of volumes and pods",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "volumeNumber, volNum, vn, v",
					Usage: "number of volumes to attach to each pod",
				},
				cli.IntFlag{
					Name:  "podNumber, podNum, pn, p",
					Usage: "number of pod to create",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			volNum := c.Int("volumeNumber")
			podNum := c.Int("podNumber")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.ProvisioningSuite{
					VolumeNumber: volNum,
					PodNumber:    podNum,
					Image:        testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getScalingCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "scaling",
		Usage:    "creates sts, then scales it to specified number of replicas, and then scales it down to 0",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "replicas, repNum, rn, r",
					Usage: "number of statefulset replicas",
					Value: 5,
				},
				cli.IntFlag{
					Name:  "volumeNumber, volNum, v",
					Usage: "number of volume to attach to each replica",
					Value: 10,
				},
				cli.BoolFlag{
					Name:  "gradual, g",
					Usage: "enable gradual scale down of replicas",
				},
				cli.StringFlag{
					Name:  "podPolicy, pp",
					Usage: "change pod policy of the statefulset",
					Value: "Parallel",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			repNum := c.Int("replicas")
			volNum := c.Int("volumeNumber")
			gradual := c.Bool("gradual")
			podPolicy := c.String("podPolicy")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.ScalingSuite{
					ReplicaNumber:    repNum,
					VolumeNumber:     volNum,
					GradualScaleDown: gradual,
					PodPolicy:        podPolicy,
					Image:            testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getVolumeIoCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:      "volumeio",
		ShortName: "vio",
		Usage:     "test volume io",
		Category:  "test",
		Flags: append(
			[]cli.Flag{
				cli.IntFlag{
					Name:  "chainNumber, cn",
					Usage: "number of parallel chains",
				},
				cli.IntFlag{
					Name:  "chainLength, cl",
					Usage: "length of a chain (number of pods to be created)",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			chNumber := c.Int("chainNumber")
			chLength := c.Int("chainLength")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&volumeio.VolumeIoSuite{
					ChainNumber: chNumber,
					ChainLength: chLength,
					Image:       testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getSnapCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:      "snapshot",
		ShortName: "snap",
		Usage:     "test full snapshot pipeline",
		Category:  "test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:  "volumeSnapshotClass, vsc",
					Usage: "define your volumeSnapshotClass",
				},
				cli.IntFlag{
					Name:  "snapshotAmount, sa",
					Usage: "define the amount of snapshots to create",
					Value: 3,
				},
				cli.StringFlag{
					Name:  "size, s",
					Usage: "volume size to be created",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			snapClass := c.String("volumeSnapshotClass")
			snapshotAmount := c.Int("snapshotAmount")
			size := c.String("size")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.SnapSuite{
					SnapClass:  snapClass,
					SnapAmount: snapshotAmount,
					VolumeSize: size,
					Image:      testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getVolumeGroupSnapCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:      "volume-group-snapshot",
		ShortName: "vgs",
		Usage:     "test volume group snapshot",
		Category:  "test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:  "volumeSnapshotClass, vsc",
					Usage: "define your volumeSnapshotClass",
				},
				cli.StringFlag{
					Name:  "volumeSize",
					Usage: "volume size to be created",
				},
				cli.StringFlag{
					Name:  "volumeLabel",
					Usage: "label to be added for the volumes",
				},
				cli.StringFlag{
					Name:  "volumeGroupName",
					Usage: "name of the volume group/snapshot",
				},
				cli.StringFlag{
					Name:  "driver",
					Usage: "driver name to be used",
				},
				cli.StringFlag{
					Name:  "reclaimPolicy",
					Usage: "set the member reclaim policy, Delete, Retain",
				},
				cli.StringFlag{
					Name:  "accessMode",
					Usage: "set the volume access mode",
				},
				cli.IntFlag{
					Name:  "volumeNumber",
					Usage: "number of volume to create for the group",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			snapClass := c.String("volumeSnapshotClass")
			volSize := c.String("volumeSize")
			volumeLabel := c.String("volumeLabel")
			vgsName := c.String("volumeGroupName")
			reclaimPolicy := c.String("reclaimPolicy")
			accessMode := c.String("accessMode")
			numberOfVols := c.Int("volumeNumber")
			driver := c.String("driver")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.VolumeGroupSnapSuite{
					SnapClass:       snapClass,
					VolumeSize:      volSize,
					AccessMode:      accessMode,
					VolumeGroupName: vgsName,
					VolumeLabel:     volumeLabel,
					ReclaimPolicy:   reclaimPolicy,
					VolumeNumber:    numberOfVols,
					Driver:          driver,
					Image:           testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getBlockSnapCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:      "blocksnap",
		ShortName: "bs",
		Usage:     "mount filesystem, write data, take snap, mount as raw block and check the data",
		Category:  "test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:  "volumeSnapshotClass, vsc",
					Usage: "define your volumeSnapshotClass",
				},
				cli.StringFlag{
					Name:  "size, s",
					Usage: "volume size to be created",
				},
				cli.StringFlag{
					Name:  "access-mode, am",
					Usage: "volume access mode",
				},
			},
			globalFlags...,
		),
		Before: updatePath,
		Action: func(c *cli.Context) error {
			snapClass := c.String("volumeSnapshotClass")
			size := c.String("size")
			accessMode := c.String("access-mode")
			testImage, err := getTestImage(c.String("image-config"))
			if err != nil {
				return fmt.Errorf("failed to get test image: %s", err)
			}
			s := []suites.Interface{
				&suites.BlockSnapSuite{
					SnapClass:  snapClass,
					VolumeSize: size,
					AccessMode: accessMode,
					Image:      testImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getPostgresCommand(globalFlags []cli.Flag) cli.Command {
	return cli.Command{
		Name:     "psql",
		Usage:    "install postgresql with helm and run benchmark",
		Category: "test",
		Flags: append(
			[]cli.Flag{
				cli.StringFlag{
					Name:  "size, s",
					Usage: "size of volumes to be created by helm chart",
					Value: "32Gi",
				},
				cli.BoolFlag{
					Name:  "replication, r",
					Usage: "enable postresql replication",
				},
				cli.IntFlag{
					Name:  "slave-replicas, sr",
					Usage: "number of slave replicas to be created",
					Value: 2,
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			size := c.String("size")
			replication := c.Bool("replication")
			slaves := c.Int("slave-replicas")
			imageConfigPath := c.String("image-config")
			var postgresImage string
			if imageConfigPath != "" {
				img, err := readImageConfig(imageConfigPath)
				if err != nil {
					return fmt.Errorf("failed to find image Config: %s", err)
				}
				for _, img := range img.Images {
					postgresImage = img.Postgres
				}
			}
			s := []suites.Interface{
				&suites.PostgresqlSuite{
					ConfigPath:        c.String("config"),
					VolumeSize:        size,
					EnableReplication: replication,
					SlaveReplicas:     slaves,
					Image:             postgresImage,
				},
			}

			sr, ss := createSuiteRunner(c, s)
			sr.RunSuites(ss)

			return nil
		},
	}
}

func getEphemeralCreationCommand(globalFlags []cli.Flag) cli.Command {
	// remove sc flag
	var scIndex int
	for i, flag := range globalFlags {
		if flag.GetName() == "sc, storage, storageclass" {
			scIndex = i
			break
		}
	}
	globalFlags = append(globalFlags[:scIndex], globalFlags[scIndex+1:]...)
	return cli.Command{
		Name:      "ephemeral-volume",
		ShortName: "ephemeral",
		Usage:     "Ephemeral volume testing",
		Category:  "test",
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
				cli.IntFlag{
					Name:  "pods, p",
					Usage: "number of pods to create",
				},
				cli.StringFlag{
					Name:  "pod-name, pname",
					Usage: "custom name for pod and cloned volume pod to create",
				},
				cli.StringFlag{
					Name:     "csi-attributes, attr",
					Usage:    "CSI attributes properties file for the ephemeral volume",
					Required: true,
				},
			},
			globalFlags...,
		),
		Action: func(c *cli.Context) error {
			pods := c.Int("pods")
			driver := c.String("driver")
			desc := c.String("description")
			fsType := c.String("fs-type")
			podName := c.String("pod-name")
			attributesFile := c.String("csi-attributes")
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
