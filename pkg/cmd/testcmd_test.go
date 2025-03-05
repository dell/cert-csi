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

/*
	func Test_readImageConfig(t *testing.T) {
		//type args struct {
		//	configFilePath string
		//}
		tests := []struct {
			name        string
			configFile  string
			expected    testcore.Images
			expectedErr error
		}{
			// TODO: Add test cases.
			{
				name:       "Valid config file",
				configFile: "../k8sclient/testdata/config.yaml",
				expected: testcore.Images{
					Images: []testcore.Image{
						{
							Test:     "test-image-1",
							Postgres: "postgres-image-1",
						},
					},
				},
				expectedErr: nil,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := readImageConfig(tt.configFile)
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("readImageConfig() = %v, want %v", got, tt.expected)
				}
				if !reflect.DeepEqual(got, tt.expectedErr) {
					t.Errorf("readImageConfig() = %v, want %v", err, tt.expectedErr)
				}
			})
		}
	}
*/
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

func TestGetVolumeCreationCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("number", 5, "number of volumes to create")
	set.String("size", "3Gi", "volume size to be created")
	set.String("access-mode", "ReadWriteOnce", "volume access mode")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeCreationCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetVolumeMigrateCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("target-sc", "test-sc", "target storage class")
	set.Int("volumeNumber", 5, "number of volumes to migrate")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("short", "flag", "provide this flag if you want short version of the test without deleting old sts and creating new sts")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeMigrateCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetRemoteReplicationProvisioningCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("remote-config-path", "../k8sclient/testdata/config.yaml", "Config file path for remote cluster")
	set.Int("volumeNumber", 5, "number of volumes to replicate")
	set.Bool("no-failover", false, "set to `true` if you don't want to execute failover/reprotect actions")
	set.String("volumeSize", "3Gi", "volume size to be created")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getRemoteReplicationProvisioningCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetReplicationCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to replicate")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("volumeSnapshotClass", "test-vsc", "volumeSnapshotClass to be used")
	set.String("volumeSize", "3Gi", "volume size to be created")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getReplicationCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetCloneVolumeCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to clone")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getCloneVolumeCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetMultiAttachVolCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Bool("block", true, "specifies if block volume should be created")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("access-mode", "ReadWriteOnce", "volume access mode")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getMultiAttachVolCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetVolumeExpansionCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to clone")
	set.Int("podNumber", 5, "number of pods to create")
	set.Bool("block", true, "specifies if block volume should be created")
	set.String("initialSize", "3Gi", "Initial size of the volumes to be created")
	set.String("expandedSize", "5Gi", "Expanded size of the volumes to be created")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeExpansionCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetVolumeHealthMetricsCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to clone")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("volumeSize", "3Gi", "volume size to be created")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeHealthMetricsCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetProvisioningCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("volumeNumber", 5, "number of volumes to clone")
	set.Int("podNumber", 5, "number of pods to create")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getProvisioningCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetScalingCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("replicas", 5, "number of statefulset replicas")
	set.Int("podNumber", 5, "number of volume to attach to each replica")
	set.Bool("gradual", true, "set to `true` if you want gradual scaling")
	set.String("podPolicy", "Parallel", "change pod policy of the statefulset")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getScalingCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetVolumeIoCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("chainNumber", 5, "number of parallel chains")
	set.Int("chainLength", 5, "")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "length of a chain (number of pods to be created)")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeIoCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetSnapCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Int("snapshotAmopunt", 3, "define the amount of snapshots to create")
	set.String("volumeSnapshotClass", "test-vsc", "volumeSnapshotClass to be used")
	set.String("size", "3Gi", "volume size to be created")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getSnapCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetVolumeGroupSnapCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("volumeSnapshotClass", "test-vsc", "volumeSnapshotClass to be used")
	set.String("volumeSize", "3Gi", "volume size to be created")
	set.String("volumeLabel", "test-label", "volume label to be used")
	set.String("volumeGroupName", "test-group", "name of the volume group/snapshot")
	set.String("driver", "test-driver", "name of the driver")
	set.String("reclaimPolicy", "Retain", "reclaim policy of the volume group/snapshot")
	set.String("accessMode", "ReadWriteOnce", "access mode of the volume group/snapshot")
	set.Int("volumeNumber", 5, "number of volume to create for the group")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getVolumeGroupSnapCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetBlockSnapCommandAction(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("volumeSnapshotClass", "test-vsc", "volumeSnapshotClass to be used")
	set.String("size", "3Gi", "volume size to be created")
	set.String("access-mode", "ReadWriteOnce", "volume access mode")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getBlockSnapCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
	actionFunc(ctx)
}

func TestGetPostgresCommand(_ *testing.T) {
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.Bool("replication", true, "set to `true` if you want to enable replication")
	set.String("size", "3Gi", "volume size to be created")
	set.Int("slave-replicas", 2, "number of slave replicas")
	set.String("timeout", "10s", "volume creation timeout")
	set.String("cooldown", "10s", "volume creation cooldown")
	ctx := cli.NewContext(nil, set, nil)
	command := getPostgresCommand([]cli.Flag{})
	// Call the action function
	action := command.Action
	actionFunc := action.(func(c *cli.Context) error)
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
	// set.String("name", "storage", "name of the storage class")
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
