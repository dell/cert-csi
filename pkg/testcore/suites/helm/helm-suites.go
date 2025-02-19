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
	"bytes"
	"context"
	"fmt"
	"github.com/dell/cert-csi/pkg/testcore/suites/common"
	"os"
	"strconv"
	"strings"

	"github.com/dell/cert-csi/pkg/helm"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/dell/cert-csi/pkg/testcore"

	log "github.com/sirupsen/logrus"
)

// PostgresqlSuite configuration struct
type PostgresqlSuite struct {
	ConfigPath        string
	VolumeSize        string
	EnableReplication bool
	Image             string
	SlaveReplicas     int
}

// Run executes the test suite
func (ps *PostgresqlSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (delFunc func() error, e error) {
	if ps.VolumeSize == "" {
		log.Info("Using default volume size : 32Gi")
		ps.VolumeSize = "32Gi"
	}

	if ps.Image == "" {
		ps.Image = "docker.io/bitnami/postgresql:11.8.0-debian-10-r72"
		log.Infof("Using default image: %s", ps.Image)
	}

	pvcClient := clients.PVCClient
	podClient := clients.PodClient

	namespace := pvcClient.Namespace
	releaseName := "cert-csi-psql"
	psqlPass := "secretpassword"

	hc, err := helm.NewClient(namespace, ps.ConfigPath, clients.PVCClient.Timeout)
	if err != nil {
		return delFunc, err
	}

	if err := hc.AddRepository("bitnami", "https://charts.bitnami.com/bitnami"); err != nil {
		return delFunc, err
	}

	if err := hc.UpdateRepositories(); err != nil {
		return delFunc, err
	}

	// define values
	vals := map[string]interface{}{
		"postgresqlPassword": psqlPass,
		"persistence": map[string]interface{}{
			"size":         ps.VolumeSize,
			"storageClass": storageClass,
		},
		"replication": map[string]interface{}{
			"enabled":                ps.EnableReplication,
			"slaveReplicas":          ps.SlaveReplicas,
			"synchronousCommit":      "on",
			"numSynchronousReplicas": ps.SlaveReplicas / 2,
		},
	}

	if err := hc.InstallChart(releaseName, "bitnami", "postgresql", vals); err != nil {
		return delFunc, err
	}

	podconf := testcore.PsqlPodConfig(psqlPass, ps.Image)
	podTmpl := podClient.MakePod(podconf)

	pod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if pod.HasError() {
		return delFunc, pod.GetError()
	}

	if err := podClient.Exec(ctx, pod.Object, []string{
		"pgbench", "-i", "-p", "5432", "-d", "helm", "--host", "cert-csi-psql-postgresql", "-U", "helm",
	}, os.Stdout, os.Stderr, false); err != nil {
		return delFunc, err
	}

	res := bytes.NewBufferString("")
	if err := podClient.Exec(ctx, pod.Object, []string{
		"pgbench", "-c", "50", "-j", "4", "-p", "5432", "-T", "60", "-d", "helm", "--host", "cert-csi-psql-postgresql", "-U", "helm",
	}, res, os.Stderr, false); err != nil {
		return delFunc, err
	}
	// Print the output, for some reason there is no verbosity level so we just trim important part
	sliced := strings.Split(res.String(), "\n")
	if len(sliced) > 10 {
		output := bytes.NewBufferString("")
		for _, s := range sliced[len(sliced)-11:] {
			output.WriteString(s)
			output.WriteString("\n")
		}
		log.Info(strings.TrimRight(output.String(), "\n"))
	} else {
		log.Info(res.String())
	}

	if err := hc.UninstallChart(releaseName); err != nil {
		return delFunc, err
	}

	return delFunc, nil
}

// GetObservers returns all observers
func (*PostgresqlSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return common.GetAllObservers(obsType)
}

// GetClients creates and returns pvc, pod, va, metrics clients
func (*PostgresqlSuite) GetClients(namespace string, client *k8sclient.KubeClient) (*k8sclient.Clients, error) {
	pvcClient, pvcErr := client.CreatePVCClient(namespace)
	if pvcErr != nil {
		return nil, pvcErr
	}

	podClient, podErr := client.CreatePodClient(namespace)
	if podErr != nil {
		return nil, podErr
	}

	vaClient, vaErr := client.CreateVaClient(namespace)
	if vaErr != nil {
		return nil, vaErr
	}

	metricsClient, mcErr := client.CreateMetricsClient(namespace)
	if mcErr != nil {
		return nil, mcErr
	}

	return &k8sclient.Clients{
		PVCClient:         pvcClient,
		PodClient:         podClient,
		VaClient:          vaClient,
		StatefulSetClient: nil,
		MetricsClient:     metricsClient,
	}, nil
}

// GetNamespace returns PostgresqlSuite namespace
func (*PostgresqlSuite) GetNamespace() string {
	return "psql-test"
}

// GetName returns PostgresqlSuite name
func (ps *PostgresqlSuite) GetName() string {
	return "PostgresqlSuite"
}

// Parameters returns formatted string of parameters
func (ps *PostgresqlSuite) Parameters() string {
	return fmt.Sprintf("{replicas: %d, volumeSize: %s, replication: %s}", ps.SlaveReplicas, ps.VolumeSize,
		strconv.FormatBool(ps.EnableReplication))
}
