package commonparams

import (
	"reflect"
	"testing"
)

func TestGetRemoteClusterID(t *testing.T) {
	want := "replication.storage.dell.com/remoteClusterID"
	if got := GetRemoteClusterID(); got != want {
		t.Errorf("GetRemoteClusterID() = %v, want %v", got, want)
	}
}

func TestGetReplicationGroupName(t *testing.T) {
	want := "replication.storage.dell.com/replicationGroupName"
	if got := GetReplicationGroupName(); got != want {
		t.Errorf("GetReplicationGroupName() = %v, want %v", got, want)
	}
}

func TestGetRemoteStorageClassName(t *testing.T) {
	want := "replication.storage.dell.com/remoteStorageClassName"
	if got := GetRemoteStorageClassName(); got != want {
		t.Errorf("GetRemoteStorageClassName() = %v, want %v", got, want)
	}
}

func TestGetLocalPVCAnnotations(t *testing.T) {
	want := []string{
		"replication.storage.dell.com/remoteClusterID",
		"replication.storage.dell.com/replicationGroupName",
		"replication.storage.dell.com/remoteStorageClassName",
	}
	if got := GetLocalPVCAnnotations(); !reflect.DeepEqual(got, want) {
		t.Errorf("GetLocalPVCAnnotations() = %v, want %v", got, want)
	}
}

func TestGetLocalPVAnnotations(t *testing.T) {
	want := []string{
		"replication.storage.dell.com/replicationGroupName",
		"replication.storage.dell.com/remoteStorageClassName",
	}
	if got := GetLocalPVAnnotation(); !reflect.DeepEqual(got, want) {
		t.Errorf("GetLocalPVAnnotations() = %v, want %v", got, want)
	}
}

func TestGetLocalPVLabels(t *testing.T) {
	want := []string{
		"replication.storage.dell.com/replicationGroupName",
	}
	if got := GetLocalPVLabels(); !reflect.DeepEqual(got, want) {
		t.Errorf("GetLocalPVLabels() = %v, want %v", got, want)
	}
}

func TestGetRemotePVAnnotations(t *testing.T) {
	want := []string{
		"replication.storage.dell.com/replicationGroupName",
	}
	if got := GetRemotePVAnnotations(); !reflect.DeepEqual(got, want) {
		t.Errorf("GetRemotePVAnnotations() = %v, want %v", got, want)
	}
}

func TestGetRemotePVLabels(t *testing.T) {
	want := []string{
		"replication.storage.dell.com/replicationGroupName",
	}
	if got := GetRemotePVLabels(); !reflect.DeepEqual(got, want) {
		t.Errorf("GetRemotePVLabels() = %v, want %v", got, want)
	}
}

func TestGetLocalPVCLabels(t *testing.T) {
	want := []string{
		"replication.storage.dell.com/remoteClusterID",
		"replication.storage.dell.com/replicationGroupName",
	}
	if got := GetLocalPVCLabels(); !reflect.DeepEqual(got, want) {
		t.Errorf("GetLocalPVCLabels() = %v, want %v", got, want)
	}
}
