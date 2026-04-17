// Translated from the Python sev-snp-measure tool:
// Copyright 2022- IBM Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0
// Source: https://github.com/virtee/sev-snp-measure/blob/46664d0347fb07c5ac2cb8ab5bf5aebc09fc67ab/sevsnpmeasure/gctx.py

package snp

import (
	"crypto/sha512"
	"encoding/binary"
)

const (
	// LaunchDigestSize is the size of the SNP launch digest in bytes (SHA-384).
	LaunchDigestSize = sha512.Size384 // 48 bytes
	vmsaGPA          = uint64(0xFFFFFFFFF000)
	pageSize         = 4096
)

const (
	pageTypeNormal     = uint8(0x01)
	pageTypeVMSA       = uint8(0x02)
	pageTypeZero       = uint8(0x03)
	pageTypeUnmeasured = uint8(0x04)
	pageTypeSecrets    = uint8(0x05)
	pageTypeCPUID      = uint8(0x06)
)

var zeros [LaunchDigestSize]byte

// GCTX holds the SNP guest launch digest state.
type GCTX struct {
	ld [LaunchDigestSize]byte
}

// NewGCTX creates a new GCTX with an all-zero seed.
func NewGCTX() *GCTX {
	return &GCTX{}
}

// NewGCTXWithSeed creates a new GCTX with the provided seed.
func NewGCTXWithSeed(seed [LaunchDigestSize]byte) *GCTX {
	return &GCTX{ld: seed}
}

// LaunchDigest returns the current launch digest.
func (g *GCTX) LaunchDigest() [LaunchDigestSize]byte {
	return g.ld
}

// update calls the PAGE_INFO hash update as defined in SNP spec 8.17.2 Table 67.
func (g *GCTX) update(pageType uint8, gpa uint64, contents [LaunchDigestSize]byte) {
	const pageInfoLen = 0x70
	isIMI := uint8(0)
	vmpl3Perms := uint8(0)
	vmpl2Perms := uint8(0)
	vmpl1Perms := uint8(0)

	// Build the PAGE_INFO structure (112 bytes = 0x70)
	var info [pageInfoLen]byte
	copy(info[0:LaunchDigestSize], g.ld[:])
	copy(info[LaunchDigestSize:LaunchDigestSize*2], contents[:])
	binary.LittleEndian.PutUint16(info[LaunchDigestSize*2:], pageInfoLen)
	info[LaunchDigestSize*2+2] = pageType
	info[LaunchDigestSize*2+3] = isIMI
	info[LaunchDigestSize*2+4] = vmpl3Perms
	info[LaunchDigestSize*2+5] = vmpl2Perms
	info[LaunchDigestSize*2+6] = vmpl1Perms
	info[LaunchDigestSize*2+7] = 0
	binary.LittleEndian.PutUint64(info[LaunchDigestSize*2+8:], gpa)

	g.ld = sha512.Sum384(info[:])
}

// UpdateNormalPages measures len(data)/pageSize normal pages starting at startGPA.
// data must be page-aligned.
func (g *GCTX) UpdateNormalPages(startGPA uint64, data []byte) {
	for offset := 0; offset < len(data); offset += pageSize {
		page := data[offset : offset+pageSize]
		g.update(pageTypeNormal, startGPA+uint64(offset), sha512.Sum384(page))
	}
}

// UpdateVMSAPage measures a single VMSA page.
func (g *GCTX) UpdateVMSAPage(data [pageSize]byte) {
	g.update(pageTypeVMSA, vmsaGPA, sha512.Sum384(data[:]))
}

// UpdateZeroPages measures length/pageSize zero pages starting at gpa.
func (g *GCTX) UpdateZeroPages(gpa uint64, lengthBytes int) {
	for offset := 0; offset < lengthBytes; offset += pageSize {
		g.update(pageTypeZero, gpa+uint64(offset), zeros)
	}
}

// UpdateUnmeasuredPages measures length/pageSize unmeasured pages starting at gpa.
func (g *GCTX) UpdateUnmeasuredPages(gpa uint64, lengthBytes int) {
	for offset := 0; offset < lengthBytes; offset += pageSize {
		g.update(pageTypeUnmeasured, gpa+uint64(offset), zeros)
	}
}

// UpdateSecretsPage measures a secrets page at gpa.
func (g *GCTX) UpdateSecretsPage(gpa uint64) {
	g.update(pageTypeSecrets, gpa, zeros)
}

// UpdateCPUIDPage measures a CPUID page at gpa.
func (g *GCTX) UpdateCPUIDPage(gpa uint64) {
	g.update(pageTypeCPUID, gpa, zeros)
}
