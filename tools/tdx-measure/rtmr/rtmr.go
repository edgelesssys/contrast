// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package rtmr

import (
	"bytes"
	"crypto"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
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
	fmt.Fprintf(os.Stderr, "Extending RTMR with %s\n", hex.EncodeToString(bytes[:]))
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
		"4a41e320ca0bc7bc3c56c3aba1228e10fd946d95391a7638828319a7a82cefd5bb33eceb9fdf14c06f694c0583077610",
		"864cf9b429c91c2ff0295e2ce7d73737f347c413bb071e4b5e014727da5109a75588ccfd61b4d634f946af5c1207812b",
		"b08a2773870bc58f493030265cc1e433b6e54c839f3e8fad5287f732890e32f1225315deb05266e793165b8aa3dbad2c",
	}
	for _, hash := range configHashes {
		var buffer [48]byte
		if _, err := hex.Decode(buffer[:], []byte(hash)); err != nil {
			return [48]byte{}, fmt.Errorf("can't decode config hash %s: %w", hash, err)
		}
		rtmr.Extend(buffer)
	}

	rtmr.extendVariableValue([]byte{0, 0}) // BootOrder
	boot0000, err := buildEfiLoadOption()
	if err != nil {
		return [48]byte{}, err
	}
	rtmr.extendVariableValue(boot0000)

	return rtmr.Get(), nil
}

// CalcRtmr1 calculates RTMR[1] for the given kernel.
func CalcRtmr1(kernelFile, initrdFile []byte) ([48]byte, error) {
	var rtmr Rtmr

	kernelHash, err := hashKernel(kernelFile, initrdFile)
	if err != nil {
		return [48]byte{}, fmt.Errorf("can't hash kernel: %w", err)
	}
	if len(kernelHash) != 48 {
		return [48]byte{}, fmt.Errorf("kernel hash has unexpected length: %d", len(kernelHash))
	}
	rtmr.Extend([48]byte(kernelHash))

	// https://github.com/tianocore/edk2/blob/0f3867fa6ef0553e26c42f7d71ff6bdb98429742/OvmfPkg/Tcg/TdTcg2Dxe/TdTcg2Dxe.c#L2155
	rtmr.hashAndExtend([]byte("Calling EFI Application from Boot Option"))

	// https://github.com/tianocore/edk2/blob/efaf8931bbfa33a81b8792fbf9e2ccc239d53204/OvmfPkg/Tcg/TdTcg2Dxe/TdTcg2Dxe.c#L2171
	rtmr.extendSeparator()

	// https://github.com/tianocore/edk2/blob/0f3867fa6ef0553e26c42f7d71ff6bdb98429742/OvmfPkg/Tcg/TdTcg2Dxe/TdTcg2Dxe.c#L2243
	rtmr.hashAndExtend([]byte("Exit Boot Services Invocation"))
	// https://github.com/tianocore/edk2/blob/0f3867fa6ef0553e26c42f7d71ff6bdb98429742/OvmfPkg/Tcg/TdTcg2Dxe/TdTcg2Dxe.c#L2254
	rtmr.hashAndExtend([]byte("Exit Boot Services Returned with Success"))
	return rtmr.Get(), nil
}

// CalcRtmr2 calculates RTMR[2] for the given kernel command line and initrd.
func CalcRtmr2(cmdLine string, initrdFile []byte) ([48]byte, error) {
	var rtmr Rtmr

	// TODO(msanft): find out which component silently adds this string to the commandline.
	// Suspects: QEMU-TDX, OVMF-TDX, Linux EFI Stub
	cmdLine += " initrd=initrd"

	// https://elixir.bootlin.com/linux/v6.11.8/source/drivers/firmware/efi/libstub/efi-stub-helper.c#L342
	codepoints := utf16.Encode([]rune(cmdLine))
	bytes := make([]byte, (len(codepoints)+1)*2)
	for i, codepoint := range codepoints {
		binary.LittleEndian.PutUint16(bytes[i*2:][:2], codepoint)
	}
	rtmr.hashAndExtend(bytes)

	// https://elixir.bootlin.com/linux/v6.11.8/source/drivers/firmware/efi/libstub/efi-stub-helper.c#L625
	rtmr.hashAndExtend(initrdFile)

	return rtmr.Get(), nil
}

func hashKernel(kernelFile, initrdFile []byte) ([]byte, error) {
	patchKernel(kernelFile, initrdFile)

	kernel, err := authenticode.Parse(bytes.NewReader(kernelFile))
	if err != nil {
		return nil, fmt.Errorf("can't parse kernel: %w", err)
	}

	return kernel.Hash(crypto.SHA384), nil
}

func patchKernel(kernelFile, initrdFile []byte) {
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

	// https://github.com/qemu/qemu/blob/f48c205fb42be48e2e47b7e1cd9a2802e5ca17b0/hw/i386/x86.c#L1036
	// Qemu patches the initrd address into the kernel header. This is unnecessary because OVMF
	// will take the initrd address from fw_cfg and patch it again after measuring the kernel. In
	// order to have predictable kernel measurements, we patch qemu to set this placeholder instead
	// of a real address.
	initrdAddr := 0x80000000
	initrdSize := len(initrdFile)

	// https://github.com/qemu/qemu/blob/f48c205fb42be48e2e47b7e1cd9a2802e5ca17b0/hw/i386/x86.c#L1044
	binary.LittleEndian.PutUint32(kernelFile[0x218:][:4], uint32(initrdAddr))

	// https://github.com/qemu/qemu/blob/f48c205fb42be48e2e47b7e1cd9a2802e5ca17b0/hw/i386/x86.c#L1045
	binary.LittleEndian.PutUint32(kernelFile[0x21C:][:4], uint32(initrdSize))
}
