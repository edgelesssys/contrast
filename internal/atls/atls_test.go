// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package atls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"testing"

	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/oid"
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
	for typ, hex := range map[string]string{
		"rsa":   rsaTestKey,
		"ecdsa": ecdsaTestKey,
	} {
		t.Run(typ, func(t *testing.T) {
			require := require.New(t)
			der, err := manifest.HexString(hex).Bytes()
			require.NoError(err)
			key, err := x509.ParsePKCS8PrivateKey(der)
			require.NoError(err)

			// Just verify that publicKey does not panic.
			_ = publicKey(key)
		})
	}
}

var (
	rsaTestKey   = `30820276020100300d06092a864886f70d0101010500048202603082025c02010002818100bca0b6909e86dde330b4a30879c55355cef98316a63f53f8cafe2c7829c6fce9571bea6544e3e8a7a38263306fe409a2a6d1d98084487644cfc0cf6af9d495475a0c7227b7a06dc370de8993932a7c4135549680590b7d30e3a00c31dc2c3f83ee18b543fcc660c8fe4d818b922e49e0ff85086c0aa989ef0151c341d2d66e35020301000102818100818fc636699ceb55bcc3a66410f817b88dd4d654bd562c406c75cf67ae126eef7b94c218530c5466a929cb259f053c150b8e825e02fe9eb5bf19899eca0159929de7b92ec023def5481c8f98866c12a648e05ee6a1d68a515f3c3bdd40215b0189f55146a680830b02a3dea1562eb7e2b99808eac65c2943ab5c009e52cd8761024100e23065e13b75df9c4f2232a191468ad7f3033167689792e6006ba934b3679c05de56ac45b1f94cf4656d949ffe3a0284f2835c7b7674ef67d4702986b99f837d024100d57d0030121c94efc34805003f5d07b24b4b42899c88d7ab0a40f73a9fd2668001b7b35605610869e7c10553df6e8247ad6190148c2e76bc2caf1f7569d2a31902404382e095c6729b48835218bca2a8e47e2a3974c081b664112464fdff0de149ef727a7a36df3522e3fb76269b4e7d300d507926dc6ef1de17269047c4bf98bddd02407ad181f525c649acb1f4d1e3b59048a83b06de0d9aff62cba4878173b9946aa183db7211afe085dd9f957d02268d45e804881742aaeee42217b6dbeb496903a90240504b88dd81d534e61347e02d42124ef9ff154cf56a960f3b0a9ec15f79ed45e9135bda1b5b2ce3982047281d485922ac5e0b21ef4dd8293ee8bcb1a0737885cb`
	ecdsaTestKey = `308187020100301306072a8648ce3d020106082a8648ce3d030107046d306b0201010420b713f884cd98c12d43c947e5ab2b4640f04741594d4551440f0f6113f9f3fa0ba14403420004d0260069e1538790da66f894f3e9558ee33ce8c9c59b5b7ffe2e3d333676aeee0ee0714cb77ff7fc02a45d31255e981343747564d529c8e351ee9c87fdaceaae`
)
