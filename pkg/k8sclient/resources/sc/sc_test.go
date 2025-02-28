package sc_test

import (
	"context"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/stretchr/testify/suite"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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

func (suite *ScTestSuite) TestSC_Get_NonExistent() {
	scClient, err := suite.kubeClient.CreateSCClient()
	suite.NoError(err)
	suite.Run("sc get non-existent", func() {
		sc := scClient.Get(context.Background(), "non-existent-sc")
		suite.Error(sc.GetError(), "expected error for non-existent storage class")
	})
}

func (suite *ScTestSuite) TestSC_Delete_NonExistent() {
	scClient, err := suite.kubeClient.CreateSCClient()
	suite.NoError(err)
	suite.Run("sc delete non-existent", func() {
		err = scClient.Delete(context.Background(), "non-existent-sc")
		suite.Error(err, "expected error for deleting non-existent storage class")
	})
}

func (suite *ScTestSuite) TestSC_DuplicateStorageClass() {
	scClient, err := suite.kubeClient.CreateSCClient()
	suite.NoError(err)
	suite.Run("sc duplicate storage class", func() {
		sourceSc := &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{Name: "source-sc"},
		}
		newSc := scClient.DuplicateStorageClass("new-sc", sourceSc)
		suite.NotNil(newSc, "expected non-nil duplicated storage class")
		suite.Equal("new-sc", newSc.GetName(), "expected 'new-sc' name")
	})
}

func TestScSuite(t *testing.T) {
	suite.Run(t, new(ScTestSuite))
}
