// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package tdvf

import (
	"encoding/binary"
	"testing"
)

// buildTDVF assembles a minimal TDVF image whose section table holds the given
// sections. The layout mirrors what parseTdvfSections walks: a data region,
// then the TDVF metadata, then a GUIDed table pointing at it, then the footer.
// It is used to reach the per-section range validation with otherwise
// well-formed firmware.
func buildTDVF(sections []tdvfSection) []byte {
	metadata := make([]byte, 16+len(sections)*32)
	copy(metadata[0:4], metadataSignature)
	binary.LittleEndian.PutUint32(metadata[4:8], uint32(len(metadata)))
	binary.LittleEndian.PutUint32(metadata[8:12], 1)
	binary.LittleEndian.PutUint32(metadata[12:16], uint32(len(sections)))
	for i, s := range sections {
		e := metadata[16+i*32:]
		binary.LittleEndian.PutUint32(e[0:4], s.DataOffset)
		binary.LittleEndian.PutUint32(e[4:8], s.RawDataSize)
		binary.LittleEndian.PutUint64(e[8:16], s.MemoryAddress)
		binary.LittleEndian.PutUint64(e[16:24], s.MemoryDataSize)
		binary.LittleEndian.PutUint32(e[24:28], uint32(s.Type))
		binary.LittleEndian.PutUint32(e[28:32], uint32(s.Attributes))
	}

	// One GUIDed table entry, read backwards as [offset(4)][length(2)][guid(16)],
	// carrying the metadata offset. It is followed by the footer entry
	// [table length(2)][footer guid(16)] and 32 trailing bytes, so the footer
	// GUID sits 48 bytes from the end.
	const entryLen = 22
	const footerEntryLen = 18
	// data region large enough that any in-range section fits.
	dataRegion := make([]byte, 512)
	image := append([]byte{}, dataRegion...)
	metadataStart := len(image)
	image = append(image, metadata...)

	tableStart := len(image)
	image = append(image, make([]byte, entryLen+footerEntryLen+32)...)

	// metadata offset is measured from the end of the image.
	metadataOffset := uint32(len(image) - metadataStart)
	binary.LittleEndian.PutUint32(image[tableStart:], metadataOffset)
	binary.LittleEndian.PutUint16(image[tableStart+4:], entryLen)
	copy(image[tableStart+6:], tdxMetadataOffsetGUID)

	// footer: table length covers both entries, then the footer GUID.
	binary.LittleEndian.PutUint16(image[len(image)-50:], entryLen+footerEntryLen)
	copy(image[len(image)-48:], tableFooterGUID)
	return image
}

// TestParseTdvfSectionsMalformed ensures the TDVF parser rejects malformed or
// short firmware with an error instead of panicking on an out-of-range slice.
// Both exported entry points (FindCfv and CalculateMrTd) drive the parser, and
// a silent panic here would crash the build-time RTMR computation.
func TestParseTdvfSectionsMalformed(t *testing.T) {
	// Valid footer GUID but a table length running past the start of the buffer.
	inconsistentTableLength := make([]byte, 64)
	copy(inconsistentTableLength[len(inconsistentTableLength)-48:], tableFooterGUID)
	binary.LittleEndian.PutUint16(inconsistentTableLength[len(inconsistentTableLength)-50:], 0xffff)

	// Well-formed through the section table, but the section's data range
	// [DataOffset, DataOffset+RawDataSize) overflows the firmware buffer.
	badSectionRange := buildTDVF([]tdvfSection{
		{DataOffset: 0, RawDataSize: 0xffffffff},
	})

	testCases := map[string][]byte{
		"empty":                               nil,
		"sub-48-byte":                         make([]byte, 32),
		"footer present, inconsistent length": inconsistentTableLength,
		"section data range exceeds firmware": badSectionRange,
	}

	for name, firmware := range testCases {
		t.Run(name, func(t *testing.T) {
			if _, err := FindCfv(firmware); err == nil {
				t.Errorf("FindCfv(%s) = (_, nil), want an error", name)
			}
			if _, err := CalculateMrTd(firmware, ""); err == nil {
				t.Errorf("CalculateMrTd(%s) = (_, nil), want an error", name)
			}
		})
	}
}
