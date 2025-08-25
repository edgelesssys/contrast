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

	"github.com/edgelesssys/contrast/internal/cryptsetup"
	"github.com/edgelesssys/contrast/securemount/internal/api"
)

func setupLuksAndMount(ctx context.Context, log *slog.Logger, r *api.SecureMountRequest, p *SecureMountParams) (retErr error) {
	if err := os.WriteFile(p.KeyFile, p.Key, 0o644); err != nil {
		return fmt.Errorf("writing key to file: %w", err)
	}
	defer func() {
		if err := os.Remove(p.KeyFile); err != nil {
			retErr = fmt.Errorf("remove: %w", err)
		}
	}()

	device := cryptsetup.NewDevice(p.DevicePath, "/run/confidential-containers/header", p.KeyFile)

	if err := device.Format(ctx); err != nil {
		return fmt.Errorf("formatting device with luksFormat: %w", err)
	}
	log.Info("Formatted device with LUKS2")

	if err := device.Open(ctx, p.MapperName); err != nil {
		return fmt.Errorf("opening device with luksOpen: %w", err)
	}
	log.Info("Opened LUKS device")

	if out, err := runCmd(ctx, "mkfs.ext4", "-F", p.MapperDevice); err != nil {
		return fmt.Errorf("mkfs.ext4 failed: %w: %s", err, out)
	}
	log.Info("Created ext4 filesystem")

	if err := os.MkdirAll(r.MountPoint, 0o755); err != nil {
		_, closeErr := runCmd(ctx, "cryptsetup", "luksClose", p.MapperName)
		return fmt.Errorf("failed to create mountpoint: %w, error closing luks device: %w", err, closeErr)
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
