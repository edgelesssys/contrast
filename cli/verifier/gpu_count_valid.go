// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// GPUCountValid verifies that GPU workloads request at least as many vCPUs as GPUs.
type GPUCountValid struct{}

// Verify checks that pods request at least as many vCPUs as GPUs.
// Up to one guest NUMA node per passed-through GPU is created, and every guest NUMA node needs at least one vCPU.
// A pod with fewer vCPUs than GPUs can randomly fail to boot.
// Strictly we only need vCPUs >= min(|GPUs|, |host NUMA nodes|).
func (v *GPUCountValid) Verify(toVerify any) error {
	var findings error

	kuberesource.MapPodSpec(toVerify, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if spec == nil || spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			return spec
		}

		gpuCount := kuberesource.GetPodGPUCount(spec)
		if gpuCount == 0 {
			return spec
		}

		cpuCount := kuberesource.GetPodCPUCount(spec)
		if cpuCount < gpuCount {
			findings = errors.Join(findings, fmt.Errorf("pod failed verification: GPU workloads require at least as many vCPUs as GPUs, but %d vCPUs were requested for %d GPUs", cpuCount, gpuCount))
		}

		return spec
	})

	return findings
}
