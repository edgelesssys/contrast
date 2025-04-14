// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package transitengine provides all functionality related to the transit engine API endpoints: decrypt and encrypt.
// It is organized in a layered approach, keeping http request processing separated from the underlying crypto
// business logic(crypto.go).
package transitengine

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/oid"
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

// httpError is a json struct holding http error related fields, used for sending json error response bodies and logging.
type httpError struct {
	code          int
	Errors        []string `json:"errors,omitempty"`
	reqMethod     string
	reqURI        string
	reqRemoteAddr string
}

type stateAuthority interface {
	GetState() (*authority.State, error)
}

// NewTransitEngineAPI sets up the transit engine API with a provided seedEngineAuthority.
func NewTransitEngineAPI(authority stateAuthority, logger *slog.Logger) (*http.Server, error) {
	privKeyAPI, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed creating transit engine API private key")
	}
	return &http.Server{
		TLSConfig: &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			GetConfigForClient: func(_ *tls.ClientHelloInfo) (*tls.Config, error) {
				logger.Debug("call getConfigForClient")
				state, err := authority.GetState()
				if err != nil {
					return nil, fmt.Errorf("getting state: %w", err)
				}
				return &tls.Config{
					ClientCAs:  state.CA().GetMeshCACertPool(),
					ClientAuth: tls.RequireAndVerifyClientCert,
					MinVersion: tls.VersionTLS12,
					GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
						return getCertificate(privKeyAPI, authority)
					},
				}, nil
			},
		},
		Handler: newTransitEngineMux(authority, logger),
	}, nil
}

func newTransitEngineMux(authority stateAuthority, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	// 'name' wildcard is kept to reflect existing transit engine API specifications:
	// https://openbao.org/api-docs/secret/transit/#encrypt-data
	// name <=> workloadSecretID, which should be used for the key derivation.
	mux.Handle("/v1/transit/encrypt/{name}", authorizationMiddleware(getEncryptHandler(authority, logger)))
	mux.Handle("/v1/transit/decrypt/{name}", authorizationMiddleware(getDecryptHandler(authority, logger)))

	return mux
}

func getEncryptHandler(authority stateAuthority, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workloadSecretID := r.PathValue("name")
		if workloadSecretID == "" {
			writeHTTPError(w, httpError{
				code:          http.StatusBadRequest,
				Errors:        []string{"Invalid URL format"},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
		var encReq encryptionRequest
		if err := parseRequest(r, &encReq); err != nil {
			writeHTTPError(w, httpError{
				code:          http.StatusBadRequest,
				Errors:        []string{fmt.Sprintf("parsing encryption request: %v", err)},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
		key, err := deriveEncryptionKey(authority, fmt.Sprintf("%d_%s", encReq.KeyVersion, workloadSecretID))
		if err != nil {
			writeHTTPError(w, httpError{
				code:          http.StatusInternalServerError,
				Errors:        []string{fmt.Sprintf("key derivation: %v", err)},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
		ciphertextContainer, err := symmetricEncryptRaw(key, encReq.Plaintext, encReq.AssociatedData)
		if err != nil {
			writeHTTPError(w, httpError{
				code:          http.StatusInternalServerError,
				Errors:        []string{fmt.Sprintf("encrypting: %v", err)},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
		ciphertextContainer.keyVersion = encReq.KeyVersion
		var encResp encryptionResponse
		encResp.Ciphertext = ciphertextContainer
		if err = writeJSONResponse(w, encResp); err != nil {
			writeHTTPError(w, httpError{
				code:          http.StatusInternalServerError,
				Errors:        []string{fmt.Sprintf("writing response: %v", err)},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
	}
}

func getDecryptHandler(authority stateAuthority, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workloadSecretID := r.PathValue("name")
		if workloadSecretID == "" {
			writeHTTPError(w, httpError{
				code:          http.StatusBadRequest,
				Errors:        []string{"Invalid URL format"},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
		var decReq decryptionRequest
		if err := parseRequest(r, &decReq); err != nil {
			writeHTTPError(w, httpError{
				code:          http.StatusBadRequest,
				Errors:        []string{fmt.Sprintf("parsing decryption request: %v", err)},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
		key, err := deriveEncryptionKey(authority, fmt.Sprintf("%d_%s", decReq.CiphertextContainer.keyVersion, workloadSecretID))
		if err != nil {
			writeHTTPError(w, httpError{
				code:          http.StatusInternalServerError,
				Errors:        []string{fmt.Sprintf("key derivation: %v", err)},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
		plaintext, err := symmetricDecryptRaw(key, decReq.CiphertextContainer, decReq.AssociatedData)
		if err != nil {
			writeHTTPError(w, httpError{
				code:          http.StatusInternalServerError,
				Errors:        []string{fmt.Sprintf("decrypting: %v", err)},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
		var decResp decryptionResponse
		decResp.Plaintext = plaintext
		if err = writeJSONResponse(w, decResp); err != nil {
			writeHTTPError(w, httpError{
				code:          http.StatusInternalServerError,
				Errors:        []string{fmt.Sprintf("writing response: %v", err)},
				reqMethod:     r.Method,
				reqURI:        r.RequestURI,
				reqRemoteAddr: r.RemoteAddr,
			}, logger)
			return
		}
	}
}

func authorizeWorkloadSecret(workloadSecretID string, r *http.Request) error {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return fmt.Errorf("no client certs provided")
	}
	extensionWSID, err := extractWorkloadSecretExtension(r.TLS.PeerCertificates[0])
	if err != nil {
		return fmt.Errorf("missing required workloadSecretID cert extension:%w", err)
	}
	if workloadSecretID == extensionWSID {
		return nil
	}
	return fmt.Errorf("mismatching workloadSecretIDs: name:%s, extension:%s", workloadSecretID, extensionWSID)
}

// deriveEncryptionKey derives the workload secret used as the encryption key by receiving the seedengine of the current state.
func deriveEncryptionKey(authority stateAuthority, workloadSecretID string) ([]byte, error) {
	state, err := authority.GetState()
	if err != nil {
		return nil, err
	}
	derivedWorkloadSecret, err := state.SeedEngine().DeriveWorkloadSecret(workloadSecretID)
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

// writeHTTPError sends the httpError handed in as json body error response and logs prior.
func writeHTTPError(w http.ResponseWriter, httpErr httpError, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpErr.code)
	logger.Error("HTTP error", "error", httpErr)

	_ = json.NewEncoder(w).Encode(httpErr) //nolint:errchkjson
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

func extractWorkloadSecretExtension(cert *x509.Certificate) (string, error) {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(oid.WorkloadSecretOID) {
			var value []byte
			_, err := asn1.Unmarshal(ext.Value, &value)
			if err != nil {
				return "", fmt.Errorf("failed to parse extension: %w", err)
			}
			return string(value), nil
		}
	}
	return "", fmt.Errorf("extension not found")
}

func authorizationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workloadSecretID := r.PathValue("name")
		if err := authorizeWorkloadSecret(workloadSecretID, r); err != nil {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getCertificate(privKeyAPI *ecdsa.PrivateKey, authority stateAuthority) (*tls.Certificate, error) {
	state, err := authority.GetState()
	if err != nil {
		return nil, fmt.Errorf("getting state: %w", err)
	}
	dnsNames := []string{}
	for _, policyEntry := range state.Manifest().Policies {
		if policyEntry.Role == manifest.RoleCoordinator {
			dnsNames = append(dnsNames, policyEntry.SANs...)
		}
	}
	meshCertPEM, err := state.CA().NewAttestedMeshCert(dnsNames, nil, &privKeyAPI.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create mesh cert: %w", err)
	}
	meshCertDER, _ := pem.Decode(meshCertPEM)
	if meshCertDER == nil {
		return nil, fmt.Errorf("failed to decode mesh cert: %w", err)
	}
	intermCertDER, _ := pem.Decode(state.CA().GetIntermCACert())
	if intermCertDER == nil {
		return nil, fmt.Errorf("failed to decode intermediate cert: %w", err)
	}
	certChain := tls.Certificate{
		Certificate: [][]byte{meshCertDER.Bytes, intermCertDER.Bytes},
		PrivateKey:  privKeyAPI,
	}
	return &certChain, nil
}
