package ca

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
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
	now := time.Now()
	notBefore := now.Add(-time.Hour)
	notAfter := now.AddDate(10, 0, 0)

	root := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "system:coordinator:root"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	rootPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating root private key: %w", err)
	}
	rootPEM, err := createCert(root, root, &rootPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, fmt.Errorf("creating root certificate: %w", err)
	}

	intermed := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "system:coordinator:meshCA"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	intermPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating intermediate private key: %w", err)
	}
	intermPEM, err := createCert(intermed, root, &intermPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, fmt.Errorf("creating intermediate certificate: %w", err)
	}

	meshCA := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "system:coordinator:meshCA"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	meshCAPEM, err := createCert(meshCA, meshCA, &intermPrivKey.PublicKey, intermPrivKey)
	if err != nil {
		return nil, fmt.Errorf("creating mesh certificate: %w", err)
	}

	return &CA{
		rootPrivKey:   rootPrivKey,
		rootCert:      root,
		rootPEM:       rootPEM,
		intermPrivKey: intermPrivKey,
		intermCert:    intermed,
		intermPEM:     intermPEM,
		meshCACert:    meshCA,
		meshCAPEM:     meshCAPEM,
		namespace:     namespace,
	}, nil
}

// NewAttestedMeshCert creates a new attested mesh certificate.
func (c *CA) NewAttestedMeshCert(dnsNames []string, extensions []pkix.Extension, subjectPublicKey any) ([]byte, error) {
	now := time.Now()
	certTemplate := &x509.Certificate{
		Subject:               pkix.Name{CommonName: dnsNames[0]},
		Issuer:                pkix.Name{CommonName: "system:coordinator:meshCA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(1, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		ExtraExtensions:       extensions,
		DNSNames:              dnsNames,
	}

	certPEM, err := createCert(certTemplate, c.meshCACert, subjectPublicKey, c.intermPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return certPEM, nil
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

func createCert(template, parent *x509.Certificate, pub, priv any) ([]byte, error) {
	if parent == nil {
		return nil, errors.New("parent cannot be nil")
	}
	if template == nil {
		return nil, errors.New("cert cannot be nil")
	}
	if template.SerialNumber != nil {
		return nil, errors.New("cert serial number must be nil")
	}

	serialNum, err := crypto.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, fmt.Errorf("generating serial number: %w", err)
	}
	template.SerialNumber = serialNum

	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %w", err)
	}

	certPEM := new(bytes.Buffer)
	if err := pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return nil, fmt.Errorf("encoding certificate: %w", err)
	}

	return certPEM.Bytes(), nil
}
