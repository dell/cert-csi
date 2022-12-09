package commonparams

const (
	// RemoteClusterID represents remote cluster ID
	RemoteClusterID = "replication.storage.dell.com/remoteClusterID"
	// ReplicationGroupName represents replication group name
	ReplicationGroupName = "replication.storage.dell.com/replicationGroupName"
	// RemotePV represents remove PV
	RemotePV = "replication.storage.dell.com/remotePV"
	// RemoteStorageClassName represents remote storage class
	RemoteStorageClassName = "replication.storage.dell.com/remoteStorageClassName"
	// RemoteVolume represents remote volume
	RemoteVolume = "replication.storage.dell.com/remoteVolume"
	// DriverName represents driver name
	DriverName = "replication.storage.dell.com/driverName"
)

var (
	// LocalPVCAnnotations represents local PVC annotations
	LocalPVCAnnotations = []string{RemoteClusterID, ReplicationGroupName, RemoteStorageClassName}
	// LocalPVCLabels represents local PVC labels
	LocalPVCLabels = []string{RemoteClusterID, ReplicationGroupName}
	// LocalPVAnnotation represents local PV annotation
	LocalPVAnnotation = []string{ReplicationGroupName, RemoteStorageClassName}
	// LocalPVLabels represents local PV labels
	LocalPVLabels = []string{ReplicationGroupName}
	// RemotePVAnnotations represents remote PV annotations
	RemotePVAnnotations = []string{ReplicationGroupName}
	// RemotePVLabels represents remote PV labels
	RemotePVLabels = []string{ReplicationGroupName}
)
