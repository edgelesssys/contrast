// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/mount"
	"github.com/spf13/cobra"
)

const (
	// tmpPassphrase is the path to a temporary passphrase file, used for initial formatting.
	tmpPassphrase = "/dev/shm/key"
	// encryptionPassphrase is the path to the disk encryption passphrase file.
	encryptionPassphrasePrefix = "/dev/shm/disk-key"
)

// luksVolume struct holds the representative attributes related to a LUKS encrypted volume.
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

// NewSetupEncryptedMountCmd creates a Cobra subcommand of the initializer to set up the specified encrypted LUKS volume.
func NewSetupEncryptedMountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setupEncryptedMount -d [device-path] -m [mount-point]",
		Short: "setupEncryptedMount on block device at [device-path] with decrypted mapper device at [mount-point]",
		Long: `Set up an LUKS encrypted VolumeMount on the provided VolumeDevice
		located at the specified [device-path] and mount the decrypted mapper
		device to the provided [mount-point].

		In certain deployments, we require a persistent volume claim configured
		as block storage to be encrypted by the initializer binary.
		Therefore we expose the defined PVC as a block VolumeDevice to our
		initializer container. This allows the initializer to setup the
		encryption on the block device located at [device-path] using cryptsetup,
		the encryption passphrase is derived from the UUID of the LUKS formatted
		block device and the current workload secret.

		The mapped decrypted block device can then be shared with other containers
		on the pod by setting up a shared VolumeMount on the specified [mount-point],
		where the mapper device will be mounted to.`,
		RunE: setupEncryptedMount,
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
	if err := mount.SetupMount(ctx, logger, "/dev/mapper/"+luksVolume.mappingName, luksVolume.volumeMountPoint); err != nil {
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
	blk, err := mount.Blkid(ctx, luksVolume.devicePath)
	if err != nil {
		return err
	}
	workloadSecretBytes, err := os.ReadFile(workloadSecretPath)
	if err != nil {
		return fmt.Errorf("reading workload secret: %w", err)
	}
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
