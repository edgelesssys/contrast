// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"github.com/edgelesssys/contrast/internal/manifest"
	corev1 "k8s.io/api/core/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	metav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// IsContrastWorkload applies IsContrastPod to the resource and returns the result.
func IsContrastWorkload(resource any) (ret bool) {
	MapPodSpec(resource, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		ret = IsContrastPod(spec)
		return spec
	})
	return ret
}

// IsCoordinator reports whether the resource is a Coordinator.
//
// This function considers all pod-generating resources in Kubernetes that carry the
// pod-role=coordinator label.
func IsCoordinator(resource any) bool {
	var ret bool
	MapPodSpecWithMeta(resource, func(meta *metav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (*metav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
		if meta != nil && meta.Labels[ContrastRoleLabelKey] == string(manifest.RoleCoordinator) {
			ret = true
		}
		return meta, spec
	})
	return ret
}

// PodVCPUCount returns the number of VM CPUs Kata will configure for this PodSpec.
//
// The calculation has two parts:
//  1. Kubernetes calculates the requirements for the entire pod.
//     https://github.com/kubernetes/kubernetes/blob/v1.31.5/pkg/api/v1/resource/helpers.go#L152
//  2. Kata adds an additional CPU for overhead by guest services.
//     https://github.com/kata-containers/kata-containers/blob/4.3.0/src/runtime/virtcontainers/sandbox.go#L2586-L2587
func PodVCPUCount(spec *applycorev1.PodSpecApplyConfiguration) int64 {
	var regular, init int64
	for _, container := range spec.Containers {
		regular += cpuLimitMilli(container.Resources)
	}
	for _, container := range spec.InitContainers {
		cpus := cpuLimitMilli(container.Resources)
		init += cpus
		if container.RestartPolicy != nil && *container.RestartPolicy == corev1.ContainerRestartPolicyAlways {
			regular += cpus
		}
	}
	return (max(regular, init, cpuLimitMilli(spec.Resources))+999)/1000 + 1
}

func cpuLimitMilli(resources *applycorev1.ResourceRequirementsApplyConfiguration) int64 {
	if resources != nil && resources.Limits != nil {
		return resources.Limits.Cpu().MilliValue()
	}
	return 0
}
