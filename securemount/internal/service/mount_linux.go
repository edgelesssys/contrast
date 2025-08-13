// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/edgelesssys/contrast/securemount/internal/api"
)

func setupLuksAndMount(ctx context.Context, r *api.SecureMountRequest, p *SecureMountParams) error {
	if err := os.WriteFile(p.KeyFile, p.Key, 0o644); err != nil {
		return fmt.Errorf("writing key file: %w", err)
	}

	args := []string{
		p.DeviceID,
		// The device should never already be encrypted.
		// Setting this to true skips the LUKS setup.
		"false",
		r.MountPoint,
		p.KeyFile,
		p.DataIntegrity,
	}

	if out, err := runCmd(ctx, "luks-encrypt-storage", args...); err != nil {
		return fmt.Errorf("LUKS encrypting storage: %w, %s", err, out)
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
