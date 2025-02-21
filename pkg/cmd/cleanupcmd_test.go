package cmd

import (
	"github.com/urfave/cli"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCleanupCommand(t *testing.T) {
	expected := cli.Command{
		Name:     "cleanup",
		Usage:    "cleanups all existing namespaces and resources, that can be created by cert-csi",
		Category: "main",
		Flags: []cli.Flag{
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
		},
	}

	actual := GetCleanupCommand()

	assert.Equal(t, expected, actual)
}
