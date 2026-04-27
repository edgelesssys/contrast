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

// CPUCountValid verifies that all workloads have a valid vCPU count.
type CPUCountValid struct{}

// Verify checks that pods do not require more than the currently supported 8 vCPUs.
func (v *CPUCountValid) Verify(toVerify any) error {
	var findings error

	kuberesource.MapPodSpec(toVerify, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if spec == nil || spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			return spec
		}

		cpuCount := kuberesource.GetPodCPUCount(spec)
		if cpuCount < 1 || cpuCount > 220 {
			// TODO(charludo): find way to add pod name to error message
			findings = errors.Join(findings, fmt.Errorf("pod failed verification: currently only 0-7 additional vCPUs are supported (1-8 vCPUs in total), but %d were requested", cpuCount))
		}

		return spec
	})

	return findings
}
