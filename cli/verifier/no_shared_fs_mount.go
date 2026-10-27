// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

const errorMessageTemplate = "volumeMount %q refers to an unsupported %s; supported types are ConfigMap, DownwardAPI, EmptyDir, Ephemeral, Projected and Secret"

// NoSharedFSMount verifies that no filesystems are shared with the root.
func NoSharedFSMount(spec *applycorev1.PodSpecApplyConfiguration) error {
	var findings error
	for _, container := range spec.Containers {
		for _, mount := range container.VolumeMounts {
			volumeReferenced := false
			for _, volume := range spec.Volumes {
				if *mount.Name != *volume.Name {
					continue
				}
				volumeReferenced = true

				if isSupportedVolume(volume) {
					continue
				}

				// a volume with invalid type is referenced
				findings = errors.Join(findings, fmt.Errorf(errorMessageTemplate, *mount.Name, "volume type"))
			}
			if !volumeReferenced {
				findings = errors.Join(findings, fmt.Errorf(errorMessageTemplate, *mount.Name, "volume claim template"))
			}
		}
	}
	return findings
}

func isSupportedVolume(volume applycorev1.VolumeApplyConfiguration) bool {
	if volume.ConfigMap != nil {
		return true
	}
	if volume.DownwardAPI != nil {
		return true
	}
	if volume.EmptyDir != nil {
		return volume.EmptyDir.Medium != nil && *volume.EmptyDir.Medium == v1.StorageMediumMemory
	}
	if volume.Ephemeral != nil {
		e := volume.Ephemeral
		if e.VolumeClaimTemplate == nil ||
			e.VolumeClaimTemplate.Spec == nil ||
			e.VolumeClaimTemplate.Spec.VolumeMode == nil ||
			*e.VolumeClaimTemplate.Spec.VolumeMode != v1.PersistentVolumeBlock {
			return false
		}
		return true
	}
	if volume.Projected != nil {
		return true
	}
	if volume.Secret != nil {
		return true
	}
	return false
}
