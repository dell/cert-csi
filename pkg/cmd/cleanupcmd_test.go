package cmd

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/urfave/cli"
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
		Action: func(_ *cli.Context) error {
			return nil
		},
	}

	actual := GetCleanupCommand()

	// Check the non-Action fields. We will test Action in next function
	expected.Action = nil
	actual.Action = nil
	assert.Equal(t, expected, actual)
}

func TestGetCleanupCommandAction(t *testing.T) {
	// Create dummy KubeConfig
	kubeConfig, _ := createDummyKubeConfig(t.TempDir(), t)

	clientCtx := &clientTestContext{t: t}

	k8sclient.FuncNewClientSet = func(_ *rest.Config) (kubernetes.Interface, error) {
		fakeClient, err := createFakeKubeClient(clientCtx)
		assert.NoError(t, err)
		return fakeClient, nil
	}

	app := cli.NewApp()
	// Default context
	set := flag.NewFlagSet("test", 0)
	set.String("config", "", "config for connecting to kubernetes")
	set.String("timeout", "1m", "set the timeout value for all of the resources (accepts format like 2h30m15s) default is 30s")
	set.Bool("yes", false, "include this flag to auto approve cleanup cmd. Could be useful if you are running cert-csi from non-interactive environment")
	falseBoolContext := cli.NewContext(app, set, nil)

	// Invalid config context
	set2 := flag.NewFlagSet("test2", 0)
	set2.String("config", "", "config for connecting to kubernetes")
	set2.String("timeout", "1m", "set the timeout value for all of the resources (accepts format like 2h30m15s) default is 30s")
	set2.Bool("yes", true, "include this flag to auto approve cleanup cmd. Could be useful if you are running cert-csi from non-interactive environment")
	invalidConfigContext := cli.NewContext(nil, set2, nil)

	// Valid config context
	set4 := flag.NewFlagSet("test2", flag.ContinueOnError)
	set4.String("config", kubeConfig, "config for connecting to kubernetes")
	set4.String("timeout", "1m", "set the timeout value for all of the resources (accepts format like 2h30m15s) default is 30s")
	set4.Bool("yes", true, "include this flag to auto approve cleanup cmd. Could be useful if you are running cert-csi from non-interactive environment")
	validContext := cli.NewContext(nil, set4, nil)

	// Get the cleanup command
	cmd := GetCleanupCommand()

	tests := []struct {
		name      string
		context   *cli.Context
		expectErr bool
	}{
		{
			name:      "Testing with false bool context",
			context:   falseBoolContext,
			expectErr: false,
		},
		{
			name:      "Testing with invalid config context",
			context:   invalidConfigContext,
			expectErr: true,
		},
		{
			name:      "Testing with true bool and valid config context",
			context:   validContext,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Override the default os.Stdout with our buffer
			stdout := os.Stdout
			os.Stdout = os.NewFile(0, "stdout")

			// Mock the user input
			input := "n"

			// Create a new strings.Reader from the input
			reader := strings.NewReader(input)

			// Create a new os.File from the strings.Reader
			stdinFile := os.NewFile(0, "stdin")
			stdinFile.ReadFrom(reader)

			// Override the default os.Stdin with our file
			oldStdin := os.Stdin
			os.Stdin = stdinFile

			// Call the action function
			action := cmd.Action
			actionFunc := action.(func(c *cli.Context) error)
			actionFunc(tt.context)

			// Restore the original os.Stdin
			os.Stdin = oldStdin

			// Restore the original os.Stdout
			os.Stdout = stdout
		})
	}
}
