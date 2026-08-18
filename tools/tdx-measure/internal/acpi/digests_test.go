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
		"1714c92f524640a6763417944c2c0e59b81b8560c94e8f8cb3e0b30c895c2eb90b6fd380dce3e4b89b8356766c7eacee",
		"0d00216b9fcdff6a6dfb7bb92b8ccf35b6804d5d195f2b9dd34fbecd0585433cf567a520f3cefb45a792469f859430b2",
		"05e27a7651b031e69216a487d39aba403766b546bba04b2bebfdc5bc160275a75249cfe147a753b799b5023a17124da4",
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
