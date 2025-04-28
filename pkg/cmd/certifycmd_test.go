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
	"context"
	"flag"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/testcore/runner"
	"github.com/dell/cert-csi/pkg/testcore/suites"
	"github.com/urfave/cli"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
)

func TestGetCertifyCommand(t *testing.T) {
	// Test the functionality of the GetCertifyCommand function
	command := GetCertifyCommand()
	assert.Equal(t, "certify", command.Name)
	assert.Equal(t, "certify csi-driver", command.Usage)
	assert.Equal(t, "main", command.Category)

	// Test the flags of the command
	assert.Equal(t, "cert-config", command.Flags[0].GetName())
	assert.Equal(t, "image-config", command.Flags[1].GetName())
	assert.Equal(t, "config, conf, c", command.Flags[2].GetName())
	assert.Equal(t, "namespace, ns", command.Flags[3].GetName())
	assert.Equal(t, "longevity, long, l", command.Flags[4].GetName())
	assert.Equal(t, "timeout, t", command.Flags[5].GetName())
	assert.Equal(t, "sequential, sq", command.Flags[6].GetName())
	assert.Equal(t, "no-cleanup, nc", command.Flags[7].GetName())
	assert.Equal(t, "no-cleanup-on-fail, ncof", command.Flags[8].GetName())
}

func TestGetAction(t *testing.T) {
	confPath, err := createDummyKubeConfig(t.TempDir(), t)
	assert.NoError(t, err)

	FuncNewClientSetOriginal := k8sclient.FuncNewClientSet
	defer func() {
		k8sclient.FuncNewClientSet = FuncNewClientSetOriginal
	}()

	clientCtx := &clientTestContext{t: t}

	k8sclient.FuncNewClientSet = func(_ *rest.Config) (kubernetes.Interface, error) {
		fakeClient, err := createFakeKubeClient(clientCtx)
		assert.NoError(t, err)

		sc2 := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "fake",
			},
			Provisioner: "no.provisioner.exists/fake",
		}
		_, err = fakeClient.StorageV1().StorageClasses().Create(context.Background(), sc2, metav1.CreateOptions{})
		assert.NoError(t, err)

		sc := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "fake-nfs",
			},
			Provisioner: "no.provisioner.exists/fake",
		}
		_, err = fakeClient.StorageV1().StorageClasses().Create(context.Background(), sc, metav1.CreateOptions{})
		assert.NoError(t, err)

		return fakeClient, nil
	}

	app := cli.NewApp()

	set := flag.NewFlagSet("unit-test", flag.ContinueOnError)
	set.String("cert-config", "./test-certify-config.yaml", "cert config file")
	set.String("longevity", "30s", "longevity")
	set.String("config", confPath, "cert config file")
	set.String("timeout", "10s", "timeout")
	set.String("volumeSnapshotClass", "fake", "volumeSnapshotClass")
	set.String("observer-type", "event", "observer type")
	set.String("driver-name", "fake", "")

	ctx := cli.NewContext(app, set, nil)
	ExecuteRunSuite = func(_ *runner.SuiteRunner, _ map[string][]suites.Interface) {}
	err = getAction(ctx)
	assert.NoError(t, err, "getAction should not return an error")
}
