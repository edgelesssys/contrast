// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package ca

import (
	"crypto/ecdsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"sync"
	"testing"

	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCA(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	rootCAKey := newKey(t, 0)
	meshCAKey := newKey(t, 1)

	ca, err := New(rootCAKey, meshCAKey)
	require.NoError(err)
	assert.NotNil(ca)
	assert.NotNil(ca.rootCAPrivKey)
	assert.NotNil(ca.rootCAPEM)
	assert.NotNil(ca.intermPrivKey)
	assert.NotNil(ca.intermCAPEM)

	root := pool(t, ca.rootCAPEM)

	cert := parsePEMCertificate(t, ca.intermCAPEM)

	opts := x509.VerifyOptions{Roots: root}

	_, err = cert.Verify(opts)
	require.NoError(err)
}

func TestAttestedMeshCert(t *testing.T) {
	testCases := map[string]struct {
		dnsNames   []string
		extensions []pkix.Extension
		subjectPub any
		wantErr    bool
		wantIPs    int
	}{
		"valid": {
			dnsNames:   []string{"foo", "bar"},
			extensions: []pkix.Extension{},
			subjectPub: newKey(t, 0).Public(),
		},
		"ips": {
			dnsNames:   []string{"foo", "192.0.2.1"},
			extensions: []pkix.Extension{},
			subjectPub: newKey(t, 0).Public(),
			wantIPs:    1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			rootCAKey := newKey(t, 1)
			meshCAKey := newKey(t, 2)
			ca, err := New(rootCAKey, meshCAKey)
			require.NoError(err)

			pem, err := ca.NewAttestedMeshCert(tc.dnsNames, tc.extensions, tc.subjectPub)
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.NotNil(pem)

			cert := parsePEMCertificate(t, pem)
			assert.Len(cert.IPAddresses, tc.wantIPs)
		})
	}
}

func TestCreateCert(t *testing.T) {
	testCases := map[string]struct {
		template *x509.Certificate
		parent   *x509.Certificate
		pub      any
		priv     any
		wantErr  bool
	}{
		"parent signed": {
			template: &x509.Certificate{},
			parent:   &x509.Certificate{},
			pub:      newKey(t, 0).Public(),
			priv:     newKey(t, 1),
		},
		"template nil": {
			parent:  &x509.Certificate{},
			pub:     newKey(t, 0).Public(),
			priv:    newKey(t, 1),
			wantErr: true,
		},
		"parent nil": {
			template: &x509.Certificate{},
			pub:      newKey(t, 0).Public(),
			priv:     newKey(t, 1),
			wantErr:  true,
		},
		"pub nil": {
			template: &x509.Certificate{},
			parent:   &x509.Certificate{},
			priv:     newKey(t, 0),
			wantErr:  true,
		},
		"priv nil": {
			template: &x509.Certificate{},
			parent:   &x509.Certificate{},
			pub:      newKey(t, 0).Public(),
			wantErr:  true,
		},
		"serial number already set": {
			template: &x509.Certificate{SerialNumber: big.NewInt(1)},
			parent:   &x509.Certificate{},
			pub:      newKey(t, 0).Public(),
			priv:     newKey(t, 1),
			wantErr:  true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			cert, pem, err := createCert(tc.template, tc.parent, tc.pub, tc.priv)
			if tc.wantErr {
				assert.Error(err)
				return
			}

			require.NoError(t, err)
			parsedCert := parsePEMCertificate(t, pem)
			assert.Equal(*parsedCert.SerialNumber, *cert.SerialNumber)
			assert.Equal(parsedCert.Subject, cert.Subject)
			assert.Equal(parsedCert.SubjectKeyId, cert.SubjectKeyId)
			assert.Equal(parsedCert.Issuer, cert.Issuer)
			assert.Equal(parsedCert.AuthorityKeyId, cert.AuthorityKeyId)
		})
	}
}

func TestCAConcurrent(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	rootCAKey := newKey(t, 0)
	meshCAKey := newKey(t, 1)
	ca, err := New(rootCAKey, meshCAKey)
	require.NoError(err)

	wg := sync.WaitGroup{}
	getIntermCert := func() {
		defer wg.Done()
		assert.NotEmpty(ca.GetIntermCACert())
	}
	getMeshCACert := func() {
		defer wg.Done()
		assert.NotEmpty(ca.GetMeshCACert())
	}
	getRootCACert := func() {
		defer wg.Done()
		assert.NotEmpty(ca.GetRootCACert())
	}
	newMeshCert := func() {
		defer wg.Done()
		_, err := ca.NewAttestedMeshCert([]string{"foo", "bar"}, []pkix.Extension{}, newKey(t, 2).Public())
		assert.NoError(err)
	}

	wg.Add(4 * 5)
	go getIntermCert()
	go getIntermCert()
	go getIntermCert()
	go getIntermCert()
	go getIntermCert()

	go getMeshCACert()
	go getMeshCACert()
	go getMeshCACert()
	go getMeshCACert()
	go getMeshCACert()

	go getRootCACert()
	go getRootCACert()
	go getRootCACert()
	go getRootCACert()
	go getRootCACert()

	go newMeshCert()
	go newMeshCert()
	go newMeshCert()
	go newMeshCert()
	go newMeshCert()

	wg.Wait()
}

func TestCertValidity(t *testing.T) {
	require := require.New(t)
	rootCAKey := newKey(t, 0)
	meshCAKey := newKey(t, 1)
	key := newKey(t, 2)

	ca, err := New(rootCAKey, meshCAKey)
	require.NoError(err)
	crt, err := ca.NewAttestedMeshCert([]string{"localhost"}, nil, key.Public())
	require.NoError(err)

	assertValidPEMCert(t, ca.GetRootCACert())
	assertValidPEMCert(t, ca.GetMeshCACert())
	assertValidPEMCert(t, ca.GetIntermCACert())
	assertValidPEMCert(t, crt)
}

func assertValidPEMCert(t *testing.T, pem []byte) {
	crt := parsePEMCertificate(t, pem)
	if crt.IsCA {
		assert.NotEmpty(t, crt.SubjectKeyId, "TLSv3 requires a Subject Key ID for CA certificates")
	}
	if crt.Issuer.CommonName != crt.Subject.CommonName {
		assert.NotEmpty(t, crt.AuthorityKeyId, "TLSv3 requires an Authority Key ID for non-root certificates")
	}
	assert.Equal(t, 3, crt.Version, "certificate should be TLSv3")
}

// TestCARecovery asserts that certificates issued by a CA verify correctly under a new CA using the same keys.
func TestCARecovery(t *testing.T) {
	require := require.New(t)
	rootCAKey := newKey(t, 0)
	meshCAKey := newKey(t, 1)

	oldCA, err := New(rootCAKey, meshCAKey)
	require.NoError(err)

	newCA, err := New(rootCAKey, meshCAKey)
	require.NoError(err)

	key := newKey(t, 2)
	oldCert, err := oldCA.NewAttestedMeshCert([]string{"localhost"}, nil, key.Public())
	require.NoError(err)
	newCert, err := newCA.NewAttestedMeshCert([]string{"localhost"}, nil, key.Public())
	require.NoError(err)

	require.NotEqual(oldCA.GetRootCACert(), newCA.GetRootCACert())
	require.NotEqual(oldCert, newCert)

	require.Equal(parsePEMCertificate(t, oldCA.GetIntermCACert()).SubjectKeyId, parsePEMCertificate(t, oldCA.GetMeshCACert()).SubjectKeyId)

	// Clients are represented by their configured root certificate and the
	// additional intermediates they should have received from the server.
	clients := map[string]x509.VerifyOptions{
		"old-root": {Roots: pool(t, oldCA.GetRootCACert()), Intermediates: pool(t, oldCA.GetIntermCACert())},
		"new-root": {Roots: pool(t, newCA.GetRootCACert()), Intermediates: pool(t, newCA.GetIntermCACert())},
		"old-mesh": {Roots: pool(t, oldCA.GetMeshCACert())},
		"new-mesh": {Roots: pool(t, newCA.GetMeshCACert())},
	}

	servers := map[string]*x509.Certificate{
		"old": parsePEMCertificate(t, oldCert),
		"new": parsePEMCertificate(t, newCert),
	}

	for clientName, client := range clients {
		t.Run("client="+clientName, func(t *testing.T) {
			for serverName, server := range servers {
				t.Run("server="+serverName, func(t *testing.T) {
					_, err = server.Verify(client)
					assert.NoError(t, err)
				})
			}
		})
	}
}

func pool(t *testing.T, pem []byte) *x509.CertPool {
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(pem))
	return pool
}

func newKey(t *testing.T, id int) *ecdsa.PrivateKey {
	return testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[id])
}

func parsePEMCertificate(t *testing.T, data []byte) *x509.Certificate {
	block, _ := pem.Decode(data)
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}
