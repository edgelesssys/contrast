// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package validators

import (
	"context"
	"encoding/asn1"
	"testing"

	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAny(t *testing.T) {
	oid := asn1.ObjectIdentifier{1, 2, 3}
	for name, tc := range map[string]struct {
		validators []*stubValidator
		numTried   int
		err        error
	}{
		"vacuous truth": {err: ErrOIDNotSupported},
		"single validator passes through": {
			validators: []*stubValidator{{err: assert.AnError}},
			numTried:   1,
			err:        assert.AnError,
		},
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

func TestWithFixedOID(t *testing.T) {
	require := require.New(t)
	inputOID := oid.RawSNPReport
	expectedOID := oid.RawTDXReport
	s := &stubValidator{}
	v := WithFixedOID(expectedOID, s)

	require.NoError(v.Validate(t.Context(), inputOID, nil, nil))
	require.True(expectedOID.Equal(s.oid))
}

func TestNames(t *testing.T) {
	for name, tc := range map[string]struct {
		v        Validator
		wantName string
	}{
		"plain": {
			v:        &stubValidator{name: "foo"},
			wantName: "foo",
		},
		"Any": {
			v:        Any(&stubValidator{name: "foo"}, &stubValidator{name: "bar"}),
			wantName: "Any(foo, bar)",
		},
		"Any-0": {
			v:        Any(),
			wantName: "<no validator>",
		},
		"Any-1": {
			v:        Any(&stubValidator{name: "foo"}),
			wantName: "foo",
		},
		"WithFixedOID": {
			v:        WithFixedOID(asn1.ObjectIdentifier{1, 2, 3}, &stubValidator{name: "foo"}),
			wantName: "foo",
		},
		"Nested": {
			v: Any(
				Any(&stubValidator{name: "foo"}, &stubValidator{name: "bar"}),
				WithFixedOID(asn1.ObjectIdentifier{1, 2, 3}, &stubValidator{name: "baz"}),
			),
			wantName: "Any(Any(foo, bar), baz)",
		},
		"Named": {
			v:        Named("bar", &stubValidator{name: "foo"}),
			wantName: "bar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.wantName, tc.v.String())
		})
	}
}

type stubValidator struct {
	err  error
	name string

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

func (s *stubValidator) String() string {
	return s.name
}

var (
	_ Validator = ValidatorFunc(nil)
	_ Validator = (*anyOf)(nil)
	_ Validator = (*named)(nil)
	_ Validator = (*withFixedOID)(nil)
	_ Validator = (*noValidator)(nil)
)
