// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/userapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// maxSetManifestBodySize limits the accepted request body size. A manifest and its policies
// are much smaller than this, but they're the largest input the Coordinator accepts.
// At 128 B per manifest policy entry, this is sufficient for more than 100k entries.
const maxSetManifestBodySize = 16 << 20 // 16 MiB

// errTooManySignatures rejects requests with more than one signature.
var errTooManySignatures = errors.New("at most one signature is supported")

// ManifestSetter sets a manifest. It is a *userapi.Server at runtime, but can be stubbed in tests.
type ManifestSetter interface {
	SetManifest(context.Context, *userapi.SetManifestRequest) (*userapi.SetManifestResponse, error)
}

// SetManifestHandler handles POST requests to /v1/manifest.
//
// It's a thin translation layer in front of the gRPC UserAPI's SetManifest: the request is
// converted to its protobuf equivalent, handled by the same server logic, and the response is
// converted back to JSON.
//
// The endpoint is deliberately not idempotent. Replaying a request that was already applied fails with
// [apitypes.ErrorCodeManifestConflict] instead of applying the manifest twice.
// A client that wants to retry has to re-read the current transition hash and re-sign the request.
type SetManifestHandler struct {
	UserAPI ManifestSetter
}

// ServeHTTP implements [http.Handler].
func (h *SetManifestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !checkMediaType(w, r) {
		return
	}

	bodyReader := http.MaxBytesReader(w, r.Body, maxSetManifestBodySize)
	defer bodyReader.Close()
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, maxBytesErr)
			return
		}
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	var req apitypesv1.SetManifestRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Signatures) > 1 {
		writeJSONError(w, http.StatusBadRequest, errTooManySignatures)
		return
	}
	var signature []byte
	if len(req.Signatures) == 1 {
		signature = req.Signatures[0]
	}

	resp, err := h.UserAPI.SetManifest(r.Context(), &userapi.SetManifestRequest{
		Manifest:  req.Manifest,
		Policies:  req.Policies,
		Signature: signature,
	})
	if err != nil {
		writeJSONError(w, httpStatusFromGRPC(err), err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(setManifestResponse(resp)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
	}
}

// setManifestResponse converts the gRPC response to its wire-format equivalent.
func setManifestResponse(resp *userapi.SetManifestResponse) *apitypesv1.SetManifestResponse {
	out := &apitypesv1.SetManifestResponse{
		Version: constants.Version,
		RootCA:  resp.GetRootCA(),
		MeshCA:  resp.GetMeshCA(),
	}
	doc := resp.GetSeedSharesDoc()
	if doc == nil {
		return out
	}

	shares := make([]apitypesv1.SeedShare, 0, len(doc.GetSeedShares()))
	for _, share := range doc.GetSeedShares() {
		shares = append(shares, apitypesv1.SeedShare{
			PublicKey:     share.GetPublicKey(),
			EncryptedSeed: share.GetEncryptedSeed(),
		})
	}
	out.SeedSharesDoc = &apitypesv1.SeedShareDocument{
		SeedShares: shares,
		Salt:       doc.GetSalt(),
	}
	return out
}

// httpStatusFromGRPC maps the gRPC status codes returned by the UserAPI to HTTP status codes.
func httpStatusFromGRPC(err error) int {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	default:
		return http.StatusInternalServerError
	}
}
