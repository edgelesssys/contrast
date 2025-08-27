// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/edgelesssys/contrast/internal/cryptsetup"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// cryptsetupFlags holds configuration for mounting a LUKS encrypted device.
type cryptsetupFlags struct {
	devicePath       string
	volumeMountPoint string
}

// NewCryptsetupCmd creates a Cobra subcommand that mounts an encrypted LUKS volume.
func NewCryptsetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cryptsetup -d [device-path] -m [mount-point]",
		Short: "cryptsetup on block device at [device-path] with decrypted mapper device at [mount-point]",
		Long: `Set up an LUKS encrypted VolumeMount on the provided VolumeDevice
		located at the specified [device-path] and mount the decrypted mapper
		device to the provided [mount-point].

		In certain deployments, we require a persistent volume claim configured
		as block storage to be encrypted by the initializer binary.
		Therefore we expose the defined PVC as a block VolumeDevice to our
		initializer container. This allows the initializer to setup the
		encryption on the block device located at [device-path] using cryptsetup
		with the current workload secret as passphrase.

		The mapped decrypted block device can then be shared with other containers
		on the pod by setting up a shared VolumeMount on the specified [mount-point],
		where the mapper device will be mounted to.`,
		RunE: runCryptsetup,
	}
	cmd.Flags().StringP("device-path", "d", "", "path to the volume device to be encrypted")
	cmd.Flags().StringP("mount-point", "m", "", "mount point of decrypted mapper device")
	must(cmd.MarkFlagRequired("device-path"))
	must(cmd.MarkFlagRequired("mount-point"))

	return cmd
}

// parseCryptsetupFlags returns struct of type cryptsetupFlags, representing the provided flags to the subcommand, which is then used to setup the encrypted volume mount.
func parseCryptsetupFlags(cmd *cobra.Command) (*cryptsetupFlags, error) {
	devicePath, err := cmd.Flags().GetString("device-path")
	if err != nil {
		return nil, err
	}
	mountPoint, err := cmd.Flags().GetString("mount-point")
	if err != nil {
		return nil, err
	}
	return &cryptsetupFlags{
		devicePath:       devicePath,
		volumeMountPoint: mountPoint,
	}, nil
}

func runCryptsetup(cmd *cobra.Command, _ []string) error {
	flags, err := parseCryptsetupFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing cryptsetup flags: %w", err)
	}
	log, err := logger.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: creating logger: %v\n", err)
		return err
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	return setupEncryptedMount(ctx, log, flags)
}

// setupEncryptedMount sets up an encrypted LUKS volume mount using the device path and mount point, provided to the setupEncryptedMount subcommand.
func setupEncryptedMount(ctx context.Context, log *slog.Logger, flags *cryptsetupFlags) (retErr error) {
	log = log.With("devicePath", flags.devicePath, "mountPoint", flags.volumeMountPoint)
	mapperHash := sha256.Sum256([]byte(flags.devicePath + flags.volumeMountPoint))
	mappingName := hex.EncodeToString(mapperHash[:8])

	cryptDev, err := cryptsetup.NewDevice(flags.devicePath, workloadSecretPath)
	if err != nil {
		return err
	}
	isLuks, err := cryptDev.IsLuks(ctx)
	if err != nil {
		return err
	}
	if !isLuks {
		log.Info("Device is not a LUKS device, formatting it")
		if err := cryptDev.Format(ctx); err != nil {
			return fmt.Errorf("formatting device %s as LUKS: %w", flags.devicePath, err)
		}
		log.Info("Device formatted successfully")
	} else {
		log.Info("Device is already a LUKS device")
	}
	log.Info("Opening LUKS device", "mappingName", mappingName)
	if err := cryptDev.Open(ctx, mappingName); err != nil {
		return fmt.Errorf("opening LUKS device %s: %w", flags.devicePath, err)
	}
	log.Info("LUKS device opened successfully", "mappingName", mappingName)
	//nolint: contextcheck // The context might be canceled, we still want to close the device.
	defer func() {
		if retErr != nil {
			log.Info("An error occurred, closing LUKS device")
			if err := cryptDev.Close(context.Background(), mappingName); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("closing LUKS device: %w", err))
			}
		}
	}()

	log.Info("Setting up mount point")
	if err := setupMount(ctx, log, "/dev/mapper/"+mappingName, flags.volumeMountPoint); err != nil {
		return err
	}
	log.Info("Mount point set up successfully")

	return nil
}

// setupMount formats the device to ext4 in case it is not ext4 formatted and mounts it to the provided mount point.
func setupMount(ctx context.Context, log *slog.Logger, devPath, mountPoint string) error {
	log = log.With("devPath", devPath, "mountPoint", mountPoint)
	isExt4, err := isExt4(devPath)
	if err != nil {
		return fmt.Errorf("checking if device is ext4: %w", err)
	}
	if !isExt4 {
		log.Info("No ext4 filesystem identified, creating new ext4 filesystem")
		if err := wipeExt4Blocks(ctx, devPath); err != nil {
			return fmt.Errorf("wiping ext4 blocks: %w", err)
		}
		if err := mkfsExt4(ctx, devPath); err != nil {
			return fmt.Errorf("formatting device %s to ext4: %w", devPath, err)
		}
		log.Info("Created ext4 filesystem on device")
	} else {
		log.Info("ext4 filesystem present on device")
	}
	log.Info("Mounting device")
	if err := mount(ctx, devPath, mountPoint); err != nil {
		return err
	}

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

func must(err error) {
	if err != nil {
		panic(err)
	}
}
