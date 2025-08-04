// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

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
					findings = errors.Join(findings, fmt.Errorf("the volume %q has an unsupported type, which may cause unexpected behavior, please make sure your volumes are either a configMap, downwardAPI, emptyDir, ephemeral, projected or secret", *volume.Name))
				}
				if !volumeReferenced {
					findings = errors.Join(findings, fmt.Errorf("mount %q doesn't referenced any existing volume", *mount.Name))
				}
			}
		}
		return spec
	})
	if isNonCC {
		// we don't care about non-confidential pods
		return nil
	}

	// get all stateful sets from the resources to verify
	unstructuredResources, err := kuberesource.ResourcesToUnstructured([]any{toVerify})
	if err != nil {
		return err
	}
	var volumeClaims []corev1.PersistentVolumeClaim
	for _, r := range unstructuredResources {
		switch r.GetKind() {
		case "StatefulSet":
			var statefulSet appsv1.StatefulSet
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(r.UnstructuredContent(), &statefulSet)
			if err != nil {
				return err
			}
			volumeClaims = append(volumeClaims, statefulSet.Spec.VolumeClaimTemplates...)
		}
	}

	// verify all volume claims
	for _, claim := range volumeClaims {
		if *claim.Spec.VolumeMode != "Block" {
			findings = errors.Join(findings, fmt.Errorf("volume claim %q does not have volumeMode=Block, which is unsupported", claim.Name))
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
