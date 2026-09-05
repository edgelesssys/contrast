// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package acpi

import "crypto/sha512"

// OVMFDataDigests hashes OVMF's fixed ACPI DATA sequence.
// Digests cover raw fw_cfg blobs before relocation and checksum patches.
func OVMFDataDigests(blobsDir string) ([][48]byte, error) {
	files := OVMFMeasuredFiles()
	digests := make([][48]byte, 0, len(files))
	for _, name := range files {
		blob, err := readBlob(blobsDir, name)
		if err != nil {
			return nil, err
		}
		digests = append(digests, sha512.Sum384(blob))
	}
	return digests, nil
}
