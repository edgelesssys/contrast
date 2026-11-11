// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"context"
	"crypto/hpke"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/manifest"
)

var (
	errMethodNotAllowed   = errors.New("method not allowed")
	errContentType        = errors.New("invalid Content-Type")
	errNonceLength        = errors.New("invalid nonce length")
	errGettingState       = errors.New("getting state")
	errGettingHistory     = errors.New("getting history")
	errGettingAttestation = errors.New("getting attestation report")
	errGeneratingHPKEKey  = errors.New("generating HPKE key")
)

// StateGuard is a stateguard.Guard at runtime, but can be stubbed in tests.
type StateGuard interface {
	GetState(context.Context) (*stateguard.State, error)
	GetHistory(ctx context.Context) ([][]byte, map[manifest.HexString][]byte, error)
}

// AttestationHandler handles POST requests to the unversioned attestation endpoint, which predates API versioning.
type AttestationHandler struct {
	Issuer     atls.Issuer
	StateGuard StateGuard
}

func (h *AttestationHandler) getResponse(ctx context.Context, nonce []byte) (*apitypesv1.AttestationResponse, int, error) {
	// state knows the latest transition
	state, err := h.StateGuard.GetState(ctx)
	switch {
	case errors.Is(err, stateguard.ErrNoState):
		return nil, http.StatusPreconditionFailed, userapi.ErrNoManifest
	case errors.Is(err, stateguard.ErrStaleState):
		return nil, http.StatusPreconditionFailed, userapi.ErrNeedsRecovery
	case err != nil:
		return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGettingState, err)
	}

	coordinatorState, err := getCoordinatorState(ctx, h.StateGuard, state)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	transitionHash := state.LatestTransition().TransitionHash
	reportData := apitypesv1.ConstructReportData(nonce, transitionHash[:], coordinatorState)
	attestation, err := h.Issuer.Issue(ctx, reportData)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGettingAttestation, err)
	}

	resp := &apitypesv1.AttestationResponse{
		Version:           constants.Version,
		AttestationType:   h.Issuer.OID(),
		RawAttestationDoc: attestation,
		CoordinatorState:  *coordinatorState,
	}

	return resp, http.StatusOK, nil
}

func (h *AttestationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serveAttestation(w, r, h.getResponse)
}

// VersionedAttestationHandler handles POST requests to the attestation endpoint of API version v1.
//
// Unlike the unversioned endpoint, it also answers if the Coordinator has no manifest or awaits recovery,
// and reports which of these is the case. Each response carries a fresh HPKE public key of the Coordinator,
// whose private key is kept in PendingKeys for a follow-up request.
type VersionedAttestationHandler struct {
	Issuer     atls.Issuer
	StateGuard StateGuard
	// CapabilitiesDigest is the digest of the capabilities, see [apitypes.CapabilitiesResponse.Digest].
	// It is returned with the attestation and bound into the report data.
	CapabilitiesDigest [sha256.Size]byte
	// PendingKeys receives the private key of the HPKE key pair generated for each exchange.
	PendingKeys *PendingKeys
}

func (h *VersionedAttestationHandler) getResponse(ctx context.Context, nonce []byte) (*apitypesv1.AttestationResponse, int, error) {
	resp := &apitypesv1.AttestationResponse{
		Version:            constants.Version,
		AttestationType:    h.Issuer.OID(),
		CapabilitiesDigest: h.CapabilitiesDigest[:],
	}

	// The transition digest stays empty unless the Coordinator is active, as does the Coordinator state.
	var transitionDigest []byte
	state, err := h.StateGuard.GetState(ctx)
	switch {
	case errors.Is(err, stateguard.ErrNoState):
		resp.Status = apitypesv1.StatusUninitialized
	case errors.Is(err, stateguard.ErrStaleState):
		resp.Status = apitypesv1.StatusStale
	case err != nil:
		return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGettingState, err)
	default:
		resp.Status = apitypesv1.StatusActive
		coordinatorState, err := getCoordinatorState(ctx, h.StateGuard, state)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		resp.CoordinatorState = *coordinatorState
		latest := state.LatestTransition()
		transitionDigest = latest.TransitionHash[:]
		resp.LatestTransitionSignature = latest.Signature
	}

	// The key isn't bound to the Coordinator's state, so that a Coordinator awaiting recovery can hand one out, too.
	hpkeKey, err := hpke.MLKEM768X25519().GenerateKey()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGeneratingHPKEKey, err)
	}
	resp.HPKEPubKey = hpkeKey.PublicKey().Bytes()

	reportData, err := apitypesv1.ConstructAttestationReportData(nonce, transitionDigest, resp)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("constructing report data: %w", err)
	}
	resp.RawAttestationDoc, err = h.Issuer.Issue(ctx, reportData)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGettingAttestation, err)
	}

	// Only keep the key once there is a report vouching for it.
	h.PendingKeys.Put([apitypesv1.NonceSize]byte(nonce), hpkeKey)

	return resp, http.StatusOK, nil
}

func (h *VersionedAttestationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serveAttestation(w, r, h.getResponse)
}

// getCoordinatorState collects the state an active Coordinator reports on its attestation endpoints.
func getCoordinatorState(ctx context.Context, guard StateGuard, state *stateguard.State) (*apitypesv1.CoordinatorState, error) {
	manifests, policies, err := guard.GetHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGettingHistory, err)
	}

	ca := state.CA()
	coordinatorState := &apitypesv1.CoordinatorState{
		Manifests: manifests,
		RootCA:    ca.GetRootCACert(),
		MeshCA:    ca.GetMeshCACert(),
	}
	for _, policy := range policies {
		coordinatorState.Policies = append(coordinatorState.Policies, policy)
	}
	return coordinatorState, nil
}

// serveAttestation handles a request to an attestation endpoint. The endpoints share their request format
// and differ only in the response, which getResponse provides for the nonce of the request.
func serveAttestation(w http.ResponseWriter, r *http.Request, getResponse func(ctx context.Context, nonce []byte) (*apitypesv1.AttestationResponse, int, error)) {
	if r.Method != apitypesv1.AttestMethod {
		w.Header().Set("Allow", apitypesv1.AttestMethod)
		writeJSONError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}
	if mediaType != "application/json" {
		writeJSONError(w, http.StatusUnsupportedMediaType, errContentType)
		return
	}

	// Limit size to a small value to avoid abuse (nonce only expected).
	bodyReader := http.MaxBytesReader(w, r.Body, 1024)
	defer bodyReader.Close()
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeJSONError(w, http.StatusRequestEntityTooLarge, maxBytesErr)
			return
		}
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	var req apitypesv1.AttestationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	if len(req.Nonce) != apitypesv1.NonceSize {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("%w: got %d, expected %d", errNonceLength, len(req.Nonce), apitypesv1.NonceSize))
		return
	}

	ctx := r.Context()
	resp, errCode, err := getResponse(ctx, req.Nonce)
	if err != nil {
		writeJSONError(w, errCode, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(resp); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	log.Print(err.Error())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	apiErr := &apitypesv1.AttestationError{
		Version:    constants.Version,
		StatusCode: status,
		Code:       errorCode(err, status),
		Err:        err.Error(),
	}
	if errEncode := json.NewEncoder(w).Encode(apiErr); errEncode != nil {
		log.Printf("encoding error response %v failed: %v", err, errEncode)
	}
}
