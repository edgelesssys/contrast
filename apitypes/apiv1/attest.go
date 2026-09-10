// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apiv1

import (
	"bytes"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/json"
	"fmt"

	"github.com/edgelesssys/contrast/apitypes"
)

// ReportDataSize is the size of the SNP/TDX REPORTDATA fields.
const ReportDataSize = 64

// AttestationRequest is the wire-format for incoming /attest requests.
// The nonce is expected to be base64-encoded.
type AttestationRequest struct {
	Nonce []byte `json:"nonce"`
}

// AttestationError is returned by the /attest endpoint if the request was not successful.
//
// It is an alias of [apitypes.APIError], which all Contrast HTTP API endpoints return on error.
type AttestationError = apitypes.APIError

// AttestationResponse contains all fields required for application-level verification.
type AttestationResponse struct {
	// Version is the Coordinator version.
	Version string `json:"version"`
	// RawAttestationDoc is a raw attestation report.
	RawAttestationDoc []byte `json:"raw_attestation_doc"`
	// AttestationType is the OID used to identify the type of attestation document.
	//
	// Outside of unit tests, this will always be an OID from the internal/oid package.
	AttestationType asn1.ObjectIdentifier `json:"attestation_type"`

	CoordinatorState
}

// UnmarshalAttestationResponse parses a JSON-serialized AttestationResponse.
//
// If parsing fails, it tries to find a version indicator in the data and reports it back to the
// caller.
func UnmarshalAttestationResponse(data []byte) (*AttestationResponse, error) {
	var resp AttestationResponse
	origErr := json.NewDecoder(bytes.NewBuffer(data)).Decode(&resp)
	if origErr == nil {
		return &resp, nil
	}

	var unstructured map[string]any
	if err := json.NewDecoder(bytes.NewBuffer(data)).Decode(&unstructured); err != nil {
		return nil, &unmarshalError{err: origErr}
	}
	switch version := unstructured["version"].(type) {
	case string:
		return nil, &unmarshalError{version: version, err: origErr}
	default:
		return nil, &unmarshalError{err: origErr}
	}
}

type unmarshalError struct {
	version string
	err     error
}

func (e *unmarshalError) Error() string {
	version := "unknown"
	if e.version != "" {
		version = e.version
	}
	return fmt.Sprintf("unmarshalling API response (server version %s): %s", version, e.err.Error())
}

func (e *unmarshalError) Unwrap() error {
	return e.err
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
//
// capabilitiesDigest is the SHA-256 digest of the raw /capabilities response body.
// Binding it into the report data lets clients detect tampering with the unauthenticated capabilities endpoint.
func ConstructReportData(nonce []byte, transitionDigest []byte, capabilitiesDigest []byte, state *CoordinatorState) [ReportDataSize]byte {
	// reportdata = sha256(nonce || sha256(transition) || sha256(root-ca) || sha256(mesh-ca) || sha256(capabilities))
	rootCADigest := sha256.Sum256(state.RootCA)
	meshCADigest := sha256.Sum256(state.MeshCA)

	reportdata := append([]byte{}, nonce...)
	reportdata = append(reportdata, transitionDigest...)
	reportdata = append(reportdata, rootCADigest[:]...)
	reportdata = append(reportdata, meshCADigest[:]...)
	reportdata = append(reportdata, capabilitiesDigest...)
	hash32 := sha256.Sum256(reportdata)

	var hash64 [64]byte
	copy(hash64[:], hash32[:])

	return hash64
}
