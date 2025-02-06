// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package transitengine

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/edgelesssys/contrast/coordinator/internal/seedengine"
)

// seedEngineAuthority defines an interface holding a GetSeedEngine method for deriving the encryption key based with the provided seedengine.
type seedEngineAuthority interface {
	GetSeedEngine() (*seedengine.SeedEngine, error)
}

type (
	// b64Plaintext describes a base64-encoded plaintext.
	b64Plaintext []byte
	// prefixb64Ciphertext describes a base64-encoded ciphertext with prefix 'vault:v1:'.
	prefixb64Ciphertext struct {
		ciphertext []byte
		prefix     string
	}
)

// NewTransitEngineAPI sets up the transit engine API with a provided seedEngineAuthority.
func NewTransitEngineAPI(authority seedEngineAuthority, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	// avoid explicit routing
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 4 || pathParts[1] != "transit" {
			http.NotFound(w, r)
			return
		}
		action := pathParts[2]

		switch action {
		case "encrypt":
			getEncryptHandler(authority, logger).ServeHTTP(w, r)
		case "decrypt":
			getDecryptHandler(authority, logger).ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	return mux
}

func getEncryptHandler(authority seedEngineAuthority, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParts := strings.Split(r.URL.Path, "/")
		version, workloadSecret := urlParts[0], urlParts[4]
		key, err := deriveEncryptionKey(authority, workloadSecret+version)
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

func getDecryptHandler(authority seedEngineAuthority, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParts := strings.Split(r.URL.Path, "/")
		version, workloadSecret := urlParts[0], urlParts[4]
		key, err := deriveEncryptionKey(authority, workloadSecret+version)
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
func deriveEncryptionKey(authority seedEngineAuthority, workloadSecretID string) ([]byte, error) {
	seedEngine, err := authority.GetSeedEngine()
	if err != nil {
		return nil, err
	}
	// TODO(jmxnzo): authentication of client certs <-> parsed workloadSecretID.
	derivedWorkloadSecret, err := seedEngine.DeriveWorkloadSecret(workloadSecretID)
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
	// TODO(jmxnzo): Read symOpts from HTTP request params
	opts.convergent = false

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

	opts.convergent = false

	// If not convergent, extract nonce and update ciphertext
	if !opts.convergent {
		opts.nonce = prefixCiphertext.ciphertext[:aesGCMNonceSize]
		prefixCiphertext.ciphertext = prefixCiphertext.ciphertext[aesGCMNonceSize:]
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
	decoded, err := b64.StdEncoding.DecodeString(plaintextBase64)
	if err != nil {
		return err
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
	decoded, err := b64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return err
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
