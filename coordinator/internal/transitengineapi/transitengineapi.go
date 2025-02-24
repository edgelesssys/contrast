// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// transitengine provides all functionality related to the transit engine API endpoints: decrypt and encrypt.
// It is organized in a layered approach, keeping http request processing separated from the underlying crypto
// business logic(crypto.go).
package transitengine

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/edgelesssys/contrast/coordinator/internal/authority"
)

const (
	// aesGCMNonceSize specifies the default nonce size in bytes used in AES GCM, used during parsing of ciphertextContainer.
	aesGCMNonceSize = 12
	// aesGCMKeySize specifies the default key size in bytes to use AES-256 GCM.
	aesGCMKeySize = 32
)

type (
	// encryptionRequest holds the request-specific plaintext and currently supported, optional query parameters: associatedData and keyVersion.
	encryptionRequest struct {
		Plaintext      []byte `json:"plaintext"`
		KeyVersion     uint32 `json:"key_version"`
		AssociatedData []byte `json:"associated_data,omitempty"`
	}
	// decryptionRequest holds the request-specific ciphertextContainer and currently supported, optional query parameters: associatedData.
	decryptionRequest struct {
		CiphertextContainer ciphertextContainer `json:"ciphertext"`
		AssociatedData      []byte              `json:"associated_data,omitempty"`
	}
	// encryptionResponse holds the response-specific ciphertextContainer.
	encryptionResponse struct {
		Ciphertext ciphertextContainer `json:"ciphertext"`
	}
	// decryptionResponse holds the response-specific plaintext.
	decryptionResponse struct {
		Plaintext []byte `json:"plaintext"`
	}
)

type stateAuthority interface {
	GetState() (*authority.State, error)
}

// NewTransitEngineAPI sets up the transit engine API with a provided seedEngineAuthority.
func NewTransitEngineAPI(authority stateAuthority, _ *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	// 'name' wildcard is kept to reflect existing transit engine API specifications:
	// https://openbao.org/api-docs/secret/transit/#encrypt-data
	// name <=> workloadSecretID, which should be used for the key derivation.
	mux.Handle("/v1/transit/encrypt/{name}", getEncryptHandler(authority))
	mux.Handle("/v1/transit/decrypt/{name}", getDecryptHandler(authority))

	return mux
}

func getEncryptHandler(authority stateAuthority) http.HandlerFunc {
	// TODO(jmxnzo): Implement Vault json error bodies
	return func(w http.ResponseWriter, r *http.Request) {
		workloadSecretID := r.PathValue("name")
		if workloadSecretID == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		var encReq encryptionRequest
		if err := parseRequest(r, &encReq); err != nil {
			http.Error(w, fmt.Sprintf("parsing encryption request: %v", err), http.StatusBadRequest)
			return
		}
		key, err := deriveEncryptionKey(authority, fmt.Sprintf("%d_%s", encReq.KeyVersion, workloadSecretID))
		if err != nil {
			http.Error(w, fmt.Sprintf("key derivation: %v", err), http.StatusInternalServerError)
			return
		}
		ciphertextContainer, err := symmetricEncryptRaw(key, encReq.Plaintext, encReq.AssociatedData)
		if err != nil {
			http.Error(w, fmt.Sprintf("encrypting: %v", err), http.StatusInternalServerError)
			return
		}
		ciphertextContainer.keyVersion = encReq.KeyVersion
		var encResp encryptionResponse
		encResp.Ciphertext = ciphertextContainer
		if err = writeJSONResponse(w, encResp); err != nil {
			http.Error(w, fmt.Sprintf("writing response: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func getDecryptHandler(authority stateAuthority) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workloadSecretID := r.PathValue("name")
		if workloadSecretID == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		var decReq decryptionRequest
		if err := parseRequest(r, &decReq); err != nil {
			http.Error(w, fmt.Sprintf("parsing decryption request: %v", err), http.StatusBadRequest)
			return
		}
		key, err := deriveEncryptionKey(authority, fmt.Sprintf("%d_%s", decReq.CiphertextContainer.keyVersion, workloadSecretID))
		if err != nil {
			http.Error(w, fmt.Sprintf("key derivation: %v", err), http.StatusInternalServerError)
			return
		}
		plaintext, err := symmetricDecryptRaw(key, decReq.CiphertextContainer, decReq.AssociatedData)
		if err != nil {
			http.Error(w, fmt.Sprintf("decrypting: %v", err), http.StatusInternalServerError)
			return
		}
		var decResp decryptionResponse
		decResp.Plaintext = plaintext
		if err = writeJSONResponse(w, decResp); err != nil {
			http.Error(w, fmt.Sprintf("writing response: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// deriveEncryptionKey derives the workload secret used as the encryption key by receiving the seedengine of the current state.
func deriveEncryptionKey(authority stateAuthority, workloadSecretID string) ([]byte, error) {
	state, err := authority.GetState()
	if err != nil {
		return nil, err
	}
	// TODO(jmxnzo): authentication of client certs <-> parsed workloadSecretID.
	derivedWorkloadSecret, err := state.SeedEngine.DeriveWorkloadSecret(workloadSecretID)
	if err != nil {
		return nil, err
	}
	if len(derivedWorkloadSecret) < aesGCMKeySize {
		return nil, fmt.Errorf("derived key too small, expected key length: %d", aesGCMKeySize)
	}
	return derivedWorkloadSecret[:aesGCMKeySize], nil
}

// writeJSONResponse wraps any payload inside a "data" object and sends it as an HTTP response.
func writeJSONResponse(w http.ResponseWriter, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]any{
		"data": payload,
	}
	return json.NewEncoder(w).Encode(response)
}

// parseRequest parses the given HTTP request body into the struct.
func parseRequest(r *http.Request, into any) error {
	defer r.Body.Close()
	if err := validateContentType(r); err != nil {
		return err
	}
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		return err
	}
	return nil
}

// validateContentType ensures that if Content-Type is present, it is set to application/json.
func validateContentType(r *http.Request) error {
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		if !strings.HasPrefix(contentType, "application/json") {
			return fmt.Errorf("invalid content-type: %s (want application/json)", contentType)
		}
	}
	return nil
}

// extractVersion is a helper function checking the version string format 'vX' and extracting the corresponding version as uint32.
func extractVersion(versionStr string) (uint32, error) {
	re := regexp.MustCompile(`^v(\d+)$`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid format: %s", versionStr)
	}
	version, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse version: %w", err)
	}

	return uint32(version), nil
}
