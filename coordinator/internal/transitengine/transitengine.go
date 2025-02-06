// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// transitengine provides all functionality related to the transit engine API endpoints: decrypt and encrypt.
// It is organized in a layered approach, keeping http request processing separated from the underlying crypto
// business logic(crypto.go).
package transitengine

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/edgelesssys/contrast/coordinator/internal/authority"
)

type (
	// b64Plaintext describes a base64-encoded plaintext.
	b64Plaintext []byte
	// prefixb64Ciphertext describes a base64-encoded ciphertext with prefix 'vault:v1:'.
	prefixb64Ciphertext struct {
		ciphertext []byte
		prefix     string
	}
)

type stateAuthority interface {
	GetState() (*authority.State, error)
}

// NewTransitEngineAPI sets up the transit engine API with a provided seedEngineAuthority.
func NewTransitEngineAPI(authority stateAuthority, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/{version}/transit/encrypt/{name}", getEncryptHandler(authority, logger))
	mux.Handle("/{version}/transit/decrypt/{name}", getDecryptHandler(authority, logger))

	return mux
}

func getEncryptHandler(authority stateAuthority, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := r.PathValue("version")
		workloadSecretID := r.PathValue("name")
		if version == "" || workloadSecretID == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		key, err := deriveEncryptionKey(authority, workloadSecretID+version)
		if err != nil {
			http.Error(w, fmt.Sprint("key derivation: %w", err), http.StatusInternalServerError)
			return
		}
		plaintext, opts, err := parseEncryptionRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprint("parsing encryption request: %w", err), http.StatusBadRequest)
			return
		}
		ciphertext, err := symmetricEncryptRaw(key, plaintext, opts)
		if err != nil {
			http.Error(w, fmt.Sprint("encrypting: %w", err), http.StatusInternalServerError)
			return
		}
		prefixCiphertext := prefixb64Ciphertext{
			ciphertext: ciphertext,
			prefix:     version,
		}
		if err = writeJSONResponse(w, prefixCiphertext); err != nil {
			http.Error(w, fmt.Sprint("writing response: %w", err), http.StatusInternalServerError)
			return
		}
		logger.Debug("Request successful", "addr", r.RemoteAddr, "method", r.Method, "url", r.URL)
	}
}

func getDecryptHandler(authority stateAuthority, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := r.PathValue("version")
		workloadSecretID := r.PathValue("name")
		if version == "" || workloadSecretID == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		key, err := deriveEncryptionKey(authority, workloadSecretID+version)
		if err != nil {
			http.Error(w, fmt.Sprint("key derivation: %w", err), http.StatusInternalServerError)
			return
		}
		prefixCiphertext, opts, err := parseDecryptionRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprint("parsing decryption request: %w", err), http.StatusBadRequest)
			return
		}
		plaintext, err := symmetricDecryptRaw(key, prefixCiphertext.ciphertext, opts)
		if err != nil {
			http.Error(w, fmt.Sprint("decrypting: %w", err), http.StatusInternalServerError)
			return
		}
		b64Plaintext := b64Plaintext(plaintext)
		if err = writeJSONResponse(w, b64Plaintext); err != nil {
			http.Error(w, fmt.Sprint("writing response: %w", err), http.StatusInternalServerError)
			return
		}
		logger.Debug("Request successful", "addr", r.RemoteAddr, "method", r.Method, "url", r.URL)
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
	return derivedWorkloadSecret[:aesGCMKeySize], nil
}

// parseEncryptionRequest parses the HTTP request body into b64Plaintext
// and extracts symOpts from HTTP request parameters.
func parseEncryptionRequest(r *http.Request) (b64Plaintext, symOpts, error) {
	defer r.Body.Close()

	var plaintext b64Plaintext
	var opts symOpts
	if err := json.NewDecoder(r.Body).Decode(&plaintext); err != nil {
		return b64Plaintext{}, symOpts{}, err
	}
	if _, exists := r.Header["associated_data"]; exists {
		associatedData, err := decodeBase64(r.Header.Get("associated_data"))
		if err != nil {
			return b64Plaintext{}, symOpts{}, fmt.Errorf("associated_data: %w", err)
		}
		opts.associatedData = associatedData
	}
	if _, exists := r.Header["nonce"]; exists {
		nonce, err := decodeBase64(r.Header.Get("nonce"))
		if err != nil {
			return b64Plaintext{}, symOpts{}, fmt.Errorf("nonce: %w", err)
		}
		if len(nonce) != aesGCMNonceSize {
			return b64Plaintext{}, symOpts{}, errors.New("nonce must be 12 byte")
		}
		opts.nonce = nonce
	}
	return plaintext, opts, nil
}

// parseDecryptionRequest parses the HTTP request body into prefixb64Ciphertext
// and extracts symOpts from HTTP request parameters.
func parseDecryptionRequest(r *http.Request) (prefixb64Ciphertext, symOpts, error) {
	defer r.Body.Close()

	var prefixCiphertext prefixb64Ciphertext
	var opts symOpts
	if err := json.NewDecoder(r.Body).Decode(&prefixCiphertext); err != nil {
		return prefixb64Ciphertext{}, symOpts{}, err
	}
	if _, exists := r.Header["associated_data"]; exists {
		associatedData, err := decodeBase64(r.Header.Get("associated_data"))
		if err != nil {
			return prefixb64Ciphertext{}, symOpts{}, fmt.Errorf("associated_data: %w", err)
		}
		opts.associatedData = associatedData
	}
	return prefixCiphertext, opts, nil
}

// writeJSONResponse wraps any payload inside a "data" object and sends it as an HTTP response.
func writeJSONResponse(w http.ResponseWriter, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"data": payload,
	}
	return json.NewEncoder(w).Encode(response)
}

// UnmarshalJSON extracts "plaintext" and decodes it from Base64.
func (p *b64Plaintext) UnmarshalJSON(data []byte) error {
	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	plaintextBase64, exists := obj["plaintext"]
	if !exists {
		return fmt.Errorf("missing 'plaintext' key in JSON")
	}
	decoded, err := decodeBase64(plaintextBase64)
	if err != nil {
		return fmt.Errorf("decoding b64plaintext: %w", err)
	}
	*p = decoded
	return nil
}

// MarshalJSON wraps b64Plaintext in an object with the key "plaintext".
func (p b64Plaintext) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"plaintext": b64.StdEncoding.EncodeToString(p),
	})
}

// UnmarshalJSON extracts the "ciphertext" field, removes the dynamic "vault:vX:" prefix, and decodes Base64.
func (p *prefixb64Ciphertext) UnmarshalJSON(data []byte) error {
	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	encoded, exists := obj["ciphertext"]
	if !exists {
		return fmt.Errorf("missing 'ciphertext' key in JSON")
	}
	// Split "vault:vX:base64" format
	parts := strings.SplitN(encoded, ":", 3)
	if len(parts) < 3 {
		return fmt.Errorf("invalid format: missing version prefix")
	}
	p.prefix = parts[1]
	decoded, err := decodeBase64(parts[2])
	if err != nil {
		return fmt.Errorf("decoding prefixb64Ciphertext: %w", err)
	}
	p.ciphertext = decoded
	return nil
}

// MarshalJSON wraps prefixb64Ciphertext inside a JSON object with "ciphertext" as the key.
func (p prefixb64Ciphertext) MarshalJSON() ([]byte, error) {
	encoded := b64.StdEncoding.EncodeToString(p.ciphertext)
	version := p.prefix
	if version == "" {
		version = "v1"
	}
	// Convert to "vault:vX:base64" format
	versioned := fmt.Sprintf("vault:%s:%s", version, encoded)

	return json.Marshal(map[string]string{
		"ciphertext": versioned,
	})
}

// decodeBase64 is a helper function, ensuring that the b64 encoded string is not empty and returning the base64 decoding.
func decodeBase64(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, errors.New("empty base64 string")
	}

	decoded, err := b64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
