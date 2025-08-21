// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/edgelesssys/contrast/internal/cryptsetup"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/mount"
	"github.com/spf13/cobra"
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
	}
	if err := cryptDev.Open(ctx, mappingName); err != nil {
		return fmt.Errorf("opening LUKS device %s: %w", flags.devicePath, err)
	}
	//nolint: contextcheck // The context might be canceled, we still want to close the device.
	defer func() {
		if retErr != nil {
			log.Info("An error occurred, closing LUKS device")
			if err := cryptDev.Close(context.Background(), mappingName); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("closing LUKS device: %w", err))
			}
		}
	}()
	if err := mount.SetupMount(ctx, log, "/dev/mapper/"+mappingName, flags.volumeMountPoint); err != nil {
		return err
	}

	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
