// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package validator

import (
	"encoding/binary"
	"testing"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/stretchr/testify/assert"
)

func TestGetDigestSNP(t *testing.T) {
	for name, tc := range map[string]struct {
		hostdata []byte
		err      error
	}{
		"success": {hostdata: newSlice(32)},
		"error":   {err: assert.AnError},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			qp := &stubSNPQuoteProvider{
				hostData: tc.hostdata,
				err:      tc.err,
			}
			g := &snpDigestGetter{qp}
			digest, err := g.GetDigest()
			assert.ErrorIs(err, tc.err)
			assert.Equal(tc.hostdata, digest)
		})
	}
}

type stubSNPQuoteProvider struct {
	hostData []byte
	err      error
}

func (s *stubSNPQuoteProvider) IsSupported() bool {
	return true
}

func (s *stubSNPQuoteProvider) GetRawQuote(reportData [64]byte) ([]uint8, error) {
	if s.err != nil {
		return nil, s.err
	}
	// This fakes an SNP report just enough so that it passes validation and contains the desired hostdata.
	raw := make([]byte, abi.ReportSize)
	if s.hostData != nil {
		copy(raw[0xC0:0xE0], s.hostData)
	}
	copy(raw[0x50:0x90], reportData[:])
	binary.LittleEndian.PutUint64(raw[0x08:0x10], 1<<17) // specified reserved field in Policy
	return raw, nil
}

func (s *stubSNPQuoteProvider) Product() *sevsnp.SevProduct {
	panic("not implemented")
}
