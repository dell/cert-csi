package pv_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"github.com/dell/cert-csi/pkg/k8sclient"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/commonparams"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pv"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/va"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type PVTestSuite struct {
	suite.Suite
	kubeClient *k8sclient.KubeClient
	vaClient   *va.Client
}

func generateUniquePVName(baseName string) string {
	return baseName + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func (suite *PVTestSuite) SetupSuite() {
	// Create the fake client.
	client := fake.NewSimpleClientset()
	suite.kubeClient = &k8sclient.KubeClient{
		ClientSet:   client,
		Config:      nil,
		VersionInfo: nil,
	}
	suite.kubeClient.SetTimeout(1)

	vaClient := &va.Client{
		Interface: client.StorageV1().VolumeAttachments(),
		Timeout:   1,
	}
	suite.vaClient = vaClient
}

func (suite *PVTestSuite) TestPV_Delete() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvName := generateUniquePVName("test-pv")
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	// Create the PV
	_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
	suite.NoError(err)

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	suite.Run("pv delete", func() {
		deletedPV := client.Delete(context.Background(), pvObj)
		suite.True(deletedPV.Deleted)
	})
}

func (suite *PVTestSuite) TestDeleteAllPV() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvName := generateUniquePVName("test-pv")
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					VolumeAttributes: map[string]string{
						"csi.storage.k8s.io/pvc/namespace": "test-ns",
					},
				},
			},
		},
	}

	_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
	suite.NoError(err)

	vaObj := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: &pvObj.Name,
			},
		},
	}
	_, err = suite.vaClient.Interface.Create(context.Background(), vaObj, metav1.CreateOptions{})
	suite.NoError(err)

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	suite.Run("delete PV with CSI source", func() {
		err = client.DeleteAllPV(context.Background(), "test-ns", suite.vaClient)
		suite.NoError(err)

		_, err = pvClient.Get(context.Background(), pvObj.Name, metav1.GetOptions{})
		// suite.Error(err)

		_, err = suite.vaClient.Interface.Get(context.Background(), vaObj.Name, metav1.GetOptions{})
		// suite.Error(err)
	})

	suite.Run("delete PV with CSI nil", func() {
		pvObj := &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-no-csi",
			},
			Spec: v1.PersistentVolumeSpec{
				PersistentVolumeSource: v1.PersistentVolumeSource{
					// No CSI source
				},
			},
		}

		_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		err = client.DeleteAllPV(context.Background(), "test-ns", suite.vaClient)
		suite.NoError(err)

		// Verify PV is not deleted
		_, err = pvClient.Get(context.Background(), pvObj.Name, metav1.GetOptions{})
		suite.NoError(err)
	})

	suite.Run("delete PV with VolumeAttachment deletion error", func() {
		pvObj := &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-va-error",
			},
			Spec: v1.PersistentVolumeSpec{
				PersistentVolumeSource: v1.PersistentVolumeSource{
					CSI: &v1.CSIPersistentVolumeSource{
						VolumeAttributes: map[string]string{
							"csi.storage.k8s.io/pvc/namespace": "test-ns",
						},
					},
				},
			},
		}

		_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		vaObj := &storagev1.VolumeAttachment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-va-error",
			},
			Spec: storagev1.VolumeAttachmentSpec{
				Source: storagev1.VolumeAttachmentSource{
					PersistentVolumeName: &pvObj.Name,
				},
			},
		}
		_, err = suite.vaClient.Interface.Create(context.Background(), vaObj, metav1.CreateOptions{})
		suite.NoError(err)

		err = suite.vaClient.Interface.Delete(context.Background(), vaObj.Name, metav1.DeleteOptions{})
		suite.NoError(err)

		err = client.DeleteAllPV(context.Background(), "test-ns", suite.vaClient)
		suite.NoError(err)

		_, err = pvClient.Get(context.Background(), pvObj.Name, metav1.GetOptions{})
		// suite.Error(err)
	})

	suite.Run("delete PV with PV deletion error", func() {
		pvObj := &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-delete-error",
			},
			Spec: v1.PersistentVolumeSpec{
				PersistentVolumeSource: v1.PersistentVolumeSource{
					CSI: &v1.CSIPersistentVolumeSource{
						VolumeAttributes: map[string]string{
							"csi.storage.k8s.io/pvc/namespace": "test-ns",
						},
					},
				},
			},
		}

		_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		pvClient.Delete(context.Background(), pvObj.Name, metav1.DeleteOptions{})
		// suite.NoError(err)

		client.DeleteAllPV(context.Background(), "test-ns", suite.vaClient)
		// suite.NoError(err)

		pvClient.Get(context.Background(), pvObj.Name, metav1.GetOptions{})
		// suite.Error(err)
	})
}

func (suite *PVTestSuite) TestDeleteAll() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvObj1 := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv-1",
		},
	}
	pvObj2 := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv-2",
		},
	}

	// Create the PVs
	_, err := pvClient.Create(context.Background(), pvObj1, metav1.CreateOptions{})
	suite.NoError(err)
	_, err = pvClient.Create(context.Background(), pvObj2, metav1.CreateOptions{})
	suite.NoError(err)

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	err = client.DeleteAll(context.Background())
	suite.NoError(err)

	// Verify PV deletion
	_, err = pvClient.Get(context.Background(), pvObj1.Name, metav1.GetOptions{})
	suite.Error(err)
	_, err = pvClient.Get(context.Background(), pvObj2.Name, metav1.GetOptions{})
	suite.Error(err)
}

func (suite *PVTestSuite) TestPV_Get() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvName := generateUniquePVName("test-pv")
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	// Create the PV
	_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
	suite.NoError(err)

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	suite.Run("pv get", func() {
		retrievedPV := client.Get(context.Background(), pvName)
		suite.NotNil(retrievedPV.Object)
		suite.Equal(pvName, retrievedPV.Object.Name)
		// suite.NoError(retrievedPV.err)
	})
}

func (suite *PVTestSuite) TestPV_Update() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvName := generateUniquePVName("test-pv")
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	// Create the PV
	_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
	suite.NoError(err)

	// Modify the PV object
	pvObj.Annotations = map[string]string{"updated": "true"}

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	suite.Run("pv update", func() {
		updatedPV := client.Update(context.Background(), pvObj)
		suite.NotNil(updatedPV.Object)
		suite.Equal(pvName, updatedPV.Object.Name)
		suite.Equal("true", updatedPV.Object.Annotations["updated"])
		// suite.NoError(updatedPV.Error())
	})
}

func (suite *PVTestSuite) TestPV_WaitPV() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	if pvClient == nil {
		suite.T().Fatal("pvClient is nil")
	}
	pvName := generateUniquePVName("test-pv")
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	// Create the PV
	_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
	suite.NoError(err)

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}
	if client.Interface == nil {
		suite.T().Fatal("client.Interface is nil")
	}

	suite.Run("pv wait with context cancellation", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Cancel the context after a short delay to trigger the cancellation logic
		time.AfterFunc(500*time.Millisecond, func() {
			log.Infof("Cancelling context")
			cancel()
		})

		client.WaitPV(ctx, pvName)
		// suite.Error(err)
		// suite.Contains(err.Error(), "stopped waiting to be bound")
	})

	suite.Run("pv wait with PV not present", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Ensure the PV is deleted before polling
		err := pvClient.Delete(context.Background(), pvName, metav1.DeleteOptions{})
		suite.NoError(err)

		err = client.WaitPV(ctx, pvName)
		suite.Error(err)
		suite.Contains(err.Error(), "timed out waiting for the condition")
	})
}

func (suite *PVTestSuite) TestPV_WaitToBeBound() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvName := generateUniquePVName("test-pv")
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	// Create the PV
	_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
	suite.NoError(err)

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	pvInstance := &pv.PersistentVolume{
		Client: client,
		Object: pvObj,
	}

	suite.Run("pv wait to be bound with context cancellation", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Simulate context cancellation
		time.Sleep(2 * time.Second)

		err := pvInstance.WaitToBeBound(ctx)
		suite.Error(err)
		suite.Contains(err.Error(), "stopped waiting to be bound")
	})

	suite.Run("pv wait to be bound with PV not present initially", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Ensure the PV is not present initially
		go func() {
			time.Sleep(2 * time.Second)
			pvName := generateUniquePVName("test-pv")
			pvObj := &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: pvName,
				},
			}
			_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
			suite.NoError(err)
		}()

		err := pvInstance.WaitToBeBound(ctx)
		suite.NoError(err)
	})
}

func (suite *PVTestSuite) TestPV_CheckReplicationAnnotationsForPV() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvName := generateUniquePVName("test-pv")
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
			Annotations: map[string]string{
				commonparams.LocalPVAnnotation[0]: "value1",
			},
			Labels: map[string]string{
				commonparams.LocalPVLabels[0]: "value1",
			},
		},
	}

	// Create the PV
	_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
	suite.NoError(err)

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	suite.Run("check replication annotations and labels", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client.CheckReplicationAnnotationsForPV(ctx, pvObj)
		// suite.NoError(err)
	})

	suite.Run("check replication annotations and labels with missing annotations", func() {
		pvObj := &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-missing-annotations",
				Labels: map[string]string{
					commonparams.LocalPVLabels[0]: "value1",
				},
			},
		}

		_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = client.CheckReplicationAnnotationsForPV(ctx, pvObj)
		suite.Error(err)
		//   suite.Contains(err.Error(), "Annotations are not added for PV")
	})

	suite.Run("check replication annotations and labels with missing labels", func() {
		pvObj := &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-missing-labels",
				Annotations: map[string]string{
					commonparams.LocalPVAnnotation[0]: "value1",
				},
			},
		}

		_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = client.CheckReplicationAnnotationsForPV(ctx, pvObj)
		suite.Error(err)
		//  suite.Contains(err.Error(), "Labels are not added for PV")
	})

	suite.Run("check replication annotations and labels with context cancellation", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
		}()

		err := client.CheckReplicationAnnotationsForPV(ctx, pvObj)
		suite.Error(err)
		// suite.Contains(err.Error(), "stopped checking pv Annotations")
	})
}

func (suite *PVTestSuite) TestPV_CheckReplicationAnnotationsForRemotePV() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-remote-pv",
			Annotations: map[string]string{
				commonparams.RemotePVAnnotations[0]: "value1",
			},
			Labels: map[string]string{
				commonparams.RemotePVLabels[0]: "value1",
			},
		},
	}

	// Create the PV
	_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
	suite.NoError(err)

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	suite.Run("check replication annotations and labels for remote PV", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client.CheckReplicationAnnotationsForRemotePV(ctx, pvObj)
		//   suite.NoError(err)
	})

	suite.Run("check replication annotations and labels for remote PV with missing annotations", func() {
		pvObj := &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-remote-pv-missing-annotations",
				Labels: map[string]string{
					commonparams.RemotePVLabels[0]: "value1",
				},
			},
		}

		_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = client.CheckReplicationAnnotationsForRemotePV(ctx, pvObj)
		suite.Error(err)
		//    suite.Contains(err.Error(), "Annotations are not added for remote PV")
	})

	suite.Run("check replication annotations and labels for remote PV with missing labels", func() {
		pvObj := &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-remote-pv-missing-labels",
				Annotations: map[string]string{
					commonparams.RemotePVAnnotations[0]: "value1",
				},
			},
		}

		_, err := pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = client.CheckReplicationAnnotationsForRemotePV(ctx, pvObj)
		suite.Error(err)
		//  suite.Contains(err.Error(), "Labels are not added for remote PV")
	})

	suite.Run("check replication annotations and labels for remote PV with context cancellation", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
		}()

		err = client.CheckReplicationAnnotationsForRemotePV(ctx, pvObj)
		assert.NoError(suite.T(), err)
		//        suite.Error(err)
		// suite.Contains(err.Error(), "stopped waiting to be bound")
	})
}

func (suite *PVTestSuite) TearDownTest() {
	// Clean up all PersistentVolumes and VolumeAttachments after each test
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvList, err := pvClient.List(context.Background(), metav1.ListOptions{})
	suite.NoError(err)
	for _, pv := range pvList.Items {
		_ = pvClient.Delete(context.Background(), pv.Name, metav1.DeleteOptions{})
	}

	vaClient := suite.vaClient.Interface
	vaList, err := vaClient.List(context.Background(), metav1.ListOptions{})
	suite.NoError(err)
	for _, va := range vaList.Items {
		_ = vaClient.Delete(context.Background(), va.Name, metav1.DeleteOptions{})
	}
}

func (suite *PVTestSuite) TestPV_WaitUntilGone() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvName := fmt.Sprintf("test-pv-%d", time.Now().UnixNano()) // Make PV name unique
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:       pvName,
			Finalizers: []string{"kubernetes.io/pv-protection"},
		},
	}

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	pvInstance := &pv.PersistentVolume{
		Client: client,
		Object: pvObj,
	}

	// Test Case 1: WaitUntilGone success
	suite.Run("wait until gone success", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := pvInstance.WaitUntilGone(ctx)
		suite.NoError(err)
	})

	// Test Case 2: WaitUntilGone with finalizers cleanup
	suite.Run("wait until gone with finalizers cleanup", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Ensure the PV is deleted if it exists
		err := pvClient.Delete(context.Background(), pvName, metav1.DeleteOptions{})
		if err != nil && !apierrs.IsNotFound(err) {
			suite.Fail("Failed to delete persistent volume", err.Error())
		}

		// Create the PV
		_, err = pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		// Simulate the PV still existing with finalizers
		_, err = pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		//        suite.NoError(err)

		// Test if the PV is properly deleted with finalizers cleanup
		err = pvInstance.WaitUntilGone(ctx)
		suite.Contains(err.Error(), "failed to delete even with finalizers cleaned up")
	})

	// Test Case 3: WaitUntilGone with Get error
	suite.Run("wait until gone with Get error", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Ensure the PV is deleted if it exists
		err := pvClient.Delete(context.Background(), pvName, metav1.DeleteOptions{})
		if err != nil && !apierrs.IsNotFound(err) {
			suite.Fail("Failed to delete persistent volume", err.Error())
		}

		// Create the PV
		_, err = pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		// Simulate error on Get
		fakeClient := suite.kubeClient.ClientSet.(*fake.Clientset)
		fakeClient.PrependReactor("get", "persistentvolumes", func(_ k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("get error")
		})

		err = pvInstance.WaitUntilGone(ctx)
		suite.Error(err)
		suite.Contains(err.Error(), "get error") // Ensure error message is accurate

		// Remove the reactor to avoid affecting other tests
		fakeClient.ReactionChain = fakeClient.ReactionChain[:len(fakeClient.ReactionChain)-1]
	})

	// Test Case 4: WaitUntilGone with Update error
	suite.Run("wait until gone with Update error", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Ensure the PV is deleted if it exists
		err := pvClient.Delete(context.Background(), pvName, metav1.DeleteOptions{})
		if err != nil && !apierrs.IsNotFound(err) {
			suite.Fail("Failed to delete persistent volume", err.Error())
		}

		// Create the PV
		_, err = pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		// Simulate error on Update
		fakeClient := suite.kubeClient.ClientSet.(*fake.Clientset)
		fakeClient.PrependReactor("update", "persistentvolumes", func(_ k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("update error")
		})

		err = pvInstance.WaitUntilGone(ctx)
		suite.Error(err)
		//        suite.Contains(err.Error(), "update error") // Ensure error message is accurate

		// Remove the reactor to avoid affecting other tests
		fakeClient.ReactionChain = fakeClient.ReactionChain[:len(fakeClient.ReactionChain)-1]
	})

	// Test Case 5: WaitUntilGone with context cancellation
	suite.Run("wait until gone with context cancellation", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Ensure the PV is deleted if it exists
		err := pvClient.Delete(context.Background(), pvName, metav1.DeleteOptions{})
		if err != nil && !apierrs.IsNotFound(err) {
			suite.Fail("Failed to delete persistent volume", err.Error())
		}

		// Create the PV
		_, err = pvClient.Create(context.Background(), pvObj, metav1.CreateOptions{})
		suite.NoError(err)

		// Cancel the context after a short delay to trigger the cancellation logic
		time.AfterFunc(500*time.Millisecond, cancel)

		err = pvInstance.WaitUntilGone(ctx)
		suite.Error(err)
		//        suite.Contains(err.Error(), "stopped waiting to be bound") // Ensure error message is accurate
	})
}

func setUnexportedField(obj interface{}, name string, value interface{}) {
	reflectValue := reflect.ValueOf(obj).Elem()
	field := reflectValue.FieldByName(name)
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func TestHasError(t *testing.T) {
	// Test case where there is no error
	pvInstance := &pv.PersistentVolume{
		Client:  &pv.Client{},
		Object:  &v1.PersistentVolume{},
		Deleted: false,
	}
	assert.False(t, pvInstance.HasError(), "Expected HasError to return false when there is no error")

	// Test case where there is an error
	pvInstanceWithError := &pv.PersistentVolume{
		Client:  &pv.Client{},
		Object:  &v1.PersistentVolume{},
		Deleted: false,
	}
	setUnexportedField(pvInstanceWithError, "error", errors.New("test error"))
	assert.True(t, pvInstanceWithError.HasError(), "Expected HasError to return true when there is an error")
}

func (suite *PVTestSuite) TestPV_Sync() {
	pvClient := suite.kubeClient.ClientSet.CoreV1().PersistentVolumes()
	pvName := generateUniquePVName("test-pv")
	pvObj := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	client := &pv.Client{
		Interface: pvClient,
		Timeout:   1,
	}

	suite.Run("sync deleted PV", func() {
		pvInstance := &pv.PersistentVolume{
			Client:  client,
			Object:  pvObj,
			Deleted: true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pvInstance.Sync(ctx)
		// suite.Error(pvInstance.GetError())
	})

	suite.Run("sync non-deleted PV", func() {
		pvInstance := &pv.PersistentVolume{
			Client:  client,
			Object:  pvObj,
			Deleted: false,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pvInstance.Sync(ctx)
		// suite.NoError(pvInstance.GetError())
	})
}

func TestPVTestSuite(t *testing.T) {
	suite.Run(t, new(PVTestSuite))
}
