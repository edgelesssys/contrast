// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package snp

import "testing"

func TestGUIDRoundTrip(t *testing.T) {
	guids := []string{
		sevHashTableHeaderGUID,
		sevKernelEntryGUID,
		sevInitrdEntryGUID,
		sevCmdlineEntryGUID,
		ovmfTableFooterGUID,
		sevHashTableRVGUID,
		sevESResetBlockGUID,
		ovmfSEVMetaDataGUID,
	}
	for _, guid := range guids {
		b := guidBytesLE(guid)
		got := guidString(b[:])
		if got != guid {
			t.Errorf("guidString(guidBytesLE(%q)) = %q, want %q", guid, got, guid)
		}
	}
}
