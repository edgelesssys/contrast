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
	mediaFirmwareVolumeGUID = [16]byte{
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
	mediaFirmwareFileGUID = [16]byte{
		0xdc, 0x5b, 0xc2, 0xee,
		0xf2, 0x67,
		0x95, 0x4d,
		0xb1, 0xd5,
		0xf8, 0x1b, 0x20, 0x39, 0xd1, 0x1d,
	}
)

type efiDevicePath struct {
	Type    uint8
	SubType uint8
	Data    []byte
}

// buildFilePathList assembles the device path passed by QEMU for boot option Boot0000.
func buildFilePathList(files []efiDevicePath) ([]byte, error) {
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

	for _, node := range files {
		if err := addNode(node.Type, node.SubType, node.Data); err != nil {
			return nil, err
		}
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
func buildEfiLoadOption(description string, attributes uint32, filePathList []byte) ([]byte, error) {
	// The structure of `EFI_LOAD_OPTION` is described in Unified Extensible
	// Firmware Interface (UEFI) Specification, Release 2.10 Errata A, 3.1.3
	// Load Options.

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

	// Configuration Firmware Volume (CFV).
	cvf, err := tdvf.FindCfv(firmware)
	if err != nil {
		return [48]byte{}, fmt.Errorf("can't find CFV section in firmware: %w", err)
	}
	rtmr.hashAndExtend(cvf)

	// QEMU FW CFG.BootMenu
	// This is a two byte little-endian number with values
	// - 0x0000: disabled
	// - 0x0001: enabled
	rtmr.extendVariableValue([]byte{0x00, 0x00})
	// QEMU FW CFG.BootOrder
	rtmr.extendVariableValue([]byte("/rom@genroms/linuxboot_dma.bin\x00"))

	rtmr.extendVariable(efiGlobalVariable, "SecureBoot", []byte{})
	rtmr.extendVariable(efiGlobalVariable, "PK", []byte{})
	rtmr.extendVariable(efiGlobalVariable, "KEK", []byte{})
	rtmr.extendVariable(efiImageSecurityDatabaseGUID, "db", []byte{})
	rtmr.extendVariable(efiImageSecurityDatabaseGUID, "dbx", []byte{})

	rtmr.extendSeparator()

	// TODO(freax13): Don't hard-code these, calculate them instead.
	configHashes := []string{
		// These are the hashes for the ACPI tables (ACPI DATA).
		// They might change depending on firmware/qemu version or qemu command line.
		"978413224c711ace8c588bd45f9585657572c8053410df87f94bed7254feb88b4ce82233ead3db3721198a3a215efc1b",
		"7d49579cd2b17a399b29b8fd40f2fd66bf3d0fafcbfdfd9a3b912b7d4f81dd7dba85ee15768b36214d7507dc10fc6464",
		"4f564889e597ba62b02a0f5ad95ad9f8883947deadc3275fe289f5096c01ed3db8323d70681d04f694c025ee8426be11",
	}
	for _, hash := range configHashes {
		var buffer [48]byte
		if _, err := hex.Decode(buffer[:], []byte(hash)); err != nil {
			return [48]byte{}, fmt.Errorf("can't decode config hash %s: %w", hash, err)
		}
		rtmr.Extend(buffer)
	}

	// EFI_GLOBAL_VARIABLE BootOrder
	// Content: Boot0000, Boot0001
	rtmr.extendVariableValue([]byte{0x00, 0x00, 0x01, 0x00})

	// EV_EFI_VARIABLE_BOOT
	// Boot0000: BootManagerMenuApp
	boot00000FilePathList, err := buildFilePathList(
		[]efiDevicePath{
			// The structure of PIWG Firmware Volume Device Path nodes is described in
			// UEFI Platform Initialization Specification, Version 1.8 Errata A, II-8.2
			// Firmware Volume Media Device Path.
			{Type: mediaProtocolType, SubType: piwgFirmwareVolumeSubType, Data: mediaFirmwareVolumeGUID[:]},
			// The structure of PIWG Firmware File Device Path nodes is described in
			// UEFI Platform Initialization Specification, Version 1.8 Errata A, II-8.3
			// Firmware File Media Device Path.
			{Type: mediaProtocolType, SubType: piwgFirmwareFileSubType, Data: mediaFirmwareFileGUID[:]},
		},
	)
	if err != nil {
		return [48]byte{}, err
	}
	boot0000, err := buildEfiLoadOption(
		"BootManagerMenuApp",
		loadOptionActive|loadOptionHidden|loadOptionCategoryApp,
		boot00000FilePathList,
	)
	if err != nil {
		return [48]byte{}, err
	}
	rtmr.extendVariableValue(boot0000)

	// EV_EFI_VARIABLE_BOOT
	// Boot0001: EFI Firmware Setup
	boot0001FilePathList, err := buildFilePathList(
		[]efiDevicePath{
			{Type: mediaProtocolType, SubType: piwgFirmwareVolumeSubType, Data: mediaFirmwareVolumeGUID[:]},
			{Type: mediaProtocolType, SubType: piwgFirmwareFileSubType, Data: fileGUID[:]},
		},
	)
	if err != nil {
		return [48]byte{}, err
	}
	boot0001, err := buildEfiLoadOption(
		"EFI Firmware Setup",
		loadOptionActive|loadOptionCategoryApp,
		boot0001FilePathList,
	)
	if err != nil {
		return [48]byte{}, err
	}
	rtmr.extendVariableValue(boot0001)

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

	// The following is prepended by OVMF, see
	// https://github.com/tianocore/edk2/blob/af9cc80359e320690877e4add870aa13fe889fbe/OvmfPkg/Library/X86QemuLoadImageLib/X86QemuLoadImageLib.c#L581
	// and https://github.com/tianocore/edk2/commit/a2ac0fea49996ab484c1a8761c234cc354f5a760
	cmdLine = "initrd=initrd " + cmdLine

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
