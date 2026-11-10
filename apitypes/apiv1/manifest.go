// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apiv1

import (
	"encoding/asn1"
	"net/http"
)

const (
	// ManifestPath is the path of the manifest endpoint.
	ManifestPath = "/" + Version + "/manifest"
	// SetManifestMethod is the HTTP method that sets a manifest.
	SetManifestMethod = http.MethodPost
)

// SetManifestRequest is the request body of [SetManifestMethod] [ManifestPath].
type SetManifestRequest struct {
	// Nonce is a 32-byte random value, which the response's attestation report is bound to.
	Nonce []byte `json:"nonce"`
	// Manifest is the JSON-encoded manifest to set.
	Manifest []byte `json:"manifest"`
	// Policies are the policies referenced by the manifest.
	Policies [][]byte `json:"policies"`
	// PreviousTransitionHash optionally makes the update atomic: if set, the manifest is only set
	// if the Coordinator's latest transition has this hash. It's all zeroes for the initial manifest.
	PreviousTransitionHash []byte `json:"previous_transition_hash,omitempty"`
	// Signature is a workload owner's signature over the transition the request makes, in the
	// scheme of the `contrast sign` command:
	//
	//	h   = Transition{ManifestHash: sha256(manifest), PreviousTransitionHash: latest}.Digest()
	//	sig = ECDSA-Sign(workloadOwnerKey, sha256(hex(h)))
	//
	// latest is the hash of the Coordinator's latest transition, or all zeroes for the initial manifest.
	// Updates are verified against the WorkloadOwnerPubKeys of the manifest in force, the initial manifest
	// against its own.
	//
	// It may only be empty for an initial manifest that lists no workload owner keys.
	Signature []byte `json:"signature,omitempty"`
}

// SetManifestResponse is the response body of [SetManifestMethod] [ManifestPath].
//
// It reports the state the Coordinator is in after setting the manifest, attested under the
// request's nonce, see [ConstructSetManifestReportData].
type SetManifestResponse struct {
	// Version is the Coordinator version.
	Version string `json:"version"`
	// RawAttestationDoc is a raw attestation report.
	RawAttestationDoc []byte `json:"raw_attestation_doc"`
	// AttestationType is the OID used to identify the type of attestation document.
	AttestationType asn1.ObjectIdentifier `json:"attestation_type"`
	// SeedSharesDoc is the JSON-encoded seed share document, in the format `contrast recover` reads.
	// It's only set for the initial manifest.
	//
	// It is bound into the report data as transmitted, see [ConstructSetManifestReportData].
	SeedSharesDoc []byte `json:"seed_shares_doc,omitempty"`

	CoordinatorState
}

// setManifestLabel names the report data construction of [ManifestPath].
const setManifestLabel = "contrast manifest v1"

// ConstructSetManifestReportData constructs the report data of a [SetManifestResponse].
//
//	reportData = sha256(sha256("contrast manifest v1") || nonce || transitionDigest ||
//	             sha256(rootCA) || sha256(meshCA) || sha256(seedSharesDoc))
//
// The CA certificates and the seed share document are taken from resp.
// transitionDigest is the digest of the transition the manifest was set with.
func ConstructSetManifestReportData(nonce []byte, transitionDigest []byte, resp *SetManifestResponse) ([ReportDataSize]byte, error) {
	if err := fixed("nonce", nonce, NonceSize); err != nil {
		return [ReportDataSize]byte{}, err
	}
	if err := fixed("transition digest", transitionDigest, DigestSize); err != nil {
		return [ReportDataSize]byte{}, err
	}
	return asReportData(constructDigest(
		setManifestLabel,
		nonce,
		transitionDigest,
		hashed(resp.RootCA),
		hashed(resp.MeshCA),
		hashed(resp.SeedSharesDoc),
	)), nil
}
