// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/google/go-containerregistry/pkg/name"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// ImageRefValid verifies that all image references contain valid tag and digest.
type ImageRefValid struct {
	ExcludeContrastImages bool
}

// Verify verifies that neither the tag nor the digest of image references are empty.
func (v *ImageRefValid) Verify(toVerify any) error {
	var findings error

	kuberesource.MapPodSpec(toVerify, func(
		spec *applycorev1.PodSpecApplyConfiguration,
	) *applycorev1.PodSpecApplyConfiguration {
		if spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			return spec
		}

		for _, container := range spec.Containers {
			if container.Image == nil || *container.Image == "" {
				findings = errors.Join(findings, fmt.Errorf("the container '%s' failed verification. It has no image specified", *container.Name))
				continue
			}
			if v.ExcludeContrastImages && strings.HasPrefix(*container.Image, "ghcr.io/edgelesssys/contrast") {
				continue
			}
			if _, err := name.NewDigest(*container.Image); err != nil {
				findings = errors.Join(findings, fmt.Errorf("the image reference '%s' failed verification. Ensure that it contains a digest and is in the format 'image:tag@sha256:...'", *container.Image))
			}
		}
		return spec
	})

	return findings
}
