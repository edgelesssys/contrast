// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package igvm

import (
	"encoding/binary"
	"encoding/json"
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

// MarshalBinary marshals the fixed header to a byte slice.
func (h *FixedHeader) MarshalBinary() ([]byte, error) {
	data := make([]byte, 24)
	binary.LittleEndian.PutUint32(data[0:4], h.Magic)
	binary.LittleEndian.PutUint32(data[4:8], h.FormatVersion)
	binary.LittleEndian.PutUint32(data[8:12], h.VariableHeaderOffset)
	binary.LittleEndian.PutUint32(data[12:16], h.VariableHeaderSize)
	binary.LittleEndian.PutUint32(data[16:20], h.TotalFileSize)
	binary.LittleEndian.PutUint32(data[20:24], h.Checksum)
	return data, nil
}

// UnmarshalBinary unmarshals the fixed header from a byte slice.
func (h *FixedHeader) UnmarshalBinary(data []byte) error {
	if len(data) < 24 {
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
	// VhtInvalid type.
	VhtInvalid VariableHeaderType = 0x0

	// VhtSupportedPlatform indicates a header of type [VhsSupportedPlatform].
	VhtSupportedPlatform VariableHeaderType = 0x1

	//
	// IGVM_VHT_RANGE_INIT structures.
	//

	// VhtGuestPolicy indicates a header of type [VhsGuestPolicy].
	VhtGuestPolicy VariableHeaderType = 0x101
	// VhtRelocatableRegion indicates a header of type [VhsRelocatableRegion].
	VhtRelocatableRegion VariableHeaderType = 0x102
	// VhtPageTableRelocationRegion indicates a header of type [VhsPageTableRelocationRegion].
	VhtPageTableRelocationRegion VariableHeaderType = 0x103

	//
	// IGVM_VHT_RANGE_DIRECTIVE structures.
	//

	// VhtParameterArea indicates a header of type [VhsParameterArea].
	VhtParameterArea VariableHeaderType = 0x301
	// VhtPageData indicates a header of type [VhsPageData].
	VhtPageData VariableHeaderType = 0x302
	// VhtParameterInsert indicates a header of type [VhsParameterInsert].
	VhtParameterInsert VariableHeaderType = 0x303
	// VhtVpContext indicates a header of type [VhsVpContext].
	VhtVpContext VariableHeaderType = 0x304
	// VhtRequiredMemory indicates a header of type [VhsRequiredMemory].
	VhtRequiredMemory VariableHeaderType = 0x305
	// ReservedDoNotUse MUST NOT be used.
	// It was previously used in earlier revisions as `IGVM_VHT_SHARED_BOUNDARY_GPA` but is now unused.
	ReservedDoNotUse VariableHeaderType = 0x306
	// VhtVpCountParameter indicates a header of type [VhsVpCountParameter].
	VhtVpCountParameter VariableHeaderType = 0x307
	// VhtSrat indicates a header of type [VhsSrat].
	VhtSrat VariableHeaderType = 0x308
	// VhtMadt indicates a header of type [VhsMadt].
	VhtMadt VariableHeaderType = 0x309
	// VhtMmioRanges indicates a header of type [VhsMmioRanges].
	VhtMmioRanges VariableHeaderType = 0x30A
	// VhtSnpIDBlock indicates a header of type [VhsSnpIDBlock].
	VhtSnpIDBlock VariableHeaderType = 0x30B
	// VhtMemoryMap indicates a header of type [VhsMemoryMap].
	VhtMemoryMap VariableHeaderType = 0x30C
	// VhtErrorRange indicates a header of type [VhsErrorRange].
	VhtErrorRange VariableHeaderType = 0x30D
	// VhtCommandLine indicates a header of type [VhsCommandLine].
	VhtCommandLine VariableHeaderType = 0x30E
	// VhtSlit indicates a header of type [VhsSlit].
	VhtSlit VariableHeaderType = 0x30F
	// VhtPptt indicates a header of type [VhsPptt].
	VhtPptt VariableHeaderType = 0x310
	// VhtVbsMeasurement indicates a header of type [VhsVbsMeasurement].
	VhtVbsMeasurement VariableHeaderType = 0x311
	// VhtDeviceTree indicates a header of type [VhsDeviceTree].
	VhtDeviceTree VariableHeaderType = 0x312
	// VhtEnvironmentInfoParameter indicates a header of type [VhsEnvironmentInfoParameter].
	VhtEnvironmentInfoParameter VariableHeaderType = 0x313
)

// MarshalJSON marshals the variable header type to JSON.
func (t VariableHeaderType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, t.String())), nil
}

// UnmarshalJSON unmarshals the variable header type from JSON.
func (t *VariableHeaderType) UnmarshalJSON(data []byte) error {
	s := string(data)
	if len(s) < 2 {
		return fmt.Errorf("invalid variable header type %q", s)
	}
	if s[0] != '"' || s[len(s)-1] != '"' {
		return fmt.Errorf("invalid variable header type %q", s)
	}
	v, err := VariableHeaderTypeFromString(s[1 : len(s)-1])
	if err != nil {
		return fmt.Errorf("unmarshaling variable header type: %w", err)
	}
	*t = v
	return nil
}

// String method for human-readable output.
func (t VariableHeaderType) String() string {
	switch t {
	case VhtInvalid:
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
	case VhtSnpIDBlock:
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

// VariableHeaderTypeFromString converts a string to a VariableHeaderType.
func VariableHeaderTypeFromString(s string) (VariableHeaderType, error) {
	switch s {
	case "VhtSupportedPlatform":
		return VhtSupportedPlatform, nil
	case "VhtGuestPolicy":
		return VhtGuestPolicy, nil
	case "VhtRelocatableRegion":
		return VhtRelocatableRegion, nil
	case "VhtPageTableRelocationRegion":
		return VhtPageTableRelocationRegion, nil
	case "VhtParameterArea":
		return VhtParameterArea, nil
	case "VhtPageData":
		return VhtPageData, nil
	case "VhtParameterInsert":
		return VhtParameterInsert, nil
	case "VhtVpContext":
		return VhtVpContext, nil
	case "VhtRequiredMemory":
		return VhtRequiredMemory, nil
	case "ReservedDoNotUse":
		return ReservedDoNotUse, nil
	case "VhtVpCountParameter":
		return VhtVpCountParameter, nil
	case "VhtSrat":
		return VhtSrat, nil
	case "VhtMadt":
		return VhtMadt, nil
	case "VhtMmioRanges":
		return VhtMmioRanges, nil
	case "VhtSnpIdBlock":
		return VhtSnpIDBlock, nil
	case "VhtMemoryMap":
		return VhtMemoryMap, nil
	case "VhtErrorRange":
		return VhtErrorRange, nil
	case "VhtCommandLine":
		return VhtCommandLine, nil
	case "VhtSlit":
		return VhtSlit, nil
	case "VhtPptt":
		return VhtPptt, nil
	case "VhtVbsMeasurement":
		return VhtVbsMeasurement, nil
	case "VhtDeviceTree":
		return VhtDeviceTree, nil
	case "VhtEnvironmentInfoParameter":
		return VhtEnvironmentInfoParameter, nil
	default:
		return VhtInvalid, fmt.Errorf("unknown variable header type %q", s)
	}
}

// VariableHeader represents a variable header in IGVM.
type VariableHeader struct {
	Type    VariableHeaderType
	Length  uint32
	Content []byte
	Padding []byte // Unmarshal only.
}

// MarshalBinary marshals the variable header to a byte slice.
func (h *VariableHeader) MarshalBinary() ([]byte, error) {
	paddingLen := paddingForAlignment(h.Length)
	data := make([]byte, 8+int(h.Length)+int(paddingLen))
	binary.LittleEndian.PutUint32(data[0:4], uint32(h.Type))
	binary.LittleEndian.PutUint32(data[4:8], h.Length)
	if h.Length == 0 {
		return data, nil
	}
	copy(data[8:8+h.Length], h.Content)
	return data, nil
}

// UnmarshalBinary unmarshals the variable header from a byte slice.
func (h *VariableHeader) UnmarshalBinary(data []byte) error {
	h.Type = VariableHeaderType(binary.LittleEndian.Uint32(data[0:4]))
	h.Length = binary.LittleEndian.Uint32(data[4:8])
	if h.Length == 0 {
		return nil
	}
	h.Content = data[8 : 8+h.Length]
	paddingLen := paddingForAlignment(h.Length)
	if paddingLen > 0 {
		h.Padding = data[8+h.Length : 8+h.Length+paddingLen]
	}
	return nil
}

// TypedContent unmarshals the content based on the variable header type.
// It returns the marshaled struct, that can then be cast to the concrete type.
func (h *VariableHeader) TypedContent() (any, error) {
	switch h.Type {
	case VhtSnpIDBlock:
		var content VhsSnpIDBlock
		if err := content.UnmarshalBinary(h.Content); err != nil {
			return nil, fmt.Errorf("unmarshaling SnpIDBlock: %w", err)
		}
		return content, nil
	case VhtSupportedPlatform:
		var content VhsSupportedPlatform
		if err := content.UnmarshalBinary(h.Content); err != nil {
			return nil, fmt.Errorf("unmarshaling SupportedPlatfrom: %w", err)
		}
		return content, nil
	default:
		return nil, fmt.Errorf("unknown variable header type %q", h.Type)
	}
}

// MarshalJSON marshals the variable header to JSON.
func (h *VariableHeader) MarshalJSON() ([]byte, error) {
	typedContent, err := h.TypedContent()
	if err != nil {
		return json.Marshal(struct {
			Type    VariableHeaderType `json:"Type"`
			Length  uint32             `json:"Length"`
			Content []byte             `json:"Content"`
		}{
			Type:    h.Type,
			Length:  h.Length,
			Content: h.Content,
		})
	}
	return json.Marshal(struct {
		Type    VariableHeaderType `json:"Type"`
		Length  uint32             `json:"Length"`
		Content any                `json:"Content"`
	}{
		Type:    h.Type,
		Length:  h.Length,
		Content: typedContent,
	})
}

// PlatformType identifies an isolation platform.
type PlatformType uint8

const (
	// PlatformTypeNative is a native platform without any isolation.
	PlatformTypeNative PlatformType = 0x00
	// PlatformTypeVMSIsolation is a platform that supports Hyper-V's VMS isolation.
	PlatformTypeVMSIsolation PlatformType = 0x01
	// PlatformTypeSEVSNP is a platform that supports AMD SEV-SNP.
	PlatformTypeSEVSNP PlatformType = 0x02
	// PlatformTypeTDX is a platform that supports Intel TDX.
	PlatformTypeTDX PlatformType = 0x03
	// PlatformTypeSEV is a platform that supports AMD SEV. This is unstable.
	PlatformTypeSEV PlatformType = 0x04 // unstable
	// PlatformTypeSEVES is a platform that supports AMD SEV-S. This is unstable.
	PlatformTypeSEVES PlatformType = 0x05 // unstable
)

// PlatformVersion is the version of each PlatformType that is supported.
type PlatformVersion uint16

const (
	// PlatformVersionNative is the version of PlatformTypeNative.
	PlatformVersionNative uint16 = 0x1
	// PlatformVersionVMSIsolation is the version of PlatformTypeVMSIsolation.
	PlatformVersionVMSIsolation uint16 = 0x1
	// PlatformVersionSEVSNP is the version of PlatformTypeSEVSNP.
	PlatformVersionSEVSNP uint16 = 0x1
	// PlatformVersionTDX is the version of PlatformTypeTDX.
	PlatformVersionTDX uint16 = 0x1
	// PlatformVersionSEV is the version of PlatformTypeSEV. This is unstable.
	PlatformVersionSEV uint16 = 0x1 // unstable
	// PlatformVersionSEVES is the version of PlatformTypeSEVES. This is unstable.
	PlatformVersionSEVES uint16 = 0x1 // unstable

)

// VhsSupportedPlatform describes which isolation platforms are compatible with this guest image.
// A separate header is required for each supported platform.
//
// The header must appear prior to any other structures that refer to the compatibility mask this
// header defines.
type VhsSupportedPlatform struct {
	// CompatibilityMask is a bitmask that is used in following variable header structures that
	// correspond with this platform. Headers that have this corresponding bit
	// set indicates that it should be loaded if loading this specified
	// platform.
	//
	// This must have only one bit set.
	CompatibilityMask uint32
	// HighestVTL is the VTL that will be the highest VTL activated for the guest. On platforms
	// that do not support multiple VTLs, this value must be zero.
	HighestVTL uint8
	// PlatformType is the platform that is supported.
	PlatformType PlatformType
	// PlatformVersion is the version that is supported.
	PlatformVersion PlatformVersion
	// SharedGPABoundary describes the GPA at which memory above the boundary will be
	// host visible. A value of 0 indicates that this field is ignored, and the
	// platform described will manage shared memory in an enlightened manner.
	SharedGPABoundary uint64
}

// MarshalBinary marshals the supported platform to a byte slice.
func (p *VhsSupportedPlatform) MarshalBinary() ([]byte, error) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint32(data[0:4], p.CompatibilityMask)
	data[5] = p.HighestVTL
	data[6] = uint8(p.PlatformType)
	binary.LittleEndian.PutUint16(data[6:8], uint16(p.PlatformVersion))
	binary.LittleEndian.PutUint64(data[8:16], p.SharedGPABoundary)
	return data, nil
}

// UnmarshalBinary unmarshals the supported platform from a byte slice.
func (p *VhsSupportedPlatform) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("expected 16 byte but got %d", len(data))
	}
	p.CompatibilityMask = binary.LittleEndian.Uint32(data[0:4])
	p.HighestVTL = data[5]
	p.PlatformType = PlatformType(data[6])
	p.PlatformVersion = PlatformVersion(binary.LittleEndian.Uint16(data[6:8]))
	p.SharedGPABoundary = binary.BigEndian.Uint64(data[8:16])
	return nil
}

// VhsGuestPolicy describes the isolation architecture dependent guest policy.
type VhsGuestPolicy struct {
	// Policy is the isolation architecture dependent policy.
	Policy uint64
	// CompatibilityMask. See CompatibilityMask on [VhsSupportedPlatform].
	CompatibilityMask uint32
	// Reserved, must be zero.
	Reserved uint32
}

// VhsSnpPolicy describes the AMD SEV-SNP policy guest policy,
// as described in Section 4.3 Guest Policy of the AMD SEV-SNP ABI specification.
type VhsSnpPolicy struct {
	// ABIMajor is the minimum SEV SNP ABI version needed to run the guest's minor version number.
	ABIMinor uint8
	// ABIMajor is the minimum SEV SNP ABI version needed to run the guest's major version number.
	ABIMajor uint8
	// SMT is true if symmetric multithreading is allowed.
	SMT bool
	// MigrateMA is true if the guest is allowed to have a migration agent.
	MigrateMA bool
	// Debug is true if the VM can be decrypted by the host for debugging purposes.
	Debug bool
	// SingleSocket is true if the guest may only be active on a single socket.
	SingleSocket bool
}

// VhsSnpIDBlockSignature represents the signature for the SNP ID block. See the corresponding PSP definitions.
type VhsSnpIDBlockSignature struct {
	RComp [72]uint8
	SComp [72]uint8
}

// MarshalBinary marshals the SNP ID block signature to a byte slice.
func (b *VhsSnpIDBlockSignature) MarshalBinary() ([]byte, error) {
	data := make([]byte, 144)
	copy(data[0:72], b.RComp[:])
	copy(data[72:144], b.SComp[:])
	return data, nil
}

// UnmarshalBinary unmarshals the SNP ID block signature from a byte slice.
func (b *VhsSnpIDBlockSignature) UnmarshalBinary(data []byte) error {
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

// MarshalBinary marshals the SNP ID block public key to a byte slice.
func (b *VhsSnpIDBlockPublicKey) MarshalBinary() ([]byte, error) {
	data := make([]byte, 152)
	binary.LittleEndian.PutUint32(data[0:4], b.Curve)
	binary.LittleEndian.PutUint32(data[4:8], b.Reserved)
	copy(data[8:80], b.QX[:])
	copy(data[80:152], b.QY[:])
	return data, nil
}

// UnmarshalBinary unmarshals the SNP ID block public key from a byte slice.
func (b *VhsSnpIDBlockPublicKey) UnmarshalBinary(data []byte) error {
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

// MarshalBinary marshals the SNP ID block to a byte slice.
func (b *VhsSnpIDBlock) MarshalBinary() ([]byte, error) {
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
	idKeySig, err := b.IDKeySignature.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshaling ID key signature: %w", err)
	}
	copy(data[104:248], idKeySig)
	idKeyPub, err := b.IDPublicKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshaling ID public key: %w", err)
	}
	copy(data[248:400], idKeyPub)
	authKeySig, err := b.AuthorKeySignature.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshaling author key signature: %w", err)
	}
	copy(data[400:544], authKeySig)
	authKeyPub, err := b.AuthorPublicKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshaling author public key: %w", err)
	}
	copy(data[544:696], authKeyPub)
	return data, nil
}

// UnmarshalBinary unmarshals the SNP ID block from a byte slice.
func (b *VhsSnpIDBlock) UnmarshalBinary(data []byte) error {
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
	if err := b.IDKeySignature.UnmarshalBinary(data[104:248]); err != nil {
		return fmt.Errorf("unmarshaling ID key signature: %w", err)
	}
	if err := b.IDPublicKey.UnmarshalBinary(data[248:400]); err != nil {
		return fmt.Errorf("unmarshaling ID public key: %w", err)
	}
	if err := b.AuthorKeySignature.UnmarshalBinary(data[400:544]); err != nil {
		return fmt.Errorf("unmarshaling author key signature: %w", err)
	}
	if err := b.AuthorPublicKey.UnmarshalBinary(data[544:696]); err != nil {
		return fmt.Errorf("unmarshaling author public key: %w", err)
	}
	return nil
}

func paddingForAlignment(size uint32) uint32 {
	// From the spec:
	//  Each variable header structure must begin at a file offset that is a multiple of 8 bytes,
	//  so the length field of any structure must be rounded up to 8 bytes to find the
	//  type/length information of the following structure.
	return (8 - (size % 8)) % 8
}
