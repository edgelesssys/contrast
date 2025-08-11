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

	"github.com/edgelesssys/contrast/securemount/internal/api"
)

func setupLuksAndMount(ctx context.Context, log *slog.Logger, r *api.SecureMountRequest, p *SecureMountParams) error {
	formatArgs := []string{
		"--batch-mode",
		"luksFormat",
		"--type", "luks2",
		"--sector-size", "4096",
		"--cipher", "aes-xts-plain64",
		"--integrity", "hmac-sha256",
		"--key-file", "-",
	}

	if p.DataIntegrity != "false" {
		formatArgs = append(formatArgs, "--integrity", p.DataIntegrity)
	}
	formatArgs = append(formatArgs, p.DevicePath)

	if out, err := runCmdWithInput(ctx, p.Key, "cryptsetup", formatArgs...); err != nil {
		return fmt.Errorf("formatting device with luksFormat: %w: %s", err, out)
	}
	log.Info("Formatted device with LUKS2")

	openArgs := []string{
		"luksOpen",
		"--key-file", "-",
		p.DevicePath,
		p.MapperName,
	}
	if out, err := runCmdWithInput(ctx, p.Key, "cryptsetup", openArgs...); err != nil {
		return fmt.Errorf("luksOpen failed: %w: %s", err, out)
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
