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
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/edgelesssys/contrast/tdx-measure/tdvf"
	"github.com/foxboron/go-uefi/authenticode"
)

// GPUModel represents the GPU model used in the VM.
type GPUModel int

const (
	// GPUModelNone indicates that no GPU is used.
	GPUModelNone GPUModel = iota
	// GPUModelH100 indicates that an NVIDIA H100 GPU is used.
	GPUModelH100
	// GPUModelB200 indicates that an NVIDIA B200 GPU is used.
	GPUModelB200
)

// GPUModelFromString converts a string representation of a GPU model to the corresponding [GPUModel] type.
func GPUModelFromString(s string) (GPUModel, error) {
	lower := strings.ToLower(s)
	switch lower {
	case "none":
		return GPUModelNone, nil
	case "h100":
		return GPUModelH100, nil
	case "b200":
		return GPUModelB200, nil
	default:
		return GPUModelNone, fmt.Errorf("unknown GPU model: %s", s)
	}
}

// TotalAddressableMemoryMB returns the total addressable memory in megabytes for the given GPU model.
// It replicates the semantics of [1] for all supported GPU models.
// The BAR sizes used for calculating this can be found via `lspci -v`.
// The "memory at" outputs then need to be summed up and rounded up to the next power of two.
//
// [1]: https://github.com/NVIDIA/go-nvlib/blob/9a6788d93d8ccf1d8f026604f0b8b13d254f2500/pkg/nvpci/resources.go#L89
func (g GPUModel) TotalAddressableMemoryMB() int {
	switch g {
	case GPUModelH100:
		return 262144 // ceilpow2(128GB + 32MB + 16MB) = 256GB
	case GPUModelB200:
		return 524288 // ceilpow2(256GB + 64MB + 32MB) = 512GB
	default:
		return 0
	}
}

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
func CalcRtmr0(firmware []byte, gpu GPUModel) ([48]byte, error) {
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
	if gpu != GPUModelNone {
		// QEMU FW CFG.opt/ovmf/X-PciMmio64Mb (GPU only, set by Kata)
		// See:
		// - https://github.com/kata-containers/kata-containers/blob/c7d0c270ee7dfaa6d978e6e07b99dabdaf2b9fda/src/runtime/virtcontainers/qemu.go#L817-L827
		// - https://github.com/kata-containers/kata-containers/blob/c7d0c270ee7dfaa6d978e6e07b99dabdaf2b9fda/src/runtime/virtcontainers/qemu_arch_base.go#L877-L908
		rtmr.extendVariableValue([]byte(fmt.Sprintf("%d", gpu.TotalAddressableMemoryMB())))
	}
	// QEMU FW CFG.BootOrder
	rtmr.extendVariableValue([]byte("/rom@genroms/linuxboot_dma.bin\x00"))

	rtmr.extendVariable(efiGlobalVariable, "SecureBoot", []byte{})
	rtmr.extendVariable(efiGlobalVariable, "PK", []byte{})
	rtmr.extendVariable(efiGlobalVariable, "KEK", []byte{})
	rtmr.extendVariable(efiImageSecurityDatabaseGUID, "db", []byte{})
	rtmr.extendVariable(efiImageSecurityDatabaseGUID, "dbx", []byte{})

	rtmr.extendSeparator()

	// TODO(freax13): Don't hard-code these, calculate them instead.
	acpiHashes := []string{
		// These are the hashes for the ACPI tables (ACPI DATA).
		// They might change depending on firmware/qemu version or qemu command line.
		"978413224c711ace8c588bd45f9585657572c8053410df87f94bed7254feb88b4ce82233ead3db3721198a3a215efc1b",
		"7d49579cd2b17a399b29b8fd40f2fd66bf3d0fafcbfdfd9a3b912b7d4f81dd7dba85ee15768b36214d7507dc10fc6464",
		"4f564889e597ba62b02a0f5ad95ad9f8883947deadc3275fe289f5096c01ed3db8323d70681d04f694c025ee8426be11",
	}
	var configHashes []string
	if gpu == GPUModelNone {
		configHashes = slices.Concat(acpiHashes)
	}
	for _, hash := range configHashes {
		var buffer [48]byte
		if _, err := hex.Decode(buffer[:], []byte(hash)); err != nil {
			return [48]byte{}, fmt.Errorf("can't decode config hash %s: %w", hash, err)
		}
		rtmr.Extend(buffer)
	}

	return rtmr.Get(), nil
}

// CalcRtmr1 calculates RTMR[1] for the given kernel.
func CalcRtmr1(kernelFile []byte) ([48]byte, error) {
	var rtmr Rtmr

	kernel, err := authenticode.Parse(bytes.NewReader(kernelFile))
	if err != nil {
		return [48]byte{}, fmt.Errorf("can't parse kernel: %w", err)
	}
	kernelHash := kernel.Hash(crypto.SHA384)
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
