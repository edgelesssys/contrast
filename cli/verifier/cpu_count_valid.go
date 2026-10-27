// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"fmt"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// CPUCountValid verifies that all workloads have a valid vCPU count.
func CPUCountValid(spec *applycorev1.PodSpecApplyConfiguration) error {
	cpuCount := kuberesource.GetPodCPUCount(spec)
	if cpuCount < 1 || cpuCount > 220 {
		return fmt.Errorf("currently only 1-220 vCPUs are supported, but %d were requested", cpuCount)
	}
	return nil
}
