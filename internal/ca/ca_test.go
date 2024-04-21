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

	ca, err := New()
	require.NoError(err)
	assert.NotNil(ca)
	assert.NotNil(ca.rootCAPrivKey)
	assert.NotNil(ca.rootCACert)
	assert.NotNil(ca.rootCAPEM)
	assert.NotNil(ca.intermPrivKey)
	assert.NotNil(ca.intermCACert)
	assert.NotNil(ca.intermCAPEM)

	root := x509.NewCertPool()
	ok := root.AppendCertsFromPEM(ca.rootCAPEM)
	assert.True(ok)

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

			ca, err := New()
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

func TestRotateIntermCerts(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ca, err := New()
	require.NoError(err)

	oldIntermCert := ca.intermCACert
	oldintermPEM := ca.intermCAPEM
	oldMeshCACert := ca.meshCACert
	oldMeshCAPEM := ca.meshCAPEM

	err = ca.RotateIntermCerts()
	assert.NoError(err)
	assert.NotEqual(oldIntermCert, ca.intermCACert)
	assert.NotEqual(oldintermPEM, ca.intermCAPEM)
	assert.NotEqual(oldMeshCACert, ca.meshCACert)
	assert.NotEqual(oldMeshCAPEM, ca.meshCAPEM)
}

func TestCAConcurrent(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ca, err := New()
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
	rotateIntermCerts := func() {
		defer wg.Done()
		assert.NoError(ca.RotateIntermCerts())
	}
	newMeshCert := func() {
		defer wg.Done()
		_, err := ca.NewAttestedMeshCert([]string{"foo", "bar"}, []pkix.Extension{}, newKey(require).Public())
		assert.NoError(err)
	}

	wg.Add(5 * 5)
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

	go rotateIntermCerts()
	go rotateIntermCerts()
	go rotateIntermCerts()
	go rotateIntermCerts()
	go rotateIntermCerts()

	go newMeshCert()
	go newMeshCert()
	go newMeshCert()
	go newMeshCert()
	go newMeshCert()

	wg.Wait()
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
