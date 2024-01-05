package ca

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/edgelesssys/nunki/internal/crypto"
)

// CA is a cross-signing certificate authority.
type CA struct {
	rootPrivKey *ecdsa.PrivateKey
	rootCert    *x509.Certificate
	rootPEM     []byte

	// The intermPrivKey is used for both the intermediate and meshCA certificates.
	intermPrivKey *ecdsa.PrivateKey

	intermCert *x509.Certificate
	intermPEM  []byte

	meshCACert *x509.Certificate
	meshCAPEM  []byte

	namespace string
}

// New creates a new CA.
func New(namespace string) (*CA, error) {
	rootSerialNumber, err := crypto.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, err
	}

	root := &x509.Certificate{
		SerialNumber:          rootSerialNumber,
		Subject:               pkix.Name{CommonName: "system:coordinator:root"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	rootPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}
	rootBytes, err := x509.CreateCertificate(rand.Reader, root, root, &rootPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create root certificate: %w", err)
	}
	rootPEM := new(bytes.Buffer)
	if err := pem.Encode(rootPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rootBytes,
	}); err != nil {
		return nil, fmt.Errorf("failed to encode root certificate: %w", err)
	}

	intermSerialNumber, err := crypto.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, err
	}
	intermed := &x509.Certificate{
		SerialNumber:          intermSerialNumber,
		Subject:               pkix.Name{CommonName: "system:coordinator:meshCA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	intermPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}
	intermBytes, err := x509.CreateCertificate(rand.Reader, intermed, root, &intermPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create intermediate certificate: %w", err)
	}
	intermPEM := new(bytes.Buffer)
	if err := pem.Encode(intermPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: intermBytes,
	}); err != nil {
		return nil, fmt.Errorf("failed to encode intermediate certificate: %w", err)
	}

	intermCASerialNumber, err := crypto.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, err
	}
	meshCA := &x509.Certificate{
		SerialNumber:          intermCASerialNumber,
		Subject:               pkix.Name{CommonName: "system:coordinator:meshCA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	meshCABytes, err := x509.CreateCertificate(rand.Reader, meshCA, meshCA, &intermPrivKey.PublicKey, intermPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create meshCA certificate: %w", err)
	}
	meshCAPEM := new(bytes.Buffer)
	if err := pem.Encode(meshCAPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: meshCABytes,
	}); err != nil {
		return nil, fmt.Errorf("failed to encode meshCA certificate: %w", err)
	}

	return &CA{
		rootPrivKey:   rootPrivKey,
		rootCert:      root,
		rootPEM:       rootPEM.Bytes(),
		intermPrivKey: intermPrivKey,
		intermCert:    intermed,
		intermPEM:     intermPEM.Bytes(),
		meshCACert:    meshCA,
		meshCAPEM:     meshCAPEM.Bytes(),
		namespace:     namespace,
	}, nil
}

// NewAttestedMeshCert creates a new attested mesh certificate.
func (c *CA) NewAttestedMeshCert(dnsNames []string, extensions []pkix.Extension, subjectPublicKey any) ([]byte, error) {
	serialNumber, err := crypto.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	certTemplate := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: dnsNames[0]},
		Issuer:                pkix.Name{CommonName: "system:coordinator:meshCA"},
		NotBefore:             now.Add(-2 * time.Hour),
		NotAfter:              now.Add(354 * 24 * time.Hour),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		ExtraExtensions:       extensions,
		DNSNames:              dnsNames,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, c.meshCACert, subjectPublicKey, c.intermPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := new(bytes.Buffer)
	if err := pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return nil, fmt.Errorf("failed to encode certificate: %w", err)
	}

	return certPEM.Bytes(), nil
}

// GetRootCACert returns the root certificate of the CA in PEM format.
func (c *CA) GetRootCACert() []byte {
	return c.rootPEM
}

// GetIntermCert returns the intermediate certificate of the CA in PEM format.
func (c *CA) GetIntermCert() []byte {
	return c.intermPEM
}

// GetMeshCACert returns the mesh root certificate of the CA in PEM format.
func (c *CA) GetMeshCACert() []byte {
	return c.meshCAPEM
}
