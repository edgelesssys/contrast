// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package mount

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

// SetupMount formats the device to ext4 in case it is not ext4 formatted and mounts it to the provided mount point.
func SetupMount(ctx context.Context, logger *slog.Logger, devPath, mountPoint string) error {
	isExt4, err := isExt4(devPath)
	if err != nil {
		return fmt.Errorf("checking if device is ext4: %w", err)
	}
	if !isExt4 {
		logger.Info("Device is not ext4", "dev", devPath)
		if err := wipeExt4Blocks(ctx, devPath); err != nil {
			return fmt.Errorf("wiping ext4 blocks: %w", err)
		}
		if err := mkfsExt4(ctx, devPath); err != nil {
			return fmt.Errorf("formatting device %s to ext4: %w", devPath, err)
		}
		logger.Info("Device formatted to ext4 successfully", "dev", devPath)
	} else {
		logger.Info("Device is already ext4 formatted", "dev", devPath)
	}
	logger.Info("Mounting device", "dev", devPath, "mountPoint", mountPoint)
	if err := mount(ctx, devPath, mountPoint); err != nil {
		return err
	}
	logger.Info("Device mounted successfully", "dev", devPath, "mountPoint", mountPoint)

	return nil
}

// isExt4 checks for presence of ext4 magic bytes on a device.
//
// Note: We previously used 'blkid' to check the filesystem type, but it tries
// to read blocks that aren't initialized after dm-verity is set up. We implement
// this simple manual check, knowing it will only read the expected block which
// is manually initialized by wipeExt4Blocks before it is used by mkfs.ext4.
func isExt4(devPath string) (bool, error) {
	const (
		expectMagic = uint16(0xef53) // Magic number for ext4 filesystems.
		offset      = 1080           // Offset of ext4 superblock (1024) + offset of magic within superblock (56).
	)
	f, err := os.Open(devPath)
	if err != nil {
		return false, fmt.Errorf("opening device: %w", err)
	}
	defer f.Close()
	if _, err := f.Seek(offset, 0); err != nil {
		return false, fmt.Errorf("seeking magic offset: %w", err)
	}
	var magic uint16
	if err := binary.Read(f, binary.LittleEndian, &magic); err != nil && errors.Is(err, syscall.EIO) {
		// When the device wasn't formatted before, we expect integrity data to be invalid.
		// In this case, we expect an I/O error when reading from the device.
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("reading magic bytes: %w", err)
	}
	return magic == expectMagic, nil
}

var numsRegexp = regexp.MustCompile(`\d+`)

func wipeExt4Blocks(ctx context.Context, devPath string) error {
	// Run mkfs.ext4 in dry-run mode to get the blocks that would be used by the filesystem.
	cmd := exec.CommandContext(ctx, "mkfs.ext4", "-F", "-n", devPath)
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("executing %s: %w, stderr: %q", cmd.String(), err, exitErr.Stderr)
	} else if err != nil {
		return fmt.Errorf("executing %s: %w, output: %q", cmd.String(), err, out)
	}

	delimiter := "Superblock backups stored on blocks:"
	_, blockList, ok := strings.Cut(string(out), delimiter)
	if !ok {
		return fmt.Errorf("parsing mkfs.ext4 output: delimiter %q not found in output %q", delimiter, out)
	}
	blockNums := numsRegexp.FindAllString(blockList, -1)
	if len(blockNums) == 0 {
		return fmt.Errorf("parsing mkfs.ext4 output: no block numbers found in output %q", out)
	}
	blockNums = append(blockNums, "0")

	for _, blockNum := range blockNums {
		cmd := exec.CommandContext(ctx, "dd", "if=/dev/zero", "bs=4k", "count=1", "oflag=direct",
			"of="+devPath, "seek="+blockNum)
		var exitErr *exec.ExitError
		if out, err := cmd.CombinedOutput(); err != nil && errors.As(err, &exitErr) {
			return fmt.Errorf("running %s: %w, stdout: %q, stderr: %q", cmd.String(), err, out, exitErr.Stderr)
		} else if err != nil {
			return fmt.Errorf("running %s: %w, output: %q", cmd.String(), err, out)
		}
	}
	return nil
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
	if out, err := exec.CommandContext(ctx, "mount", "-o", "sync,data=journal", devPath, mountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("mount: %w, output: %q", err, out)
	}
	return nil
}
