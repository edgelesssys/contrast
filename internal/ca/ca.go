// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package ca

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/edgelesssys/contrast/internal/cryptohelpers"
)

// CA is a cross-signing certificate authority.
//
// It is configured with two private keys (root and intermediate) and generates corresponding
// CA certificates (root, intermediate and mesh) when created with New. The mesh certificate is
// self-signed and used for issuing workload certificates with NewAttestedMeshCert. It is usually
// bound to a single manifest. The intermediate cert uses the same key as the mesh cert, but is
// signed by the root key and thus links the workload cert to the root cert. The idea of
// cross-signing workload certs was adapted from MarbleRun, see:
// https://docs.edgeless.systems/marblerun/architecture/security#public-key-infrastructure-and-certificate-authority
type CA struct {
	rootCAPrivKey *ecdsa.PrivateKey
	rootCAPEM     []byte

	intermPrivKey *ecdsa.PrivateKey

	intermCAPEM []byte

	meshCACert *x509.Certificate
	meshCAPEM  []byte

	meshCACertPool *x509.CertPool
}

// New creates a new CA.
func New(rootPrivKey, intermPrivKey *ecdsa.PrivateKey) (*CA, error) {
	now := time.Now()
	notBefore := now.Add(-time.Hour)
	notAfter := now.AddDate(10, 0, 0)

	rootTemplate := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "system:coordinator:root"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	rootCert, rootPEM, err := createCert(rootTemplate, rootTemplate, &rootPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, fmt.Errorf("creating root certificate: %w", err)
	}

	notAfter = now.AddDate(10, 0, 0)
	intermCACertTemplate := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "system:coordinator:intermediate"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	_, intermCAPEM, err := createCert(intermCACertTemplate, rootCert, &intermPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, fmt.Errorf("creating intermediate certificate: %w", err)
	}

	meshCACertTemplate := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "system:coordinator:intermediate"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	meshCACert, meshCAPEM, err := createCert(meshCACertTemplate, meshCACertTemplate, &intermPrivKey.PublicKey, intermPrivKey)
	if err != nil {
		return nil, fmt.Errorf("creating mesh certificate: %w", err)
	}
	meshCACertPool := x509.NewCertPool()
	if !meshCACertPool.AppendCertsFromPEM(meshCAPEM) {
		return nil, fmt.Errorf("creating mesh CA cert pool")
	}
	ca := CA{
		rootCAPrivKey:  rootPrivKey,
		rootCAPEM:      rootPEM,
		intermPrivKey:  intermPrivKey,
		intermCAPEM:    intermCAPEM,
		meshCACert:     meshCACert,
		meshCAPEM:      meshCAPEM,
		meshCACertPool: meshCACertPool,
	}

	return &ca, nil
}

// NewAttestedMeshCert creates a new attested mesh certificate.
func (c *CA) NewAttestedMeshCert(names []string, extensions []pkix.Extension, subjectPublicKey any) ([]byte, error) {
	var dnsNames []string
	var ips []net.IP
	var uris []*url.URL
	for _, name := range names {
		// If a string parses correctly as an IP address, it is not a valid DNS name anyway, so we
		// can split the SANs into DNS and IP by that predicate.
		if ip := net.ParseIP(name); ip != nil {
			ips = append(ips, ip)
		} else if uri, err := url.Parse(name); err == nil && uri.Scheme != "" {
			// Similarly, if a string parses as a URL with scheme, it's not a valid DNS name and
			// we can safely add it to the URI SANs.
			uris = append(uris, uri)
		} else {
			dnsNames = append(dnsNames, name)
		}
	}

	now := time.Now()
	certTemplate := &x509.Certificate{
		Subject:               pkix.Name{CommonName: dnsNames[0]},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(1, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		ExtraExtensions:       extensions,
		DNSNames:              dnsNames,
		IPAddresses:           ips,
		URIs:                  uris,
	}

	_, certPEM, err := createCert(certTemplate, c.meshCACert, subjectPublicKey, c.intermPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return certPEM, nil
}

// GetRootCACert returns the root certificate of the CA in PEM format.
func (c *CA) GetRootCACert() []byte {
	return c.rootCAPEM
}

// GetIntermCAPrivKey returns the intermediate private key of the CA.
func (c *CA) GetIntermCAPrivKey() *ecdsa.PrivateKey {
	return c.intermPrivKey
}

// GetIntermCACert returns the intermediate CA certificate in PEM format.
func (c *CA) GetIntermCACert() []byte {
	return c.intermCAPEM
}

// GetMeshCACert returns the mesh CA certificate of the CA in PEM format.
func (c *CA) GetMeshCACert() []byte {
	return c.meshCAPEM
}

// GetMeshCACertPool returns a certificate pool, containing the current mesh CA certificate.
func (c *CA) GetMeshCACertPool() *x509.CertPool {
	return c.meshCACertPool
}

// createCert issues a new certificate for pub, based on template, signed by parent with priv.
//
// It returns the certificate both in PEM encoding and as an x509 struct.
func createCert(template, parent *x509.Certificate, pub, priv any) (*x509.Certificate, []byte, error) {
	if parent == nil {
		return nil, nil, errors.New("parent cannot be nil")
	}
	if template == nil {
		return nil, nil, errors.New("cert cannot be nil")
	}
	if template.SerialNumber != nil {
		return nil, nil, errors.New("cert serial number must be nil")
	}

	serialNum, err := cryptohelpers.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, nil, fmt.Errorf("generating serial number: %w", err)
	}
	template.SerialNumber = serialNum

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("creating certificate: %w", err)
	}

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing the created certificate: %w", err)
	}

	return cert, certPem, nil
}
