// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cache

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCRLFreshness(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCRLSign | x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &key.PublicKey, key)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)
	crlDER, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now().Add(-time.Minute),
		NextUpdate: time.Now().Add(12 * time.Hour),
	}, ca, key)
	require.NoError(t, err)
	crlPEM := pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: crlDER})

	for _, c := range []struct {
		name string
		body []byte
	}{
		{"der", crlDER},
		{"pem", crlPEM},
	} {
		t.Run(c.name, func(t *testing.T) {
			d, ok := crlFreshness(c.body)
			require.True(t, ok, "expected a CRL-derived freshness")
			assert.Greater(t, d, 11*time.Hour)
			assert.LessOrEqual(t, d, 12*time.Hour)
		})
	}

	_, ok := crlFreshness([]byte("not a crl"))
	assert.False(t, ok, "garbage should not be treated as a CRL")
}

func TestCacheControlMaxAge(t *testing.T) {
	cases := []struct {
		name   string
		status int
		header string
		want   time.Duration
		ok     bool
	}{
		{"empty", http.StatusOK, "", 0, false},
		{"max-age 60", http.StatusOK, "max-age=60", 60 * time.Second, true},
		{"public max-age", http.StatusOK, "public, max-age=3600", 3600 * time.Second, true},
		{"no-store", http.StatusOK, "no-store, max-age=60", 0, false},
		{"no-cache", http.StatusOK, "no-cache", 0, false},
		{"zero", http.StatusOK, "max-age=0", 0, false},
		{"garbage", http.StatusOK, "max-age=NaN", 0, false},
		{"must-understand understood", http.StatusOK, "no-store, must-understand, max-age=60", 60 * time.Second, true},
		{"must-understand unknown status", http.StatusNoContent, "no-store, must-understand, max-age=60", 0, false},
		{"must-understand no-cache", http.StatusOK, "no-cache, must-understand, max-age=60", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := http.Header{}
			if c.header != "" {
				h.Set("Cache-Control", c.header)
			}
			got, ok := cacheControlMaxAge(c.status, h)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestPutGetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	require.NoError(t, err)
	h := http.Header{}
	h.Set("Cache-Control", "max-age=120")
	_, err = c.Put("https://example/x", 200, h, []byte("hello"))
	require.NoError(t, err)
	e, fresh := c.Get("https://example/x")
	require.NotNil(t, e)
	assert.True(t, fresh)
	assert.Equal(t, "hello", string(e.Body))

	// Reopen from disk.
	c2, err := New(dir)
	require.NoError(t, err)
	e2, fresh2 := c2.Get("https://example/x")
	require.NotNil(t, e2)
	assert.True(t, fresh2)
	assert.Equal(t, "hello", string(e2.Body))
}

func TestStaleEntryStillReturned(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	require.NoError(t, err)
	h := http.Header{}
	h.Set("Cache-Control", "max-age=1")
	e, err := c.Put("https://example/x", 200, h, []byte("hi"))
	require.NoError(t, err)
	e.FreshUntil = time.Now().Add(-time.Hour)
	got, fresh := c.Get("https://example/x")
	require.NotNil(t, got, "entry vanished")
	assert.False(t, fresh, "expected stale, got fresh")
}
