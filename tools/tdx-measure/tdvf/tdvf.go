// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// This package implements parsing of firmware files following the Intel® TDX
// Virtual Firmware Design Guide.
package tdvf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/edgelesssys/contrast/tdx-measure/mrtd"
)

var (
	tableFooterGUID       = []byte{0xde, 0x82, 0xb5, 0x96, 0xb2, 0x1f, 0xf7, 0x45, 0xba, 0xea, 0xa3, 0x66, 0xc5, 0x5a, 0x08, 0x2d}
	tdxMetadataOffsetGUID = []byte{0x35, 0x65, 0x7a, 0xe4, 0x4a, 0x98, 0x98, 0x47, 0x86, 0x5e, 0x46, 0x85, 0xa7, 0xbf, 0x8e, 0xc2}
	metadataSignature     = "TDVF"
)

// Find the offset for the TDVF metadata as described in Intel® TDX Virtual
// Firmware Design Guide, 11.1 TDVF Metadata Location.
func findTdvfMetadataOffset(firmware []byte) (uint32, error) {
	footerGUID := firmware[len(firmware)-48:][:16]
	if !bytes.Equal(footerGUID, tableFooterGUID) {
		return 0, errors.New("can't find table footer GUID")
	}

	offset := len(firmware) - 50
	tableLength := binary.LittleEndian.Uint16(firmware[offset:][:2])
	endOffset := len(firmware) - 32 - int(tableLength)

	for endOffset < offset {
		entryBlockGUID := firmware[offset-16:][:16]
		entryBlockLength := binary.LittleEndian.Uint16(firmware[offset-18:][:2])

		if bytes.Equal(entryBlockGUID, tdxMetadataOffsetGUID) {
			tdxOffset := binary.LittleEndian.Uint32(firmware[offset-22:][:4])
			return tdxOffset, nil
		}

		offset -= int(entryBlockLength)
	}

	return 0, errors.New("can't find TDX metadata offset block entry")
}

type tdvfSection struct {
	DataOffset     uint32
	RawDataSize    uint32
	MemoryAddress  uint64
	MemoryDataSize uint64
	Type           tdvfSectionType
	Attributes     tdvfSectionAttributes
}

type tdvfSectionType uint32

const cfv tdvfSectionType = 1

type tdvfSectionAttributes uint32

const (
	mrExtend tdvfSectionAttributes = 1 << iota
	pageAug
)

// Parse the TDVF descriptor described in Intel® TDX Virtual Firmware Design
// Guide, 11.2 TDVF descriptor.
func parseTdvfSections(firmware []byte) ([]tdvfSection, error) {
	offset, err := findTdvfMetadataOffset(firmware)
	if err != nil {
		return nil, fmt.Errorf("can't locate TDX firmware metadata offset: %w", err)
	}

	metadata := firmware[len(firmware)-int(offset):]
	metadataHeader := metadata[:16]

	if string(metadataHeader[:4]) != metadataSignature {
		return nil, errors.New("unexpected signature")
	}

	// We don't need this, we only consider NumberOfSectionEntries.
	length := binary.LittleEndian.Uint32(metadataHeader[4:][:4])
	_ = length

	version := binary.LittleEndian.Uint32(metadataHeader[8:][:4])
	if version != 1 {
		return nil, fmt.Errorf("expected version 1, got %v", version)
	}

	numberOfSectionEntries := binary.LittleEndian.Uint32(metadataHeader[12:][:4])
	sections := make([]tdvfSection, numberOfSectionEntries)
	sectionsData := metadata[16:]
	for i := range numberOfSectionEntries {
		sectionData := sectionsData[i*32:][:32]
		section := &sections[i]
		section.DataOffset = binary.LittleEndian.Uint32(sectionData[0:][:4])
		section.RawDataSize = binary.LittleEndian.Uint32(sectionData[4:][:4])
		section.MemoryAddress = binary.LittleEndian.Uint64(sectionData[8:][:8])
		section.MemoryDataSize = binary.LittleEndian.Uint64(sectionData[16:][:8])
		section.Type = tdvfSectionType(binary.LittleEndian.Uint32(sectionData[24:][:4]))
		section.Attributes = tdvfSectionAttributes(binary.LittleEndian.Uint32(sectionData[28:][:4]))
	}

	return sections, nil
}

// CalculateMrTd calculates the MRTD for a TDVF-conformant firmware as
// described in Intel® TDX Virtual Firmware Design Guide, 11.2 TDVF descriptor.
func CalculateMrTd(firmware []byte) ([48]byte, error) {
	launchContext := mrtd.NewLaunchContext()

	sections, err := parseTdvfSections(firmware)
	if err != nil {
		return [48]byte{}, fmt.Errorf("can't parse TDVF sections: %w", err)
	}

	for _, section := range sections {
		err := launchContext.AddRegion(section.MemoryAddress, section.MemoryDataSize)
		if err != nil {
			return [48]byte{}, fmt.Errorf("can't add region: %w", err)
		}

		if section.Attributes&mrExtend != 0 {
			data := firmware[section.DataOffset:][:section.RawDataSize]
			err = launchContext.ExtendRegion(section.MemoryAddress, data)
			if err != nil {
				return [48]byte{}, fmt.Errorf("can't extend region: %w", err)
			}
		}
	}

	digest := launchContext.Finalize()
	return digest, nil
}

// FindCfv looks up the CFV (Configuration Firmware Volume) in the firmware file.
func FindCfv(firmware []byte) ([]byte, error) {
	sections, err := parseTdvfSections(firmware)
	if err != nil {
		return nil, fmt.Errorf("can't parse TDVF sections: %w", err)
	}

	for _, section := range sections {
		if section.Type == cfv {
			data := firmware[section.DataOffset:][:section.RawDataSize]
			return data, nil
		}
	}

	return nil, errors.New("can't find CFV section")
}
