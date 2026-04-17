// Translated from the Python sev-snp-measure tool:
// Copyright 2022- IBM Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0
// Source: https://github.com/virtee/sev-snp-measure/blob/46664d0347fb07c5ac2cb8ab5bf5aebc09fc67ab/sevsnpmeasure/ovmf.py

package snp

import (
	"encoding/binary"
	"fmt"
	"os"
)

const fourGB = uint64(0x100000000)

// SectionType represents the type of an OVMF SEV metadata section.
type SectionType uint32

// Section type constant definitions.
const (
	SectionTypeSNPSecMem     SectionType = 1
	SectionTypeSNPSecrets    SectionType = 2
	SectionTypeCPUID         SectionType = 3
	SectionTypeSVSMCAA       SectionType = 4
	SectionTypeSNPKernelHash SectionType = 0x10
)

// MetadataSection describes one entry from the OVMF SEV metadata.
type MetadataSection struct {
	GPA  uint32
	Size uint32
	Type SectionType
}

// OVMF holds the parsed OVMF firmware image.
type OVMF struct {
	data          []byte
	gpa           uint64
	table         map[string][]byte
	metadataItems []MetadataSection
}

// Well-known GUIDs appearing in the OVMF footer table (as ASCII UUID strings).
const (
	ovmfTableFooterGUID = "96b582de-1fb2-45f7-baea-a366c55a082d"
	sevHashTableRVGUID  = "7255371f-3a3b-4b04-927b-1da6efa8d454"
	sevESResetBlockGUID = "00f771de-1a7e-4fcb-890e-68c77e2fb44e"
	ovmfSEVMetaDataGUID = "dc886566-984a-4798-a75e-5585a7bf67cc"
)

// NewOVMF reads and parses an OVMF firmware file.
func NewOVMF(filename string) (*OVMF, error) {
	return newOVMF(filename, fourGB)
}

func newOVMF(filename string, endAt uint64) (*OVMF, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading OVMF file: %w", err)
	}
	o := &OVMF{
		data:  data,
		table: make(map[string][]byte),
	}
	if err := o.parseFooterTable(); err != nil {
		return nil, err
	}
	if err := o.parseSEVMetadata(); err != nil {
		return nil, err
	}
	o.gpa = endAt - uint64(len(data))
	return o, nil
}

// Data returns the raw OVMF binary.
func (o *OVMF) Data() []byte { return o.data }

// GPA returns the base guest physical address of the OVMF image.
func (o *OVMF) GPA() uint64 { return o.gpa }

// MetadataItems returns the parsed SEV metadata section descriptors.
func (o *OVMF) MetadataItems() []MetadataSection { return o.metadataItems }

// HasMetadataSection reports whether a section of the given type exists.
func (o *OVMF) HasMetadataSection(t SectionType) bool {
	for _, s := range o.metadataItems {
		if s.Type == t {
			return true
		}
	}
	return false
}

// SEVHashesTableGPA returns the GPA of the SEV hashes table, or an error.
func (o *OVMF) SEVHashesTableGPA() (uint32, error) {
	entry, ok := o.table[sevHashTableRVGUID]
	if !ok {
		return 0, fmt.Errorf("SEV_HASH_TABLE_RV_GUID not found in OVMF table")
	}
	if len(entry) < 4 {
		return 0, fmt.Errorf("SEV_HASH_TABLE_RV_GUID entry too short")
	}
	return binary.LittleEndian.Uint32(entry[:4]), nil
}

// IsSEVHashesTableSupported reports whether the OVMF supports kernel/initrd/cmdline measurement.
func (o *OVMF) IsSEVHashesTableSupported() bool {
	gpa, err := o.SEVHashesTableGPA()
	return err == nil && gpa != 0
}

// SEVESResetEIP returns the SEV-ES reset EIP from the OVMF footer table.
func (o *OVMF) SEVESResetEIP() (uint32, error) {
	entry, ok := o.table[sevESResetBlockGUID]
	if !ok {
		return 0, fmt.Errorf("SEV_ES_RESET_BLOCK_GUID not found in OVMF table")
	}
	if len(entry) < 4 {
		return 0, fmt.Errorf("SEV_ES_RESET_BLOCK_GUID entry too short")
	}
	return binary.LittleEndian.Uint32(entry[:4]), nil
}

// parseFooterTable parses the OVMF footer GUID table.
// The table is located immediately before the last 32 bytes of the firmware.
// Each entry has: data bytes, then a 2-byte size, then a 16-byte GUID.
// The table is traversed from back to front.
func (o *OVMF) parseFooterTable() error {
	const entryHeaderSize = 18 // uint16 size + 16 byte GUID
	size := len(o.data)

	startOfFooterTable := size - 32 - entryHeaderSize
	if startOfFooterTable < 0 {
		return nil
	}

	footerEntry := o.data[startOfFooterTable:]
	footerSize := binary.LittleEndian.Uint16(footerEntry[0:2])
	footerGUID := guidString(footerEntry[2:18])

	if footerGUID != ovmfTableFooterGUID {
		return nil
	}

	tableSize := int(footerSize) - entryHeaderSize
	if tableSize < 0 {
		return nil
	}

	tableBytes := o.data[startOfFooterTable-tableSize : startOfFooterTable]

	for len(tableBytes) >= entryHeaderSize {
		tail := tableBytes[len(tableBytes)-entryHeaderSize:]
		entrySize := int(binary.LittleEndian.Uint16(tail[0:2]))
		if entrySize < entryHeaderSize {
			return fmt.Errorf("invalid OVMF footer table entry size %d", entrySize)
		}
		entryGUID := guidString(tail[2:18])
		dataStart := len(tableBytes) - entrySize
		if dataStart < 0 {
			return fmt.Errorf("OVMF footer table entry extends past table start")
		}
		entryData := tableBytes[dataStart : len(tableBytes)-entryHeaderSize]
		o.table[entryGUID] = entryData
		tableBytes = tableBytes[:len(tableBytes)-entrySize]
	}
	return nil
}

// parseSEVMetadata parses the OVMF SEV metadata section descriptors.
func (o *OVMF) parseSEVMetadata() error {
	entry, ok := o.table[ovmfSEVMetaDataGUID]
	if !ok {
		return nil
	}
	if len(entry) < 4 {
		return fmt.Errorf("OVMF_SEV_META_DATA_GUID entry too short")
	}
	offsetFromEnd := binary.LittleEndian.Uint32(entry[:4])
	start := len(o.data) - int(offsetFromEnd)
	if start < 0 || start+16 > len(o.data) {
		return fmt.Errorf("SEV metadata header out of range")
	}

	// OvmfSevMetadataHeader: signature(4) + size(4) + version(4) + num_items(4) = 16 bytes
	sig := o.data[start : start+4]
	if string(sig) != "ASEV" {
		return fmt.Errorf("wrong SEV metadata signature: %q", sig)
	}
	mdSize := binary.LittleEndian.Uint32(o.data[start+4 : start+8])
	version := binary.LittleEndian.Uint32(o.data[start+8 : start+12])
	numItems := binary.LittleEndian.Uint32(o.data[start+12 : start+16])
	if version != 1 {
		return fmt.Errorf("wrong SEV metadata version: %d", version)
	}

	const headerSize = 16
	const itemSize = 12 // gpa(4) + size(4) + section_type(4)
	itemsStart := start + headerSize
	itemsEnd := start + int(mdSize)
	if itemsEnd > len(o.data) || itemsEnd < itemsStart {
		return fmt.Errorf("SEV metadata items out of range")
	}
	items := o.data[itemsStart:itemsEnd]

	for i := range int(numItems) {
		off := i * itemSize
		if off+itemSize > len(items) {
			return fmt.Errorf("SEV metadata item %d out of range", i)
		}
		gpa := binary.LittleEndian.Uint32(items[off : off+4])
		sz := binary.LittleEndian.Uint32(items[off+4 : off+8])
		st := SectionType(binary.LittleEndian.Uint32(items[off+8 : off+12]))
		o.metadataItems = append(o.metadataItems, MetadataSection{GPA: gpa, Size: sz, Type: st})
	}
	return nil
}

// guidString converts 18 bytes (footer table entry: 2-byte size + 16-byte GUID in bytes_le)
// at offset 2 to a lowercase UUID string matching Python's str(uuid.UUID(bytes_le=...)).
// Input is the 16-byte bytes_le GUID.
func guidString(b []byte) string {
	// UUID bytes_le layout:
	// time_low (4 bytes, LE) | time_mid (2 bytes, LE) | time_hi (2 bytes, LE) | rest (8 bytes, BE)
	if len(b) < 16 {
		return ""
	}
	timeLow := binary.LittleEndian.Uint32(b[0:4])
	timeMid := binary.LittleEndian.Uint16(b[4:6])
	timeHi := binary.LittleEndian.Uint16(b[6:8])
	return fmt.Sprintf("%08x-%04x-%04x-%x-%x",
		timeLow, timeMid, timeHi,
		b[8:10], b[10:16],
	)
}
