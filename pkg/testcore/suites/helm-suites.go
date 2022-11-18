package suites

import (
	"bytes"
	"cert-csi/pkg/helm"
	"cert-csi/pkg/k8sclient"
	"cert-csi/pkg/observer"
	"cert-csi/pkg/testcore"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

type PostgresqlSuite struct {
	ConfigPath        string
	VolumeSize        string
	EnableReplication bool
	SlaveReplicas     int
}

func (ps *PostgresqlSuite) Run(ctx context.Context, storageClass string, clients *k8sclient.Clients) (e error, delFunc func() error) {
	if ps.VolumeSize == "" {
		log.Info("Using default volume size : 32Gi")
		ps.VolumeSize = "32Gi"
	}

	pvcClient := clients.PVCClient
	podClient := clients.PodClient

	namespace := pvcClient.Namespace
	releaseName := "cert-csi-psql"
	psqlPass := "secretpassword"

	hc, err := helm.NewClient(namespace, ps.ConfigPath, clients.PVCClient.Timeout)
	if err != nil {
		return err, delFunc
	}

	if err := hc.AddRepository("bitnami", "https://charts.bitnami.com/bitnami"); err != nil {
		return err, delFunc
	}

	if err := hc.UpdateRepositories(); err != nil {
		return err, delFunc
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
		return err, delFunc
	}

	podconf := testcore.PsqlPodConfig(psqlPass)
	podTmpl := podClient.MakePod(podconf)

	pod := podClient.Create(ctx, podTmpl).Sync(ctx)
	if pod.HasError() {
		return pod.GetError(), delFunc
	}

	if err := podClient.Exec(ctx, pod.Object, []string{
		"pgbench", "-i", "-p", "5432", "-d", "postgres", "--host", "cert-csi-psql-postgresql", "-U", "postgres",
	}, os.Stdout, os.Stderr, false); err != nil {
		return err, delFunc
	}

	res := bytes.NewBufferString("")
	if err := podClient.Exec(ctx, pod.Object, []string{
		"pgbench", "-c", "50", "-j", "4", "-p", "5432", "-T", "60", "-d", "postgres", "--host", "cert-csi-psql-postgresql", "-U", "postgres",
	}, res, os.Stderr, false); err != nil {
		return err, delFunc
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
		return err, delFunc
	}

	return nil, delFunc
}

func (*PostgresqlSuite) GetObservers(obsType observer.Type) []observer.Interface {
	return getAllObservers(obsType)
}

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

func (*PostgresqlSuite) GetNamespace() string {
	return "psql-test"
}

func (ps *PostgresqlSuite) GetName() string {
	return "PostgresqlSuite"
}

func (ps *PostgresqlSuite) Parameters() string {
	return fmt.Sprintf("{replicas: %d, volumeSize: %s, replication: %s}", ps.SlaveReplicas, ps.VolumeSize,
		strconv.FormatBool(ps.EnableReplication))
}
