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
	"regexp"
	"strconv"
	"strings"

	"github.com/edgelesssys/contrast/coordinator/internal/authority"
)

const (
	// aesGCMNonceSize specifies the default nonce size in bytes used in AES GCM, used during parsing of ciphertextContainer.
	aesGCMNonceSize = 12
	// aesGCMKeySize specifies the default key size in bytes AES GCM.
	aesGCMKeySize = 16
)

type (
	// b64Plaintext describes a base64-encoded plaintext.
	b64Plaintext []byte
	// ciphertextContainer describes a base64-encoded ciphertext prepended with the nonce and specified key version.
	ciphertextContainer struct {
		nonce      []byte
		ciphertext []byte
		version    int
	}
	// encryptionRequest holds the request-specific b64Plaintext and currently supported, optional query parameters: associatedData.
	encryptionRequest struct {
		plaintext      b64Plaintext
		associatedData []byte
	}
	// decryptionRequest holds the request-specific ciphertextContainer and currently supported, optional query parameters: associatedData.
	decryptionRequest struct {
		ciphertextContainer ciphertextContainer
		associatedData      []byte
	}
)

type stateAuthority interface {
	GetState() (*authority.State, error)
}

// NewTransitEngineAPI sets up the transit engine API with a provided seedEngineAuthority.
func NewTransitEngineAPI(authority stateAuthority, _ *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/{version}/transit/encrypt/{name}", getEncryptHandler(authority))
	mux.Handle("/{version}/transit/decrypt/{name}", getDecryptHandler(authority))

	return mux
}

func getEncryptHandler(authority stateAuthority) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strVersion := r.PathValue("version")
		workloadSecretID := r.PathValue("name")
		if strVersion == "" || workloadSecretID == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		version, err := extractVersion(strVersion)
		if err != nil {
			http.Error(w, fmt.Sprint("URL version: %w", err), http.StatusBadRequest)
			return
		}
		key, err := deriveEncryptionKey(authority, workloadSecretID+strVersion)
		if err != nil {
			http.Error(w, fmt.Sprint("key derivation: %w", err), http.StatusInternalServerError)
			return
		}
		encReq, err := parseEncryptionRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprint("parsing encryption request: %w", err), http.StatusBadRequest)
			return
		}
		ciphertextContainer, err := symmetricEncryptRaw(key, encReq.plaintext, encReq.associatedData)
		if err != nil {
			http.Error(w, fmt.Sprint("encrypting: %w", err), http.StatusInternalServerError)
			return
		}
		ciphertextContainer.version = version
		if err = writeJSONResponse(w, ciphertextContainer); err != nil {
			http.Error(w, fmt.Sprint("writing response: %w", err), http.StatusInternalServerError)
			return
		}
	}
}

func getDecryptHandler(authority stateAuthority) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strVersion := r.PathValue("version")
		workloadSecretID := r.PathValue("name")
		if strVersion == "" || workloadSecretID == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		version, err := extractVersion(strVersion)
		if err != nil {
			http.Error(w, fmt.Sprint("URL version: %w", err), http.StatusBadRequest)
			return
		}
		key, err := deriveEncryptionKey(authority, workloadSecretID+strVersion)
		if err != nil {
			http.Error(w, fmt.Sprint("key derivation: %w", err), http.StatusInternalServerError)
			return
		}
		decReq, err := parseDecryptionRequest(r, version)
		if err != nil {
			http.Error(w, fmt.Sprint("parsing decryption request: %w", err), http.StatusBadRequest)
			return
		}
		b64Plaintext, err := symmetricDecryptRaw(key, decReq.ciphertextContainer, decReq.associatedData)
		if err != nil {
			http.Error(w, fmt.Sprint("decrypting: %w", err), http.StatusInternalServerError)
			return
		}
		if err = writeJSONResponse(w, b64Plaintext); err != nil {
			http.Error(w, fmt.Sprint("writing response: %w", err), http.StatusInternalServerError)
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
	return derivedWorkloadSecret[:aesGCMKeySize], nil
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

// parseEncryptionRequest returns the given HTTP request body as encryptionRequest holding
// the b64Plaintext and the optional query parameter associatedData.
func parseEncryptionRequest(r *http.Request) (encryptionRequest, error) {
	defer r.Body.Close()
	if err := validateContentType(r); err != nil {
		return encryptionRequest{}, err
	}
	var encReq encryptionRequest
	if err := json.NewDecoder(r.Body).Decode(&encReq); err != nil {
		return encryptionRequest{}, err
	}
	return encReq, nil
}

// parseDecryptionRequest returns the given HTTP request body as decryptionRequest holding
// the ciphertextContainer and the optional query parameter associatedData.
// Checks that the URL version matches the received ciphertext request version.
func parseDecryptionRequest(r *http.Request, version int) (decryptionRequest, error) {
	defer r.Body.Close()
	if err := validateContentType(r); err != nil {
		return decryptionRequest{}, err
	}
	var decReq decryptionRequest
	if err := json.NewDecoder(r.Body).Decode(&decReq); err != nil {
		return decryptionRequest{}, err
	}
	if version != decReq.ciphertextContainer.version {
		return decryptionRequest{}, errors.New("Mismatch URL and ciphertext key version")
	}
	return decReq, nil
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

// newCiphertextContainer returns a new ciphertextContainer, holding the version prefix and decoded base64 nonce and ciphertext.
func newCiphertextContainer(encoded string) (ciphertextContainer, error) {
	// Split "vault:vX:base64" format
	parts := strings.SplitN(encoded, ":", 3)
	if len(parts) < 3 {
		return ciphertextContainer{}, fmt.Errorf("invalid ciphertext format")
	}
	version, err := extractVersion(parts[1])
	if err != nil {
		return ciphertextContainer{}, fmt.Errorf("ciphertext version: %w", err)
	}
	fullCiphertext, err := decodeBase64(parts[2])
	if err != nil {
		return ciphertextContainer{}, fmt.Errorf("decoding ciphertext: %w", err)
	}

	return ciphertextContainer{
		version:    version,
		nonce:      fullCiphertext[:aesGCMNonceSize],
		ciphertext: fullCiphertext[aesGCMNonceSize:],
	}, nil
}

// MarshalJSON wraps b64Plaintext in an object with the key "plaintext".
func (p b64Plaintext) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"plaintext": b64.StdEncoding.EncodeToString(p),
	})
}

// MarshalJSON wraps ciphertextContainer inside a JSON object with "ciphertext" as the key.
func (c ciphertextContainer) MarshalJSON() ([]byte, error) {
	encodedNonce := b64.StdEncoding.EncodeToString(c.nonce)
	encodedCiphertext := b64.StdEncoding.EncodeToString(c.ciphertext)
	// Convert to "vault:vX:base64" format
	versioned := fmt.Sprintf("vault:v%d:%s%s", c.version, encodedNonce, encodedCiphertext)

	return json.Marshal(map[string]string{
		"ciphertext": versioned,
	})
}

// UnmarshalJSON creates a encryptionRequest, holding the request-specific b64Plaintext and associatedData if present.
func (e *encryptionRequest) UnmarshalJSON(data []byte) error {
	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	encPlaintext, exists := obj["plaintext"]
	if !exists {
		return fmt.Errorf("missing 'plaintext' key in JSON")
	}
	plaintext, err := decodeBase64(encPlaintext)
	if err != nil {
		return fmt.Errorf("decoding plaintext: %w", err)
	}
	e.plaintext = plaintext
	if encAssociatedData, exists := obj["associated_data"]; exists {
		e.associatedData, err = decodeBase64(encAssociatedData)
		if err != nil {
			return fmt.Errorf("decoding associated_data: %w", err)
		}
	}
	return nil
}

// UnmarshalJSON creates a decryptionRequest, holding the request-specific ciphertextContainer and associatedData if present.
func (d *decryptionRequest) UnmarshalJSON(data []byte) error {
	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	encCiphertext, exists := obj["ciphertext"]
	if !exists {
		return fmt.Errorf("missing 'ciphertext' key in JSON")
	}
	var err error
	d.ciphertextContainer, err = newCiphertextContainer(encCiphertext)
	if err != nil {
		return err
	}
	if encAssociatedData, exists := obj["associated_data"]; exists {
		d.associatedData, err = decodeBase64(encAssociatedData)
		if err != nil {
			return fmt.Errorf("decoding associated_data: %w", err)
		}
	}
	return nil
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

// extractVersion is a helper function checking the version string format 'vX' and extracting the corresponding version as int.
func extractVersion(versionStr string) (int, error) {
	re := regexp.MustCompile(`^v(\d+)$`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid format: %s", versionStr)
	}
	return strconv.Atoi(matches[1])
}
