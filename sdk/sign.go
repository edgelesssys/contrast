// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/edgelesssys/contrast/internal/history"
)

// SignManifestTransition signs the transition to manifestBytes with a workload owner key, for use as the
// signature of a request that sets a manifest.
//
// previousTransitionHash is the hash of the Coordinator's latest transition, see [CoordinatorState.LatestTransitionHash],
// or empty for the initial manifest. The signature is only valid for a Coordinator still at that transition.
//
// The signature scheme is the same for all API versions, and for the gRPC API.
func SignManifestTransition(key *ecdsa.PrivateKey, manifestBytes []byte, previousTransitionHash []byte) ([]byte, error) {
	tr := &history.Transition{ManifestHash: history.Digest(manifestBytes)}
	switch len(previousTransitionHash) {
	case 0:
	case history.HashSize:
		tr.PreviousTransitionHash = [history.HashSize]byte(previousTransitionHash)
	default:
		return nil, fmt.Errorf("invalid previous transition hash length: got %d, want %d", len(previousTransitionHash), history.HashSize)
	}
	digest := tr.OwnerSigningDigest()
	return ecdsa.SignASN1(rand.Reader, key, digest[:])
}
