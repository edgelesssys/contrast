// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package atls provides config generation functions to bootstrap attested TLS connections.
package atls

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/internal/atls/reportdata"
	"github.com/edgelesssys/contrast/internal/attestation"
	contrastcrypto "github.com/edgelesssys/contrast/internal/crypto"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// NoValidators skips validation of the server's attestation document.
	NoValidators = []Validator{}
	// NoIssuer skips embedding the client's attestation document.
	NoIssuer Issuer
	// NoMetrics skips collecting metrics for attestation failures.
	NoMetrics prometheus.Counter

	// ErrNoValidAttestationExtensions is returned when no valid attestation document certificate extensions are found.
	ErrNoValidAttestationExtensions = errors.New("no valid attestation document certificate extensions found")
	// ErrNoMatchingValidators is returned when no validator matches the attestation document.
	ErrNoMatchingValidators = errors.New("no matching validators found")
)

// CreateAttestationServerTLSConfig creates a tls.Config object with a self-signed certificate and an embedded attestation document.
// Pass a list of validators to enable mutual aTLS.
// If issuer is nil, no attestation will be embedded.
func CreateAttestationServerTLSConfig(issuer Issuer, validators []Validator, attestationFailures prometheus.Counter) (*tls.Config, error) {
	getConfigForClient, err := getATLSConfigForClientFunc(issuer, validators, attestationFailures)
	if err != nil {
		return nil, fmt.Errorf("get aTLS config for client: %w", err)
	}

	return &tls.Config{
		GetConfigForClient: getConfigForClient,
	}, nil
}

// CreateAttestationClientTLSConfig creates a tls.Config object that verifies a certificate with an embedded attestation document.
//
// ATTENTION: The returned config is configured with a nonce and uses the input context. It must
// only be used for a single connection.
//
// If no validators are set, the server's attestation document will not be verified.
// If issuer is nil, the client will be unable to perform mutual aTLS.
func CreateAttestationClientTLSConfig(ctx context.Context, issuer Issuer, validators []Validator, privKey crypto.PrivateKey) (*tls.Config, error) {
	clientNonce, err := contrastcrypto.GenerateRandomBytes(contrastcrypto.RNGLengthDefault)
	if err != nil {
		return nil, err
	}
	clientConn := &clientConnection{
		issuer:      issuer,
		validators:  validators,
		clientNonce: clientNonce,
		privKey:     privKey,
	}

	return &tls.Config{
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return clientConn.verify(ctx, rawCerts, verifiedChains)
		},
		GetClientCertificate: clientConn.getCertificate, // use custom certificate for mutual aTLS connections
		InsecureSkipVerify:   true,                      // disable default verification because we use our own verify func
		MinVersion:           tls.VersionTLS12,
		NextProtos: []string{
			encodeNonceToNextProtos(clientNonce),
			"h2", // grpc-go requires us to advertise HTTP/2 (h2) over ALPN
		},
	}, nil
}

// Issuer issues an attestation document.
type Issuer interface {
	Getter
	Issue(ctx context.Context, reportData [64]byte) (quote []byte, err error)
}

// Validator is able to validate an attestation document.
type Validator interface {
	Getter
	Validate(ctx context.Context, attDoc []byte, reportData []byte) error
	fmt.Stringer
}

// getATLSConfigForClientFunc returns a config setup function that is called once for every client connecting to the server.
// This allows for different server configuration for every client.
// In aTLS this is used to generate unique nonces for every client.
func getATLSConfigForClientFunc(issuer Issuer, validators []Validator, attestationFailures prometheus.Counter) (func(*tls.ClientHelloInfo) (*tls.Config, error), error) {
	// generate key for the server
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	// this function will be called once for every client
	return func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
		// generate nonce for this connection
		serverNonce, err := contrastcrypto.GenerateRandomBytes(contrastcrypto.RNGLengthDefault)
		if err != nil {
			return nil, fmt.Errorf("generate nonce: %w", err)
		}

		serverConn := &serverConnection{
			privKey:             priv,
			issuer:              issuer,
			validators:          validators,
			attestationFailures: attestationFailures,
			serverNonce:         serverNonce,
		}

		cfg := &tls.Config{
			GetCertificate: serverConn.getCertificate,
			MinVersion:     tls.VersionTLS12,
			ClientAuth:     tls.RequestClientCert, // request client certificate but don't require it
			NextProtos:     []string{"h2"},        // grpc-go requires us to advertise HTTP/2 (h2) over ALPN
		}

		// ugly hack: abuse acceptable client CAs as a channel to transmit the nonce
		if cfg.ClientCAs, err = encodeNonceToCertPool(serverNonce, priv); err != nil {
			return nil, fmt.Errorf("encode nonce: %w", err)
		}

		// enable mutual aTLS if any validators are set
		if len(validators) > 0 {
			cfg.ClientAuth = tls.RequireAnyClientCert // validity of certificate will be checked by our custom verify function
			cfg.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				return serverConn.verify(chi.Context(), rawCerts, verifiedChains)
			}
		}

		return cfg, nil
	}, nil
}

// getCertificate creates a client or server certificate for aTLS connections.
// The certificate uses certificate extensions to embed an attestation document generated using nonce.
func getCertificate(ctx context.Context, issuer Issuer, priv crypto.PrivateKey, pub any, nonce []byte) (*tls.Certificate, error) {
	serialNumber, err := contrastcrypto.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, err
	}

	var extensions []pkix.Extension

	// create and embed attestation if quote Issuer is available
	if issuer != nil {
		pubBytes, err := x509.MarshalPKIXPublicKey(pub)
		if err != nil {
			return nil, err
		}

		// create attestation document using the nonce send by the remote party
		reportData := reportdata.Construct(pubBytes, nonce)
		attDoc, err := issuer.Issue(ctx, reportData)
		if err != nil {
			return nil, err
		}

		extensions = append(extensions, pkix.Extension{Id: issuer.OID(), Value: attDoc})
	}

	// create certificate that includes the attestation document as extension
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:    serialNumber,
		Subject:         pkix.Name{CommonName: "Contrast"},
		NotBefore:       now.Add(-2 * time.Hour),
		NotAfter:        now.Add(2 * time.Hour),
		ExtraExtensions: extensions,
	}
	cert, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{Certificate: [][]byte{cert}, PrivateKey: priv}, nil
}

// processCertificate parses the certificate and verifies it.
// If successful returns the certificate and its hashed public key, an error otherwise.
func processCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) (*x509.Certificate, []byte, error) {
	// parse certificate
	if len(rawCerts) == 0 {
		return nil, nil, errors.New("rawCerts is empty")
	}
	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return nil, nil, fmt.Errorf("parse certificate: %w", err)
	}

	// verify self-signed certificate
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	_, err = cert.Verify(x509.VerifyOptions{Roots: roots})
	if err != nil {
		return nil, nil, fmt.Errorf("verify certificate: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal public key: %w", err)
	}
	return cert, pubBytes, nil
}

// verifyEmbeddedReport verifies an aTLS certificate by validating the attestation document embedded in the TLS certificate.
//
// It will check against all applicable validator for the type of attestation document, and return success on the first match.
func verifyEmbeddedReport(ctx context.Context, validators []Validator, cert *x509.Certificate, peerPublicKey, nonce []byte) (retErr error) {
	// For better error reporting, let's keep track of whether we've found a valid extension at all..
	var foundExtension bool
	// .. and whether we've found a matching validator.
	var foundMatchingValidator bool

	expectedReportData := reportdata.Construct(peerPublicKey, nonce)

	// We'll need to have a look at all extensions in the certificate to find the attestation document.
	for _, ex := range cert.Extensions {
		// Optimization: Skip the extension early before heading into the m*n complexity of the validator check
		// if the extension is not an attestation document.
		if !attestation.IsAttestationDocumentExtension(ex.Id) {
			continue
		}

		// We have a valid attestation document. Let's check it against all applicable validators.
		foundExtension = true
		for _, validator := range validators {
			// Optimization: Skip the validator if it doesn't match the attestation type of the document.
			if !ex.Id.Equal(validator.OID()) {
				continue
			}

			// We've found a matching validator. Let's validate the document.
			foundMatchingValidator = true

			validationErr := validator.Validate(ctx, ex.Value, expectedReportData[:])
			if validationErr == nil {
				// The validator has successfully verified the document. We can exit.
				return nil
			}
			// Otherwise, we'll keep track of the error and continue with the next validator.
			retErr = errors.Join(retErr, fmt.Errorf(" validator %s failed: %w", validator.String(), validationErr))
		}
	}

	if !foundExtension {
		return ErrNoValidAttestationExtensions
	}

	if !foundMatchingValidator {
		return ErrNoMatchingValidators
	}

	// The joined error should reveal the atls nonce once to maintain readability.
	if retErr != nil {
		retErr = fmt.Errorf("with AtlsConnectionNonce %s: %w", hex.EncodeToString(nonce), retErr)
	}

	// If we're here, an error must've happened during validation.
	return retErr
}

// encodeNonceToCertPool returns a cert pool that contains a certificate whose CN is the base64-encoded nonce.
func encodeNonceToCertPool(nonce []byte, privKey *ecdsa.PrivateKey) (*x509.CertPool, error) {
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
		Subject:      pkix.Name{CommonName: base64.StdEncoding.EncodeToString(nonce)},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pool, nil
}

// decodeNonceFromAcceptableCAs interprets the CN of acceptableCAs[0] as base64-encoded nonce and returns the decoded nonce.
// acceptableCAs should have been received by a client where the server used encodeNonceToCertPool to transmit the nonce.
func decodeNonceFromAcceptableCAs(acceptableCAs [][]byte) ([]byte, error) {
	if len(acceptableCAs) != 1 {
		return nil, errors.New("unexpected acceptableCAs length")
	}
	var rdnSeq pkix.RDNSequence
	if _, err := asn1.Unmarshal(acceptableCAs[0], &rdnSeq); err != nil {
		return nil, fmt.Errorf("unmarshal acceptableCAs: %w", err)
	}

	// https://github.com/golang/go/blob/19309779ac5e2f5a2fd3cbb34421dafb2855ac21/src/crypto/x509/pkix/pkix.go#L188
	oidCommonName := asn1.ObjectIdentifier{2, 5, 4, 3}

	for _, rdnSet := range rdnSeq {
		for _, rdn := range rdnSet {
			if rdn.Type.Equal(oidCommonName) {
				nonce, ok := rdn.Value.(string)
				if !ok {
					return nil, errors.New("unexpected RDN type")
				}
				nonceDecoded, err := base64.StdEncoding.DecodeString(nonce)
				if err != nil {
					return nil, fmt.Errorf("decode nonce: %w", err)
				}
				return nonceDecoded, nil
			}
		}
	}

	return nil, errors.New("CN not found")
}

var (
	errNoNonce         = errors.New("no nonce in supported protocols or SNI")
	errVersionMismatch = errors.New("proto refers to an unsupported atls version")
)

const noncePrefix = `atls:%s:nonce:`

var (
	preferredAtlsVersion  = "v1"
	supportedAtlsVersions = []string{preferredAtlsVersion}
)

func encodeNonceToNextProtos(nonce []byte) string {
	return fmt.Sprintf("%s%x", fmt.Appendf(nil, noncePrefix, preferredAtlsVersion), nonce)
}

func decodeNonceFromSupportedProtos(protos []string) ([]byte, error) {
	for _, proto := range protos {
		if !strings.HasPrefix(proto, "atls") {
			continue
		}

		nonceHex, err := extractNonce(proto)
		if err != nil {
			return nil, err
		}

		nonce, err := hex.DecodeString(nonceHex)
		if err != nil {
			return nil, fmt.Errorf("decoding nonce: %w", err)
		}
		return nonce, nil
	}

	return nil, errNoNonce
}

func extractNonce(proto string) (string, error) {
	for _, version := range supportedAtlsVersions {
		if nonceHex, ok := strings.CutPrefix(proto, fmt.Sprintf(noncePrefix, version)); ok {
			return nonceHex, nil
		}
	}
	return "", fmt.Errorf("%w: %q", errVersionMismatch, proto)
}

// clientConnection holds state for client to server connections.
type clientConnection struct {
	issuer      Issuer
	validators  []Validator
	clientNonce []byte
	privKey     crypto.PrivateKey
}

// verify the validity of an aTLS server certificate.
func (c *clientConnection) verify(ctx context.Context, rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	cert, pubBytes, err := processCertificate(rawCerts, verifiedChains)
	if err != nil {
		return fmt.Errorf("process certificate: %w", err)
	}

	// don't perform verification of attestation document if no validators are set
	if len(c.validators) == 0 {
		return nil
	}

	return verifyEmbeddedReport(ctx, c.validators, cert, pubBytes, c.clientNonce)
}

// getCertificate generates a client certificate for mutual aTLS connections.
func (c *clientConnection) getCertificate(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	priv := c.privKey

	if priv == nil {
		// generate and hash key
		var err error
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
	}

	// ugly hack: abuse acceptable client CAs as a channel to receive the nonce
	serverNonce, err := decodeNonceFromAcceptableCAs(cri.AcceptableCAs)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}

	return getCertificate(cri.Context(), c.issuer, priv, publicKey(priv), serverNonce)
}

func publicKey(key crypto.PrivateKey) crypto.PublicKey {
	typedKey, ok := key.(interface {
		Public() crypto.PublicKey
	})
	if !ok {
		// All standard library implementations of private keys implement this interface - see
		// https://pkg.go.dev/crypto#PrivateKey. Since we only ever expect keys from the standard
		// library, trying to work around the incompatibility is not going to get us anywhere and
		// panicking is justified.
		panic(fmt.Sprintf("private key of type %T does not implement Public()", key))
	}
	return typedKey.Public()
}

// serverConnection holds state for server to client connections.
type serverConnection struct {
	issuer              Issuer
	validators          []Validator
	attestationFailures prometheus.Counter
	privKey             crypto.PrivateKey
	serverNonce         []byte
}

// verify the validity of a clients aTLS certificate.
// Only needed for mutual aTLS.
func (c *serverConnection) verify(ctx context.Context, rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	cert, pubBytes, err := processCertificate(rawCerts, verifiedChains)
	if err != nil {
		return fmt.Errorf("process certificate: %w", err)
	}

	err = verifyEmbeddedReport(ctx, c.validators, cert, pubBytes, c.serverNonce)
	if err != nil && c.attestationFailures != nil {
		c.attestationFailures.Inc()
	}
	return err
}

// getCertificate generates a client certificate for aTLS connections.
// Can be used for mutual as well as basic aTLS.
func (c *serverConnection) getCertificate(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	clientNonce, err := getNonce(chi)
	if err != nil {
		return nil, err
	}
	// create aTLS certificate using the nonce as extracted from the client-hello message
	return getCertificate(chi.Context(), c.issuer, c.privKey, publicKey(c.privKey), clientNonce)
}

func getNonce(chi *tls.ClientHelloInfo) ([]byte, error) {
	// Try to get the nonce from ALPN first.
	clientNonce, err := decodeNonceFromSupportedProtos(chi.SupportedProtos)
	if err == nil {
		return clientNonce, nil
	}
	if !errors.Is(err, errNoNonce) {
		return nil, fmt.Errorf("decoding nonce from SupportedProtos: %w", err)
	}

	// Fall back to base64-encoded nonce in SNI.
	if chi.ServerName == "" {
		return nil, errNoNonce
	}
	clientNonce, err = base64.StdEncoding.DecodeString(chi.ServerName)
	if err != nil {
		return nil, fmt.Errorf("decoding nonce from SNI: %w", err)
	}
	return clientNonce, nil
}

// FakeIssuer fakes an issuer and can be used for tests.
type FakeIssuer struct {
	Getter
}

// NewFakeIssuer creates a new FakeIssuer with the given OID.
func NewFakeIssuer(oid Getter) *FakeIssuer {
	return &FakeIssuer{oid}
}

// Issue marshals the user data and returns it.
func (FakeIssuer) Issue(_ context.Context, reportData [64]byte) ([]byte, error) {
	return json.Marshal(FakeAttestationDoc{ReportData: reportData[:]})
}

// FakeValidator fakes a validator and can be used for tests.
type FakeValidator struct {
	Getter
	err error // used for package internal testing only
}

// NewFakeValidator creates a new FakeValidator with the given OID.
func NewFakeValidator(oid Getter) *FakeValidator {
	return &FakeValidator{oid, nil}
}

// NewFakeValidators returns a slice with a single FakeValidator.
func NewFakeValidators(oid Getter) []Validator {
	return []Validator{NewFakeValidator(oid)}
}

// Validate unmarshals the attestation document and verifies the nonce.
func (v FakeValidator) Validate(_ context.Context, attDoc []byte, reportData []byte) error {
	var doc FakeAttestationDoc
	if err := json.Unmarshal(attDoc, &doc); err != nil {
		return err
	}

	if !bytes.Equal(doc.ReportData, reportData) {
		return fmt.Errorf("invalid reportData: expected %x, got %x", doc.ReportData, reportData)
	}

	return v.err
}

// String returns the name as identifier of the validator.
func (v *FakeValidator) String() string {
	return ""
}

// FakeAttestationDoc is a fake attestation document used for testing.
type FakeAttestationDoc struct {
	ReportData []byte
}

// Getter returns an ASN.1 Object Identifier.
type Getter interface {
	OID() asn1.ObjectIdentifier
}
