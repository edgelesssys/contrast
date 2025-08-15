// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"

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
	var volumeMounts []applycorev1.VolumeMountApplyConfiguration
	kuberesource.MapPodSpec(toVerify, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		for _, container := range spec.Containers {
			volumeMounts = append(volumeMounts, container.VolumeMounts...)
		}
		return spec
	})

	// get all volumes that are referenced in the resource
	var volumes []applycorev1.VolumeApplyConfiguration
	kuberesource.MapPodSpec(toVerify, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		volumes = append(volumes, spec.Volumes...)
		return spec
	})

	// check if referenced volumes are okay
	for _, mount := range volumeMounts {
		for _, volume := range volumes {
			if *mount.Name != *volume.Name {
				continue
			}
			if volume.ConfigMap != nil {
				continue
			}
			if volume.DownwardAPI != nil {
				continue
			}
			if volume.EmptyDir != nil {
				continue
			}
			if volume.Ephemeral != nil {
				continue
			}
			if volume.Projected != nil {
				continue
			}
			if volume.Secret != nil {
				continue
			}

			// a volume with invalid type is referenced
			findings = errors.Join(findings, fmt.Errorf("the volume %q has an unsupported type, which may cause unexpected behavior, please make sure your volumes are either a configMap, downwardAPI, emptyDir, ephemeral, projected or secret", *volume.Name))
		}
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

	for _, claim := range volumeClaims {
		if *claim.Spec.VolumeMode != "Block" {
			findings = errors.Join(findings, fmt.Errorf("volume claim %q does not have volumeMode=Block, which is unsupported", claim.Name))
		}
	}

	return findings
}
