// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package acpi

import (
	"crypto/sha512"
	"encoding/hex"
	"os"
	"testing"
)

// Known Kata 4.0 digests in OVMFMeasuredFiles order.
var referenceDataDigests = map[string][]string{
	"default": {
		"a98ebc08f45482c21c329b93a51b926a299c4c366d4e79d6ba586e4a8c92a66e9d593b176ffdf32476f0dd81bb68f95b",
		"6cd5eb8fa5c659b3ac4172a1c678aa1bba3118e107dcc5cdea0deec83b50e152703822396ef3422fd020bbf0ffdc3491",
		"764a603f33825a43eff061cf6154516d8304eb420a08c06b7e3d7a956932b82efa591701b1854a1fb75a8deb54b2aefc",
	},
	"legacy-serial": {
		"8bf8a7115101120057677c1b839246af7528c59894df19d21de2cbc2368110d7a55f9b70641d78d8183efb0af0ecf69a",
		"10fa3f97624e4236b059cd90226d4d23e177f5dc2daf0d840d0ddfd95c0d9a478994b07d75258a869b1a895a5a244868",
		"19b296088ef7e494ae63fc51fe4010f826b638f6c1a443ea439a1953e7a568cfb54af33827fcd8d83f6b570e55851475",
	},
}

// Nix sets these paths for the real-blob regression.
var blobsDirEnv = map[string]string{
	"default":       "ACPI_BLOBS_DEFAULT_DIR",
	"legacy-serial": "ACPI_BLOBS_LEGACY_SERIAL_DIR",
}

func writeTestBlob(t *testing.T, blobsDir, name string, data []byte) {
	t.Helper()
	if err := WriteBlob(blobsDir, name, data); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// TestOVMFDataDigestsSynthetic checks ordering and raw SHA-384 hashing.
func TestOVMFDataDigestsSynthetic(t *testing.T) {
	dir := t.TempDir()

	loader := []byte("dummy-table-loader-blob")
	rsdp := []byte("dummy-rsdp-blob")
	tables := []byte("dummy-tables-blob-contents")
	writeTestBlob(t, dir, "etc/table-loader", loader)
	writeTestBlob(t, dir, "etc/acpi/rsdp", rsdp)
	writeTestBlob(t, dir, "etc/acpi/tables", tables)

	digests, err := OVMFDataDigests(dir)
	if err != nil {
		t.Fatalf("OVMFDataDigests: %v", err)
	}

	want := [][48]byte{
		sha512.Sum384(loader),
		sha512.Sum384(rsdp),
		sha512.Sum384(tables),
	}
	if len(digests) != len(want) {
		t.Fatalf("got %d digests, want %d", len(digests), len(want))
	}
	for i := range want {
		if digests[i] != want[i] {
			t.Errorf("digest %d: got %s, want %s", i,
				hex.EncodeToString(digests[i][:]), hex.EncodeToString(want[i][:]))
		}
	}
}

// TestOVMFDataDigestsRealBlobs checks generated blobs against known digests.
func TestOVMFDataDigestsRealBlobs(t *testing.T) {
	for topology, want := range referenceDataDigests {
		t.Run(topology, func(t *testing.T) {
			dir := os.Getenv(blobsDirEnv[topology])
			if dir == "" {
				t.Skipf("%s unset; skipping real-blob regression", blobsDirEnv[topology])
			}
			digests, err := OVMFDataDigests(dir)
			if err != nil {
				t.Fatalf("OVMFDataDigests: %v", err)
			}
			if len(digests) != len(want) {
				t.Fatalf("got %d digests, want %d", len(digests), len(want))
			}
			for i, w := range want {
				if got := hex.EncodeToString(digests[i][:]); got != w {
					t.Errorf("digest %d: got %s, want %s", i, got, w)
				}
			}
		})
	}
}
