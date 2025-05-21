// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

/*
This library provides structs and methods for the Independent Guest Virtual
Machine (IGVM) file format.

The IGVM format is [specified as rust crate], this library is based on
the 0.3.4 version of the specification. From the specification:

The IGVM file format is designed to encapsulate all information required to
launch a virtual machine on any given virtualization stack, with support for
different isolation technologies such as AMD SEV-SNP and Intel TDX.

At a conceptual level, this file format is a set of commands created by the
tool that generated the file, used by the loader to construct the initial
guest state. The file format also contains measurement information that the
underlying platform will use to confirm that the file was loaded correctly
and signed by the appropriate authorities.

[specified as rust crate]: https://github.com/microsoft/igvm/blob/igvm_defs-v0.3.4/igvm_defs/src/lib.rs
*/
package igvm

import (
	"fmt"
	"hash/crc32"
)

// IGVM is an Independent Guest Virtual Machine (IGVM) image.
type IGVM struct {
	Header          FixedHeader
	VariableHeaders []VariableHeader
	FileData        []byte
}

// UpdateChecksum updates the checksum of the IGVM struct.
func (i *IGVM) UpdateChecksum() error {
	i.Header.Checksum = 0
	data, err := i.marshalBinaryHeaders()
	if err != nil {
		return fmt.Errorf("marshaling headers to binary: %w", err)
	}
	i.Header.Checksum = crc32.ChecksumIEEE(data)
	return nil
}

// MarshalBinary marshals the IGVM struct into a byte slice.
func (i *IGVM) MarshalBinary() ([]byte, error) {
	headers, err := i.marshalBinaryHeaders()
	if err != nil {
		return nil, err
	}
	// File data is not included in the checksum calculation.
	return append(headers, i.FileData...), nil
}

func (i *IGVM) marshalBinaryHeaders() ([]byte, error) {
	data, err := i.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}
	for _, vhs := range i.VariableHeaders {
		vhsData, err := vhs.MarshalBinary()
		if err != nil {
			return nil, err
		}
		data = append(data, vhsData...)
	}
	return data, nil
}

// UnmarshalBinary unmarshals the byte slice into the IGVM struct.
func (i *IGVM) UnmarshalBinary(data []byte) error {
	if err := i.Header.UnmarshalBinary(data[:24]); err != nil {
		return err
	}
	index := i.Header.VariableHeaderOffset
	for index < i.Header.VariableHeaderOffset+i.Header.VariableHeaderSize {
		var vhs VariableHeader
		if err := vhs.UnmarshalBinary(data[index:]); err != nil {
			return err
		}
		i.VariableHeaders = append(i.VariableHeaders, vhs)
		index += 8 + vhs.Length + uint32(len(vhs.Padding))
	}
	i.FileData = data[index:]
	return nil
}
