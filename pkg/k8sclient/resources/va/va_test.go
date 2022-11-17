package va_test

import (
	"cert-csi/pkg/k8sclient"
	"context"
	"github.com/stretchr/testify/suite"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

type VaTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
}

func (suite *VaTestSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()

	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)
}

func (suite *VaTestSuite) TestVaClient_WaitUntilNoneLeft() {
	va, err := suite.kubeClient.ClientSet.StorageV1().VolumeAttachments().Create(context.Background(), &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
	}, metav1.CreateOptions{})
	suite.NoError(err)

	vaClient, err := suite.kubeClient.CreateVaClient("test-namespace")
	vaClient.CustomTimeout = time.Second
	suite.NoError(err)

	suite.Run("timeout error", func() {
		err := vaClient.WaitUntilNoneLeft(context.Background())
		suite.Error(err)
	})

	vaClient, err = suite.kubeClient.CreateVaClient("test-namespace")
	suite.NoError(err)
	err = suite.kubeClient.ClientSet.StorageV1().VolumeAttachments().Delete(context.Background(), va.Name, metav1.DeleteOptions{})
	suite.NoError(err)

	suite.Run("all deleted", func() {
		err := vaClient.WaitUntilNoneLeft(context.Background())
		suite.NoError(err)
	})
}

func TestVaTestSuite(t *testing.T) {
	suite.Run(t, new(VaTestSuite))
}
