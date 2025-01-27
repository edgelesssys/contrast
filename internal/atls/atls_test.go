// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package atls

import (
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"testing"

	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyEmbeddedReport(t *testing.T) {
	fakeAttDoc := FakeAttestationDoc{}
	attDocBytes, err := json.Marshal(fakeAttDoc)
	assert.NoError(t, err)

	testCases := map[string]struct {
		cert       *x509.Certificate
		validators []Validator
		wantErr    bool
		targetErr  error
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
			validators: NewFakeValidators(stubSNPValidator{}),
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
			validators: NewFakeValidators(stubSNPValidator{}),
		},
		"skip non-matching validator": {
			cert: &x509.Certificate{
				Extensions: []pkix.Extension{
					{
						Id: []int{4, 5, 6},
					},
					{
						Id:    oid.RawSNPReport,
						Value: attDocBytes,
					},
				},
			},
			validators: append(NewFakeValidators(stubSNPValidator{}), NewFakeValidator(stubFooValidator{})),
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
			validators: NewFakeValidators(stubSNPValidator{}),
			wantErr:    true,
		},
		"no extensions": {
			cert:       &x509.Certificate{},
			validators: nil,
			targetErr:  ErrNoValidAttestationExtensions,
			wantErr:    true,
		},
		"no matching validator": {
			cert: &x509.Certificate{
				Extensions: []pkix.Extension{
					{
						Id: oid.RawSNPReport,
					},
				},
			},
			validators: nil,
			targetErr:  ErrNoMatchingValidators,
			wantErr:    true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			err := verifyEmbeddedReport(tc.validators, tc.cert, nil, nil)
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

type stubSNPValidator struct{}

func (v stubSNPValidator) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

type stubFooValidator struct{}

func (v stubFooValidator) OID() asn1.ObjectIdentifier {
	return []int{1, 2, 3}
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
