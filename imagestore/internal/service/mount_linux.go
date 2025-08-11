// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/imagestore/internal/api"
	"github.com/edgelesssys/contrast/internal/cryptsetup"
)

func setupLuksAndMount(ctx context.Context, log *slog.Logger, r *api.SecureMountRequest, p *SecureImageStoreParams) (retErr error) {
	if err := os.WriteFile(p.KeyFile, p.Key, 0o644); err != nil {
		return fmt.Errorf("writing key to file: %w", err)
	}
	defer func() {
		if err := os.Remove(p.KeyFile); err != nil {
			retErr = fmt.Errorf("remove: %w", err)
		}
	}()

	device, err := cryptsetup.NewDevice(p.DevicePath, p.KeyFile, p.MapperName)
	if err != nil {
		return fmt.Errorf("preparing device: %w", err)
	}

	if err := device.Format(ctx); err != nil {
		return fmt.Errorf("formatting device with luksFormat: %w", err)
	}
	log.Info("Formatted device with LUKS2")

	if err := device.Open(ctx); err != nil {
		return fmt.Errorf("opening device with luksOpen: %w", err)
	}
	log.Info("Opened LUKS device")

	if err := device.MkfsExt4(ctx); err != nil {
		return fmt.Errorf("formatting with mkfs.ext4: %w", err)
	}
	log.Info("Set up ext4 filesystem")

	if err := os.MkdirAll(r.MountPoint, 0o755); err != nil {
		return fmt.Errorf("failed to create mountpoint: %w", err)
	}

	var mountArgs []string
	if len(r.Flags) > 0 {
		mountArgs = append([]string{"-o", strings.Join(r.Flags, ",")}, p.MapperDevice, r.MountPoint)
	} else {
		mountArgs = []string{p.MapperDevice, r.MountPoint}
	}
	if out, err := runCmd(ctx, "mount", mountArgs...); err != nil {
		_, closeErr := runCmd(ctx, "cryptsetup", "luksClose", p.MapperName)
		return fmt.Errorf("mounting: %w: %s, error closing luks device: %w", err, out, closeErr)
	}

	return nil
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
