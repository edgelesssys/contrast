// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/imagestore/internal/api"
)

// SecureImageStoreService is the struct for which the SecureImageStore ttRPC service is implemented.
type SecureImageStoreService struct {
	Logger *slog.Logger
}

// SecureMount is a ttRPC service which pulls and mounts docker images.
func (s *SecureImageStoreService) SecureMount(ctx context.Context, req *api.SecureMountRequest) (response *api.SecureMountResponse, retErr error) {
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

	if err := setupLuksAndMount(ctx, log, req, params); err != nil {
		return nil, fmt.Errorf("creating and mounting LUKS device: %w", err)
	}

	log.Info("Securely mounted device", "target", req.MountPoint)
	return &api.SecureMountResponse{
		MountPath: req.MountPoint,
	}, nil
}
