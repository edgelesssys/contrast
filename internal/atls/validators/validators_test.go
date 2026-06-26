// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package validators

import (
	"context"
	"encoding/asn1"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAny(t *testing.T) {
	oid := asn1.ObjectIdentifier{1, 2, 3}
	for name, tc := range map[string]struct {
		validators []*stubValidator
		numTried   int
		err        error
	}{
		"vacuous truth": {err: ErrOIDNotSupported},
		"OID not supported": {
			validators: []*stubValidator{{err: ErrOIDNotSupported}, {err: ErrOIDNotSupported}},
			numTried:   2,
			err:        ErrOIDNotSupported,
		},
		"matching validator first": {
			validators: []*stubValidator{
				{},
				{err: ErrOIDNotSupported},
			},
			numTried: 1,
		},
		"matching validator second": {
			validators: []*stubValidator{
				{err: ErrOIDNotSupported},
				{},
			},
			numTried: 2,
		},
		"no validator passes": {
			validators: []*stubValidator{
				{err: assert.AnError},
				{err: assert.AnError},
			},
			numTried: 2,
			err:      assert.AnError,
		},
		"second validator passes": {
			validators: []*stubValidator{
				{err: assert.AnError},
				{},
			},
			numTried: 2,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			var validators []Validator
			for _, v := range tc.validators {
				validators = append(validators, v)
			}

			doc := []byte{1, 2, 3}
			rd := []byte{4, 5, 6}
			v := Any(validators...)
			err := v.Validate(t.Context(), oid, doc, rd)
			if tc.err != nil {
				assert.ErrorIs(err, tc.err)
			} else {
				assert.NoError(err)
			}

			for i := range len(tc.validators) {
				v := tc.validators[i]
				if i < tc.numTried {
					assert.True(oid.Equal(v.oid))
					assert.Equal(doc, v.doc)
					assert.Equal(rd, v.reportData)
				} else {
					assert.Nil(v.oid)
					assert.Nil(v.doc)
					assert.Nil(v.reportData)
				}
			}
		})
	}
}

type stubValidator struct {
	err error

	// saved arguments to Validate
	oid        asn1.ObjectIdentifier
	doc        []byte
	reportData []byte
}

// Validate validates an attestation doc and returns an error if validation failed.
//
// If validation passes, the validator guarantees that the given reportData was present in the
// attestation document.
func (s *stubValidator) Validate(_ context.Context, oid asn1.ObjectIdentifier, doc []byte, rd []byte) error {
	s.oid = oid
	s.doc = doc
	s.reportData = rd
	return s.err
}
