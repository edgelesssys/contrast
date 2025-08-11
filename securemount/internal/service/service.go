// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/securemount/internal/api"
	"golang.org/x/sync/singleflight"
)

var mountGroup singleflight.Group

// SecureMountService is the struct for which the SecureMount ttRPC service is implemented.
type SecureMountService struct {
	Logger *slog.Logger
}

// SecureMount is a ttRPC service which pulls and mounts docker images.
func (s *SecureMountService) SecureMount(ctx context.Context, r *api.SecureMountRequest) (response *api.SecureMountResponse, retErr error) {
	log := s.Logger.With(slog.String("mount_point", r.MountPoint))
	log.Info("Handling secure mount request")
	log.Debug("Secure mount request details", "options", r.Options, "flags", r.Flags)

	defer func() {
		if retErr != nil {
			log.Error("Request failed", "err", retErr)
		}
	}()

	p, err := getAndVerifyParams(r)
	if err != nil {
		return nil, fmt.Errorf("verifying request parameters: %w", err)
	}
	log = log.With(
		slog.String("device_path", p.DevicePath),
		slog.String("mapper", p.MapperDevice),
	)

	_, err, repeat := mountGroup.Do(r.MountPoint, func() (any, error) {
		return nil, setupLuksAndMount(ctx, log, r, p)
	})
	if err != nil {
		return nil, fmt.Errorf("creating and mounting LUKS device: %w", err)
	}
	if repeat {
		return nil, fmt.Errorf("repeated mount attempt")
	}

	log.Info("Securely mounted device", "target", r.MountPoint)
	return &api.SecureMountResponse{
		MountPath: r.MountPoint,
	}, nil
}
