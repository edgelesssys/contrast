// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/kuberesource"

	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

const errorMessageTemplate = "volumeMount %q refers to an unsupported %s; supported types are ConfigMap, DownwardAPI, EmptyDir, Ephemeral, Projected and Secret"

// NoSharedFSMount verifies that no filesystems are shared with the root.
type NoSharedFSMount struct{}

// Verify verifies that no filesystems are shared with the root.
func (v *NoSharedFSMount) Verify(toVerify any) error {
	var findings error

	// get all volume mounts that are referenced in containers
	isNonCC := false
	kuberesource.MapPodSpec(toVerify, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			// this isn't a confidential pod so we don't need to check further
			isNonCC = true
			return spec
		}
		// verify all volume mounts of all containers in pod
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
		return spec
	})
	if isNonCC {
		// we don't care about non-confidential pods
		return nil
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
		return true
	}
	if volume.Ephemeral != nil {
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
