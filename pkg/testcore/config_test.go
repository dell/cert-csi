/*
 *
 * Copyright © 2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package testcore

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/statefulset"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/volumegroupsnapshot"
	v1 "k8s.io/api/core/v1"
)

func TestVolumeCreationConfig(t *testing.T) {
	tests := []struct {
		storageclass string
		claimSize    string
		name         string
		accessMode   string
		want         *pvc.Config
	}{
		{
			storageclass: "test-sc",
			claimSize:    "1Gi",
			name:         "test-name",
			accessMode:   "ReadWriteOnce",
			want: &pvc.Config{
				Name:             "test-name",
				NamePrefix:       "vol-create-test-",
				ClaimSize:        "1Gi",
				AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				StorageClassName: &[]string{"test-sc"}[0],
			},
		},
		{
			storageclass: "test-sc",
			claimSize:    "2Gi",
			name:         "test-name-2",
			accessMode:   "ReadWriteMany",
			want: &pvc.Config{
				Name:             "test-name-2",
				NamePrefix:       "vol-create-test-",
				ClaimSize:        "2Gi",
				AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
				StorageClassName: &[]string{"test-sc"}[0],
			},
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("storageclass=%s,claimSize=%s,name=%s,accessMode=%s", tt.storageclass, tt.claimSize, tt.name, tt.accessMode), func(t *testing.T) {
			got := VolumeCreationConfig(tt.storageclass, tt.claimSize, tt.name, tt.accessMode)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VolumeCreationConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiAttachVolumeConfig(t *testing.T) {
	tests := []struct {
		storageclass string
		claimSize    string
		want         *pvc.Config
	}{
		{
			storageclass: "test-sc",
			claimSize:    "1Gi",
			want: &pvc.Config{
				NamePrefix:       "vol-multi-pod-test-",
				ClaimSize:        "1Gi",
				AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				StorageClassName: &[]string{"test-sc"}[0],
			},
		},
		{
			storageclass: "test-sc",
			claimSize:    "2Gi",
			want: &pvc.Config{
				NamePrefix:       "vol-multi-pod-test-",
				ClaimSize:        "2Gi",
				AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				StorageClassName: &[]string{"test-sc"}[0],
			},
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("storageclass=%s,claimSize=%s", tt.storageclass, tt.claimSize), func(t *testing.T) {
			got := MultiAttachVolumeConfig(tt.storageclass, tt.claimSize, "test-name")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MultiAttachVolumeConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapacityTrackingPodConfig(t *testing.T) {
	tests := []struct {
		name           string
		pvcNames       []string
		podName        string
		containerImage string
		want           *pod.Config
	}{
		{
			name:           "Test with single PVC",
			pvcNames:       []string{"pvc1"},
			podName:        "test-pod-1",
			containerImage: "test-image-1",
			want: &pod.Config{
				Name:           "test-pod-1",
				NamePrefix:     "pod-capacity-tracking-test-",
				PvcNames:       []string{"pvc1"},
				VolumeName:     "vol",
				MountPath:      "/data",
				ContainerName:  "capacity-tracking-test",
				ContainerImage: "test-image-1",
				Command:        []string{"/bin/bash"},
				Args:           []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
			},
		},
		{
			name:           "Test with multiple PVCs",
			pvcNames:       []string{"pvc1", "pvc2"},
			podName:        "test-pod-2",
			containerImage: "test-image-2",
			want: &pod.Config{
				Name:           "test-pod-2",
				NamePrefix:     "pod-capacity-tracking-test-",
				PvcNames:       []string{"pvc1", "pvc2"},
				VolumeName:     "vol",
				MountPath:      "/data",
				ContainerName:  "capacity-tracking-test",
				ContainerImage: "test-image-2",
				Command:        []string{"/bin/bash"},
				Args:           []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CapacityTrackingPodConfig(tt.pvcNames, tt.podName, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CapacityTrackingPodConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvisioningPodConfig(t *testing.T) {
	tests := []struct {
		name           string
		pvcNames       []string
		podName        string
		containerImage string
		want           *pod.Config
	}{
		{
			name:           "Test with single PVC",
			pvcNames:       []string{"pvc1"},
			podName:        "test-pod-1",
			containerImage: "test-image-1",
			want: &pod.Config{
				Name:           "test-pod-1",
				NamePrefix:     "pod-prov-test-",
				PvcNames:       []string{"pvc1"},
				VolumeName:     "vol",
				MountPath:      "/data",
				ContainerName:  "prov-test",
				ContainerImage: "test-image-1",
				Command:        []string{"/bin/bash"},
				Args:           []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProvisioningPodConfig(tt.pvcNames, tt.podName, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProvisioningPodConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeHealthPodConfig(t *testing.T) {
	tests := []struct {
		name           string
		pvcNames       []string
		podName        string
		containerImage string
		want           *pod.Config
	}{
		{
			name:           "Test with single PVC",
			pvcNames:       []string{"pvc1"},
			podName:        "test-pod-1",
			containerImage: "test-image-1",
			want: &pod.Config{
				Name:           "test-pod-1",
				NamePrefix:     "pod-volume-health-test-",
				PvcNames:       []string{"pvc1"},
				VolumeName:     "vol",
				MountPath:      "/data",
				ContainerName:  "volume-health-test",
				ContainerImage: "test-image-1",
				Command:        []string{"/bin/bash"},
				Args:           []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VolumeHealthPodConfig(tt.pvcNames, tt.podName, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VolumeHealthPodConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIoWritePodConfig(t *testing.T) {
	tests := []struct {
		name           string
		pvcNames       []string
		podName        string
		containerImage string
		want           *pod.Config
	}{
		{
			name:           "Test with single PVC",
			pvcNames:       []string{"pvc1"},
			podName:        "test-pod-1",
			containerImage: "test-image-1",
			want: &pod.Config{
				Name:           "test-pod-1",
				NamePrefix:     "iowriter-test-",
				PvcNames:       []string{"pvc1"},
				VolumeName:     "vol",
				MountPath:      "/data",
				ContainerName:  "iowriter",
				ContainerImage: "test-image-1",
				Command:        []string{"/bin/bash"},
				Args:           []string{"-c", " trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IoWritePodConfig(tt.pvcNames, tt.podName, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IoWritePodConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockSnapPodConfig(t *testing.T) {
	tests := []struct {
		name           string
		pvcNames       []string
		containerImage string
		want           *pod.Config
	}{
		{
			name:           "Test with single PVC",
			pvcNames:       []string{"pvc1"},
			containerImage: "test-image-1",
			want: &pod.Config{
				NamePrefix:     "bs-test-",
				VolumeName:     "vol",
				MountPath:      "/data",
				ContainerName:  "iowriter",
				ContainerImage: "test-image-1",
				PvcNames:       []string{"pvc1"},
				Command:        []string{"/bin/bash"},
				Args:           []string{"-c", " trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
				Capabilities:   []v1.Capability{"SYS_ADMIN"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BlockSnapPodConfig(tt.pvcNames, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BlockSnapPodConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiAttachPodConfig(t *testing.T) {
	tests := []struct {
		name           string
		pvcNames       []string
		containerImage string
		want           *pod.Config
	}{
		{
			name:           "Test with single PVC",
			pvcNames:       []string{"pvc1"},
			containerImage: "test-image-1",
			want: &pod.Config{
				NamePrefix:     "iowriter-test-",
				PvcNames:       []string{"pvc1"},
				VolumeName:     "vol",
				MountPath:      "/data",
				ContainerName:  "iowriter",
				ContainerImage: "test-image-1",
				Command:        []string{"/bin/bash"},
				Args:           []string{"-c", " trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
				Capabilities:   []v1.Capability{"SYS_ADMIN"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MultiAttachPodConfig(tt.pvcNames, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MultiAttachPodConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScalingStsConfig(t *testing.T) {
	tests := []struct {
		name           string
		storageclass   string
		claimSize      string
		volumeNumber   int
		podPolicy      string
		containerImage string
		want           *statefulset.Config
	}{
		{
			name:           "Test with default values",
			storageclass:   "standard",
			claimSize:      "1Gi",
			volumeNumber:   1,
			podPolicy:      "OrderedReady",
			containerImage: "test-image-1",
			want: &statefulset.Config{
				VolumeNumber:        1,
				Replicas:            1,
				VolumeName:          "vol",
				MountPath:           "/data",
				StorageClassName:    "standard",
				PodManagementPolicy: "OrderedReady",
				ClaimSize:           "1Gi",
				ContainerName:       "scale-test",
				ContainerImage:      "test-image-1",
				NamePrefix:          "sts-scale-test",
				Command:             []string{"/bin/bash"},
				Args:                []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
				Labels:              map[string]string{"app": "unified-test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScalingStsConfig(tt.storageclass, tt.claimSize, tt.volumeNumber, tt.podPolicy, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ScalingStsConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPsqlPodConfig(t *testing.T) {
	tests := []struct {
		name           string
		password       string
		containerImage string
		want           *pod.Config
	}{
		{
			name:           "Test with default password",
			password:       "default-password",
			containerImage: "test-image-1",
			want: &pod.Config{
				NamePrefix:     "psql-client-pod-",
				ContainerName:  "psql-client",
				ContainerImage: "test-image-1",
				Command:        []string{"/bin/bash"},
				Args:           []string{"-c", " trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
				EnvVars: []v1.EnvVar{{
					Name:  "PGPASSWORD",
					Value: "default-password",
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PsqlPodConfig(tt.password, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PsqlPodConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEphemeralPodConfig(t *testing.T) {
	tests := []struct {
		name           string
		podName        string
		csiVolSrc      v1.CSIVolumeSource
		containerImage string
		want           *pod.Config
	}{
		{
			name:           "Test with default CSI Volume Source",
			podName:        "test-pod-1",
			csiVolSrc:      v1.CSIVolumeSource{Driver: "test-driver-1"},
			containerImage: "test-image-1",
			want: &pod.Config{
				Name:            "test-pod-1",
				NamePrefix:      "pod-ephemeral-test-",
				VolumeName:      "ephemeral-vol",
				MountPath:       "/data",
				ContainerName:   "prov-test",
				ContainerImage:  "test-image-1",
				Command:         []string{"/bin/bash"},
				Args:            []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
				CSIVolumeSource: v1.CSIVolumeSource{Driver: "test-driver-1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EphemeralPodConfig(tt.podName, tt.csiVolSrc, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EphemeralPodConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeMigrateStsConfig(t *testing.T) {
	tests := []struct {
		name           string
		storageclass   string
		claimSize      string
		volumeNumber   int
		podNumber      int32
		podPolicy      string
		containerImage string
		want           *statefulset.Config
	}{
		{
			name:           "Test with default values",
			storageclass:   "standard",
			claimSize:      "1Gi",
			volumeNumber:   1,
			podNumber:      1,
			podPolicy:      "OrderedReady",
			containerImage: "test-image-1",
			want: &statefulset.Config{
				VolumeNumber:        1,
				Replicas:            1,
				VolumeName:          "vol",
				MountPath:           "/data",
				StorageClassName:    "standard",
				PodManagementPolicy: "OrderedReady",
				ClaimSize:           "1Gi",
				ContainerName:       "volume-migrate-test",
				ContainerImage:      "test-image-1",
				NamePrefix:          "sts-volume-migrate-test",
				Command:             []string{"/bin/bash"},
				Args:                []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
				Labels:              map[string]string{"app": "unified-test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VolumeMigrateStsConfig(tt.storageclass, tt.claimSize, tt.volumeNumber, tt.podNumber, tt.podPolicy, tt.containerImage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VolumeMigrateStsConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVolumeGroupSnapConfig(t *testing.T) {
	tests := []struct {
		name          string
		vgsName       string
		driver        string
		reclaimPolicy string
		snapClass     string
		volumeLabel   string
		namespace     string
		want          *volumegroupsnapshot.Config
	}{
		{
			name:          "Test with provided vgsName",
			vgsName:       "vgs-test-1",
			driver:        "test-driver-1",
			reclaimPolicy: "Retain",
			snapClass:     "snap-class-1",
			volumeLabel:   "volume-label-1",
			namespace:     "namespace-1",
			want: &volumegroupsnapshot.Config{
				Name:          "vgs-test-1",
				DriverName:    "test-driver-1",
				ReclaimPolicy: "Retain",
				SnapClass:     "snap-class-1",
				VolumeLabel:   "volume-label-1",
				Namespace:     "namespace-1",
			},
		},
		{
			name:          "Test with empty vgsName",
			vgsName:       "",
			driver:        "test-driver-2",
			reclaimPolicy: "Delete",
			snapClass:     "snap-class-2",
			volumeLabel:   "volume-label-2",
			namespace:     "namespace-2",
			want: &volumegroupsnapshot.Config{
				DriverName:    "test-driver-2",
				ReclaimPolicy: "Delete",
				SnapClass:     "snap-class-2",
				VolumeLabel:   "volume-label-2",
				Namespace:     "namespace-2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VolumeGroupSnapConfig(tt.vgsName, tt.driver, tt.reclaimPolicy, tt.snapClass, tt.volumeLabel, tt.namespace)
			if tt.vgsName == "" {
				if got.Name == "" || !reflect.DeepEqual(got.DriverName, tt.want.DriverName) || !reflect.DeepEqual(got.ReclaimPolicy, tt.want.ReclaimPolicy) || !reflect.DeepEqual(got.SnapClass, tt.want.SnapClass) || !reflect.DeepEqual(got.VolumeLabel, tt.want.VolumeLabel) || !reflect.DeepEqual(got.Namespace, tt.want.Namespace) {
					t.Errorf("VolumeGroupSnapConfig() = %v, want %v", got, tt.want)
				} else {
					// Check if the generated name follows the expected pattern
					if got.Name[:9] != "vgs-test-" {
						t.Errorf("Generated name %v does not follow the expected pattern", got.Name)
					}
				}
			} else {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("VolumeGroupSnapConfig() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestGetAccessMode(t *testing.T) {
	tests := []struct {
		name       string
		AccessMode string
		want       []v1.PersistentVolumeAccessMode
	}{
		{
			name:       "Test ReadOnlyMany",
			AccessMode: "ReadOnlyMany",
			want:       []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany},
		},
		{
			name:       "Test ReadWriteMany",
			AccessMode: "ReadWriteMany",
			want:       []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
		},
		{
			name:       "Test ReadWriteOncePod",
			AccessMode: "ReadWriteOncePod",
			want:       []v1.PersistentVolumeAccessMode{v1.ReadWriteOncePod},
		},
		{
			name:       "Test default case",
			AccessMode: "UnknownMode",
			want:       []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAccessMode(tt.AccessMode); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAccessMode() = %v, want %v", got, tt.want)
			}
		})
	}
}
