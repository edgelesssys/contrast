// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/spf13/cobra"
)

const (
	// tmpPassphrase is the path to a temporary passphrase file, used for initial formatting.
	tmpPassphrase = "/dev/shm/key"
	// encryptionPassphrase is the path to the disk encryption passphrase file.
	encryptionPassphrasePrefix = "/dev/shm/disk-key"
)

type luksVolume struct {
	devicePath           string
	mappingName          string
	volumeMountPoint     string
	encryptionPassphrase string
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// NewSetupEncryptedMountCmd creates a Cobra subcommand of the initializer to set up specified encrypted volumes.
func NewSetupEncryptedMountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setupEncryptedMount -d [device-path] -m [mount-point]",
		Short: "",
		Long:  "",
		RunE:  setupEncryptedMount,
	}
	cmd.Flags().StringP("device-path", "d", "/dev/csi0", "path to the volume device to be encrypted")
	cmd.Flags().StringP("mount-point", "m", "/state", "mount point of decrypted mapper device")
	must(cmd.MarkFlagRequired("device-path"))
	must(cmd.MarkFlagRequired("mount-point"))

	return cmd
}

// parseSetupEncryptedMountFlags returns a luksVolume, representing the provided flags to the subcommand, which is then used to setup the encrypted volume mount.
func parseSetupEncryptedMountFlags(cmd *cobra.Command) (*luksVolume, error) {
	devicePath, err := cmd.Flags().GetString("device-path")
	if err != nil {
		return nil, err
	}
	mountPoint, err := cmd.Flags().GetString("mount-point")
	if err != nil {
		return nil, err
	}
	hash := md5.Sum([]byte(devicePath + mountPoint))
	mappingName := hex.EncodeToString(hash[:8])
	return &luksVolume{
		devicePath:           devicePath,
		mappingName:          mappingName,
		volumeMountPoint:     mountPoint,
		encryptionPassphrase: encryptionPassphrasePrefix + mappingName,
	}, nil
}

// setupEncryptedMount sets up an encrypted LUKS volume mount using the device path and mount point, provided to the setupEncryptedMount subcommand.
func setupEncryptedMount(cmd *cobra.Command, _ []string) error {
	logger, err := logger.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: creating logger: %v\n", err)
		return err
	}
	luksVolume, err := parseSetupEncryptedMountFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing setupEncryptedMount flags: %w", err)
	}
	ctx := cmd.Context()
	if !isLuks(ctx, logger, luksVolume.devicePath) {
		// TODO(jmxnzo) might just use stdin instead for the initial passphrase generation
		if err := createInitPassphrase(tmpPassphrase); err != nil {
			return err
		}
		logger.Info("formatting csi device to LUKS with initial passphrase)")
		// TODO(jmxnzo) check what happens if container is terminated in between formatting
		if err := luksFormat(ctx, luksVolume.devicePath, tmpPassphrase); err != nil {
			return err
		}
		if err := createEncryptionPassphrase(ctx, luksVolume, workloadSecretPath); err != nil {
			return err
		}
		if err := luksChangeKey(ctx, luksVolume.devicePath, tmpPassphrase, luksVolume.encryptionPassphrase); err != nil {
			return err
		}
	} else {
		if err := createEncryptionPassphrase(ctx, luksVolume, workloadSecretPath); err != nil {
			return err
		}
	}
	if err := openEncryptedDevice(ctx, luksVolume); err != nil {
		return err
	}
	// The decrypted devices with <name> will always be mapped to /dev/mapper/<name> by default.
	if err := setupMount(ctx, logger, "/dev/mapper/"+luksVolume.mappingName, luksVolume.volumeMountPoint); err != nil {
		return err
	}

	if err := os.WriteFile("/done", []byte(""), 0o644); err != nil {
		return fmt.Errorf("Creating startup probe done directory:%w", err)
	}
	// Waits for SIGTERM signal and then properly release the resources.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)
	<-signalChan

	return nil
}

// setupMount mounts the csi device to the state disk mount point.
func setupMount(ctx context.Context, logger *slog.Logger, devPath, mountPath string) error {
	blk, err := blkid(ctx, devPath)
	if errors.Is(err, errNotIdentified) {
		logger.Info("csi device not identified, assuming first start, formatting", "device", devPath)
		if err := mkfsExt4(ctx, devPath); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if blk.Type != "ext4" {
		logger.Info("csi device is not ext4, assuming first start, formatting", "device", devPath)
		if err := mkfsExt4(ctx, devPath); err != nil {
			return err
		}
	}

	if err := mount(ctx, devPath, mountPath); err != nil {
		return err
	}
	logger.Info("csi device mounted to state disk mount point", "dev", devPath, "mountPoint", mountPath)

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

// createInitPassphrase creates a hardcoded string passphrase, to allow formatting the device to LUKS in order to get the UUID.
func createInitPassphrase(pathToPassphrase string) (err error) {
	err = os.WriteFile(pathToPassphrase, []byte("init_passphrase"), 0o644)
	if err != nil {
		return fmt.Errorf("Writing initial passphrase: %w", err)
	}
	return nil
}

// createEncryptionPassphrase writes the UUID of the device and the current workload secret to the path of encryptionPassphrase in luksVolume.
func createEncryptionPassphrase(ctx context.Context, luksVolume *luksVolume, workloadSecretPath string) error {
	blk, err := blkid(ctx, luksVolume.devicePath)
	if err != nil {
		return err
	}
	workloadSecretBytes, err := os.ReadFile(workloadSecretPath)
	if err != nil {
		return fmt.Errorf("reading workload secret: %w", err)
	}
	print(string(workloadSecretBytes))
	// Using UUID of the LUKS device ensures to not derive the same encryption key for multiple devices,
	// still allowing reconstruction when UUID of device is known.
	err = os.WriteFile(luksVolume.encryptionPassphrase, []byte(blk.UUID+string(workloadSecretBytes)), 0o644)
	if err != nil {
		return fmt.Errorf("writing encryption passphrase: %w", err)
	}
	return nil
}

// isLuks wraps the cryptsetup isLuks command and returns a bool reflecting if the device is formatted as LUKS.
func isLuks(ctx context.Context, logger *slog.Logger, devName string) bool {
	cmd := exec.CommandContext(ctx, "cryptsetup", "isLuks", "--debug", devName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		logger.Info("cryptsetup isLuks failed", "error", err)
		return false
	}
	return true
}

// luksFormat wraps the luksFormat command.
func luksFormat(ctx context.Context, devName, pathToKey string) error {
	cmd := exec.CommandContext(ctx, "cryptsetup", "luksFormat", "--pbkdf-memory=10240", devName, pathToKey)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cryptsetup luksFormat: %w, output: %q", err, out)
	}
	return nil
}

// openEncryptedDevice wraps the cryptsetup open command.
func openEncryptedDevice(ctx context.Context, luksVolume *luksVolume) error {
	cmd := exec.CommandContext(ctx, "cryptsetup", "open", luksVolume.devicePath, luksVolume.mappingName, "-d", luksVolume.encryptionPassphrase)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cryptsetup open %s failed: %w", luksVolume.devicePath, err)
	}
	return nil
}

// luksChangeKey wraps the luksChangeKey command.
func luksChangeKey(ctx context.Context, devName, oldKeyPath, newKeyPath string) error {
	cmd := exec.CommandContext(ctx, "cryptsetup", "luksChangeKey", "--pbkdf-memory=10240", devName, "--key-file", oldKeyPath, newKeyPath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cryptsetup luksChangeKey %s failed: %w", devName, err)
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
