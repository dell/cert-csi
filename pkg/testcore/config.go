/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"crypto/rand"
	"fmt"

	"github.com/dell/cert-csi/pkg/k8sclient/resources/pod"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/pvc"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/statefulset"
	"github.com/dell/cert-csi/pkg/k8sclient/resources/volumegroupsnapshot"

	v1 "k8s.io/api/core/v1"
)

// Image represents an image configuration.
type Image struct {
	Test     string
	Postgres string
}

// Images represents an array of images.
type Images struct {
	Images []Image `yaml:"images"`
}

// VolumeCreationConfig config to use in volumecreation suite
func VolumeCreationConfig(storageclass string, claimSize string, Name string, AccessMode string) *pvc.Config {
	accessMode := GetAccessMode(AccessMode)

	return &pvc.Config{
		Name:             Name,
		NamePrefix:       "vol-create-test-",
		ClaimSize:        claimSize,
		AccessModes:      accessMode,
		StorageClassName: &storageclass,
	}
}

// MultiAttachVolumeConfig config allows to create volume with any AccessMode
func MultiAttachVolumeConfig(storageclass string, claimSize string, AccessMode string) *pvc.Config {
	accessMode := GetAccessMode(AccessMode)
	return &pvc.Config{
		NamePrefix:       "vol-multi-pod-test-",
		ClaimSize:        claimSize,
		AccessModes:      accessMode,
		StorageClassName: &storageclass,
	}
}

// CapacityTrackingPodConfig config to use in capacity-tracking suite
func CapacityTrackingPodConfig(pvcNames []string, podName string, containerImage string) *pod.Config {
	return &pod.Config{
		Name:           podName,
		NamePrefix:     "pod-capacity-tracking-test-",
		PvcNames:       pvcNames,
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "capacity-tracking-test",
		ContainerImage: containerImage,
		Command:        []string{`/bin/bash`},
		Args:           []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
	}
}

// ProvisioningPodConfig config to use in provisioning suite
func ProvisioningPodConfig(pvcNames []string, podName string, containerImage string) *pod.Config {
	return &pod.Config{
		Name:           podName,
		NamePrefix:     "pod-prov-test-",
		PvcNames:       pvcNames,
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "prov-test",
		ContainerImage: containerImage,
		Command:        []string{`/bin/bash`},
		Args:           []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
	}
}

// VolumeHealthPodConfig config to use in provisioning suite
func VolumeHealthPodConfig(pvcNames []string, podName string, containerImage string) *pod.Config {
	return &pod.Config{
		Name:           podName,
		NamePrefix:     "pod-volume-health-test-",
		PvcNames:       pvcNames,
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "volume-health-test",
		ContainerImage: containerImage,
		Command:        []string{`/bin/bash`},
		Args:           []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
	}
}

// IoWritePodConfig config to use in io suite
func IoWritePodConfig(pvcNames []string, podName string, containerImage string) *pod.Config {
	return &pod.Config{
		Name:           podName,
		NamePrefix:     "iowriter-test-",
		PvcNames:       pvcNames,
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "iowriter",
		ContainerImage: containerImage,
		Command:        []string{`/bin/bash`},
		Args:           []string{"-c", " trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
	}
}

// BlockSnapPodConfig config to use in blockvolsnapshot suite
func BlockSnapPodConfig(pvcNames []string, containerImage string) *pod.Config {
	return &pod.Config{
		NamePrefix:     "bs-test-",
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "iowriter",
		ContainerImage: containerImage,
		PvcNames:       pvcNames,
		Command:        []string{`/bin/bash`},
		Args:           []string{"-c", " trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
		Capabilities:   []v1.Capability{"SYS_ADMIN"},
	}
}

// MultiAttachPodConfig config to use in MultiAttachSuite
func MultiAttachPodConfig(pvcNames []string, containerImage string) *pod.Config {
	return &pod.Config{
		NamePrefix:     "iowriter-test-",
		PvcNames:       pvcNames,
		VolumeName:     "vol",
		MountPath:      "/data",
		ContainerName:  "iowriter",
		ContainerImage: containerImage,
		Command:        []string{`/bin/bash`},
		Args:           []string{"-c", " trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
		Capabilities:   []v1.Capability{"SYS_ADMIN"},
	}
}

// ScalingStsConfig config to use in scaling suite
func ScalingStsConfig(storageclass string, claimSize string, volumeNumber int, podPolicy string, containerImage string) *statefulset.Config {
	return &statefulset.Config{
		VolumeNumber:        volumeNumber,
		Replicas:            1,
		VolumeName:          "vol",
		MountPath:           "/data",
		StorageClassName:    storageclass,
		PodManagementPolicy: podPolicy,
		ClaimSize:           claimSize,
		ContainerName:       "scale-test",
		ContainerImage:      containerImage,
		NamePrefix:          "sts-scale-test",
		Command:             []string{`/bin/bash`},
		Args:                []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
		Labels:              map[string]string{"app": "unified-test"},
	}
}

// PsqlPodConfig config to use for psql suite
func PsqlPodConfig(password string, containerImage string) *pod.Config {
	return &pod.Config{
		NamePrefix:     "psql-client-pod-",
		ContainerName:  "psql-client",
		ContainerImage: containerImage,
		Command:        []string{`/bin/bash`},
		Args:           []string{"-c", " trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
		EnvVars: []v1.EnvVar{{
			Name:  "PGPASSWORD",
			Value: password,
		}},
	}
}

// EphemeralPodConfig config to use in ephemeral inline volume suite
func EphemeralPodConfig(podName string, csiVolSrc v1.CSIVolumeSource, containerImage string) *pod.Config {
	return &pod.Config{
		Name:            podName,
		NamePrefix:      "pod-ephemeral-test-",
		VolumeName:      "ephemeral-vol",
		MountPath:       "/data",
		ContainerName:   "prov-test",
		ContainerImage:  containerImage,
		Command:         []string{`/bin/bash`},
		Args:            []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
		CSIVolumeSource: csiVolSrc,
	}
}

// VolumeMigrateStsConfig config to use in scaling suite
func VolumeMigrateStsConfig(storageclass string, claimSize string, volumeNumber int, podNumber int32, podPolicy string, containerImage string) *statefulset.Config {
	return &statefulset.Config{
		VolumeNumber:        volumeNumber,
		Replicas:            podNumber,
		VolumeName:          "vol",
		MountPath:           "/data",
		StorageClassName:    storageclass,
		PodManagementPolicy: podPolicy,
		ClaimSize:           claimSize,
		ContainerName:       "volume-migrate-test",
		ContainerImage:      containerImage,
		NamePrefix:          "sts-volume-migrate-test",
		Command:             []string{`/bin/bash`},
		Args:                []string{"-c", "trap 'exit 0' SIGTERM;while true; do sleep 1; done"},
		Labels:              map[string]string{"app": "unified-test"},
	}
}

// GetAccessMode returns access mode
func GetAccessMode(AccessMode string) []v1.PersistentVolumeAccessMode {
	var accessMode []v1.PersistentVolumeAccessMode
	switch AccessMode {
	case "ReadOnlyMany":
		accessMode = []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany}
	case "ReadWriteMany":
		accessMode = []v1.PersistentVolumeAccessMode{v1.ReadWriteMany}
	case "ReadWriteOncePod":
		accessMode = []v1.PersistentVolumeAccessMode{v1.ReadWriteOncePod}
	default:
		accessMode = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	}
	return accessMode
}

// VolumeGroupSnapConfig config for volume group blockvolsnapshot
func VolumeGroupSnapConfig(vgsName, driver, reclaimPolicy, snapClass, volumeLabel, namespace string) *volumegroupsnapshot.Config {
	if vgsName == "" {
		// we will generate random name
		p, _ := rand.Prime(rand.Reader, 16)
		vgsName = fmt.Sprintf("%s%d", "vgs-test-", p)
	}
	return &volumegroupsnapshot.Config{
		Name:          vgsName,
		DriverName:    driver,
		ReclaimPolicy: reclaimPolicy,
		SnapClass:     snapClass,
		VolumeLabel:   volumeLabel,
		Namespace:     namespace,
	}
}
