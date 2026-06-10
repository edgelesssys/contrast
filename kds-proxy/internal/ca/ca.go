// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	caCertFile = "ca.crt"
	caKeyFile  = "ca.key"

	caValidity   = 10 * 365 * 24 * time.Hour
	leafValidity = 30 * 24 * time.Hour
)

// CA is a persistent signing authority used to mint leaf certs on demand.
type CA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey

	mu    sync.Mutex
	leafs map[string]leafEntry
}

type leafEntry struct {
	certPEM []byte
	keyPEM  []byte
	expires time.Time
}

// LoadOrGenerate loads a CA from dir, generating and persisting one if absent.
func LoadOrGenerate(dir string) (*CA, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating CA dir: %w", err)
	}
	certPath := filepath.Join(dir, caCertFile)
	keyPath := filepath.Join(dir, caKeyFile)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		cert, key, err := parseCAPEM(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("parsing existing CA material: %w", err)
		}
		return &CA{cert: cert, key: key, leafs: map[string]leafEntry{}}, nil
	}
	if certErr != nil && !errors.Is(certErr, os.ErrNotExist) {
		return nil, certErr
	}
	if keyErr != nil && !errors.Is(keyErr, os.ErrNotExist) {
		return nil, keyErr
	}

	cert, key, err := generateCA()
	if err != nil {
		return nil, err
	}
	if err := writeCAPEM(dir, cert, key); err != nil {
		return nil, err
	}
	return &CA{cert: cert, key: key, leafs: map[string]leafEntry{}}, nil
}

// CertPEM returns the CA certificate in PEM form for distribution to clients.
func (c *CA) CertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.cert.Raw})
}

// LeafPEM returns a leaf cert + key pair (PEM-encoded) for host, minting and caching it on first use.
func (c *CA) LeafPEM(host string) (certPEM, keyPEM []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.leafs[host]; ok && time.Now().Before(e.expires) {
		return e.certPEM, e.keyPEM, nil
	}
	cert, key, err := c.mintLeaf(host)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	c.leafs[host] = leafEntry{
		certPEM: certPEM,
		keyPEM:  keyPEM,
		expires: cert.NotAfter.Add(-time.Hour),
	}
	return certPEM, keyPEM, nil
}

func (c *CA) mintLeaf(host string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		DNSNames:     []string{host},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(leafValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.cert, &key.PublicKey, c.key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func generateCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "contrast kds-proxy CA"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(caValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func randomSerial() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func parseCAPEM(certPEM, keyPEM []byte) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, nil, errors.New("invalid CA cert PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "EC PRIVATE KEY" {
		return nil, nil, errors.New("invalid CA key PEM")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func writeCAPEM(dir string, cert *x509.Certificate, key *ecdsa.PrivateKey) error {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(filepath.Join(dir, caCertFile), certPEM, 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, caKeyFile), keyPEM, 0o600)
}
