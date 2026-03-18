// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/imagestore/internal/securemountapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SecureImageStoreParams provides simpler access to the (verified) request parameters.
type SecureImageStoreParams struct {
	DeviceID      string
	DevicePath    string
	DataIntegrity string
	MapperName    string
	MapperDevice  string
	Key           []byte
	KeyFile       string
}

func getAndVerifyParams(req *securemountapi.SecureMountRequest) (*SecureImageStoreParams, error) {
	if req.MountPoint == "" {
		return nil, status.Errorf(codes.InvalidArgument, "mountpoint is required")
	}

	// Hardcoded in https://github.com/kata-containers/kata-containers/blob/7480aa636e54feb2a2e28bb79023201754997556/src/agent/src/rpc.rs#L2321
	if req.VolumeType != "block-device" {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported volmue type: %s", req.VolumeType)
	}

	deviceID, ok := req.Options["deviceId"]
	if !ok || deviceID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Options[\"deviceId\"] is required")
	}

	devicePath, err := resolveDeviceID(deviceID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "resolving device path")
	}

	// Hardcoded in https://github.com/kata-containers/kata-containers/blob/7480aa636e54feb2a2e28bb79023201754997556/src/agent/src/rpc.rs#L2323
	encryptType, ok := req.Options["encryptionType"]
	if !ok || encryptType != "luks2" {
		return nil, status.Errorf(codes.InvalidArgument, "Options[\"encryptType\"] must be LUKS")
	}

	dataIntegrity, ok := req.Options["dataIntegrity"]
	if !ok || dataIntegrity == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Options[\"dataIntegrity\"] is required")
	}

	mapperSuffix := make([]byte, 8)
	if _, err := rand.Read(mapperSuffix); err != nil {
		return nil, fmt.Errorf("generating mapper suffix: %w", err)
	}
	mapperName := fmt.Sprintf("secure-%x", mapperSuffix)
	mappedDev := filepath.Join("/dev/mapper", mapperName)

	keyFileSuffix := make([]byte, 8)
	if _, err := rand.Read(keyFileSuffix); err != nil {
		return nil, fmt.Errorf("generating mapper suffix: %w", err)
	}
	keyFile := fmt.Sprintf("/run/key_%x", keyFileSuffix)

	key := make([]byte, 64)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	return &SecureImageStoreParams{
		DeviceID:      deviceID,
		DevicePath:    devicePath,
		DataIntegrity: dataIntegrity,
		MapperName:    mapperName,
		MapperDevice:  mappedDev,
		Key:           key,
		KeyFile:       keyFile,
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
