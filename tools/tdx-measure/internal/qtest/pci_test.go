// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qtest

import "testing"

func TestPCIConfigAddress(t *testing.T) {
	if got, want := pciConfigAddress(2, 3, 1, 0x65), uint32(0x80021964); got != want {
		t.Fatalf("pciConfigAddress() = %#x, want %#x", got, want)
	}
}
