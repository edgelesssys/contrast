// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/regclient/regclient/types/ref"

	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// VersionsMatch verifies that the cli version matches the version of the used resources.
type VersionsMatch struct {
	Version string
}

// Verify verifies that the cli version matches the version of the used resources.
func (v *VersionsMatch) Verify(toVerify any) error {
	// ignore this verifier for preview and dev builds.
	if strings.Contains(v.Version, "dev") || strings.Contains(v.Version, "pre") {
		return nil
	}

	var findings error

	kuberesource.MapPodSpecWithMeta(toVerify, func(
		meta *applymetav1.ObjectMetaApplyConfiguration,
		spec *applycorev1.PodSpecApplyConfiguration,
	) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
		if spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			return meta, spec
		}

		for _, container := range spec.Containers {
			if !needsImageVersionCheck(*container.Name, meta.Annotations["contrast.edgeless.systems/pod-role"]) {
				continue
			}
			if container.Image == nil || *container.Image == "" {
				continue
			}

			r, err := ref.New(*container.Image)
			if err != nil {
				findings = errors.Join(findings, fmt.Errorf("could not parse image reference %q: %w", *container.Image, err))
				continue
			}
			if r.Tag == "" {
				continue
			}

			if r.Tag != v.Version {
				findings = errors.Join(findings, fmt.Errorf("version mismatch: you are attempting to use Contrast version %q with a %q resource of version %q",
					v.Version, *container.Name, r.Tag))
			}
		}
		return meta, spec
	})

	return findings
}

func needsImageVersionCheck(containerName string, podRole string) bool {
	if slices.Contains([]string{"contrast-initializer", "contrast-service-mesh"}, containerName) {
		return true
	}
	if containerName == "coordinator" && podRole == "coordinator" {
		return true
	}
	return false
}
