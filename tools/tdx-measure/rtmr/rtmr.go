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

const (
	mediaProtocolType         = 4
	piwgFirmwareFileSubType   = 6
	piwgFirmwareVolumeSubType = 7
	endOfPathType             = 0x7f
)

const (
	loadOptionActive      = 1 << 0
	loadOptionHidden      = 1 << 3
	loadOptionCategoryApp = 1 << 8
)

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
	fvNameGUID = [16]byte{
		0xc9, 0xbd, 0xb8, 0x7c,
		0xeb, 0xf8,
		0x34, 0x4f,
		0xaa, 0xea,
		0x3e, 0xe4, 0xaf, 0x65, 0x16, 0xa1,
	}
	fileGUID = [16]byte{
		0x21, 0xaa, 0x2c, 0x46,
		0x14, 0x76,
		0x03, 0x45,
		0x83, 0x6e,
		0x8a, 0xb6, 0xf4, 0x66, 0x23, 0x31,
	}
)

// buildFilePathList assembles the device path passed by QEMU for boot option Boot0000.
func buildFilePathList() ([]byte, error) {
	// The structure of `EFI_DEVICE_PATH_PROTOCOL` is described in Unified
	// Extensible Firmware Interface (UEFI) Specification, Release 2.10 Errata
	// A, 10.2 EFI Device Path Protocol.
	var buffer bytes.Buffer
	addNode := func(ty uint8, subType uint8, data []byte) error {
		if err := buffer.WriteByte(ty); err != nil {
			return err
		}
		if err := buffer.WriteByte(subType); err != nil {
			return err
		}
		length := len(data) + 4
		if err := binary.Write(&buffer, binary.LittleEndian, uint16(length)); err != nil {
			return err
		}
		if _, err := buffer.Write(data); err != nil {
			return err
		}
		return nil
	}

	// The structure of PIWG Firmware Volume Device Path nodes is described in
	// UEFI Platform Initialization Specification, Version 1.8 Errata A, II-8.2
	// Firmware Volume Media Device Path.
	if err := addNode(mediaProtocolType, piwgFirmwareVolumeSubType, fvNameGUID[:]); err != nil {
		return nil, err
	}

	// The structure of PIWG Firmware File Device Path nodes is described in
	// UEFI Platform Initialization Specification, Version 1.8 Errata A, II-8.3
	// Firmware File Media Device Path.
	if err := addNode(mediaProtocolType, piwgFirmwareFileSubType, fileGUID[:]); err != nil {
		return nil, err
	}

	// The structure of End Entire Device Path nodes is described in Unified
	// Extensible Firmware Interface (UEFI) Specification, Release 2.10 Errata
	// A, 10.3.1 Generic Device Path Structures, Table 10.2: Device Path End
	// Structure.
	if err := addNode(endOfPathType, 0xff, []byte{}); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// buildEfiLoadOption assembles the EFI_LOAD_OPTION passed by QEMU for boot option Boot0000.
func buildEfiLoadOption() ([]byte, error) {
	// The structure of `EFI_LOAD_OPTION` is described in Unified Extensible
	// Firmware Interface (UEFI) Specification, Release 2.10 Errata A, 3.1.3
	// Load Options.

	var attributes uint32 = loadOptionActive | loadOptionHidden | loadOptionCategoryApp
	filePathList, err := buildFilePathList()
	if err != nil {
		return nil, err
	}
	description := "UiApp"
	optionalData := []byte{}

	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.LittleEndian, attributes); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, uint16(len(filePathList))); err != nil {
		return nil, err
	}
	unicodeDescription := utf16.Encode([]rune(description))
	for _, codepoint := range unicodeDescription {
		if err := binary.Write(&buffer, binary.LittleEndian, codepoint); err != nil {
			return nil, err
		}
	}
	// null terminator
	if err := binary.Write(&buffer, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if _, err := buffer.Write(filePathList); err != nil {
		return nil, err
	}
	if _, err := buffer.Write(optionalData); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// CalcRtmr0 calculates RTMR[0] for the given firmware.
func CalcRtmr0(firmware []byte) ([48]byte, error) {
	var rtmr Rtmr

	// We don't measure the Hobs, the firmware verifies them instead.

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
		"4aafe8d74abbdba1d1a104bceacc12f88f0f6215ccefa7e725f74297ca52d8474a7db3a5f5a3be8de28e653b6b04fd13",
		"a1b0f8aab5a5ab458fb81c837d8133c99b5ea177365e93ae0335549b43ede7d5b0129d74a35d499586b09cea8435ee77",
		"4fd723ee785c7fe3107f6bd4db78587de3eb3b2841988d4e2e5cea4e1bd5bfe182c6f557397a3ac64bc9700f91901b8a",
	}
	for _, hash := range configHashes {
		var buffer [48]byte
		if _, err := hex.Decode(buffer[:], []byte(hash)); err != nil {
			panic(err)
		}
		rtmr.Extend(buffer)
	}

	rtmr.extendVariableValue([]byte{0, 0}) // BootOrder
	boot0000, err := buildEfiLoadOption()
	if err != nil {
		return [48]byte{}, err
	}
	rtmr.extendVariableValue(boot0000)

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

// CalcRtmr2 calculates RTMR[2] for the given kernel command line.
func CalcRtmr2(cmdLine string) ([48]byte, error) {
	var rtmr Rtmr

	codepoints := utf16.Encode([]rune(cmdLine))
	bytes := make([]byte, (len(codepoints)+1)*2)
	for i, codepoint := range codepoints {
		binary.LittleEndian.PutUint16(bytes[i*2:][:2], codepoint)
	}
	rtmr.hashAndExtend(bytes)

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