package helm

import (
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

func TestNewClient(t *testing.T) {
	// Set up environment variables
	os.Setenv("HELM_DRIVER", "secret")
	defer os.Unsetenv("HELM_DRIVER")

	namespace := "default"
	configPath := "/path/to/kubeconfig" // Ensure this path is valid for the test
	timeout := 300

	client, err := NewClient(namespace, configPath, timeout)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	assert.Equal(t, namespace, client.namespace)
	assert.Equal(t, configPath, client.configPath)
	assert.Equal(t, timeout, client.timeout)
	assert.NotNil(t, client.settings)
	assert.NotNil(t, client.actionConfig)
}

/*
func TestNewClient_InvalidConfig(t *testing.T) {
	namespace := "default"
	configPath := "/invalid/path/to/kubeconfig" // This path should be invalid for the test
	timeout := 300

	// Set an invalid HELM_DRIVER to simulate an error
	os.Setenv("HELM_DRIVER", "invalid-driver")
	defer os.Unsetenv("HELM_DRIVER")

	client, err := NewClient(namespace, configPath, timeout)
	assert.Error(t, err)
	assert.Nil(t, client)
}
*/

func TestAddRepository(t *testing.T) {
	// Set up environment variables
	os.Setenv("HELM_DRIVER", "secret")
	defer os.Unsetenv("HELM_DRIVER")

	// Create a temporary directory for the repository config
	tempDir := t.TempDir()
	repoFile := filepath.Join(tempDir, "repositories.yaml")
	os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)
	defer os.Unsetenv("HELM_REPOSITORY_CONFIG")

	namespace := "default"
	configPath := "/path/to/kubeconfig" // Ensure this path is valid for the test
	timeout := 300

	client, err := NewClient(namespace, configPath, timeout)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Test adding a new repository
	repoName := "test-repo"
	repoURL := "https://charts.helm.sh/stable" // Use a valid Helm chart repository URL
	err = client.AddRepository(repoName, repoURL)
	assert.NoError(t, err)

	// Verify the repository was added
	b, err := os.ReadFile(repoFile)
	assert.NoError(t, err)

	var f repo.File
	err = yaml.Unmarshal(b, &f)
	assert.NoError(t, err)
	assert.True(t, f.Has(repoName))
	assert.Equal(t, repoURL, f.Get(repoName).URL)

	// Test adding the same repository again to cover the "already exists" path
	err = client.AddRepository(repoName, repoURL)
	assert.NoError(t, err)

	// Test adding an invalid repository URL to cover the error path
	invalidRepoURL := "https://invalid-url/charts"
	err = client.AddRepository("invalid-repo", invalidRepoURL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "looks like \"https://invalid-url/charts\" is not a valid chart repository or cannot be reached")
}

func TestUpdateRepositories(t *testing.T) {
	// Set up the Helm settings
	settings := cli.New()
	settings.RepositoryConfig = "/path/to/repositories.yaml" // Adjust this path as needed

	// Create a new client
	client := &Client{
		settings: settings,
	}

	// Mock the repository file
	repoFile := repo.NewFile()
	repoFile.Add(&repo.Entry{
		Name: "my-repo",
		URL:  "https://example.com/charts",
	})
	err := repoFile.WriteFile(settings.RepositoryConfig, 0644)
	assert.NoError(t, err)

	// Call the UpdateRepositories function
	err = client.UpdateRepositories()
	assert.NoError(t, err)
}

func TestInstallChart(t *testing.T) {
	// Set up environment variables
	os.Setenv("HELM_DRIVER", "secret")
	defer os.Unsetenv("HELM_DRIVER")

	// Create a temporary directory for the repository config
	tempDir := t.TempDir()
	repoFile := filepath.Join(tempDir, "repositories.yaml")
	os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)
	defer os.Unsetenv("HELM_REPOSITORY_CONFIG")

	namespace := "default"
	configPath := "/path/to/kubeconfig" // Ensure this path is valid for the test
	timeout := 300

	// Create a new Helm client
	client, err := NewClient(namespace, configPath, timeout)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Add a test repository
	repoName := "test-repo"
	repoURL := "https://charts.helm.sh/stable" // Use a valid Helm chart repository URL
	err = client.AddRepository(repoName, repoURL)
	assert.NoError(t, err)

	// Define the release name, chart name, and values
	releaseName := "test-release"
	chartName := "nginx" // Use a valid chart name from the repository
	values := map[string]interface{}{
		"service": map[string]interface{}{
			"type": "ClusterIP",
		},
	}

	// Install the chart
	err = client.InstallChart(releaseName, repoName, chartName, values)
	assert.Error(t, err)

	// Clean up: Uninstall the chart after the test
	defer func() {
		err = client.UninstallChart(releaseName)
		assert.Error(t, err)
	}()
}

func TestUninstallChart(t *testing.T) {
	// Set up the Helm action configuration
	settings := cli.New()
	actionConfig := new(action.Configuration)
	err := actionConfig.Init(settings.RESTClientGetter(), "default", os.Getenv("HELM_DRIVER"), log.Infof)
	assert.NoError(t, err)

	// Create a new client
	client := &Client{
		namespace:    "default",
		actionConfig: actionConfig,
	}

	// Define the release name to uninstall
	releaseName := "my-release"

	// Call the UninstallChart function
	err = client.UninstallChart(releaseName)
	assert.Error(t, err)

}

func TestIsChartInstallable(t *testing.T) {
	tests := []struct {
		name      string
		chartType string
		expect    bool
		expectErr bool
	}{
		{"Empty type should be installable", "", true, false},
		{"Application type should be installable", "application", true, false},
		{"Library type should not be installable", "library", false, true},
		{"Unknown type should not be installable", "unknown", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &chart.Chart{Metadata: &chart.Metadata{Type: tt.chartType}}
			installable, err := isChartInstallable(ch)

			assert.Equal(t, tt.expect, installable)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
