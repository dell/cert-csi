package commonparams

const (
	RemoteClusterID        = "replication.storage.dell.com/remoteClusterID"
	ReplicationGroupName   = "replication.storage.dell.com/replicationGroupName"
	RemotePV               = "replication.storage.dell.com/remotePV"
	RemoteStorageClassName = "replication.storage.dell.com/remoteStorageClassName"
	RemoteVolume           = "replication.storage.dell.com/remoteVolume"
	DriverName             = "replication.storage.dell.com/driverName"
)

var (
	LocalPVCAnnotations = []string{RemoteClusterID, ReplicationGroupName, RemoteStorageClassName}
	LocalPVCLabels      = []string{RemoteClusterID, ReplicationGroupName}
	LocalPVAnnotation   = []string{ReplicationGroupName, RemoteStorageClassName}
	LocalPVLabels       = []string{ReplicationGroupName}
	RemotePVAnnotations = []string{ReplicationGroupName}
	RemotePVLabels      = []string{ReplicationGroupName}
)
