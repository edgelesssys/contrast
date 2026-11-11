// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package policies checks the policies a Coordinator reports alongside its manifest history.
package policies

import (
	"encoding/json"
	"fmt"

	"github.com/edgelesssys/contrast/internal/history"
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
		for policyRef := range m.Policies {
			// The Coordinator decodes the references, so they're not necessarily in canonical (lowercase) form.
			policyHash, err := policyRef.Bytes()
			if err != nil {
				return fmt.Errorf("decoding policy hash %q in manifest %d: %w", policyRef, i, err)
			}
			referenced[manifest.NewHexString(policyHash)] = struct{}{}
		}
	}

	reported := make(map[manifest.HexString]struct{}, len(policies))
	for _, policy := range policies {
		// Manifests reference policies by the hash the Coordinator stores them under, independent of the initdata algorithm.
		digest := history.Digest(policy)
		policyHash := manifest.NewHexString(digest[:])
		if _, ok := referenced[policyHash]; !ok {
			return fmt.Errorf("policy %s isn't referenced by any manifest", policyHash)
		}
		// The Coordinator reports a policy once per distinct reference string, so a policy referenced in different
		// cases is reported multiple times. Identical hashes mean identical policies, so duplicates are harmless.
		reported[policyHash] = struct{}{}
	}

	for policyHash := range referenced {
		if _, ok := reported[policyHash]; !ok {
			return fmt.Errorf("policy %s referenced by the manifest history is missing", policyHash)
		}
	}
	return nil
}
