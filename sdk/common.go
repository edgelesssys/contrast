// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/manifest"
)

// ValidatorsFromManifest returns a list of validators corresponding to the reference values in the given manifest.
// Originally an unexported function in the contrast CLI.
// Can be made unexported again, if we decide to move all userapi calls from the CLI to the SDK.
// Validators MUST NOT be used concurrently.
func ValidatorsFromManifest(kdsGetter *certcache.CachedHTTPSGetter, m *manifest.Manifest, log *slog.Logger) (validators.Validator, error) {
	v, err := m.CoordinatorValidator(log, kdsGetter)
	if err != nil {
		return nil, fmt.Errorf("creating coordinator validator: %w", err)
	}
	return v, nil
}
