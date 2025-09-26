// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package validator

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDigest(t *testing.T) {
	badDigest := newSlice(32)
	slices.Reverse(badDigest)
	for name, tc := range map[string]struct {
		digestGetter digestGetter
		digest       []byte
		wantErr      error
	}{
		"equal length": {
			digestGetter: &stubDigestGetter{
				digest: newSlice(32),
			},
			digest: newSlice(32),
		},
		"longer input": {
			digestGetter: &stubDigestGetter{
				digest: newSlice(32),
			},
			digest: newSlice(48),
		},
		"longer reference": {
			digestGetter: &stubDigestGetter{
				digest: newSlice(48),
			},
			digest: newSlice(32),
		},
		"short input": {
			digestGetter: &stubDigestGetter{
				digest: newSlice(32),
			},
			digest:  newSlice(31),
			wantErr: errUnexpectedDigestSize,
		},
		"short reference": {
			digestGetter: &stubDigestGetter{
				digest: newSlice(31),
			},
			digest:  newSlice(32),
			wantErr: errUnexpectedDigestSize,
		},
		"nil input": {
			digestGetter: &stubDigestGetter{
				digest: newSlice(32),
			},
			wantErr: errUnexpectedDigestSize,
		},
		"nil reference": {
			digestGetter: &stubDigestGetter{},
			digest:       newSlice(32),
			wantErr:      errUnexpectedDigestSize,
		},
		"mismatch": {
			digestGetter: &stubDigestGetter{
				digest: badDigest,
			},
			digest:  newSlice(32),
			wantErr: errDigestMismatch,
		},
		"digestGetter error": {
			digestGetter: &stubDigestGetter{
				err: assert.AnError,
			},
			digest:  newSlice(32),
			wantErr: assert.AnError,
		},
	} {
		t.Run(name, func(t *testing.T) {
			v := &Validator{
				digestGetter: tc.digestGetter,
			}
			err := v.ValidateDigest(tc.digest)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

type stubDigestGetter struct {
	digest []byte
	err    error
}

func (g *stubDigestGetter) GetDigest() ([]byte, error) {
	return g.digest, g.err
}

func newSlice(n int) []byte {
	buf := make([]byte, n)
	for i := range n {
		buf[i] = byte(i % 256)
	}
	return buf
}
