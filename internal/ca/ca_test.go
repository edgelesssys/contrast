package ca

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCA(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ca, err := New("namespace")
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
