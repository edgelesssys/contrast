package ca

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/katexochen/coordinator-kbs/internal/crypto"
)

type CA struct {
	rootPrivKey   *rsa.PrivateKey
	rootCert      *x509.Certificate
	rootPEM       []byte
	intermPrivKey *rsa.PrivateKey
	intermCert    *x509.Certificate
	intermPEM     []byte
}

func New() (*CA, error) {
	root := &x509.Certificate{
		SerialNumber:          big.NewInt(2019),
		Subject:               pkix.Name{CommonName: "system:coordinator-kbs:root"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	rootPrivKey, err := rsa.GenerateKey(rand.Reader, 4098)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}
	rootBytes, err := x509.CreateCertificate(rand.Reader, root, root, &rootPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create root certificate: %w", err)
	}
	rootPEM := new(bytes.Buffer)
	pem.Encode(rootPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rootBytes,
	})

	interm := &x509.Certificate{
		SerialNumber:          big.NewInt(2020),
		Subject:               pkix.Name{CommonName: "system:coordinator-kbs:intermediate"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	intermPrivKey, err := rsa.GenerateKey(rand.Reader, 4098)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}
	intermBytes, err := x509.CreateCertificate(rand.Reader, interm, root, &intermPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create intermediate certificate: %w", err)
	}
	intermPEM := new(bytes.Buffer)
	pem.Encode(intermPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: intermBytes,
	})

	return &CA{
		rootPrivKey:   rootPrivKey,
		rootCert:      root,
		rootPEM:       rootPEM.Bytes(),
		intermPrivKey: intermPrivKey,
		intermCert:    interm,
		intermPEM:     intermPEM.Bytes(),
	}, nil
}

func (c *CA) NewAttestedMeshCert(commonName string, extensions []pkix.Extension, subjectPublicKey any) ([]byte, error) {
	serialNumber, err := crypto.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	certTemplate := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: commonName},
		Issuer:                pkix.Name{CommonName: "system:coordinator-kbs:intermediate"},
		NotBefore:             now.Add(-2 * time.Hour),
		NotAfter:              now.Add(354 * 24 * time.Hour),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		ExtraExtensions:       extensions,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, c.intermCert, subjectPublicKey, c.intermPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return certPEM.Bytes(), nil
}

func (c *CA) GetCertChain() [][]byte {
	return [][]byte{c.intermPEM, c.rootPEM}
}
