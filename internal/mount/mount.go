// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package mount

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

// SetupMount formats the device to ext4 in case it is not ext4 formatted and mounts it to the provided mount point.
func SetupMount(ctx context.Context, logger *slog.Logger, devPath, mountPoint string) error {
	blk, err := blkid(ctx, devPath)
	if errors.Is(err, errNotIdentified) {
		logger.Info("device not identified, formatting", "device", devPath)
		if err := mkfsExt4(ctx, devPath); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if blk.Type != "ext4" {
		logger.Info("device is not ext4, formatting", "device", devPath)
		if err := mkfsExt4(ctx, devPath); err != nil {
			return err
		}
	}

	if err := mount(ctx, devPath, mountPoint); err != nil {
		return err
	}
	logger.Info("device mounted successfully", "dev", devPath, "mountPoint", mountPoint)

	return nil
}

// blk holds the main attributes of a block device used by blkid system executable command.
type blk struct {
	DevName   string
	UUID      string
	BlockSize int
	Type      string
}

var errNotIdentified = errors.New("blkid did not identify the device")

// blkid creates a blk struct of the device located at the provided devPath.
func blkid(ctx context.Context, devPath string) (*blk, error) {
	cmd := exec.CommandContext(ctx, "blkid", "-o", "export", devPath)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
		// See man page, sec EXIT STATUS.
		return nil, errNotIdentified
	} else if err != nil {
		return nil, fmt.Errorf("blkid: %w, output: %q", err, out)
	}
	return parseBlkidCommand(out)
}

// parseBlkidCommand parses the output of the blkid system command to rerpresentative Blkid struct.
func parseBlkidCommand(out []byte) (*blk, error) {
	lines := strings.Split(string(out), "\n")
	b := &blk{}

	for _, line := range lines {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("parsing blkid output line %q", line)
		}
		switch key {
		case "DEVNAME":
			b.DevName = value
		case "UUID":
			b.UUID = value
		case "TYPE":
			b.Type = value
		case "BLOCK_SIZE":
			blockSize, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("parsing BLOCK_SIZE of blkid output %q: %w", value, err)
			}
			b.BlockSize = blockSize
		}
	}
	return b, nil
}

// mkfsExt4 wraps the mkfs.ext4 command and creates an ext4 file system on the device.
func mkfsExt4(ctx context.Context, devPath string) error {
	cmd := exec.CommandContext(ctx, "mkfs.ext4", devPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkfs.ext4: %w, output: %q", err, out)
	}
	return nil
}

// mount wraps the mount command and attaches the device to the specified mountPoint.
func mount(ctx context.Context, devPath, mountPoint string) error {
	if err := os.MkdirAll(mountPoint, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if out, err := exec.CommandContext(ctx, "mount", devPath, mountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("mount: %w, output: %q", err, out)
	}
	return nil
}
