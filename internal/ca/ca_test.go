// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCA(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	rootCAKey := newKey(require)
	meshCAKey := newKey(require)

	ca, err := New(rootCAKey, meshCAKey)
	require.NoError(err)
	assert.NotNil(ca)
	assert.NotNil(ca.rootCAPrivKey)
	assert.NotNil(ca.rootCACert)
	assert.NotNil(ca.rootCAPEM)
	assert.NotNil(ca.intermPrivKey)
	assert.NotNil(ca.intermCACert)
	assert.NotNil(ca.intermCAPEM)

	root := pool(t, ca.rootCAPEM)

	cert := parsePEMCertificate(t, ca.intermCAPEM)

	opts := x509.VerifyOptions{Roots: root}

	_, err = cert.Verify(opts)
	require.NoError(err)
}

func TestAttestedMeshCert(t *testing.T) {
	req := require.New(t)

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
			subjectPub: newKey(req).Public(),
		},
		"ips": {
			dnsNames:   []string{"foo", "192.0.2.1"},
			extensions: []pkix.Extension{},
			subjectPub: newKey(req).Public(),
			wantIPs:    1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			rootCAKey := newKey(require)
			meshCAKey := newKey(require)
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
	req := require.New(t)

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
			pub:      newKey(req).Public(),
			priv:     newKey(req),
		},
		"template nil": {
			parent:  &x509.Certificate{},
			pub:     newKey(req).Public(),
			priv:    newKey(req),
			wantErr: true,
		},
		"parent nil": {
			template: &x509.Certificate{},
			pub:      newKey(req).Public(),
			priv:     newKey(req),
			wantErr:  true,
		},
		"pub nil": {
			template: &x509.Certificate{},
			parent:   &x509.Certificate{},
			priv:     newKey(req),
			wantErr:  true,
		},
		"priv nil": {
			template: &x509.Certificate{},
			parent:   &x509.Certificate{},
			pub:      newKey(req).Public(),
			wantErr:  true,
		},
		"serial number already set": {
			template: &x509.Certificate{SerialNumber: big.NewInt(1)},
			parent:   &x509.Certificate{},
			pub:      newKey(req).Public(),
			priv:     newKey(req),
			wantErr:  true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			pem, err := createCert(tc.template, tc.parent, tc.pub, tc.priv)
			if tc.wantErr {
				assert.Error(err)
				return
			}

			assert.NoError(err)
			parsePEMCertificate(t, pem)
		})
	}
}

func TestCAConcurrent(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	rootCAKey := newKey(require)
	meshCAKey := newKey(require)
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
		_, err := ca.NewAttestedMeshCert([]string{"foo", "bar"}, []pkix.Extension{}, newKey(require).Public())
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

// TestCARecovery asserts that certificates issued by a CA verify correctly under a new CA using the same keys.
func TestCARecovery(t *testing.T) {
	require := require.New(t)
	rootCAKey := newKey(require)
	meshCAKey := newKey(require)

	oldCA, err := New(rootCAKey, meshCAKey)
	require.NoError(err)

	newCA, err := New(rootCAKey, meshCAKey)
	require.NoError(err)

	key := newKey(require)
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

func newKey(require *require.Assertions) *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(err)
	return key
}

func parsePEMCertificate(t *testing.T, data []byte) *x509.Certificate {
	block, _ := pem.Decode(data)
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}
