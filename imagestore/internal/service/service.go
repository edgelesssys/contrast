// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync"

	"github.com/edgelesssys/contrast/imagestore/internal/securemountapi"
)

// SecureImageStoreService is the struct for which the SecureImageStore ttRPC service is implemented.
type SecureImageStoreService struct {
	Logger *slog.Logger
	mu     sync.Mutex
}

// SecureMount is a ttRPC service which pulls and mounts docker images.
func (s *SecureImageStoreService) SecureMount(
	ctx context.Context, req *securemountapi.SecureMountRequest,
) (response *securemountapi.SecureMountResponse, retErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := s.Logger.With(slog.String("mount_point", req.MountPoint))
	log.Info("Handling secure image store mount request")
	log.Debug("Mount request details", "options", req.Options, "flags", req.Flags)

	defer func() {
		if retErr != nil {
			log.Error("Request failed", "err", retErr)
		}
	}()

	params, err := getAndVerifyParams(req)
	if err != nil {
		return nil, fmt.Errorf("verifying request parameters: %w", err)
	}
	log = log.With(
		slog.String("device_path", params.DevicePath),
		slog.String("mapper", params.MapperDevice),
	)

	markerFile := path.Join(req.MountPoint, ".contrast_mount_point_marker")
	if _, err := os.Stat(markerFile); err == nil {
		log.Info("store is already mounted, skipping")
		return &securemountapi.SecureMountResponse{}, nil
	}
	if err := setupLuksAndMount(ctx, log, req, params); err != nil {
		return nil, fmt.Errorf("creating and mounting LUKS device: %w", err)
	}
	if _, err := os.Create(markerFile); err != nil {
		return nil, fmt.Errorf("creating marker file: %w", err)
	}

	log.Info("Securely mounted device", "target", req.MountPoint)
	return &securemountapi.SecureMountResponse{}, nil
}
