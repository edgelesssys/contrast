// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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
	"net"
	"sync"
	"time"

	"github.com/edgelesssys/contrast/internal/crypto"
)

// CA is a cross-signing certificate authority.
type CA struct {
	rootCAPrivKey *ecdsa.PrivateKey
	rootCACert    *x509.Certificate
	rootCAPEM     []byte

	// The intermPrivKey is used for both the intermediate and meshCA certificates.
	// This implements cross-signing for the leaf certificates.
	// This is also implemented in MarbleRun, see:
	// https://docs.edgeless.systems/marblerun/architecture/security#public-key-infrastructure-and-certificate-authority
	intermMux     sync.RWMutex
	intermPrivKey *ecdsa.PrivateKey

	intermCACert *x509.Certificate
	intermCAPEM  []byte

	meshCACert *x509.Certificate
	meshCAPEM  []byte
}

// New creates a new CA.
func New() (*CA, error) {
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

	ca := CA{
		rootCAPrivKey: rootPrivKey,
		rootCACert:    root,
		rootCAPEM:     rootPEM,
	}
	if err := ca.RotateIntermCerts(); err != nil {
		return nil, fmt.Errorf("rotating intermediate certificates: %w", err)
	}

	return &ca, nil
}

// NewAttestedMeshCert creates a new attested mesh certificate.
func (c *CA) NewAttestedMeshCert(names []string, extensions []pkix.Extension, subjectPublicKey any) ([]byte, error) {
	var dnsNames []string
	var ips []net.IP
	for _, name := range names {
		// If a string parses correctly as an IP address, it is not a valid DNS name anyway, so we
		// can split the SANs into DNS and IP by that predicate.
		if ip := net.ParseIP(name); ip != nil {
			ips = append(ips, ip)
		} else {
			dnsNames = append(dnsNames, name)
		}
	}

	c.intermMux.RLock()
	defer c.intermMux.RUnlock()
	now := time.Now()
	certTemplate := &x509.Certificate{
		Subject:               pkix.Name{CommonName: dnsNames[0]},
		Issuer:                pkix.Name{CommonName: "system:coordinator:intermediate"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(1, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		ExtraExtensions:       extensions,
		DNSNames:              dnsNames,
		IPAddresses:           ips,
	}

	certPEM, err := createCert(certTemplate, c.meshCACert, subjectPublicKey, c.intermPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return certPEM, nil
}

// RotateIntermCerts rotates the intermediate and mesh CA certificate.
// All existing mesh certificates will remain valid under the rootCA but
// not under the new intermediate and mesh CA certificates.
// To distribute the new intermediate and mesh CA certificates, all workloads
// should be restarted.
func (c *CA) RotateIntermCerts() error {
	c.intermMux.Lock()
	defer c.intermMux.Unlock()

	now := time.Now()
	notBefore := now.Add(-time.Hour)
	notAfter := now.AddDate(10, 0, 0)
	c.intermCACert = &x509.Certificate{
		Subject:               pkix.Name{CommonName: "system:coordinator:intermediate"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	var err error
	c.intermPrivKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating intermediate private key: %w", err)
	}
	c.intermCAPEM, err = createCert(c.intermCACert, c.rootCACert, &c.intermPrivKey.PublicKey, c.rootCAPrivKey)
	if err != nil {
		return fmt.Errorf("creating intermediate certificate: %w", err)
	}

	c.meshCACert = &x509.Certificate{
		Subject:               pkix.Name{CommonName: "system:coordinator:intermediate"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	c.meshCAPEM, err = createCert(c.meshCACert, c.meshCACert, &c.intermPrivKey.PublicKey, c.intermPrivKey)
	if err != nil {
		return fmt.Errorf("creating mesh certificate: %w", err)
	}

	return nil
}

// GetRootCACert returns the root certificate of the CA in PEM format.
func (c *CA) GetRootCACert() []byte {
	return c.rootCAPEM
}

// GetIntermCACert returns the intermediate CA certificate in PEM format.
func (c *CA) GetIntermCACert() []byte {
	return c.intermCAPEM
}

// GetMeshCACert returns the mesh CA certificate of the CA in PEM format.
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
