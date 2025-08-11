// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/securemount/internal/api"
)

// SecureMountParams provides simpler access to the (verified) request parameters.
type SecureMountParams struct {
	DeviceID      string
	DevicePath    string
	DataIntegrity string
	MapperName    string
	MapperDevice  string
	Key           []byte
}

func getAndVerifyParams(r *api.SecureMountRequest) (*SecureMountParams, error) {
	if r.MountPoint == "" {
		return nil, fmt.Errorf("mountpoint is required")
	}

	// Hardcoded in https://github.com/kata-containers/kata-containers/blob/b50777a174a2daa7af51b1599b5d1e0b265a53be/src/agent/src/rpc.rs#L2292
	if r.VolumeType != "BlockDevice" {
		return nil, fmt.Errorf("unsupported volmue type: %s", r.VolumeType)
	}

	deviceID, ok := r.Options["deviceId"]
	if !ok || deviceID == "" {
		return nil, fmt.Errorf("Options[\"deviceId\"] is required")
	}

	devicePath, err := resolveDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("resolving device path")
	}

	// Hardcoded in https://github.com/kata-containers/kata-containers/blob/b50777a174a2daa7af51b1599b5d1e0b265a53be/src/agent/src/rpc.rs#L2288
	encryptType, ok := r.Options["encryptType"]
	if !ok || encryptType != "LUKS" {
		return nil, fmt.Errorf("Options[\"encryptType\"] must be LUKS")
	}

	dataIntegrity, ok := r.Options["dataIntegrity"]
	if !ok || dataIntegrity == "" {
		return nil, fmt.Errorf("Options[\"dataIntegrity\"] is required")
	}

	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, fmt.Errorf("generating mapper suffix: %w", err)
	}
	mapperName := fmt.Sprintf("secure-%s", hex.EncodeToString(randBytes))
	mappedDev := filepath.Join("/dev/mapper", mapperName)

	key := make([]byte, 64)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	return &SecureMountParams{
		DeviceID:      deviceID,
		DevicePath:    devicePath,
		DataIntegrity: dataIntegrity,
		MapperName:    mapperName,
		MapperDevice:  mappedDev,
		Key:           key,
	}, nil
}

func resolveDeviceID(deviceID string) (string, error) {
	sysPath := filepath.Join("/sys/dev/block", deviceID)
	target, err := os.Readlink(sysPath)
	if err != nil {
		return "", err
	}
	base := filepath.Base(target)
	devPath := filepath.Join("/dev", base)
	return devPath, nil
}
