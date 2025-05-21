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
	pageSize  = 0x1000
	chunkSize = 256
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

// AddRegion adds multiple pages to the launch measurement.
//
// This function is a helper that calls `MemPageAdd` in a loop.
func (l *LaunchContext) AddRegion(gpa uint64, dataLen uint64) error {
	if dataLen%pageSize != 0 {
		return errors.New("data length is not a multiple of the page size")
	}

	numPages := dataLen / pageSize
	for i := range numPages {
		l.MemPageAdd(gpa + i*pageSize)
	}

	return nil
}

// ExtendRegion extends multiple chunks to the launch measurement.
//
// This function is a helper that calls `MrExtend` in a loop.
func (l *LaunchContext) ExtendRegion(gpa uint64, data []byte) error {
	if len(data)%chunkSize != 0 {
		return errors.New("data length is not a multiple of the chunk size")
	}

	numChunks := uint64(len(data)) / chunkSize
	for i := range numChunks {
		var chunk [256]byte
		copy(chunk[:], data[i*chunkSize:][:chunkSize])
		l.MrExtend(gpa+i*chunkSize, chunk)
	}

	return nil
}
