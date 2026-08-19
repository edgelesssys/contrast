// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qtest

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/bits"
	"strings"
)

const (
	dmaAddressHighPort = 0x514
	dmaAddressLowPort  = 0x518
	fileDirectory      = 0x19
	dmaControlRead     = 0x02
	dmaControlSelect   = 0x08

	dataAddress   = 0x08000000
	accessAddress = 0x08100000
	maxBlobSize   = accessAddress - dataAddress

	fileDirectoryEntrySize  = 64
	maxFileDirectoryEntries = 4095
)

// FWConfig provides access to QEMU fw_cfg files through qtest DMA.
type FWConfig struct {
	client    *Client
	directory map[string]fwConfigFile
}

type fwConfigFile struct {
	selector uint16
	size     uint32
}

// OpenFWConfig reads the fw_cfg file directory.
func OpenFWConfig(ctx context.Context, client *Client) (*FWConfig, error) {
	directory, err := readDirectory(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("reading fw_cfg directory: %w", err)
	}
	return &FWConfig{client: client, directory: directory}, nil
}

// ReadFile returns the contents of the named fw_cfg file.
func (f *FWConfig) ReadFile(ctx context.Context, name string) ([]byte, error) {
	entry, exists := f.directory[name]
	if !exists {
		return nil, fmt.Errorf("fw_cfg entry %q not found", name)
	}
	if entry.size > maxBlobSize {
		return nil, fmt.Errorf("fw_cfg entry %q is %d bytes, exceeding DMA scratch space %d", name, entry.size, maxBlobSize)
	}
	data, err := dmaRead(ctx, f.client, entry.selector, entry.size)
	if err != nil {
		return nil, fmt.Errorf("reading fw_cfg entry %q: %w", name, err)
	}
	return data, nil
}

func dmaRead(ctx context.Context, client *Client, selector uint16, size uint32) ([]byte, error) {
	if size > maxBlobSize {
		return nil, fmt.Errorf("fw_cfg entry size %d exceeds DMA scratch space %d", size, maxBlobSize)
	}
	access := make([]byte, 16)
	control := uint32(selector)<<16 | dmaControlSelect | dmaControlRead
	binary.BigEndian.PutUint32(access[0:4], control)
	binary.BigEndian.PutUint32(access[4:8], size)
	binary.BigEndian.PutUint64(access[8:16], dataAddress)
	if err := client.writeMemory(ctx, accessAddress, access); err != nil {
		return nil, err
	}

	// The fw_cfg DMA address register is big-endian.
	high := bits.ReverseBytes32(uint32(accessAddress >> 32))
	low := bits.ReverseBytes32(uint32(accessAddress))
	if err := client.outL(ctx, dmaAddressHighPort, high); err != nil {
		return nil, err
	}
	if err := client.outL(ctx, dmaAddressLowPort, low); err != nil {
		return nil, err
	}
	return client.readMemory(ctx, dataAddress, size)
}

func readDirectory(ctx context.Context, client *Client) (map[string]fwConfigFile, error) {
	countData, err := dmaRead(ctx, client, fileDirectory, 4)
	if err != nil {
		return nil, err
	}
	count := binary.BigEndian.Uint32(countData)
	if count == 0 || count > maxFileDirectoryEntries {
		return nil, fmt.Errorf("invalid fw_cfg file count %d", count)
	}
	raw, err := dmaRead(ctx, client, fileDirectory, 4+count*fileDirectoryEntrySize)
	if err != nil {
		return nil, err
	}
	return parseDirectory(raw)
}

func parseDirectory(raw []byte) (map[string]fwConfigFile, error) {
	if len(raw) < 4 {
		return nil, fmt.Errorf("fw_cfg directory is only %d bytes", len(raw))
	}
	count := binary.BigEndian.Uint32(raw[:4])
	if count == 0 || count > maxFileDirectoryEntries {
		return nil, fmt.Errorf("invalid fw_cfg file count %d", count)
	}
	expectedSize := 4 + int(count)*fileDirectoryEntrySize
	if len(raw) != expectedSize {
		return nil, fmt.Errorf("fw_cfg directory has %d bytes, want %d", len(raw), expectedSize)
	}

	directory := make(map[string]fwConfigFile, count)
	for index := range count {
		offset := 4 + int(index)*fileDirectoryEntrySize
		entry := raw[offset : offset+fileDirectoryEntrySize]
		name := cString(entry[8:64])
		if name == "" {
			return nil, fmt.Errorf("fw_cfg directory entry %d has an empty name", index)
		}
		if _, exists := directory[name]; exists {
			return nil, fmt.Errorf("fw_cfg directory contains duplicate entry %q", name)
		}
		directory[name] = fwConfigFile{
			selector: binary.BigEndian.Uint16(entry[4:6]),
			size:     binary.BigEndian.Uint32(entry[0:4]),
		}
	}
	return directory, nil
}

func cString(data []byte) string {
	if index := strings.IndexByte(string(data), 0); index >= 0 {
		return string(data[:index])
	}
	return string(data)
}
