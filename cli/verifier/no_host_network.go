// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"

	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// NoHostNetwork rejects pods using the host network.
//
// Using the host network namespace is not supported by Kata and may break the node.
func NoHostNetwork(spec *applycorev1.PodSpecApplyConfiguration) error {
	if spec.HostNetwork != nil && *spec.HostNetwork {
		return errors.New("hostNetwork=true is not supported by Contrast")
	}
	return nil
}
