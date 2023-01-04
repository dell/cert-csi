/*
 *
 * Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
