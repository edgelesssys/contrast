// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// See Intel® Trust Domain Extensions (Intel® TDX) Module Base Architecture
// Specification, 12.2.1 MRTD: Build-Time Measurement Register for more
// information.
package mrtd

import (
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"hash"
)

const (
	pageSize  uint64 = 0x1000
	chunkSize uint64 = 256
)

// LaunchContext tracks the measurement state of a guest during launch.
type LaunchContext struct {
	mrTd hash.Hash
}

// NewLaunchContext initializes a new `LaunchContext`.
//
// This function corresponds to the TDH.MNG.INIT SEAM call.
//
// See Intel® Trust Domain Extensions (Intel® TDX) Module Architecture
// Application Binary Interface (ABI) Reference Specification, 5.3.37
// TDH.MNG.INIT Leaf.
func NewLaunchContext() LaunchContext {
	return LaunchContext{
		mrTd: sha512.New384(),
	}
}

// MemPageAdd updates the launch context as if a page was added at the given
// address.
//
// This method corresponds to the TDH.MEM.PAGE.ADD SEAM call.
//
// See Intel® Trust Domain Extensions (Intel® TDX) Module Base Architecture
// Specification, 9.9 Adding TD Private Pages during TD Build Time:
// TDH.MEM.PAGE.ADD and Intel® Trust Domain Extensions (Intel® TDX) Module
// Architecture Application Binary Interface (ABI) Reference Specification,
// 5.3.20 TDH.MEM.PAGE.ADD Leaf.
func (l *LaunchContext) MemPageAdd(gpa uint64) {
	var buffer [128]byte
	copy(buffer[:16], []byte("MEM.PAGE.ADD"))
	binary.LittleEndian.PutUint64(buffer[16:][:8], gpa)
	l.mrTd.Write(buffer[:])
}

// MrExtend updates the launch context as if some content was added at the
// given address.
//
// This method corresponds to the TDH.MR.EXTEND SEAM call.
//
// See Intel® Trust Domain Extensions (Intel® TDX) Module Architecture
// Application Binary Interface (ABI) Reference Specification, 5.3.44
// TDH.MR.EXTEND Leaf.
func (l *LaunchContext) MrExtend(gpa uint64, content [256]byte) {
	var buffer [128]byte
	copy(buffer[:16], []byte("MR.EXTEND"))
	binary.LittleEndian.PutUint64(buffer[16:][:8], gpa)
	l.mrTd.Write(buffer[:])

	l.mrTd.Write(content[:])
}

// Finalize finalizes the calculation of the MRTD.
//
// This method corresponds to the TDH.MR.FINALIZE SEAM call.
//
// See Intel® Trust Domain Extensions (Intel® TDX) Module Architecture
// Application Binary Interface (ABI) Reference Specification, 5.3.45
// TDH.MR.FINALIZE Leaf.
func (l *LaunchContext) Finalize() [48]byte {
	var digest [48]byte
	copy(digest[:], l.mrTd.Sum([]byte{}))
	return digest
}

// WriteRegion writes [data] to the launch measurement (i.e., the MRTD).
//
// For each page in the region, it calls `TDH_MEM_PAGE_ADD`, and,
// if [extend] is true, `TDH_MR_EXTEND` for each chunk in the page.
//
// In QEMU, this is done by the `KVM_TDX_INIT_MEM_REGION` ioctl.
// See: https://github.com/qemu/qemu/blob/de074358e99b8eb5076d3efa267e44c292c90e3e/target/i386/kvm/tdx.c#L359
func (l *LaunchContext) WriteRegion(gpa uint64, data []byte, dataLen uint64, extend bool) error {
	if dataLen%pageSize != 0 {
		return fmt.Errorf("data length 0x%X is not a multiple of page size 0x%X", dataLen, pageSize)
	}

	for i := range dataLen / pageSize {
		pageOffset := i * pageSize
		pageAddress := gpa + pageOffset

		l.MemPageAdd(pageAddress)

		if !extend {
			continue
		}

		for j := range pageSize / chunkSize {
			chunkOffset := pageOffset + j*chunkSize
			chunkAddress := pageAddress + j*chunkSize

			var chunk [256]byte
			copy(chunk[:], data[chunkOffset:][:chunkSize])

			l.MrExtend(chunkAddress, chunk)
		}
	}

	return nil
}
