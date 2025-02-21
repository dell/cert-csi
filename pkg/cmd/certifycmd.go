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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dell/cert-csi/pkg/store"
	"github.com/dell/cert-csi/pkg/testcore/runner"
	"github.com/dell/cert-csi/pkg/testcore/suites"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/resource"
)

// CertConfig contains StorageClasses
type CertConfig struct {
	StorageClasses []Entry `yaml:"storageClasses"`
}

// Entry contains tests to be executed
type Entry struct {
	Name             string            `yaml:"name"`
	MinSize          string            `yaml:"minSize"`
	RawBlock         bool              `yaml:"rawBlock"`
	Expansion        bool              `yaml:"expansion"`
	Clone            bool              `yaml:"clone"`
	Snapshot         bool              `yaml:"snapshot"`
	RWX              bool              `yaml:"RWX"`
	RWOP             bool              `yaml:"RWOP"`
	VolumeHealth     bool              `yaml:"volumeHealth"`
	VGS              bool              `yaml:"VGS"`
	Ephemeral        *EphemeralParams  `yaml:"ephemeral"`
	CapacityTracking *CapacityTracking `yaml:"capacityTracking"`
}

// CapacityTracking contains parameters specific to Storage Capacity Tracking tests
type CapacityTracking struct {
	DriverNamespace string `yaml:"driverNamespace"`
	StorageClass    string
	VolumeSize      string
	PollInterval    time.Duration `yaml:"pollInterval"`
}

// EphemeralParams contains parameters specific to Ephemeral Volume tests
type EphemeralParams struct {
	Driver           string            `yaml:"driver"`
	FSType           string            `yaml:"fstype"`
	VolumeAttributes map[string]string `yaml:"volumeAttributes"`
}

// GetCertifyCommand returns certify CLI command
func GetCertifyCommand() cli.Command {
	certCmd := cli.Command{
		Name:     "certify",
		Usage:    "certify csi-driver",
		Category: "main",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "cert-config",
				Usage:    "path to certification config file",
				Required: true,
			},
			cli.StringFlag{
				Name:  "image-config",
				Usage: "path to images config file",
			},
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
				Name:  "sequential, sq",
				Usage: "include this flag to run the test suites sequentially",
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
				Name:  "volumeSnapshotClass, vsc",
				Usage: "define your volumeSnapshotClass",
			},
			cli.StringFlag{
				Name:  "reportPath, path",
				Usage: "path to folder where reports will be created (if not specified `~/.cert-csi/` will be used)",
			},
			cli.StringFlag{
				Name:  "driver-namespace, driver-ns",
				Usage: "specify the driver namespace to find the driver resources for the volume health metrics suite",
			},
			cli.StringFlag{
				Name:  "driver-name, driver",
				Usage: "specify the driver for volume group snapshot",
			},
			cli.StringFlag{
				Name:  "vgs-volume-label",
				Usage: "specify the volume label for VGS",
			},
			cli.StringFlag{
				Name:  "vgs-name",
				Usage: "specify the volume group name",
			},
		},
		Before: updatePath,
		Action: getAction,
	}

	return certCmd
}

func getAction(c *cli.Context) error {
	configFilePath := c.String("cert-config")
	configFilePath = filepath.Clean(configFilePath)
	// Read the YAML file
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("can't find config file: %w", err)
	}
	// Declare a variable to hold the unmarshal data
	var certConfig CertConfig
	// Unmarshal the YAML data into the struct
	err = yaml.Unmarshal(data, &certConfig)
	if err != nil {
		return fmt.Errorf("unable to decode Config: %w", err)
	}

	testImage, err := getTestImage(c.String("image-config"))
	if err != nil {
		return fmt.Errorf("failed to get test image: %s", err)
	}
	// Parse timeout
	timeout, err := time.ParseDuration(c.String("timeout"))
	if err != nil {
		return errors.New("timeout is wrong formatted")
	}
	timeOutInSeconds := int(timeout.Seconds())

	var scDBs []*store.StorageClassDB
	ss := make(map[string][]suites.Interface)

	for _, sc := range certConfig.StorageClasses {
		pathToDb := fmt.Sprintf("file:%s.db", sc.Name)
		DB := store.NewSQLiteStore(pathToDb) // dbs should be closed in suite runner

		scDBs = append(scDBs, &store.StorageClassDB{
			StorageClass: sc.Name,
			DB:           DB,
		})

		var s []suites.Interface

		minSize := "8Gi"
		if sc.MinSize != "" {
			minSize = sc.MinSize
		}

		s = append(s, &volumeio.VolumeIoSuite{
			VolumeNumber: 2,
			VolumeSize:   minSize,
			ChainNumber:  2,
			ChainLength:  2,
			Image:        testImage,
		})

		s = append(s, &suites.ScalingSuite{
			ReplicaNumber:    2,
			VolumeNumber:     5,
			GradualScaleDown: false,
			PodPolicy:        "Parallel",
			VolumeSize:       minSize,
			Image:            testImage,
		})

		if sc.Clone {
			s = append(s, &suites.CloneVolumeSuite{
				VolumeNumber: 1,
				PodNumber:    2,
				VolumeSize:   minSize,
				Image:        testImage,
			})
		}

		if sc.Expansion {
			expSize := resource.MustParse(minSize)
			expSize.Add(expSize)
			s = append(s, &suites.VolumeExpansionSuite{
				VolumeNumber: 1,
				PodNumber:    1,
				InitialSize:  minSize,
				ExpandedSize: expSize.String(),
				Image:        testImage,
			})

			if sc.RawBlock {
				s = append(s, &suites.VolumeExpansionSuite{
					VolumeNumber: 1,
					PodNumber:    1,
					IsBlock:      true,
					InitialSize:  minSize,
					ExpandedSize: expSize.String(),
					Image:        testImage,
				})
			}
		}

		if sc.Snapshot {
			snapClass := c.String("volumeSnapshotClass")
			if snapClass == "" {
				return errors.New("volume snapshot class required to verify `snapshot` capability")
			}
			s = append(s, &suites.SnapSuite{
				SnapAmount: 3,
				SnapClass:  snapClass,
				VolumeSize: minSize,
				Image:      testImage,
			})

			s = append(s, &suites.ReplicationSuite{
				VolumeNumber: 5,
				VolumeSize:   minSize,
				PodNumber:    2,
				SnapClass:    snapClass,
				Image:        testImage,
			})
		}

		if sc.RawBlock {
			s = append(s, &suites.MultiAttachSuite{
				PodNumber:  5,
				RawBlock:   true,
				AccessMode: "ReadWriteMany",
				VolumeSize: minSize,
				Image:      testImage,
			})
		}

		if sc.RWX {
			s = append(s, &suites.MultiAttachSuite{
				PodNumber:  5,
				RawBlock:   false,
				AccessMode: "ReadWriteMany",
				VolumeSize: minSize,
				Image:      testImage,
			})
		}

		if sc.VolumeHealth {
			s = append(s, &suites.VolumeHealthMetricsSuite{
				PodNumber:    1,
				VolumeNumber: 1,
				VolumeSize:   minSize,
				Namespace:    c.String("driver-namespace"),
				Image:        testImage,
			})
		}

		if sc.RWOP {
			s = append(s, &suites.MultiAttachSuite{
				PodNumber:  5,
				RawBlock:   false,
				AccessMode: "ReadWriteOncePod",
				VolumeSize: minSize,
				Image:      testImage,
			})
		}

		if sc.Ephemeral != nil {
			s = append(s, &suites.EphemeralVolumeSuite{
				Driver:           sc.Ephemeral.Driver,
				FSType:           sc.Ephemeral.FSType,
				PodNumber:        2,
				VolumeAttributes: sc.Ephemeral.VolumeAttributes,
				Image:            testImage,
			})
		}

		if sc.VGS {
			snapClass := c.String("volumeSnapshotClass")
			if snapClass == "" {
				return errors.New("volume snapshot class required to verify `snapshot` capability")
			}
			label := c.String("vgs-volume-label")
			if label == "" {
				return errors.New("vgs-volume-label required to verify volume group snapshot")
			}
			driverName := c.String("driver-name")
			if driverName == "" {
				return errors.New("driver-name required to verify volume group snapshot")
			}
			vgsName := c.String("vgs-name")
			if vgsName == "" {
				return errors.New("vgs-name required to verify volume group snapshot")
			}
			s = append(s, &suites.VolumeGroupSnapSuite{
				SnapClass:       snapClass,
				VolumeSize:      minSize,
				AccessMode:      "ReadWriteOnce",
				VolumeLabel:     label,
				ReclaimPolicy:   "Delete",
				VolumeNumber:    2,
				Driver:          driverName,
				VolumeGroupName: vgsName,
				Image:           testImage,
			})
		}
		if sc.CapacityTracking != nil {
			s = append(s, &suites.CapacityTrackingSuite{
				DriverNamespace: sc.CapacityTracking.DriverNamespace,
				StorageClass:    sc.Name,
				VolumeSize:      minSize,
				PollInterval:    sc.CapacityTracking.PollInterval,
				Image:           testImage,
			})
		}
		log.Infof("Suites to run with %s storage class:", color.CyanString(sc.Name))
		for i, suite := range s {
			log.Infof("%d. %s %s", i+1, color.HiMagentaString(suite.GetName()), suite.Parameters())
		}
		ss[sc.Name] = s
	}

	fmt.Println("Does it look OK? (Y)es/(n)o")
	readerCleanup := bufio.NewReader(os.Stdin)
	fmt.Print("-> ")
	charCleanup, _, err := readerCleanup.ReadRune()
	if err != nil {
		log.Error(err)
	}
	switch charCleanup {
	case 'n', 'N':
		log.Infof("Cancelling launch of certification")
		return nil
	}

	sr := runner.NewSuiteRunner(
		c.String("config"),
		c.String("namespace"),
		c.String("start-hook"),
		c.String("ready-hook"),
		c.String("finish-hook"),
		c.String("observer-type"),
		c.String("longevity"),
		c.String("driver-namespace"),
		timeOutInSeconds,
		0,
		c.Bool("sequential"),
		c.Bool("no-cleanup"),
		c.Bool("no-cleanup-on-fail"),
		c.Bool("no-metrics"),
		c.Bool("no-reports"),
		scDBs,
	)

	sr.RunSuites(ss)
	return nil
}
