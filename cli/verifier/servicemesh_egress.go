// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// ServiceMeshEgressNotEmpty verifies that the `contrast.edgeless.systems/servicemesh-egress` annotation
// isn't empty if it exists.
type ServiceMeshEgressNotEmpty struct{}

// Verify verifies that the `contrast.edgeless.systems/servicemesh-egress` annotation
// isn't empty if it exists.
func (v *ServiceMeshEgressNotEmpty) Verify(toVerify any) error {
	var findings error

	kuberesource.MapPodSpecWithMeta(toVerify, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
		for k, v := range meta.Annotations {
			if k != "contrast.edgeless.systems/servicemesh-egress" {
				continue
			}
			if v == "" {
				findings = errors.Join(findings, errors.New("empty annotation content in \"contrast.edgeless.systems/servicemesh-egress\", which is likely to be an error"))
			}
		}

		return meta, spec
	})

	return findings
}
