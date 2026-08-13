// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/userapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// maxSetManifestBodySize limits the accepted request body size. A manifest and its policies
// are much smaller than this, but they're the largest input the Coordinator accepts.
const maxSetManifestBodySize = 16 << 20 // 16 MiB

// ManifestSetter sets a manifest. It is a *userapi.Server at runtime, but can be stubbed in tests.
type ManifestSetter interface {
	SetManifest(context.Context, *userapi.SetManifestRequest) (*userapi.SetManifestResponse, error)
}

// SetManifestHandler handles POST requests to /v1/manifest.
//
// It's a thin translation layer in front of the gRPC UserAPI's SetManifest: the request is
// converted to its protobuf equivalent, handled by the same server logic, and the response is
// converted back to JSON.
type SetManifestHandler struct {
	UserAPI ManifestSetter
}

// ServeHTTP implements [http.Handler].
func (h *SetManifestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var req apitypes.SetManifestRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	resp, err := h.UserAPI.SetManifest(r.Context(), &userapi.SetManifestRequest{
		Manifest:               req.Manifest,
		Policies:               req.Policies,
		PreviousTransitionHash: req.PreviousTransitionHash,
		Signature:              req.Signature,
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
func setManifestResponse(resp *userapi.SetManifestResponse) *apitypes.SetManifestResponse {
	out := &apitypes.SetManifestResponse{
		Version: constants.Version,
		RootCA:  resp.GetRootCA(),
		MeshCA:  resp.GetMeshCA(),
	}
	doc := resp.GetSeedSharesDoc()
	if doc == nil {
		return out
	}

	shares := make([]apitypes.SeedShare, 0, len(doc.GetSeedShares()))
	for _, share := range doc.GetSeedShares() {
		shares = append(shares, apitypes.SeedShare{
			PublicKey:     share.GetPublicKey(),
			EncryptedSeed: share.GetEncryptedSeed(),
		})
	}
	out.SeedSharesDoc = &apitypes.SeedShareDocument{
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
