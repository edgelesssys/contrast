// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package policies checks the policies a Coordinator reports alongside its manifest history.
package policies

import (
	"encoding/json"
	"fmt"

	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/manifest"
)

// CheckReferenced returns an error unless policies are exactly the policies referenced by any of the manifests.
//
// The attestation binds the manifest history, but not the policies, so they must be checked against it.
func CheckReferenced(manifests [][]byte, policies [][]byte) error {
	referenced := make(map[manifest.HexString]struct{})
	for i, manifestBytes := range manifests {
		var m manifest.Manifest
		if err := json.Unmarshal(manifestBytes, &m); err != nil {
			return fmt.Errorf("unmarshalling manifest %d: %w", i, err)
		}
		for policyHash := range m.Policies {
			referenced[policyHash] = struct{}{}
		}
	}

	reported := make(map[manifest.HexString]struct{}, len(policies))
	for i, policy := range policies {
		digest, err := initdata.Raw(policy).Digest()
		if err != nil {
			return fmt.Errorf("digesting policy %d: %w", i, err)
		}
		policyHash := manifest.NewHexString(digest)
		if _, ok := referenced[policyHash]; !ok {
			return fmt.Errorf("policy %s isn't referenced by any manifest", policyHash)
		}
		if _, ok := reported[policyHash]; ok {
			return fmt.Errorf("policy %s is reported more than once", policyHash)
		}
		reported[policyHash] = struct{}{}
	}

	for policyHash := range referenced {
		if _, ok := reported[policyHash]; !ok {
			return fmt.Errorf("policy %s referenced by the manifest history is missing", policyHash)
		}
	}
	return nil
}
