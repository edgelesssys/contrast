// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	csiDevicePath       = "/dev/csi0"
	stateDiskMountPoint = "/mnt/state"
)

// setupMount mounts the csi device to the state disk mount point.
func setupMount(ctx context.Context, logger *slog.Logger) error {
	blk, err := blkid(ctx, csiDevicePath)
	if errors.Is(err, errNotIdentified) {
		logger.Info("csi device not identified, assuming first start, formatting")
		if err := mkfsExt4(ctx, csiDevicePath); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if blk.Type != "ext4" {
		logger.Info("csi device is not ext4, assuming first start, formatting")
		if err := mkfsExt4(ctx, csiDevicePath); err != nil {
			return err
		}
	}

	if err := mount(ctx, csiDevicePath, stateDiskMountPoint); err != nil {
		return err
	}
	logger.Info("csi device mounted to state disk mount point", "dev", csiDevicePath, "mountPoint", stateDiskMountPoint)

	return nil
}

type blk struct {
	DevName   string
	UUID      string
	BlockSize int
	Type      string
}

var errNotIdentified = errors.New("blkid did not identify the device")

func blkid(ctx context.Context, devName string) (*blk, error) {
	cmd := exec.CommandContext(ctx, "blkid", "-o", "export", devName)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
		// See man page, sec return code.
		return nil, errNotIdentified
	} else if err != nil {
		return nil, fmt.Errorf("blkid: %w, output: %q", err, out)
	}
	lines := strings.Split(string(out), "\n")
	b := &blk{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("parsing blkid output line %q: %w", line, err)
		}
		switch key {
		case "DEVNAME":
			b.DevName = value
		case "UUID":
			b.UUID = value
		case "TYPE":
			b.Type = value
		case "BLOCK_SIZE":
			b.BlockSize, err = strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("parsing BLOCK_SIZE of blkid output %q: %w", value, err)
			}
		}
	}
	return b, nil
}

func mkfsExt4(ctx context.Context, devName string) error {
	cmd := exec.CommandContext(ctx, "mkfs.ext4", devName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkfs.ext4: %w, output: %q", err, out)
	}
	return nil
}

func mount(ctx context.Context, devName, mountPoint string) error {
	if err := os.MkdirAll(mountPoint, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	cmd := exec.CommandContext(ctx, "mount", devName, mountPoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount: %w, output: %q", err, out)
	}
	return nil
}
