// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qtest

import (
	"encoding/binary"
	"testing"
)

func TestParseDirectory(t *testing.T) {
	raw := make([]byte, 4+2*fileDirectoryEntrySize)
	binary.BigEndian.PutUint32(raw[:4], 2)
	writeDirectoryEntry(raw[4:4+fileDirectoryEntrySize], "etc/table-loader", 0x20, 384)
	writeDirectoryEntry(raw[4+fileDirectoryEntrySize:], "etc/acpi/tables", 0x21, 131072)

	directory, err := parseDirectory(raw)
	if err != nil {
		t.Fatalf("parseDirectory: %v", err)
	}
	if got := directory["etc/table-loader"]; got != (fwConfigFile{selector: 0x20, size: 384}) {
		t.Fatalf("table-loader entry = %#v", got)
	}
	if got := directory["etc/acpi/tables"]; got != (fwConfigFile{selector: 0x21, size: 131072}) {
		t.Fatalf("tables entry = %#v", got)
	}
}

func TestParseDirectoryRejectsMalformedInput(t *testing.T) {
	tests := map[string][]byte{
		"short header": {0, 0, 0},
		"zero entries": {0, 0, 0, 0},
		"short entry":  {0, 0, 0, 1},
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDirectory(raw); err == nil {
				t.Fatal("parseDirectory returned nil error")
			}
		})
	}
}

func TestParseDirectoryRejectsDuplicateName(t *testing.T) {
	raw := make([]byte, 4+2*fileDirectoryEntrySize)
	binary.BigEndian.PutUint32(raw[:4], 2)
	writeDirectoryEntry(raw[4:4+fileDirectoryEntrySize], "duplicate", 1, 1)
	writeDirectoryEntry(raw[4+fileDirectoryEntrySize:], "duplicate", 2, 2)
	if _, err := parseDirectory(raw); err == nil {
		t.Fatal("parseDirectory returned nil error")
	}
}

func writeDirectoryEntry(entry []byte, name string, selector uint16, size uint32) {
	binary.BigEndian.PutUint32(entry[0:4], size)
	binary.BigEndian.PutUint16(entry[4:6], selector)
	copy(entry[8:64], name)
}
