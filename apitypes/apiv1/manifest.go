// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apiv1

// SetManifestRequest is the request body of POST /v1/manifest.
type SetManifestRequest struct {
	// Manifest is the JSON-encoded manifest to set.
	Manifest []byte `json:"manifest"`
	// Policies are the policies referenced by the manifest.
	Policies [][]byte `json:"policies"`
	// Signatures are workload owner's signature(s) over the manifest.
	//
	// Over HTTP this is the only supported way to authorize an update to an existing
	// manifest, because the Coordinator can't authenticate the caller by its client
	// certificate as it does for aTLS-based gRPC calls.
	//
	// The Coordinator currently accepts at most one signature.
	Signatures [][]byte `json:"signatures,omitempty"`
}

// SetManifestResponse is the response body of POST /v1/manifest.
type SetManifestResponse struct {
	// Version is the Coordinator version.
	Version string `json:"version"`
	// RootCA is the PEM-encoded certificate of the deployment's root CA.
	RootCA []byte `json:"root_ca"`
	// MeshCA is the PEM-encoded certificate of the deployment's mesh CA.
	MeshCA []byte `json:"mesh_ca"`
	// SeedSharesDoc is only set when the initial manifest was set.
	SeedSharesDoc *SeedShareDocument `json:"seed_shares_doc,omitempty"`
}

// SeedShareDocument contains the secret seed, encrypted for the seedshare owners.
type SeedShareDocument struct {
	// SeedShares holds the seed, encrypted once per seedshare owner.
	SeedShares []SeedShare `json:"seed_shares"`
	// Salt is used together with the seed to derive the Coordinator's secrets.
	Salt []byte `json:"salt"`
}

// SeedShare is the secret seed, encrypted with a single seedshare owner's public key.
type SeedShare struct {
	// PublicKey is the hex-encoded public key of the seedshare owner.
	PublicKey string `json:"public_key"`
	// EncryptedSeed is the seed, encrypted with PublicKey.
	EncryptedSeed []byte `json:"encrypted_seed"`
}
