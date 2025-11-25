// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"github.com/edgelesssys/contrast/internal/history"
)

// ReportDataSize is the size of the SNP/TDX REPORTDATA fields.
const ReportDataSize = 64

// AttestationRequest is the wire-format for incoming /attest requests.
// The nonce is expected to be base64-encoded.
type AttestationRequest struct {
	Nonce []byte `json:"nonce"`
}

// AttestationError is returned by the /attest endpoint if the request was not successful.
type AttestationError struct {
	Err string `json:"error"`
}

// AttestationResponse contains all fields required for application-level verification.
type AttestationResponse struct {
	// RawAttestationDoc is a raw attestation report.
	RawAttestationDoc []byte `json:"raw_attestation_doc"`

	CoordinatorState
}

// CoordinatorState represents the state of the Contrast Coordinator at a fixed point in time.
type CoordinatorState struct {
	// Manifests is a slice of manifests. It represents the manifest history of the Coordinator it was received from, sorted from oldest to newest.
	Manifests [][]byte `json:"manifests"`
	// Policies contains all policies that have been referenced in any manifest in Manifests. Used to verify the guarantees a deployment had over its lifetime.
	Policies [][]byte `json:"policies"`
	// PEM-encoded certificate of the deployment's root CA.
	RootCA []byte `json:"root_ca"`
	// PEM-encoded certificate of the deployment's mesh CA.
	MeshCA []byte `json:"mesh_ca"`
}

// ConstructReportData constructs an extended report data digest,
// intended for use with application-level verification.
func ConstructReportData(nonce []byte, transitionDigest []byte, state *CoordinatorState) [ReportDataSize]byte {
	// reportdata = sha256(nonce || sha256(transition) || sha256(root-ca) || sha256(mesh-ca))
	rootCADigest := history.Digest(state.RootCA)
	meshCADigest := history.Digest(state.MeshCA)

	reportdata := append([]byte{}, nonce...)
	reportdata = append(reportdata, transitionDigest...)
	reportdata = append(reportdata, rootCADigest[:]...)
	reportdata = append(reportdata, meshCADigest[:]...)
	hash32 := history.Digest(reportdata)

	var hash64 [64]byte
	copy(hash64[:], hash32[:])

	return hash64
}
