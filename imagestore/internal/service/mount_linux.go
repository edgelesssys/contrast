// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/edgelesssys/contrast/imagestore/internal/api"
	"github.com/edgelesssys/contrast/internal/cryptsetup"
)

func setupLuksAndMount(ctx context.Context, log *slog.Logger, req *api.SecureMountRequest, params *SecureImageStoreParams) (retErr error) {
	if err := os.WriteFile(params.KeyFile, params.Key, 0o600); err != nil {
		return fmt.Errorf("writing key to file: %w", err)
	}
	defer func() {
		if err := os.Remove(params.KeyFile); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("removing key file: %w", err))
		}
	}()

	device, err := cryptsetup.NewDevice(params.DevicePath, params.KeyFile, params.MapperName)
	if err != nil {
		return fmt.Errorf("preparing device: %w", err)
	}
	defer func() {
		if retErr != nil {
			if err := device.Close(ctx); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("closing luks device: %w", err))
			}
		}
	}()

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

	if err := os.MkdirAll(req.MountPoint, 0o755); err != nil {
		return fmt.Errorf("failed to create mountpoint: %w", err)
	}

	mountArgs := []string{params.MapperDevice, req.MountPoint}
	if err := runCmd(ctx, "mount", mountArgs...); err != nil {
		return fmt.Errorf("mounting: %w", err)
	}

	return nil
}

func runCmd(ctx context.Context, name string, args ...string) error {
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("executing '%s %s': %w: %s", name, args, err, out)
	}
	return nil
}
