// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package snp

import (
	"testing"

	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/stretchr/testify/require"
)

func TestClaimsToCertExtension(t *testing.T) {
	require := require.New(t)
	report := &sevsnp.Report{
		Policy: 0x00000000000f0000,
	}
	exts, err := claimsToCertExtension(report)
	require.NoError(err)

	// Check that no OIDs are used multiple times
	oidSet := make(map[string]struct{})
	for _, ext := range exts {
		oid := ext.Id.String()
		_, ok := oidSet[oid]
		require.False(ok, "OID %s used multiple times", oid)
		oidSet[oid] = struct{}{}
	}
}
