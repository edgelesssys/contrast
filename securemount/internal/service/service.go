// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/securemount/internal/api"
)

// SecureMountService is the struct for which the SecureMount ttRPC service is implemented.
type SecureMountService struct {
	Logger *slog.Logger
}

// SecureMount is a ttRPC service which pulls and mounts docker images.
func (s *SecureMountService) SecureMount(ctx context.Context, r *api.SecureMountRequest) (response *api.SecureMountResponse, retErr error) {
	log := s.Logger.With(slog.String("mount_point", r.MountPoint))
	log.Info("Handling secure mount request")
	log.Debug("Secure mount request details", "options", r.Options, "flags", r.Flags)

	defer func() {
		if retErr != nil {
			log.Error("Request failed", "err", retErr)
		}
	}()

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
	log = log.With(slog.String("device_path", devicePath))

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
	mapperName := fmt.Sprintf("secure-%s", randBytes)
	mappedDev := filepath.Join(api.Device, mapperName)

	key := make([]byte, 64)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	formatArgs := []string{
		"luksFormat",
		"--type", "luks2",
		"--batch-mode",
		"--key-file", "-",
		"--integrity", dataIntegrity,
		devicePath,
	}
	if out, err := runCmdWithInput(ctx, key, "cryptsetup", formatArgs...); err != nil {
		return nil, fmt.Errorf("formatting device with luksFormat: %w: %s", err, out)
	}
	log.Info("Formatted device with LUKS2")

	openArgs := []string{
		"luksOpen",
		"--key-file", "-",
		devicePath,
		mapperName,
	}
	if out, err := runCmdWithInput(ctx, key, "cryptsetup", openArgs...); err != nil {
		return nil, fmt.Errorf("luksOpen failed: %w: %s", err, out)
	}
	log.Info("Opened LUKS device", "mapper", mapperName)

	if out, err := runCmd(ctx, "mkfs.ext4", "-F", mappedDev); err != nil {
		return nil, fmt.Errorf("mkfs.ext4 failed: %w: %s", err, out)
	}
	log.Info("Created ext4 filesystem")

	if err := os.MkdirAll(r.MountPoint, 0o755); err != nil {
		_, closeErr := runCmd(ctx, "cryptsetup", "luksClose", mapperName)
		return nil, fmt.Errorf("failed to create mountpoint: %w, error closing luks device: %w", err, closeErr)
	}

	var mountArgs []string
	if len(r.Flags) > 0 {
		mountArgs = append([]string{"-o", strings.Join(r.Flags, ",")}, mappedDev, r.MountPoint)
	} else {
		mountArgs = []string{mappedDev, r.MountPoint}
	}
	if out, err := runCmd(ctx, "mount", mountArgs...); err != nil {
		_, closeErr := runCmd(ctx, "cryptsetup", "luksClose", mapperName)
		return nil, fmt.Errorf("mounting: %w: %s, error closing luks device: %w", err, out, closeErr)
	}
	log.Info("Securely mounted device", "target", r.MountPoint)

	return &api.SecureMountResponse{
		MountPath: r.MountPoint,
	}, nil
}

func resolveDeviceID(deivceID string) (string, error) {
	sysPath := filepath.Join("/sys/dev/block", deivceID)
	target, err := os.Readlink(sysPath)
	if err != nil {
		return "", err
	}
	base := filepath.Base(target)
	devPath := filepath.Join("/dev", base)
	return devPath, nil
}

func runCmdWithInput(ctx context.Context, input []byte, name string, args ...string) (string, error) {
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, name, args...)
	cmd.Stdin = strings.NewReader(string(input))
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
