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

package helm

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"

	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// Client for helm driver
type Client struct {
	settings     *cli.EnvSettings
	actionConfig *action.Configuration
	timeout      int
	namespace    string
	configPath   string
}

// NewClient creates new helm driver client
func NewClient(namespace, configPath string, timeout int) (*Client, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(kube.GetConfig(configPath, "", namespace), namespace, os.Getenv("HELM_DRIVER"), log.Debugf); err != nil {
		return nil, err
	}
	return &Client{
		settings:     cli.New(),
		actionConfig: actionConfig,
		timeout:      timeout,
		namespace:    namespace,
		configPath:   configPath,
	}, nil
}

// AddRepository adds new repository with provided name and url
func (c *Client) AddRepository(name, url string) error {
	repoFile := c.settings.RepositoryConfig

	// Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock()
	}
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(filepath.Clean(repoFile))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	if f.Has(name) {
		log.Infof("repository name (%s) already exists", name)
		return nil
	}

	chartRepo := repo.Entry{
		Name: name,
		URL:  url,
	}

	r, err := repo.NewChartRepository(&chartRepo, getter.All(c.settings))
	if err != nil {
		return err
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		err := errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", url)
		return err
	}

	f.Update(&chartRepo)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return err
	}
	log.Infof("%q has been added to your repositories", name)
	return nil
}

// UpdateRepositories updates charts for all helm repos
func (c *Client) UpdateRepositories() error {
	repoFile := c.settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(errors.Cause(err)) || len(f.Repositories) == 0 {
		return errors.New("no repositories found. You must add one before updating")
	}
	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(c.settings))
		if err != nil {
			return err
		}
		repos = append(repos, r)
	}

	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				log.Infof("Can't update %q chart repository (%s):\t%s", re.Config.Name, re.Config.URL, err)
			} else {
				log.Infof("Updated %q chart repository", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	log.Info("Updated repositories")
	return nil
}

// InstallChart installs chart with provided repository and values map
func (c *Client) InstallChart(releaseName, repo, chart string, vals map[string]interface{}) error {
	log.Info("Installing chart")
	client := action.NewInstall(c.actionConfig)

	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}

	client.Namespace = c.namespace
	client.ReleaseName = releaseName
	client.Wait = true // wait until chart installs
	client.Timeout = time.Duration(c.timeout) * time.Second

	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", repo, chart), c.settings)
	if err != nil {
		return err
	}

	log.Debugf("chart path: %s", cp)

	p := getter.All(c.settings)

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: c.settings.RepositoryConfig,
					RepositoryCache:  c.settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	release, err := client.Run(chartRequested, vals)
	if err != nil {
		return err
	}
	log.Infof("Installed Chart %s in namespace %s", release.Name, release.Namespace)
	return nil
}

// UninstallChart uninstalls chart with specified release name
func (c *Client) UninstallChart(releaseName string) error {
	delClient := action.NewUninstall(c.actionConfig)
	resp, err := delClient.Run(releaseName)
	if err != nil {
		return err
	}

	log.Infof("Uninstalled Chart from path: %s in namespace: %s", resp.Release.Name, resp.Release.Namespace)
	return nil
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}
