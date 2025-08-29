// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cryptsetup

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// IsExt4 checks for presence of ext4 magic bytes on a device to determine if it is formatted as ext4.
//
// Note: We previously used 'blkid' to check the filesystem type, but it tries
// to read blocks that aren't initialized after dm-integrity is set up. We implement
// this simple manual check, knowing it will only read the expected block which
// is manually initialized by wipeExt4Blocks before it is used by mkfs.ext4.
func (d *Device) IsExt4(_ context.Context) (bool, error) {
	const (
		expectMagic = uint16(0xef53) // Magic number for ext4 filesystems.
		offset      = 1080           // Offset of ext4 superblock (1024) + offset of magic within superblock (56).
	)
	f, err := os.Open(filepath.Join("/dev/mapper", d.mappingName))
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

// MkfsExt4 creates an ext4 filesystem on the device.
// It assumes the device is integrity-protected and unwiped, so it wipes the blocks
// that are used by ext4 before creating the filesystem.
func (d *Device) MkfsExt4(ctx context.Context) error {
	mappingPath := filepath.Join("/dev/mapper", d.mappingName)
	if err := wipeExt4Blocks(ctx, mappingPath); err != nil {
		return fmt.Errorf("wiping ext4 blocks: %w", err)
	}
	return mkfsExt4(ctx, mappingPath)
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
	blockNumStrs := numsRegexp.FindAllString(blockList, -1)
	if len(blockNumStrs) == 0 {
		return fmt.Errorf("parsing mkfs.ext4 output: no block numbers found in output %q", out)
	}
	blockNums := make([]int64, 0, len(blockNumStrs))
	for _, s := range blockNumStrs {
		i, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("parsing mkfs.ext4 output: parsing block number %q: %w", s, err)
		}
		if i < 0 {
			return fmt.Errorf("parsing mkfs.ext4 output: invalid block number %d", i)
		}
		blockNums = append(blockNums, int64(i))
	}
	blockNums = append(blockNums, 0)

	if err := zeroBlocksDirect(devPath, blockNums); err != nil {
		return fmt.Errorf("zeroing blocks on device %s: %w", devPath, err)
	}
	return nil
}

func zeroBlocksDirect(path string, indices []int64) error {
	// Open with O_DIRECT to bypass page cache.
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_DIRECT, 0)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer unix.Close(fd)

	const blockSize = 4096

	// Page-aligned zero buffer, required for O_DIRECT.
	buf, err := unix.Mmap(-1, 0, blockSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANONYMOUS|unix.MAP_PRIVATE)
	if err != nil {
		return fmt.Errorf("allocating zero buffer via mmap: %w", err)
	}
	defer func() { _ = unix.Munmap(buf) }()

	for _, index := range indices {
		offset := index * blockSize
		for written := 0; written < blockSize; {
			n, err := unix.Pwrite(fd, buf[written:], offset+int64(written))
			if err != nil {
				return fmt.Errorf("writing zero block at index %d (offset %d): %w", index, offset, err)
			}
			written += n
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
