// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func TestPodVCPUCount(t *testing.T) {
	container := func(cpu string) *applycorev1.ContainerApplyConfiguration {
		return applycorev1.Container().WithResources(applycorev1.ResourceRequirements().WithLimits(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu)}))
	}
	testCases := map[string]struct {
		spec *applycorev1.PodSpecApplyConfiguration
		want int64
	}{
		"empty":       {applycorev1.PodSpec(), 1},
		"fractional":  {applycorev1.PodSpec().WithContainers(container("100m")), 2},
		"regular sum": {applycorev1.PodSpec().WithContainers(container("600m"), container("600m")), 3},
		"init sum":    {applycorev1.PodSpec().WithInitContainers(container("600m"), container("600m")), 3},
		"sidecar": {applycorev1.PodSpec().
			WithContainers(container("1")).
			WithInitContainers(container("500m").WithRestartPolicy(corev1.ContainerRestartPolicyAlways)), 3},
		"pod resources": {applycorev1.PodSpec().WithResources(
			applycorev1.ResourceRequirements().WithLimits(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")}),
		), 5},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, PodVCPUCount(tc.spec))
		})
	}
}

func TestIsContrastWorkload(t *testing.T) {
	testCases := map[string]struct {
		runtimeClass string
		want         bool
	}{
		"no runtime class": {
			runtimeClass: "",
			want:         false,
		},
		"non-contrast runtime class": {
			runtimeClass: "foobar",
			want:         false,
		},
		"contrast-cc": {
			runtimeClass: "contrast-cc",
			want:         true,
		},
		"contrast-cc-metal-qemu-snp": {
			runtimeClass: "contrast-cc-metal-qemu-snp",
			want:         true,
		},
		"contrast-insecure": {
			runtimeClass: "contrast-insecure",
			want:         true,
		},
		"contrast-insecure-metal-qemu": {
			runtimeClass: "contrast-insecure-metal-qemu",
			want:         true,
		},
	}

	resourceTypes := []any{
		applycorev1.Pod("test", ""),
		applyappsv1.Deployment("test", "").WithSpec(applyappsv1.DeploymentSpec().WithTemplate(applycorev1.PodTemplateSpec())),
		applyappsv1.StatefulSet("test", "").WithSpec(applyappsv1.StatefulSetSpec().WithTemplate(applycorev1.PodTemplateSpec())),
		applyappsv1.DaemonSet("test", "").WithSpec(applyappsv1.DaemonSetSpec().WithTemplate(applycorev1.PodTemplateSpec())),
	}

	for name, tc := range testCases {
		for _, resource := range resourceTypes {
			t.Run(fmt.Sprintf("%s-%T", name, resource), func(t *testing.T) {
				spec := applycorev1.PodSpec()
				if tc.runtimeClass != "" {
					spec.WithRuntimeClassName(tc.runtimeClass)
				}
				resource = MapPodSpec(resource, func(*applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
					return spec
				})
				assert.Equal(t, tc.want, IsContrastWorkload(resource))
			})
		}
	}
}
