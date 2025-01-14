// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/sha256"
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

// parsedCryptsetupFlags holds configuration for mounting a LUKS encrypted device.
type parsedCryptsetupFlags struct {
	devicePath       string
	mappingName      string
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
		RunE: setupEncryptedMount,
	}
	cmd.Flags().StringP("device-path", "d", "", "path to the volume device to be encrypted")
	cmd.Flags().StringP("mount-point", "m", "", "mount point of decrypted mapper device")
	must(cmd.MarkFlagRequired("device-path"))
	must(cmd.MarkFlagRequired("mount-point"))

	return cmd
}

// parseCryptsetupFlags returns struct of type parsedCryptsetupFlags, representing the provided flags to the subcommand, which is then used to setup the encrypted volume mount.
func parseCryptsetupFlags(cmd *cobra.Command) (*parsedCryptsetupFlags, error) {
	devicePath, err := cmd.Flags().GetString("device-path")
	if err != nil {
		return nil, err
	}
	mountPoint, err := cmd.Flags().GetString("mount-point")
	if err != nil {
		return nil, err
	}
	mapperHash := sha256.Sum256([]byte(devicePath + mountPoint))
	mappingName := hex.EncodeToString(mapperHash[:8])
	return &parsedCryptsetupFlags{
		devicePath:       devicePath,
		mappingName:      mappingName,
		volumeMountPoint: mountPoint,
	}, nil
}

// setupEncryptedMount sets up an encrypted LUKS volume mount using the device path and mount point, provided to the setupEncryptedMount subcommand.
func setupEncryptedMount(cmd *cobra.Command, _ []string) error {
	// Register channel waiting for SIGTERM signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)
	logger, err := logger.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: creating logger: %v\n", err)
		return err
	}
	parsedCryptsetupFlags, err := parseCryptsetupFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing setupEncryptedMount flags: %w", err)
	}
	ctx := cmd.Context()
	if !isLuks(ctx, logger, parsedCryptsetupFlags.devicePath) {
		// TODO(jmxnzo) check what happens if container is terminated in between formatting
		if err := luksFormat(ctx, parsedCryptsetupFlags.devicePath, workloadSecretPath); err != nil {
			return err
		}
	}
	if err := openEncryptedDevice(ctx, parsedCryptsetupFlags, workloadSecretPath); err != nil {
		return err
	}
	// The decrypted devices with <name> will always be mapped to /dev/mapper/<name> by default.
	if err := mount.SetupMount(ctx, logger, "/dev/mapper/"+parsedCryptsetupFlags.mappingName, parsedCryptsetupFlags.volumeMountPoint); err != nil {
		return err
	}

	if err := os.WriteFile("/done", []byte(""), 0o644); err != nil {
		return fmt.Errorf("Creating startup probe done directory:%w", err)
	}
	// Wait for SIGTERM signal
	<-signalChan

	return nil
}

// isLuks wraps the cryptsetup isLuks command and returns a bool reflecting if the device is formatted as LUKS.
func isLuks(ctx context.Context, logger *slog.Logger, devName string) bool {
	if _, err := exec.CommandContext(ctx, "cryptsetup", "isLuks", "--debug", devName).CombinedOutput(); err != nil {
		logger.Info("device", devName, "is not a LUKS device or cannot be accessed", "err", err)
		return false
	}
	return true
}

// luksFormat wraps the luksFormat command.
func luksFormat(ctx context.Context, devName, pathToKey string) error {
	if out, err := exec.CommandContext(ctx, "cryptsetup", "luksFormat", "--pbkdf-memory=10240", devName, pathToKey).CombinedOutput(); err != nil {
		return fmt.Errorf("cryptsetup luksFormat: %w, output: %q", err, out)
	}
	return nil
}

// openEncryptedDevice wraps the cryptsetup open command.
func openEncryptedDevice(ctx context.Context, parsedCryptsetupFlags *parsedCryptsetupFlags, pathToKey string) error {
	if _, err := exec.CommandContext(ctx, "cryptsetup", "open", parsedCryptsetupFlags.devicePath, parsedCryptsetupFlags.mappingName, "-d", pathToKey).CombinedOutput(); err != nil {
		return fmt.Errorf("cryptsetup open %s failed: %w", parsedCryptsetupFlags.devicePath, err)
	}
	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
