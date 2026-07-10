// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package atls

import (
	"bytes"
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/edgelesssys/contrast/internal/atls/reportdata"
	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyEmbeddedReport(t *testing.T) {
	expectedReportData := reportdata.Construct(nil, nil)
	fakeAttDoc := &fakeAttestationDoc{
		ReportData: expectedReportData[:],
	}
	attDocBytes, err := json.Marshal(fakeAttDoc)
	require.NoError(t, err)

	testCases := map[string]struct {
		cert      *x509.Certificate
		validator validators.Validator
		wantErr   bool
		targetErr error
	}{
		"success": {
			cert: &x509.Certificate{
				Extensions: []pkix.Extension{
					{
						Id: oid.RawTDXReport,
					},
					{
						Id:    oid.RawSNPReport,
						Value: attDocBytes,
					},
				},
			},
			validator: newFakeValidator(oid.RawSNPReport),
		},
		"multiple matches": {
			cert: &x509.Certificate{
				Extensions: []pkix.Extension{
					{
						Id:    oid.RawSNPReport,
						Value: []byte("foo"),
					},
					{
						Id:    oid.RawSNPReport,
						Value: attDocBytes,
					},
				},
			},
			validator: newFakeValidator(oid.RawSNPReport),
		},
		"match, error": {
			cert: &x509.Certificate{
				Extensions: []pkix.Extension{
					{
						Id:    oid.RawSNPReport,
						Value: []byte("foo"),
					},
				},
			},
			validator: newFakeValidator(oid.RawSNPReport),
			wantErr:   true,
		},
		"no extensions": {
			cert:      &x509.Certificate{},
			validator: nil,
			targetErr: ErrNoValidAttestationExtensions,
			wantErr:   true,
		},
		"no matching validator": {
			cert: &x509.Certificate{
				Extensions: []pkix.Extension{
					{
						Id: oid.RawSNPReport,
					},
				},
			},
			validator: validators.Any(),
			targetErr: ErrNoMatchingValidators,
			wantErr:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			err := verifyEmbeddedReport(t.Context(), tc.validator, tc.cert, nil, nil)
			if tc.wantErr {
				require.Error(err)
				if tc.targetErr != nil {
					assert.ErrorIs(err, tc.targetErr)
				}
			} else {
				require.NoError(err)
			}
		})
	}
}

// fakeValidator fakes a validator and can be used for tests.
type fakeValidator struct {
	oid asn1.ObjectIdentifier
	err error
}

// newFakeValidator returns a single FakeValidator.
func newFakeValidator(oid asn1.ObjectIdentifier) validators.Validator {
	return &fakeValidator{oid, nil}
}

func (v fakeValidator) Validate(_ context.Context, id asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error {
	if !v.oid.Equal(id) {
		return validators.ErrOIDNotSupported
	}
	var doc fakeAttestationDoc
	if err := json.Unmarshal(attDoc, &doc); err != nil {
		return err
	}

	if !bytes.Equal(doc.ReportData, reportData) {
		return fmt.Errorf("invalid reportData: expected %x, got %x", doc.ReportData, reportData)
	}

	return v.err
}

func (v *fakeValidator) String() string {
	return ""
}

// fakeAttestationDoc is a fake attestation document used for testing.
type fakeAttestationDoc struct {
	ReportData []byte
}

// TestPublicKey ensures that all key types used by Contrast can be passed to publicKey.
func TestPublicKey(t *testing.T) {
	for typ, key := range map[string]crypto.PrivateKey{
		"rsa":   testkeys.RSA(t),
		"ecdsa": testkeys.ECDSA(t),
	} {
		t.Run(typ, func(t *testing.T) {
			// Just verify that publicKey does not panic.
			require.NotPanics(t, func() {
				_ = publicKey(key)
			})
		})
	}
}

// contextValidator fakes a validator that takes a long time to validate.
// If the inputC channel is not fed with a result, it will wait for the context to expire.
type contextValidator struct {
	inputC <-chan error
}

func (c *contextValidator) Validate(ctx context.Context, _ asn1.ObjectIdentifier, _, _ []byte) error {
	select {
	case err := <-c.inputC:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *contextValidator) String() string {
	return "contextValidator"
}

// TestContextPassdown ensures that the context argument of verifyEmbeddedReport is properly passed down to the validators.
func TestContextPassdown(t *testing.T) {
	validator := &contextValidator{make(chan error)}
	cert := &x509.Certificate{
		Extensions: []pkix.Extension{
			{
				Id: oid.RawSNPReport,
			},
		},
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	// If the context is not passed down, the select statement in ValidateContext will not return at all.
	// We expect this function to return because the context was already cancelled.
	err := verifyEmbeddedReport(ctx, validator, cert, nil, nil)
	// The contextValidator forwards the context error, so this should be canceled.
	require.ErrorIs(t, err, context.Canceled)
}

func TestNonceInALPN(t *testing.T) {
	var nonce [cryptohelpers.RNGLengthDefault]byte

	nextProto := encodeNonceToNextProtos(nonce[:])

	for name, tc := range map[string]struct {
		supportedProtos []string
		shouldFail      bool
		wantErr         error
	}{
		"no protocols": {
			shouldFail: true,
			wantErr:    errNoNonce,
		},
		"unrelated protocols": {
			supportedProtos: []string{"h2"},
			shouldFail:      true,
			wantErr:         errNoNonce,
		},
		"first": {
			supportedProtos: []string{nextProto, "h2"},
		},
		"last": {
			supportedProtos: []string{"h2", nextProto},
		},
		"bad nonce": {
			supportedProtos: []string{"atls:v1:nonce:bad nonce value"},
			shouldFail:      true,
		},
		"wrong version": {
			supportedProtos: []string{"atls:v2:nonce:02f2f9a189459c46c3eb8a40683ca4a07bbe05fc82a18cf023025481de178ab5"},
			shouldFail:      true,
			wantErr:         errVersionMismatch,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			gotNonce, err := decodeNonceFromSupportedProtos(tc.supportedProtos)
			if !tc.shouldFail {
				require.NoError(err)
				require.Equal(nonce[:], gotNonce)
				return
			}
			require.Error(err)
			if tc.wantErr != nil {
				require.ErrorIs(err, tc.wantErr)
			}
		})
	}
}

func TestGetNonce(t *testing.T) {
	wantNonce := [cryptohelpers.RNGLengthDefault]byte{42}

	for name, tc := range map[string]struct {
		clientHello *tls.ClientHelloInfo
		wantErr     error
	}{
		"ALPN": {
			clientHello: &tls.ClientHelloInfo{
				SupportedProtos: []string{encodeNonceToNextProtos(wantNonce[:])},
			},
		},
		"SNI": {
			clientHello: &tls.ClientHelloInfo{
				ServerName: base64.StdEncoding.EncodeToString(wantNonce[:]),
			},
		},
		"no nonce": {
			clientHello: &tls.ClientHelloInfo{},
			wantErr:     errNoNonce,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			nonce, err := getNonce(tc.clientHello)
			if tc.wantErr != nil {
				assert.Nil(nonce)
				assert.ErrorIs(err, tc.wantErr)
				return
			}
			assert.Equal(wantNonce[:], nonce)
			assert.NoError(err)
		})
	}
}
