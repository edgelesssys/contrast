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
	assert.NotNil(ca.rootPrivKey)
	assert.NotNil(ca.rootCert)
	assert.NotNil(ca.rootPEM)
	assert.NotNil(ca.intermPrivKey)
	assert.NotNil(ca.intermCert)
	assert.NotNil(ca.intermPEM)

	root := x509.NewCertPool()
	ok := root.AppendCertsFromPEM(ca.rootPEM)
	assert.True(ok)

	block, _ := pem.Decode(ca.intermPEM)
	require.NotNil(block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(err)

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
	}{
		"valid": {
			dnsNames:   []string{"foo", "bar"},
			extensions: []pkix.Extension{},
			subjectPub: newKey(req).Public(),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			ca, err := New()
			require.NoError(err)

			cert, err := ca.NewAttestedMeshCert(tc.dnsNames, tc.extensions, tc.subjectPub)
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.NotNil(cert)

			assertValidPEM(assert, cert)
		})
	}
}

func TestCerateCert(t *testing.T) {
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
			assertValidPEM(assert, pem)
		})
	}
}

func TestRotateIntermCerts(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ca, err := New()
	require.NoError(err)

	oldIntermCert := ca.intermCert
	oldintermPEM := ca.intermPEM
	oldMeshCACert := ca.meshCACert
	oldMeshCAPEM := ca.meshCAPEM

	err = ca.RotateIntermCerts()
	assert.NoError(err)
	assert.NotEqual(oldIntermCert, ca.intermCert)
	assert.NotEqual(oldintermPEM, ca.intermPEM)
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
		assert.NotEmpty(ca.GetIntermCert())
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

func assertValidPEM(assert *assert.Assertions, data []byte) {
	block, _ := pem.Decode(data)
	assert.NotNil(block)
	_, err := x509.ParseCertificate(block.Bytes)
	assert.NoError(err)
}
