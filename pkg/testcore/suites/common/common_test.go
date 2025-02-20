package common

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/observer"
	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

)

func TestGetAllObservers(t *testing.T) {
	tests := []struct {
		name     string
		obsType  observer.Type
		expected []observer.Interface
	}{
		{
			name:    "EVENT type",
			obsType: observer.EVENT,
			expected: []observer.Interface{
				&observer.PvcObserver{},
				&observer.VaObserver{},
				&observer.PodObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name:    "LIST type",
			obsType: observer.LIST,
			expected: []observer.Interface{
				&observer.PvcListObserver{},
				&observer.VaListObserver{},
				&observer.PodListObserver{},
				&observer.EntityNumberObserver{},
				&observer.ContainerMetricsObserver{},
			},
		},
		{
			name:     "Invalid type",
			obsType:  "",
			expected: []observer.Interface{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAllObservers(tt.obsType)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, but got %v", tt.expected, result)
			}
		})
	}
}

func TestShouldWaitForFirstConsumer(t *testing.T) {
	clientset := fake.NewSimpleClientset(&storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-class"},
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
	})

	pvcClient := &pvc.Client{ClientSet: clientset}

	ctx := context.TODO()

	// Test case where no error occurs
	result, err := ShouldWaitForFirstConsumer(ctx, "test-storage-class", pvcClient)
	assert.NoError(t, err)
	assert.True(t, result)

	// Test case where an error occurs
	clientset.Fake.PrependReactor("get", "storageclasses", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("simulated error")
	})

	result, err = ShouldWaitForFirstConsumer(ctx, "test-storage-class", pvcClient)
	assert.Error(t, err)
	assert.False(t, result)
}

func TestValidateCustomName(t *testing.T) {
	tests := []struct {
		name     string
		volumes  int
		expected bool
	}{
		{"ValidName", 1, true},
		{"InvalidNameMultipleVolumes", 2, false},
		{"InvalidNameEmpty", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var customName string
			if tt.name == "InvalidNameEmpty" {
				customName = ""
			} else {
				customName = "custom-name"
			}
			result := ValidateCustomName(customName, tt.volumes)
			assert.Equal(t, tt.expected, result)
		})
	}
}