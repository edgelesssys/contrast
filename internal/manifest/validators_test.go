// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"context"
	"encoding/asn1"
	"log/slog"
	"testing"

	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoordinatorValidator(t *testing.T) {
	for name, tc := range map[string]struct {
		// Configuration for the fake factory
		validatorOutputHash []byte
		validatorErr        error

		// Hashes allowed by manifest
		allowedHashes []HexString

		// Final validation outcome
		expectValidationErr error
	}{
		"validator fails": {
			validatorErr:        assert.AnError,
			expectValidationErr: assert.AnError,
		},
		"happy path single coordinator": {
			validatorOutputHash: []byte{1, 2, 3},
			allowedHashes:       []HexString{"010203"},
		},
		"matching coordinator first": {
			validatorOutputHash: []byte{1, 2, 3},
			allowedHashes:       []HexString{"010203", "040506"},
		},
		"matching coordinator second": {
			validatorOutputHash: []byte{4, 5, 6},
			allowedHashes:       []HexString{"010203", "040506"},
		},
		"no coordinator": {
			validatorOutputHash: []byte{4, 5, 6},
			expectValidationErr: ErrWrongCoordinatorPolicyHash,
		},
		"no match": {
			validatorOutputHash: []byte{7, 8, 9},
			allowedHashes:       []HexString{"010203", "040506"},
			expectValidationErr: ErrWrongCoordinatorPolicyHash,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			dummyValidatorFactory := func(_ *slog.Logger, _ *certcache.CachedHTTPSGetter, reportSetter attestation.ReportSetter) (validators.Validator, error) {
				return validators.ValidatorFunc(func(context.Context, asn1.ObjectIdentifier, []byte, []byte) error {
					if tc.validatorErr != nil {
						return tc.validatorErr
					}
					reportSetter.SetReport(&stubReport{hostData: tc.validatorOutputHash})
					return nil
				}), nil
			}

			v, err := coordinatorValidator(dummyValidatorFactory, tc.allowedHashes, nil, nil)
			require.NoError(err)
			require.NotNil(v)

			validationErr := v.Validate(t.Context(), nil, nil, nil)
			if tc.expectValidationErr != nil {
				require.ErrorIs(validationErr, tc.expectValidationErr)
			} else {
				require.NoError(validationErr)
			}
		})
	}

	t.Run("bad factory", func(t *testing.T) {
		require := require.New(t)

		badFactory := func(*slog.Logger, *certcache.CachedHTTPSGetter, attestation.ReportSetter) (validators.Validator, error) {
			return nil, assert.AnError
		}

		v, err := coordinatorValidator(badFactory, nil, nil, nil)
		require.NoError(err)
		require.NotNil(v)

		require.ErrorIs(v.Validate(t.Context(), nil, nil, nil), assert.AnError)
	})

	t.Run("bad validator", func(t *testing.T) {
		require := require.New(t)

		badValidatorFactory := func(*slog.Logger, *certcache.CachedHTTPSGetter, attestation.ReportSetter) (validators.Validator, error) {
			return validators.ValidatorFunc(func(context.Context, asn1.ObjectIdentifier, []byte, []byte) error {
				return nil
			}), nil
		}

		v, err := coordinatorValidator(badValidatorFactory, nil, nil, nil)
		require.NoError(err)
		require.NotNil(v)

		require.ErrorIs(v.Validate(t.Context(), nil, nil, nil), ErrBadValidator)
	})
}

type stubReport struct {
	attestation.Report

	hostData []byte
}

func (r *stubReport) HostData() []byte {
	return r.hostData
}
