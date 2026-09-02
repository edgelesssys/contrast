// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/constants"
	"google.golang.org/grpc/status"
)

var errContentType = errors.New("invalid Content-Type")

// checkMediaType ensures the request declares Content-Type: application/json.
//
// If it doesn't, an error response is written and false is returned, in which case the
// caller must not process the request any further.
func checkMediaType(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return false
	}
	if mediaType != "application/json" {
		writeJSONError(w, http.StatusUnsupportedMediaType, errContentType)
		return false
	}
	return true
}

func writeJSONError(w http.ResponseWriter, statusCode int, err error) {
	log.Print(err.Error())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	apiErr := &apitypes.APIError{
		Version:    constants.Version,
		StatusCode: statusCode,
		Code:       errorCode(err, statusCode),
		// Unwrap gRPC status errors, so that clients don't see the gRPC framing of an
		// error that didn't travel over gRPC. For other errors, this is err.Error().
		Err: status.Convert(err).Message(),
	}
	if errEncode := json.NewEncoder(w).Encode(apiErr); errEncode != nil {
		log.Printf("encoding error response %v failed: %v", err, errEncode)
	}
}
