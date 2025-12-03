// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/httpapi"
	"github.com/edgelesssys/contrast/internal/manifest"
)

var (
	errContentType        = errors.New("invalid Content-Type")
	errNonceLength        = errors.New("invalid nonce length")
	errGettingState       = errors.New("getting state")
	errGettingHistory     = errors.New("getting history")
	errGettingAttestation = errors.New("getting attestation report")
)

// StateGuard is a stateguard.Guard at runtime, but can be stubbed in tests.
type StateGuard interface {
	GetState(context.Context) (*stateguard.State, error)
	GetHistory(ctx context.Context) ([][]byte, map[manifest.HexString][]byte, error)
}

// AttestationHandler handles POST requests to /attest.
type AttestationHandler struct {
	Issuer     atls.Issuer
	StateGuard StateGuard
}

func (h *AttestationHandler) getResponse(ctx context.Context, nonce []byte) (*httpapi.AttestationResponse, int, error) {
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

	manifests, policies, err := h.StateGuard.GetHistory(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGettingHistory, err)
	}

	ca := state.CA()
	coordinatorState := &httpapi.CoordinatorState{
		Manifests: manifests,
		RootCA:    ca.GetRootCACert(),
		MeshCA:    ca.GetMeshCACert(),
	}
	for _, policy := range policies {
		coordinatorState.Policies = append(coordinatorState.Policies, policy)
	}

	transitionHash := state.LatestTransition().Digest()
	reportData := httpapi.ConstructReportData(nonce, transitionHash[:], coordinatorState)
	attestation, err := h.Issuer.Issue(ctx, reportData)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGettingAttestation, err)
	}

	resp := &httpapi.AttestationResponse{
		Version:           constants.Version,
		RawAttestationDoc: attestation,
		CoordinatorState:  *coordinatorState,
	}

	return resp, http.StatusOK, nil
}

func (h *AttestationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
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
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	var req httpapi.AttestationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	if len(req.Nonce) != 32 {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("%w: got %d, expected 32", errNonceLength, len(req.Nonce)))
		return
	}

	ctx := r.Context()
	resp, errCode, err := h.getResponse(ctx, req.Nonce)
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

	apiErr := &httpapi.AttestationError{
		Version: constants.Version,
		Err:     err.Error(),
	}
	if errEncode := json.NewEncoder(w).Encode(apiErr); err != nil {
		log.Printf("encoding error response %v: %v", err, errEncode)
	}
}
