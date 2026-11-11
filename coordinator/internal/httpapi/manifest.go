// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/userapi"
)

// maxSetManifestBodySize limits the accepted request body size.
//
// The manifest and its policies are the largest input the Coordinator accepts. The limit leaves room for the
// gRPC API's default message size limit of 4 MiB with the overhead of their base64 encoding in JSON.
const maxSetManifestBodySize = 16 << 20

var (
	errTransitionHashLength = errors.New("invalid previous transition hash length")
	errEncodingSeedShares   = errors.New("encoding seed share document")
)

// ManifestSetter sets a manifest authorized by a workload owner signature.
type ManifestSetter interface {
	SetManifestSigned(context.Context, *userapi.SetManifestRequest) (*userapi.SetManifestResponse, *stateguard.State, error)
}

// SetManifestHandler handles POST requests to the manifest endpoint of API version v1.
//
// The response reports the state the manifest was set in, attested under the nonce of the request.
type SetManifestHandler struct {
	Issuer     atls.Issuer
	StateGuard StateGuard
	UserAPI    ManifestSetter
}

// ServeHTTP implements [http.Handler].
func (h *SetManifestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req apitypesv1.SetManifestRequest
	if !decodeJSONRequest(w, r, apitypesv1.SetManifestMethod, maxSetManifestBodySize, &req) {
		return
	}
	if len(req.Nonce) != apitypesv1.NonceSize {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("%w: got %d, expected %d", errNonceLength, len(req.Nonce), apitypesv1.NonceSize))
		return
	}
	if len(req.PreviousTransitionHash) != 0 && len(req.PreviousTransitionHash) != apitypesv1.DigestSize {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("%w: got %d, expected %d", errTransitionHashLength, len(req.PreviousTransitionHash), apitypesv1.DigestSize))
		return
	}

	resp, status, err := h.setManifest(r.Context(), &req)
	if err != nil {
		writeJSONError(w, status, err)
		return
	}
	writeJSONResponse(w, resp)
}

func (h *SetManifestHandler) setManifest(ctx context.Context, req *apitypesv1.SetManifestRequest) (*apitypesv1.SetManifestResponse, int, error) {
	// An empty previous transition hash means the update isn't atomic, which the UserAPI expects as nil.
	var previousTransitionHash []byte
	if len(req.PreviousTransitionHash) != 0 {
		previousTransitionHash = req.PreviousTransitionHash
	}
	grpcResp, state, err := h.UserAPI.SetManifestSigned(ctx, &userapi.SetManifestRequest{
		Manifest:               req.Manifest,
		Policies:               req.Policies,
		PreviousTransitionHash: previousTransitionHash,
		Signature:              req.Signature,
	})
	if err != nil {
		return nil, httpStatusFromGRPC(err), err
	}

	resp := &apitypesv1.SetManifestResponse{
		Version:         constants.Version,
		AttestationType: h.Issuer.OID(),
	}
	if grpcResp.SeedSharesDoc != nil {
		// Encoded as `contrast set` writes it to disk, which is what `contrast recover` reads.
		resp.SeedSharesDoc, err = json.Marshal(grpcResp.SeedSharesDoc)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errEncodingSeedShares, err)
		}
	}

	// From here on, the manifest is set. Errors don't undo that, but the client can find out about the
	// manifest by attesting the Coordinator.
	coordinatorState, err := getCoordinatorState(ctx, h.StateGuard, state)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	resp.CoordinatorState = *coordinatorState

	transitionHash := state.LatestTransition().TransitionHash
	reportData, err := apitypesv1.ConstructSetManifestReportData(req.Nonce, transitionHash[:], resp)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("constructing report data: %w", err)
	}
	resp.RawAttestationDoc, err = h.Issuer.Issue(ctx, reportData)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGettingAttestation, err)
	}
	return resp, http.StatusOK, nil
}
