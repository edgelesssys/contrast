// Translated from the Python sev-snp-measure tool:
// Copyright 2022- IBM Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0
// Source: https://github.com/virtee/sev-snp-measure/blob/46664d0347fb07c5ac2cb8ab5bf5aebc09fc67ab/sevsnpmeasure/sev_hashes.py

package snp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strings"
)

// GUIDs for the SEV hash table structures (ASCII UUID strings).
const (
	sevHashTableHeaderGUID = "9438d606-4f22-4cc9-b479-a793d411fd21"
	sevKernelEntryGUID     = "4de79437-abd2-427f-b835-d5b172d2045b"
	sevInitrdEntryGUID     = "44baf731-3a2f-4bd7-9af1-41e29169781d"
	sevCmdlineEntryGUID    = "97d02dd8-bd20-4c94-aa78-e7714d36ab2a"
)

// guidBytesLE returns the 16-byte little-endian UUID encoding for a GUID already in the
// bytes_le-derived display format produced by guidString().
// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx where the first three groups are LE.
func guidBytesLE(s string) [16]byte {
	b, _ := hex.DecodeString(strings.ReplaceAll(s, "-", ""))
	var g [16]byte
	// timeLow (4 bytes): big-endian in canonical -> little-endian in bytes_le
	g[0], g[1], g[2], g[3] = b[3], b[2], b[1], b[0]
	// timeMid (2 bytes): big-endian -> little-endian
	g[4], g[5] = b[5], b[4]
	// timeHi (2 bytes): big-endian -> little-endian
	g[6], g[7] = b[7], b[6]
	// rest (8 bytes): unchanged
	copy(g[8:], b[8:])
	return g
}

// SevHashes holds the SHA-256 hashes of kernel, initrd, and cmdline.
type SevHashes struct {
	KernelHash  [sha256.Size]byte
	InitrdHash  [sha256.Size]byte
	CmdlineHash [sha256.Size]byte
}

// NewSevHashes computes hashes of the kernel, initrd (may be empty string), and kernel command line.
func NewSevHashes(kernelPath, initrdPath, kernelCommandLine string) (*SevHashes, error) {
	kernelData, err := os.ReadFile(kernelPath)
	if err != nil {
		return nil, err
	}
	var initrdData []byte
	if initrdPath != "" {
		initrdData, err = os.ReadFile(initrdPath)
		if err != nil {
			return nil, err
		}
	}
	var cmdline []byte
	if kernelCommandLine != "" {
		cmdline = append([]byte(kernelCommandLine), 0x00)
	} else {
		cmdline = []byte{0x00}
	}
	return &SevHashes{
		KernelHash:  sha256.Sum256(kernelData),
		InitrdHash:  sha256.Sum256(initrdData),
		CmdlineHash: sha256.Sum256(cmdline),
	}, nil
}

// SevHashTableEntry binary layout (little-endian, packed):
//
//	guid   [16]byte
//	length uint16
//	hash   [sha256.Size]byte
//
// Size = 50 bytes.
const sevHashTableEntrySize = 16 + 2 + sha256.Size

// SevHashTable binary layout (little-endian, packed):
//
//	guid    [16]byte
//	length  uint16
//	cmdline SevHashTableEntry (50 bytes)
//	initrd  SevHashTableEntry (50 bytes)
//	kernel  SevHashTableEntry (50 bytes)
//
// Size = 16 + 2 + 50*3 = 168 bytes.
const sevHashTableSize = 16 + 2 + 3*sevHashTableEntrySize

// Padded to next 16-byte boundary: (168+15)&~15 = 176.
const paddedSevHashTableSize = (sevHashTableSize + 15) &^ 15

// ConstructTable builds the SEV hash table binary, padded to 176 bytes.
// This must be identical to how QEMU generates the hash table.
func (s *SevHashes) ConstructTable() [paddedSevHashTableSize]byte {
	var buf [paddedSevHashTableSize]byte

	headerGUID := guidBytesLE(sevHashTableHeaderGUID)
	kernelGUID := guidBytesLE(sevKernelEntryGUID)
	initrdGUID := guidBytesLE(sevInitrdEntryGUID)
	cmdlineGUID := guidBytesLE(sevCmdlineEntryGUID)

	writeEntry := func(b []byte, off int, guid [16]byte, hash [sha256.Size]byte) {
		copy(b[off:], guid[:])
		binary.LittleEndian.PutUint16(b[off+16:], uint16(sevHashTableEntrySize))
		copy(b[off+18:], hash[:])
	}

	// Header: guid + length (covers SevHashTable only, not padding)
	copy(buf[0:], headerGUID[:])
	binary.LittleEndian.PutUint16(buf[16:], uint16(sevHashTableSize))

	writeEntry(buf[:], 18, cmdlineGUID, s.CmdlineHash)
	writeEntry(buf[:], 18+sevHashTableEntrySize, initrdGUID, s.InitrdHash)
	writeEntry(buf[:], 18+2*sevHashTableEntrySize, kernelGUID, s.KernelHash)

	return buf
}

// ConstructPage places the hash table in a 4096-byte page at the given byte offset.
func (s *SevHashes) ConstructPage(offsetInPage uint32) [4096]byte {
	table := s.ConstructTable()
	var page [4096]byte
	copy(page[offsetInPage:], table[:])
	return page
}
