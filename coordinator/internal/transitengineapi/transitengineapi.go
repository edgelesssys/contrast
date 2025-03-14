// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// transitengine provides all functionality related to the transit engine API endpoints: decrypt and encrypt.
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

type stateAuthority interface {
	GetState() (*authority.State, error)
}

// NewTransitEngineAPI sets up the transit engine API with a provided seedEngineAuthority.
func NewTransitEngineAPI(authority stateAuthority, port int, logger *slog.Logger) (*http.Server, error) {
	privKeyAPI, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed creating transit engine API private key")
	}
	return &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		TLSConfig: &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			GetConfigForClient: func(_ *tls.ClientHelloInfo) (*tls.Config, error) {
				logger.Debug("call getConfigForClient")
				state, err := authority.GetState()
				if err != nil {
					logger.Debug("failed getting state")
					return nil, fmt.Errorf("getting state: %w", err)
				}
				if len(state.CA.GetMeshCACert()) == 0 {
					return nil, fmt.Errorf("mesh ca cert not initialized")
				}
				meshCAPool := x509.NewCertPool()
				if !meshCAPool.AppendCertsFromPEM(state.CA.GetMeshCACert()) {
					return nil, fmt.Errorf("failed to parse mesh CA cert")
				}
				logger.Debug("loaded mesh CA cert into pool")
				return &tls.Config{
					ClientCAs:  meshCAPool,
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

// newTransitEngineMux creates the http multiplexer for the required transit engine API path,
// adding the corresponding middlewares for logging and authorization.
func newTransitEngineMux(authority stateAuthority, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	// 'name' wildcard is kept to reflect existing transit engine API specifications:
	// https://openbao.org/api-docs/secret/transit/#encrypt-data
	// name <=> workloadSecretID, which should be used for the key derivation.
	mux.Handle("/v1/transit/encrypt/{name}", loggingMiddleware(authorizationMiddleware(getEncryptHandler(authority)), logger))
	mux.Handle("/v1/transit/decrypt/{name}", loggingMiddleware(authorizationMiddleware(getDecryptHandler(authority)), logger))

	return mux
}

func getEncryptHandler(authority stateAuthority) http.HandlerFunc {
	// TODO(jmxnzo): Implement Vault json error bodies
	return func(w http.ResponseWriter, r *http.Request) {
		workloadSecretID := r.PathValue("name")
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

// auhorizeWorkloadSecret authorizes the client request by extracting the workloadSecretID
// sent as the mesh cert extension and ensures equality to the workloadSecretID handed in.
func authorizeWorkloadSecret(workloadSecretID string, r *http.Request) error {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return fmt.Errorf("No client certs provided")
	}
	extensionWSID, err := extractCertExtension(r.TLS.PeerCertificates[0], oid.WorkloadSecretOID)
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

// extractCertExtension is a helper function which checks if the extension OID exists in the certificate and returns the string representation
// of the contained bytes.
func extractCertExtension(cert *x509.Certificate, oid asn1.ObjectIdentifier) (string, error) {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(oid) {
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

// responseLogger implements a http.ResponseWriter, which further holds logging related data.
type responseLogger struct {
	http.ResponseWriter
	statusCode   int
	bodyCaptured bool
	body         []byte
}

// WriteHeader overwrite function stores the statusCode on call.
func (rl *responseLogger) WriteHeader(code int) {
	rl.statusCode = code
	rl.ResponseWriter.WriteHeader(code)
}

// Write overwrite function stores the response message in case of http.Error occurrence.
func (rl *responseLogger) Write(b []byte) (int, error) {
	// Capture the response body only if status code is an error (≥400)
	if rl.statusCode >= 400 && !rl.bodyCaptured {
		rl.body = append([]byte{}, b...)
		rl.bodyCaptured = true
	}
	return rl.ResponseWriter.Write(b)
}

// loggingMiddleware initializes a responseLogger to capture important response data and thus
// allow successful request-response logging.
func loggingMiddleware(next http.HandlerFunc, logger *slog.Logger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rl := &responseLogger{ResponseWriter: w, statusCode: 0}

		next.ServeHTTP(rl, r)

		logMsg := fmt.Sprintf("[%s] %s from %s -> %d",
			r.Method, r.RequestURI, r.RemoteAddr, rl.statusCode)

		if rl.statusCode >= 400 && rl.bodyCaptured {
			logger.Error(logMsg, "error", string(rl.body))
		} else {
			logger.Debug(logMsg)
		}
	})
}

// authorizationMiddleware reads out the workloadSecretID stored in name URL parameter and ensures
// the client request to be authorized.
func authorizationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workloadSecretID := r.PathValue("name")
		if err := authorizeWorkloadSecret(workloadSecretID, r); err != nil {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getCertificate calls the CA of the current state to issue a new mesh cert for the private key handed in.
// It returns a tls.Certificate, which holds the certChain appending the new mesh cert and the intermediate
// CA cert.
func getCertificate(privKeyAPI *ecdsa.PrivateKey, authority stateAuthority) (*tls.Certificate, error) {
	state, err := authority.GetState()
	if err != nil {
		return nil, fmt.Errorf("getting state: %w", err)
	}
	dnsNames := []string{"coordinator"}

	meshCertPEM, err := state.CA.NewAttestedMeshCert(dnsNames, nil, &privKeyAPI.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create mesh cert: %w", err)
	}
	meshCertDER, _ := pem.Decode(meshCertPEM)
	if meshCertDER == nil {
		return nil, fmt.Errorf("failed to decode mesh cert: %w", err)
	}
	intermCertDER, _ := pem.Decode(state.CA.GetIntermCACert())
	if intermCertDER == nil {
		return nil, fmt.Errorf("failed to decode intermediate cert: %w", err)
	}
	certChain := tls.Certificate{
		Certificate: [][]byte{meshCertDER.Bytes, intermCertDER.Bytes},
		PrivateKey:  privKeyAPI,
	}
	return &certChain, nil
}
