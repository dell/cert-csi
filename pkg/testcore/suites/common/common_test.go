package common

import (
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/observer"
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
