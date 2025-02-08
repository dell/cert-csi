package sc_test

import (
	"context"
	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/stretchr/testify/suite"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

type ScTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
}

func (suite *ScTestSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()
	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)
}
func (suite *ScTestSuite) TestSC_Create() {
	scClient, err := suite.kubeClient.CreateSCClient()
	scClient.Timeout = 1
	suite.NoError(err)
	suite.Run("sc create", func() {
		err = scClient.Create(context.Background(), &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "powerstore",
			},
		})
	})
}
func (suite *ScTestSuite) TestSC_Get() {
	sc, err := suite.kubeClient.ClientSet.StorageV1().StorageClasses().Create(context.Background(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "powerstore",
		},
	}, metav1.CreateOptions{
		DryRun: []string{"All"},
	})

	suite.NoError(err)
	scClient, err := suite.kubeClient.CreateSCClient()
	scClient.Timeout = 1

	suite.Run("sc get", func() {
		_ = scClient.Get(context.Background(), sc.Name)
	})
}
func (suite *ScTestSuite) TestSC_Delete() {
	scClient, err := suite.kubeClient.CreateSCClient()
	scClient.Timeout = 1
	suite.NoError(err)
	suite.Run("sc delete", func() {
		err = scClient.Delete(context.Background(), "powerstore")
	})

}
func (suite *ScTestSuite) TestSC_MakeStorageClass() {
	scClient, err := suite.kubeClient.CreateSCClient()
	scClient.Timeout = 1
	suite.NoError(err)
	suite.Run("sc make storage class", func() {
		_ = scClient.MakeStorageClass("powerstore", "powerstore")
	})
}
func TestScSuite(t *testing.T) {
	suite.Run(t, new(ScTestSuite))
}
