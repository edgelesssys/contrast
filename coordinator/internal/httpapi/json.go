// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

// decodeJSONRequest checks that r is a request with the given method and a JSON body of at most
// maxBodySize bytes, and decodes the body into v.
//
// If it isn't, an error response is written and false is returned, in which case the caller must
// not process the request any further.
func decodeJSONRequest(w http.ResponseWriter, r *http.Request, method string, maxBodySize int64, v any) bool {
	if r.Method != method {
		w.Header().Set("Allow", method)
		writeJSONError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return false
	}

	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return false
	}
	if mediaType != "application/json" {
		writeJSONError(w, http.StatusUnsupportedMediaType, errContentType)
		return false
	}

	bodyReader := http.MaxBytesReader(w, r.Body, maxBodySize)
	defer bodyReader.Close()
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, maxBytesErr)
			return false
		}
		writeJSONError(w, http.StatusBadRequest, err)
		return false
	}

	if err := json.Unmarshal(body, v); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

// writeJSONResponse writes resp as the JSON body of a successful response.
func writeJSONResponse(w http.ResponseWriter, resp any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(resp); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
	}
}
