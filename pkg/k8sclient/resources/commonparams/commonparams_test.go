package commonparams

import (
	"reflect"
	"testing"
)

func TestConstants(t *testing.T) {
	testCases := []struct {
		name string
		want string
		got  string
	}{
		{"RemoteClusterID", RemoteClusterID, "replication.storage.dell.com/remoteClusterID"},
		{"ReplicationGroupName", ReplicationGroupName, "replication.storage.dell.com/replicationGroupName"},
		{"RemoteStorageClassName", RemoteStorageClassName, "replication.storage.dell.com/remoteStorageClassName"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.want != tc.got {
				t.Errorf("want %s, got %s", tc.want, tc.got)
			}
		})
	}
}
func TestLocalPVCAnnotations(t *testing.T) {
	expected := []string{RemoteClusterID, ReplicationGroupName, RemoteStorageClassName}

	if !reflect.DeepEqual(LocalPVCAnnotations, expected) {
		t.Errorf("LocalPVCAnnotations: expected %v, got %v", expected, LocalPVCAnnotations)
	}
}

func TestLocalPVCLabels(t *testing.T) {
	expected := []string{RemoteClusterID, ReplicationGroupName}

	if !reflect.DeepEqual(LocalPVCLabels, expected) {
		t.Errorf("LocalPVCLabels: expected %v, got %v", expected, LocalPVCLabels)
	}
}

func TestLocalPVAnnotation(t *testing.T) {
	expected := []string{ReplicationGroupName, RemoteStorageClassName}

	if !reflect.DeepEqual(LocalPVAnnotation, expected) {
		t.Errorf("LocalPVAnnotation: expected %v, got %v", expected, LocalPVAnnotation)
	}
}

func TestLocalPVLabels(t *testing.T) {
	expected := []string{ReplicationGroupName}

	if !reflect.DeepEqual(LocalPVLabels, expected) {
		t.Errorf("LocalPVLabels: expected %v, got %v", expected, LocalPVLabels)
	}
}

func TestRemotePVAnnotations(t *testing.T) {
	expected := []string{ReplicationGroupName}

	if !reflect.DeepEqual(RemotePVAnnotations, expected) {
		t.Errorf("RemotePVAnnotations: expected %v, got %v", expected, RemotePVAnnotations)
	}
}

func TestRemotePVLabels(t *testing.T) {
	expected := []string{ReplicationGroupName}

	if !reflect.DeepEqual(RemotePVLabels, expected) {
		t.Errorf("RemotePVLabels: expected %v, got %v", expected, RemotePVLabels)
	}
}
