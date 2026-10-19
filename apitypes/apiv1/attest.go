// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apiv1

import (
	"bytes"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
)

// ReportDataSize is the size of the SNP/TDX REPORTDATA fields.
const ReportDataSize = 64

const (
	// AttestPath is the path of the attestation endpoint.
	AttestPath = "/" + Version + "/attest"
	// LegacyAttestPath is the path of the unversioned attestation endpoint, which predates API versioning.
	//
	// It shares the request and response types of [AttestPath], but its response carries only the fields
	// that predate API versioning, and its report data has a layout of its own, see [ConstructReportData].
	// Unlike [AttestPath], it fails if the Coordinator isn't [StatusActive].
	LegacyAttestPath = "/attest"
	// AttestMethod is the HTTP method of both attestation endpoints.
	AttestMethod = http.MethodPost
)

// AttestationRequest is the wire-format for incoming /attest requests.
// The nonce is expected to be base64-encoded.
type AttestationRequest struct {
	Nonce []byte `json:"nonce"`
}

// AttestationError is returned by the /attest endpoint if the request was not successful.
//
// It is an alias of [apitypes.APIError], which all Contrast HTTP API endpoints return on error.
type AttestationError = apitypes.APIError

// Status tells whether a Coordinator has a manifest and the secrets belonging to it.
type Status string

const (
	// StatusUninitialized means no manifest has been set yet.
	StatusUninitialized Status = "uninitialized"
	// StatusActive means the Coordinator has a manifest and its secrets, and serves its state.
	StatusActive Status = "active"
	// StatusStale means the Coordinator has persisted state but no secrets, e.g. after a restart, and needs to be recovered.
	StatusStale Status = "stale"
)

// digestByte returns the single byte a status is bound into a digest as.
func (s Status) digestByte() (byte, error) {
	switch s {
	case StatusUninitialized:
		return 0x00, nil
	case StatusActive:
		return 0x01, nil
	case StatusStale:
		return 0x02, nil
	default:
		return 0, fmt.Errorf("unknown Coordinator status %q", string(s))
	}
}

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
	// CapabilitiesDigest is the digest of the Coordinator's capabilities, see [apitypes.CapabilitiesResponse.Digest].
	//
	// It is bound into the report data, see [ConstructAttestationReportData].
	// Clients must use it to validate the API version they negotiated.
	// The unversioned [LegacyAttestPath] endpoint doesn't set this field.
	CapabilitiesDigest []byte `json:"capabilities_digest,omitempty"`
	// Status is the state of the Coordinator. Unless it is [StatusActive], the CoordinatorState is empty.
	//
	// It is bound into the report data, see [ConstructAttestationReportData].
	// The unversioned [LegacyAttestPath] endpoint doesn't set this field.
	Status Status `json:"status,omitempty"`
	// HPKEPubKey is an ephemeral MLKEM768-X25519 public key of the Coordinator, valid only briefly and only for this exchange.
	// Clients use it to seal secrets to the Coordinator in a follow-up request, which they identify by the nonce of this exchange.
	//
	// It is bound into the report data, see [ConstructAttestationReportData].
	// The unversioned [LegacyAttestPath] endpoint doesn't set this field.
	HPKEPubKey []byte `json:"hpke_pub_key,omitempty"`

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

// ConstructReportData constructs the extended report data digest of the [LegacyAttestPath] endpoint.
func ConstructReportData(nonce []byte, transitionDigest []byte, state *CoordinatorState) [ReportDataSize]byte {
	// reportdata = sha256(nonce || sha256(transition) || sha256(root-ca) || sha256(mesh-ca))
	rootCADigest := sha256.Sum256(state.RootCA)
	meshCADigest := sha256.Sum256(state.MeshCA)

	reportdata := append([]byte{}, nonce...)
	reportdata = append(reportdata, transitionDigest...)
	reportdata = append(reportdata, rootCADigest[:]...)
	reportdata = append(reportdata, meshCADigest[:]...)
	hash32 := sha256.Sum256(reportdata)

	var hash64 [64]byte
	copy(hash64[:], hash32[:])

	return hash64
}

// attestLabel names the report data construction of the [AttestPath] endpoint.
const attestLabel = "contrast attest v1"

// ConstructAttestationReportData constructs the report data of the [AttestPath] endpoint.
//
//	reportData = sha256(sha256("contrast attest v1") || nonce || status || transitionDigest ||
//	             sha256(rootCA) || sha256(meshCA) || sha256(hpkePubKey) || capabilitiesDigest)
//
// The status, the CA certificates, the HPKE public key and the capabilities digest are taken from resp.
// transitionDigest is the digest of the latest transition, and must be empty if the Coordinator isn't
// [StatusActive], in which case it enters as zeroes.
//
// Binding the capabilities digest lets clients detect tampering with the unauthenticated capabilities endpoint.
func ConstructAttestationReportData(nonce []byte, transitionDigest []byte, resp *AttestationResponse) ([ReportDataSize]byte, error) {
	if err := fixed("nonce", nonce, NonceSize); err != nil {
		return [ReportDataSize]byte{}, err
	}
	status, err := resp.Status.digestByte()
	if err != nil {
		return [ReportDataSize]byte{}, err
	}
	if len(transitionDigest) == 0 && resp.Status != StatusActive {
		transitionDigest = make([]byte, DigestSize)
	}
	if err := fixed("transition digest", transitionDigest, DigestSize); err != nil {
		return [ReportDataSize]byte{}, err
	}
	if err := fixed("capabilities digest", resp.CapabilitiesDigest, DigestSize); err != nil {
		return [ReportDataSize]byte{}, err
	}

	return asReportData(constructDigest(
		attestLabel,
		nonce,
		[]byte{status},
		transitionDigest,
		hashed(resp.RootCA),
		hashed(resp.MeshCA),
		hashed(resp.HPKEPubKey),
		resp.CapabilitiesDigest,
	)), nil
}
