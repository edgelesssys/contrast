// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package rtmr

import (
	"bytes"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"unicode/utf16"

	"github.com/edgelesssys/contrast/tdx-measure/tdvf"
	"github.com/foxboron/go-uefi/authenticode"
)

// Rtmr tracks the state of a "Run-Time Measurement Register".
type Rtmr struct {
	value [48]byte
}

// Extend adds a new measurement to the RTMR as specified in Intel® Trust
// Domain Extensions (Intel® TDX) Module Base Architecture Specification,
// 12.2.2 RTMR: Run-Time Measurement Registers.
func (r *Rtmr) Extend(bytes [48]byte) {
	hash := sha512.New384()
	hash.Write(r.value[:])
	hash.Write(bytes[:])
	copy(r.value[:], hash.Sum([]byte{}))
}

// Get returns the value currently stored inside the RTMR.
func (r *Rtmr) Get() [48]byte {
	return r.value
}

// Some helpers for common events:

func (r *Rtmr) hashAndExtend(bytes []byte) {
	r.Extend(sha512.Sum384(bytes))
}

func (r *Rtmr) extendVariable(uuid [16]byte, name string, data []byte) {
	unicodeName := utf16.Encode([]rune(name))

	varlog := make([]byte, 16+8+8+len(unicodeName)*2+len(data))
	copy(varlog[0:16], uuid[:])
	binary.LittleEndian.PutUint64(varlog[16:24], uint64(len(unicodeName)))
	binary.LittleEndian.PutUint64(varlog[24:32], uint64(len(data)))
	for i, codepoint := range unicodeName {
		binary.LittleEndian.PutUint16(varlog[32+i*2:][:2], codepoint)
	}
	copy(varlog[32+len(unicodeName)*2:], data)
	r.hashAndExtend(varlog)
}

func (r *Rtmr) extendVariableValue(data []byte) {
	r.hashAndExtend(data)
}

func (r *Rtmr) extendSeparator() {
	r.hashAndExtend([]byte{0, 0, 0, 0})
}

var (
	efiGlobalVariable = [16]byte{
		0x61, 0xdf, 0xe4, 0x8b,
		0xca, 0x93,
		0xd2, 0x11,
		0xaa, 0x0d,
		0x00, 0xe0, 0x98, 0x03, 0x2b, 0x8c,
	}
	efiImageSecurityDatabaseGUID = [16]byte{
		0xcb, 0xb2, 0x19, 0xd7,
		0x3a, 0x3d,
		0x96, 0x45,
		0xa3, 0xbc,
		0xda, 0xd0, 0x0e, 0x67, 0x65, 0x6f,
	}
)

// CalcRtmr0 calculates RTMR[0] for the given firmware.
func CalcRtmr0(firmware []byte) ([48]byte, error) {
	var rtmr Rtmr

	// This is the hash for the TDVF handoff blob.
	// TODO(freax13): Don't hard-code this, calculate it instead.
	hobHash := "40d4d32837fad8e321850fa6d62e231e9381f83037febf152ae3c67ea492315e5d9e7587ce35c7dbced397ca3ef404ca"
	var buffer [48]byte
	if _, err := hex.Decode(buffer[:], []byte(hobHash)); err != nil {
		panic(err)
	}
	rtmr.Extend(buffer)

	cvf, err := tdvf.FindCfv(firmware)
	if err != nil {
		return [48]byte{}, fmt.Errorf("can't find CFV section in firmware: %w", err)
	}
	rtmr.hashAndExtend(cvf)

	rtmr.extendVariable(efiGlobalVariable, "SecureBoot", []byte{})
	rtmr.extendVariable(efiGlobalVariable, "PK", []byte{})
	rtmr.extendVariable(efiGlobalVariable, "KEK", []byte{})
	rtmr.extendVariable(efiImageSecurityDatabaseGUID, "db", []byte{})
	rtmr.extendVariable(efiImageSecurityDatabaseGUID, "dbx", []byte{})

	rtmr.extendSeparator()

	// These are the hashes for some fw_cfg/ACPI related measurements.
	// TODO(freax13): Don't hard-code these, calculate them instead.
	configHashes := []string{
		"d0d923dc2c67e4f8cc506741e3962e6d408e2b5db7a4486d77922690f9f5ee74e3e4d30b0f04b6acc433b834237f0fde",
		"bdbe3030a4109e7889d2b745545aaf20e6139dadc13d912eff12aae1b8122fef3ec19085f6a8dd169d1eeae5c3e8a38d",
		"8abc60b76a03c41cfdde44342a3599ace0c47c0744c259ae0046bbbc18f51e75db3fc14cb76f3af07bc3a0b733b3e8f6",
	}
	for _, hash := range configHashes {
		var buffer [48]byte
		if _, err := hex.Decode(buffer[:], []byte(hash)); err != nil {
			panic(err)
		}
		rtmr.Extend(buffer)
	}

	rtmr.extendVariableValue([]byte{0, 0}) // BootOrder
	// This is an EFI path.
	// TODO(freax13): Don't just hard-code the whole thing.
	boot0000 := "090100002c0055006900410070007000000004071400c9bdb87cebf8344faaea3ee4af6516a10406140021aa2c4614760345836e8ab6f46623317fff0400"
	boot, err := hex.DecodeString(boot0000)
	if err != nil {
		panic(err)
	}
	rtmr.extendVariableValue(boot)

	rtmr.extendSeparator()

	return rtmr.Get(), nil
}

// CalcRtmr1 calculates RTMR[1] for the given kernel.
func CalcRtmr1(kernelFile []byte) ([48]byte, error) {
	var rtmr Rtmr
	kernelHashContent, err := hashKernel(kernelFile)
	if err != nil {
		return [48]byte{}, fmt.Errorf("can't hash kernel: %w", err)
	}
	rtmr.hashAndExtend(kernelHashContent)
	rtmr.hashAndExtend([]byte("Calling EFI Application from Boot Option"))
	rtmr.hashAndExtend([]byte("Exit Boot Services Invocation"))
	rtmr.hashAndExtend([]byte("Exit Boot Services Returned with Success"))
	return rtmr.Get(), nil
}

func hashKernel(kernelFile []byte) ([]byte, error) {
	patchKernel(kernelFile)

	kernel, err := authenticode.Parse(bytes.NewReader(kernelFile))
	if err != nil {
		return nil, fmt.Errorf("can't parse kernel: %w", err)
	}

	return kernel.HashContent.Bytes(), nil
}

func patchKernel(kernelFile []byte) {
	// QEMU patches some header bytes in the kernel before loading it into memory.
	// Sources:
	// - https://gitlab.com/qemu-project/qemu/-/blob/28ae3179fc52d2e4d870b635c4a412aab99759e7/hw/i386/x86-common.c#L837
	// - https://github.com/confidential-containers/td-shim/blob/51833bd509fbac49487bc4d483b7efd70d95e8b8/td-shim-tools/src/bin/td-payload-reference-calculator/main.rs#L65
	// - Intel® TDX Virtual Firmware Design Guide, 12.2 UEFI Kernel Image Hash Calculation.

	kernelFile[0x210] = 0xb0

	kernelFile[0x211] = 0x81

	kernelFile[0x224] = 0x00
	kernelFile[0x225] = 0xfe

	kernelFile[0x228] = 0x00
	kernelFile[0x229] = 0x00
	kernelFile[0x22A] = 0x02
	kernelFile[0x22B] = 0x00
}
