// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package igvm

import (
	"encoding/binary"
	"fmt"
)

// MagicValue is the required magic number in IGVM_FIXED_HEADER.
// This is ASCII "IGVM" in little-endian format.
const MagicValue uint32 = 0x4D564749

// FixedHeader represents the version 1 fixed header at the start of every IGVM file.
type FixedHeader struct {
	Magic                uint32
	FormatVersion        uint32
	VariableHeaderOffset uint32
	VariableHeaderSize   uint32
	TotalFileSize        uint32
	Checksum             uint32
}

// BinaryMarshal marshals the fixed header to a byte slice.
func (h *FixedHeader) BinaryMarshal() ([]byte, error) {
	data := make([]byte, 24)
	binary.LittleEndian.PutUint32(data[0:4], h.Magic)
	binary.LittleEndian.PutUint32(data[4:8], h.FormatVersion)
	binary.LittleEndian.PutUint32(data[8:12], h.VariableHeaderOffset)
	binary.LittleEndian.PutUint32(data[12:16], h.VariableHeaderSize)
	binary.LittleEndian.PutUint32(data[16:20], h.TotalFileSize)
	binary.LittleEndian.PutUint32(data[20:24], h.Checksum)
	return data, nil
}

// BinaryUnmarshal unmarshals the fixed header from a byte slice.
func (h *FixedHeader) BinaryUnmarshal(data []byte) error {
	if len(data) != 24 {
		return fmt.Errorf("expected 24 bytes, but got %d", len(data))
	}
	h.Magic = binary.LittleEndian.Uint32(data[0:4])
	h.FormatVersion = binary.LittleEndian.Uint32(data[4:8])
	h.VariableHeaderOffset = binary.LittleEndian.Uint32(data[8:12])
	h.VariableHeaderSize = binary.LittleEndian.Uint32(data[12:16])
	h.TotalFileSize = binary.LittleEndian.Uint32(data[16:20])
	h.Checksum = binary.LittleEndian.Uint32(data[20:24])
	return nil
}

// VariableHeaderType represents the type of each structure in the variable header section.
type VariableHeaderType uint32

const (
	// Invalid type
	Invalid VariableHeaderType = 0x0

	// IGVM_VHT_RANGE_PLATFORM structures
	VhtSupportedPlatform VariableHeaderType = 0x1

	// IGVM_VHT_RANGE_INIT structures
	VhtGuestPolicy               VariableHeaderType = 0x101
	VhtRelocatableRegion         VariableHeaderType = 0x102
	VhtPageTableRelocationRegion VariableHeaderType = 0x103

	// IGVM_VHT_RANGE_DIRECTIVE structures
	VhtParameterArea            VariableHeaderType = 0x301
	VhtPageData                 VariableHeaderType = 0x302
	VhtParameterInsert          VariableHeaderType = 0x303
	VhtVpContext                VariableHeaderType = 0x304
	VhtRequiredMemory           VariableHeaderType = 0x305
	ReservedDoNotUse            VariableHeaderType = 0x306
	VhtVpCountParameter         VariableHeaderType = 0x307
	VhtSrat                     VariableHeaderType = 0x308
	VhtMadt                     VariableHeaderType = 0x309
	VhtMmioRanges               VariableHeaderType = 0x30A
	VhtSnpIdBlock               VariableHeaderType = 0x30B
	VhtMemoryMap                VariableHeaderType = 0x30C
	VhtErrorRange               VariableHeaderType = 0x30D
	VhtCommandLine              VariableHeaderType = 0x30E
	VhtSlit                     VariableHeaderType = 0x30F
	VhtPptt                     VariableHeaderType = 0x310
	VhtVbsMeasurement           VariableHeaderType = 0x311
	VhtDeviceTree               VariableHeaderType = 0x312
	VhtEnvironmentInfoParameter VariableHeaderType = 0x313
)

// String method for human-readable output
func (t VariableHeaderType) String() string {
	switch t {
	case Invalid:
		return "Invalid"
	case VhtSupportedPlatform:
		return "VhtSupportedPlatform"
	case VhtGuestPolicy:
		return "VhtGuestPolicy"
	case VhtRelocatableRegion:
		return "VhtRelocatableRegion"
	case VhtPageTableRelocationRegion:
		return "VhtPageTableRelocationRegion"
	case VhtParameterArea:
		return "VhtParameterArea"
	case VhtPageData:
		return "VhtPageData"
	case VhtParameterInsert:
		return "VhtParameterInsert"
	case VhtVpContext:
		return "VhtVpContext"
	case VhtRequiredMemory:
		return "VhtRequiredMemory"
	case ReservedDoNotUse:
		return "ReservedDoNotUse"
	case VhtVpCountParameter:
		return "VhtVpCountParameter"
	case VhtSrat:
		return "VhtSrat"
	case VhtMadt:
		return "VhtMadt"
	case VhtMmioRanges:
		return "VhtMmioRanges"
	case VhtSnpIdBlock:
		return "VhtSnpIdBlock"
	case VhtMemoryMap:
		return "VhtMemoryMap"
	case VhtErrorRange:
		return "VhtErrorRange"
	case VhtCommandLine:
		return "VhtCommandLine"
	case VhtSlit:
		return "VhtSlit"
	case VhtPptt:
		return "VhtPptt"
	case VhtVbsMeasurement:
		return "VhtVbsMeasurement"
	case VhtDeviceTree:
		return "VhtDeviceTree"
	case VhtEnvironmentInfoParameter:
		return "VhtEnvironmentInfoParameter"
	default:
		return fmt.Sprintf("UnknownType(0x%X)", uint32(t))
	}
}

// VariableHeader represents a variable header in IGVM.
type VariableHeader struct {
	Type    VariableHeaderType
	Length  uint32
	Content []byte
	Padding []byte // Unmarshal only.
}

// BinaryMarshal marshals the variable header to a byte slice.
func (h *VariableHeader) BinaryMarshal() ([]byte, error) {
	var paddingLen int
	if h.Length%8 != 0 {
		paddingLen = 8 - int(h.Length)%8 // Round to 8 byte alignment
	}
	data := make([]byte, 8+int(h.Length)+paddingLen)
	binary.LittleEndian.PutUint32(data[0:4], uint32(h.Type))
	binary.LittleEndian.PutUint32(data[4:8], h.Length)
	if h.Length == 0 {
		return data, nil
	}
	copy(data[8:8+h.Length], h.Content)
	return data, nil
}

// BinaryUnmarshal unmarshals the variable header from a byte slice.
func (h *VariableHeader) BinaryUnmarshal(data []byte) error {
	h.Type = VariableHeaderType(binary.LittleEndian.Uint32(data[0:4]))
	h.Length = binary.LittleEndian.Uint32(data[4:8])
	if h.Length == 0 {
		return nil
	}
	h.Content = data[8 : 8+h.Length]
	if h.Length%8 != 0 {
		paddingLen := 8 - h.Length%8 // Round to 8 byte alignment
		h.Padding = data[8+h.Length : 8+h.Length+paddingLen]
	}
	return nil
}

// // VhsRequiredMemory describes memory the IGVM file expects to be present in the
// // guest. This is a hint to the loader that the guest will not function without
// // memory present at the specified range, and should terminate the load process
// // if memory is not present.
// //
// // This memory may or may not be measured, depending on the other structures
// // this range overlaps with in the variable header section.
// //
// // Note that the guest cannot rely on memory being present at this location at
// // runtime, as a malicious host may choose to ignore this header.
// type VhsRequiredMemory struct {
// 	GPA               uint64
// 	CompatibilityMask uint32
// 	NumberOfBytes     uint32
// 	Flags             RequiredMemoryFlags
// 	Reserved          uint32
// }

// // RequiredMemoryFlags represents flags for VhsRequiredMemory.
// type RequiredMemoryFlags struct {
// 	Vtl2Protectable bool
// 	Reserved        uint32 // 31 bits reserved
// }

// // VhsMemoryRange describes memory via a range of pages.
// type VhsMemoryRange struct {
// 	StartingGPAPageNumber uint64
// 	NumberOfPages         uint64
// }

// // VhsMmioRanges describes the MMIO ranges for the guest for a
// // [`IgvmVariableHeaderType::IGVM_VHT_MMIO_RANGES`] parameter.
// //
// // Note that this structure can only define two mmio ranges, for a full
// // reporting of the guest's mmio ranges, the
// // [`IgvmVariableHeaderType::IGVM_VHT_DEVICE_TREE`] parameter should be used
// // instead.
// type VhsMmioRanges struct {
// 	MMIORanges [2]VhsMemoryRange
// }

// VhsSnpIDBlockSignature represents the signature for the SNP ID block. See the corresponding PSP definitions.
type VhsSnpIDBlockSignature struct {
	RComp [72]uint8
	SComp [72]uint8
}

// BinaryMarshal marshals the SNP ID block signature to a byte slice.
func (b *VhsSnpIDBlockSignature) BinaryMarshal() ([]byte, error) {
	data := make([]byte, 144)
	copy(data[0:72], b.RComp[:])
	copy(data[72:144], b.SComp[:])
	return data, nil
}

// BinaryUnmarshal unmarshals the SNP ID block signature from a byte slice.
func (b *VhsSnpIDBlockSignature) BinaryUnmarshal(data []byte) error {
	if len(data) != 144 {
		return fmt.Errorf("expected 144 bytes, but got %d", len(data))
	}
	copy(b.RComp[:], data[0:72])
	copy(b.SComp[:], data[72:144])
	return nil
}

// VhsSnpIDBlockPublicKey represents the public key for the SNP ID block. See the corresponding PSP definitions.
type VhsSnpIDBlockPublicKey struct {
	Curve    uint32
	Reserved uint32
	QX       [72]uint8
	QY       [72]uint8
}

// BinaryMarshal marshals the SNP ID block public key to a byte slice.
func (b *VhsSnpIDBlockPublicKey) BinaryMarshal() ([]byte, error) {
	data := make([]byte, 152)
	binary.LittleEndian.PutUint32(data[0:4], b.Curve)
	binary.LittleEndian.PutUint32(data[4:8], b.Reserved)
	copy(data[8:80], b.QX[:])
	copy(data[80:152], b.QY[:])
	return data, nil
}

// BinaryUnmarshal unmarshals the SNP ID block public key from a byte slice.
func (b *VhsSnpIDBlockPublicKey) BinaryUnmarshal(data []byte) error {
	if len(data) != 152 {
		return fmt.Errorf("expected 144 bytes, but got %d", len(data))
	}
	b.Curve = binary.LittleEndian.Uint32(data[0:4])
	b.Reserved = binary.LittleEndian.Uint32(data[4:8])
	copy(b.QX[:], data[8:80])
	copy(b.QY[:], data[80:152])
	return nil
}

// VhsSnpIDBlock describes the AMD SEV-SNP ID block.
//
// AuthorKeyEnabled is set to 0x1 if an author key is to be used, with the
// following corresponding author keys populated. Otherwise, the author key
// fields must be zero.
//
// Other fields share the same meaning as defined in the SNP API specification.
//
// TODO: doc links for fields to SNP spec.
type VhsSnpIDBlock struct {
	CompatibilityMask  uint32
	AuthorKeyEnabled   uint8
	Reserved           [3]uint8
	LD                 [48]uint8
	FamilyID           [16]uint8
	ImageID            [16]uint8
	Version            uint32
	GuestSVN           uint32
	IDKeyAlgorithm     uint32
	AuthorKeyAlgorithm uint32
	IDKeySignature     VhsSnpIDBlockSignature
	IDPublicKey        VhsSnpIDBlockPublicKey
	AuthorKeySignature VhsSnpIDBlockSignature
	AuthorPublicKey    VhsSnpIDBlockPublicKey
}

// BinaryMarshal marshals the SNP ID block to a byte slice.
func (b *VhsSnpIDBlock) BinaryMarshal() ([]byte, error) {
	data := make([]byte, 696)
	binary.LittleEndian.PutUint32(data[0:4], b.CompatibilityMask)
	data[4] = b.AuthorKeyEnabled
	copy(data[5:8], b.Reserved[:])
	copy(data[8:56], b.LD[:])
	copy(data[56:72], b.FamilyID[:])
	copy(data[72:88], b.ImageID[:])
	binary.LittleEndian.PutUint32(data[88:92], b.Version)
	binary.LittleEndian.PutUint32(data[92:96], b.GuestSVN)
	binary.LittleEndian.PutUint32(data[96:100], b.IDKeyAlgorithm)
	binary.LittleEndian.PutUint32(data[100:104], b.AuthorKeyAlgorithm)
	idKeySig, err := b.IDKeySignature.BinaryMarshal()
	if err != nil {
		return nil, fmt.Errorf("marshaling ID key signature: %w", err)
	}
	copy(data[104:248], idKeySig)
	idKeyPub, err := b.IDPublicKey.BinaryMarshal()
	if err != nil {
		return nil, fmt.Errorf("marshaling ID public key: %w", err)
	}
	copy(data[248:400], idKeyPub)
	authKeySig, err := b.AuthorKeySignature.BinaryMarshal()
	if err != nil {
		return nil, fmt.Errorf("marshaling author key signature: %w", err)
	}
	copy(data[400:544], authKeySig)
	authKeyPub, err := b.AuthorPublicKey.BinaryMarshal()
	if err != nil {
		return nil, fmt.Errorf("marshaling author public key: %w", err)
	}
	copy(data[544:696], authKeyPub)
	return data, nil
}

// BinaryUnmarshal unmarshals the SNP ID block from a byte slice.
func (b *VhsSnpIDBlock) BinaryUnmarshal(data []byte) error {
	if len(data) != 696 {
		return fmt.Errorf("expected 696 bytes, but got %d", len(data))
	}
	b.CompatibilityMask = binary.LittleEndian.Uint32(data[0:4])
	b.AuthorKeyEnabled = data[4]
	copy(b.Reserved[:], data[5:8])
	copy(b.LD[:], data[8:56])
	copy(b.FamilyID[:], data[56:72])
	copy(b.ImageID[:], data[72:88])
	b.Version = binary.LittleEndian.Uint32(data[88:92])
	b.GuestSVN = binary.LittleEndian.Uint32(data[92:96])
	b.IDKeyAlgorithm = binary.LittleEndian.Uint32(data[96:100])
	b.AuthorKeyAlgorithm = binary.LittleEndian.Uint32(data[100:104])
	if err := b.IDKeySignature.BinaryUnmarshal(data[104:248]); err != nil {
		return fmt.Errorf("unmarshaling ID key signature: %w", err)
	}
	if err := b.IDPublicKey.BinaryUnmarshal(data[248:400]); err != nil {
		return fmt.Errorf("unmarshaling ID public key: %w", err)
	}
	if err := b.AuthorKeySignature.BinaryUnmarshal(data[400:544]); err != nil {
		return fmt.Errorf("unmarshaling author key signature: %w", err)
	}
	if err := b.AuthorPublicKey.BinaryUnmarshal(data[544:696]); err != nil {
		return fmt.Errorf("unmarshaling author public key: %w", err)
	}
	return nil
}

// // VhsVbsMeasurement describes a VBS measurement to be used with Hyper-V's VBS
// // isolation architecture.
// //
// // TODO: doc fields.
// type VhsVbsMeasurement struct {
// 	CompatibilityMask     uint32
// 	Version               uint32
// 	ProductID             uint32
// 	ModuleID              uint32
// 	SecurityVersion       uint32
// 	PolicyFlags           uint32
// 	BootDigestAlgo        uint32
// 	SigningAlgo           uint32
// 	BootMeasurementDigest [64]byte
// 	Signature             [256]byte
// 	PublicKey             [512]byte
// }

// // MemoryMapEntryType is the type of memory described by a memory map entry or device tree node.
// type MemoryMapEntryType uint16

// const (
// 	// MemoryMapEntryTypeMemory describes a normal memory region.
// 	MemoryMapEntryTypeMemory MemoryMapEntryType = 0x0
// 	// MemoryMapEntryTypePlatformReserved is platform reserved memory.
// 	MemoryMapEntryTypePlatformReserved MemoryMapEntryType = 0x1
// 	// MemoryMapEntryTypePersistent is persistent memory (PMEM).
// 	MemoryMapEntryTypePersistent MemoryMapEntryType = 0x2
// 	// MemoryMapEntryTypeVTL2Protectable is memory where VTL2 protections that deny access to lower VTLs can be applied.
// 	// Some isolation architectures only allow VTL2 protections on certain memory ranges.
// 	MemoryMapEntryTypeVTL2Protectable MemoryMapEntryType = 0x3
// 	// MemoryMapEntryTypeSpecificPurpose is specific Purpose memory (SPM). This is memory with special properties
// 	// reserved for specific purposes and shouldn't be used by the firmware
// 	// or operating system. This corresponds with the UEFI memory map entry
// 	// flag EFI_MEMORY_SP, introduced in UEFI 2.8.
// 	// See https://uefi.org/specs/UEFI/2.10/07_Services_Boot_Services.html
// 	MemoryMapEntryTypeSpecificPurpose MemoryMapEntryType = 0x4 // Unstable feature
// 	// MemoryMapEntryTypeHidden memory is visible in the memory map but is hidden from any other
// 	// enumeration that may be used to expose available memory to the VM.
// 	MemoryMapEntryTypeHidden MemoryMapEntryType = 0x5 // Unstable feature
// )

// func (m MemoryMapEntryType) String() string {
// 	switch m {
// 	case MemoryMapEntryTypeMemory:
// 		return "MEMORY"
// 	case MemoryMapEntryTypePlatformReserved:
// 		return "PLATFORMRESERVED"
// 	case MemoryMapEntryTypePersistent:
// 		return "PERSISTENT"
// 	case MemoryMapEntryTypeVTL2Protectable:
// 		return "VTL2PROTECTABLE"
// 	case MemoryMapEntryTypeSpecificPurpose:
// 		return "SPECIFICPURPOSE"
// 	case MemoryMapEntryTypeHidden:
// 		return "HIDDEN"
// 	default:
// 		return "UNKNOWN"
// 	}
// }

// // VhsMemoryMapEntry is deposited by the loader for memory map entries for
// // [`IgvmVariableHeaderType::IGVM_VHT_MEMORY_MAP`] that describe memory
// // available to the guest.
// //
// // A well-behaved loader will report these in sorted order, with a final entry
// // with `number_of_pages` with zero signifying the last entry.
// type VhsMemoryMapEntry struct {
// 	// StartingGPAPageNumber is the starting gpa page number for this range of memory.
// 	StartingGPAPageNumber uint64
// 	// NumberOfPages is the number of pages in this range of memory.
// 	NumberOfPages uint64
// 	// EntryType is the type of memory this entry represents.
// 	EntryType MemoryMapEntryType
// 	// Flags about this memory entry.
// 	Flags uint16
// 	// Reserved.
// 	Reserved uint32
// }

// // VbsDigestAlgorithm represents the signature algorithm for VBS digest.
// type VbsDigestAlgorithm uint32

// const (
// 	// VbsDigestAlgorithmInvalid is an invalid digest algorithm.
// 	VbsDigestAlgorithmInvalid VbsDigestAlgorithm = 0x0
// 	// VbsDigestAlgorithmSha256 is the SHA-256 digest algorithm.
// 	VbsDigestAlgorithmSha256 VbsDigestAlgorithm = 0x1
// )

// // VbsSigningAlgorithm represents the signature algorithm for VBS measurement.
// type VbsSigningAlgorithm uint32

// const (
// 	// VbsSigningAlgorithmInvalidSig is an invalid signature algorithm.
// 	VbsSigningAlgorithmInvalidSig VbsSigningAlgorithm = 0x0
// 	// VbsSigningAlgorithmEcdsaP384 is the ECDSA signature algorithm with P-384 curve.
// 	VbsSigningAlgorithmEcdsaP384 VbsSigningAlgorithm = 0x1
// )
