// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func (s *ImagePullerService) createAndMountContainer(log *slog.Logger, imageID, bundlePath string) (string, error) {
	rootfs := filepath.Join(bundlePath, "rootfs")

	// Store the rootfs path as the container's metadata so that orphaned containers can be reclaimed later (see cleanupOrphanedContainers)
	container, err := s.Store.CreateContainer("", nil, imageID, "", rootfs, nil)
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}
	log.Info("Created container", "id", container.ID)

	mountPoint, err := s.Store.Mount(container.ID, "")
	if err != nil {
		return "", fmt.Errorf("mounting container: %w", err)
	}
	log.Debug("Mounted in store", "mountPoint", mountPoint)

	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		return "", fmt.Errorf("creating bundle path: %w", err)
	}

	if err := unix.Mount(mountPoint, rootfs, "", unix.MS_BIND, ""); err != nil {
		return "", fmt.Errorf("binding mount %s to %s: %w", mountPoint, rootfs, err)
	}

	return rootfs, nil
}
