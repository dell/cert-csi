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

package cmd

import (
	"flag"
	"os"
	"testing"

	"github.com/urfave/cli"

	"github.com/dell/cert-csi/pkg/testcore/runner"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	"github.com/stretchr/testify/assert"
)

func TestGetTestCommand(t *testing.T) {
	cmd := GetTestCommand()
	if cmd.Name != "test" {
		t.Errorf("expected command name 'test', got '%s'", cmd.Name)
	}
}

func TestGetTestImage(t *testing.T) {
	type args struct {
		imageConfigPath string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getTestImage(tt.args.imageConfigPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTestImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getTestImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadImageConfig(t *testing.T) {
	// Test case: Valid config file path
	t.Run("No config file path", func(t *testing.T) {
		configFilePath := "../k8sclient/testdata/config.yaml"
		_, err := readImageConfig(configFilePath)
		assert.Error(t, err)
	})
	// Test case: Invalid config file path
	t.Run("Empty config file path to read", func(t *testing.T) {
		configFilePath := "../k8sclient/testdata/empty_imageconfig.yaml"
		_, err := readImageConfig(configFilePath)
		assert.NoError(t, err)
	})
}

func Test_getTestImage(t *testing.T) {
	imageConfigPath := "../k8sclient/testdata/empty_imageconfig.yaml"
	_, err := getTestImage(imageConfigPath)
	assert.NoError(t, err)
}

func TestGetVolumeCreationCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("number", 5, "number of volumes to create")
	set.String("size", "3Gi", "volume size to be created")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("access-mode", "ReadWriteOnce", "volume access mode")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeCreationCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetVolumeMigrateCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.String("target-sc", "test-sc", "target storage class")
	set.Int("volumeNumber", 5, "number of volumes to migrate")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("short", "flag", "provide this flag if you want short version of the test without deleting old sts and creating new sts")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeMigrateCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetRemoteReplicationProvisioningCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.String("remote-config-path", "../k8sclient/testdata/config.yaml", "Config file path for remote cluster")
	set.Int("volumeNumber", 5, "number of volumes to replicate")
	set.Bool("no-failover", false, "set to `true` if you don't want to execute failover/reprotect actions")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("volumeSize", "3Gi", "volume size to be created")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getRemoteReplicationProvisioningCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetReplicationCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to replicate")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("volumeSnapshotClass", "test-vsc", "volumeSnapshotClass to be used")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("volumeSize", "3Gi", "volume size to be created")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getReplicationCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetCloneVolumeCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to clone")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getCloneVolumeCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetMultiAttachVolCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Bool("block", true, "specifies if block volume should be created")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("access-mode", "ReadWriteOnce", "volume access mode")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getMultiAttachVolCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetVolumeExpansionCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to clone")
	set.Int("podNumber", 5, "number of pods to create")
	set.Bool("block", true, "specifies if block volume should be created")
	set.String("initialSize", "3Gi", "Initial size of the volumes to be created")
	set.String("expandedSize", "5Gi", "Expanded size of the volumes to be created")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeExpansionCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetVolumeHealthMetricsCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to clone")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("volumeSize", "3Gi", "volume size to be created")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeHealthMetricsCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetProvisioningCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to clone")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getProvisioningCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetScalingCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("replicas", 5, "number of statefulset replicas")
	set.Int("podNumber", 5, "number of volume to attach to each replica")
	set.Bool("gradual", true, "set to `true` if you want gradual scaling")
	set.String("podPolicy", "Parallel", "change pod policy of the statefulset")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("observer-type", "event", "observer type")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getScalingCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetVolumeIoCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("chainNumber", 5, "number of parallel chains")
	set.Int("chainLength", 5, "")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "length of a chain (number of pods to be created)")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeIoCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetSnapCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Int("snapshotAmopunt", 3, "define the amount of snapshots to create")
	set.String("volumeSnapshotClass", "test-vsc", "volumeSnapshotClass to be used")
	set.String("size", "3Gi", "volume size to be created")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getSnapCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetBlockSnapCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.String("volumeSnapshotClass", "test-vsc", "volumeSnapshotClass to be used")
	set.String("size", "3Gi", "volume size to be created")
	set.String("access-mode", "ReadWriteOnce", "volume access mode")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getBlockSnapCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetPostgresCommand(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	set := flag.NewFlagSet("test", 0)
	set.Bool("replication", true, "set to `true` if you want to enable replication")
	set.String("size", "3Gi", "volume size to be created")
	set.Int("slave-replicas", 2, "number of slave replicas")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("observer-type", "event", "observer type")
	ctx := cli.NewContext(nil, set, nil)
	command := getPostgresCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteRunCmdSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	actionFunc(ctx)
}

func TestGetEphemeralCreationCommandAction(t *testing.T) {
	// Default context
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)
	tempFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatalf("Could not create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	configData := `
	key1=value1
	key2=value2
	`
	if _, err := tempFile.WriteString(configData); err != nil {
		t.Fatalf("Could not write to temporary file: %v", err)
	}
	set := flag.NewFlagSet("test", 0)
	set.String("config", kubeConfig, "config for connecting to kubernetes")
	set.String("driver", "test-driver", "name of the driver")
	set.String("fs-type", "ext4", "FS Type for ephemeral inline volume")
	set.Int("pods", 1, "number of pods to create")
	set.String("pod-name", "test-pod", "custom name for pod and cloned volume pod to create")
	set.String("csi-attributes", tempFile.Name(), "CSI attributes for ephemeral volume")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	set.Bool("no-cleanup", true, "set to `true` if you want ephemeral volumes to be deleted")
	ctx := cli.NewContext(nil, set, nil)
	flagName := []cli.Flag{
		&cli.StringFlag{
			Name: "sc",
		},
	}
	command := getEphemeralCreationCommand(flagName)
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	ExecuteFuncSuite = func(_ *runner.FunctionalSuiteRunner, _ []suites.Interface) {}
	actionFunc(ctx)
}
