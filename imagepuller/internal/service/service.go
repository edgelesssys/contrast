// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containers/storage"
	"github.com/edgelesssys/contrast/imagepuller/internal/api"
	"golang.org/x/sys/unix"
)

// ImagePullerService is the struct for which the PullImage ttRPC service is implemented.
type ImagePullerService struct {
	Logger *slog.Logger
	Store  storage.Store
}

// PullImage is a ttRPC service which pulls and mounts docker images.
func (s *ImagePullerService) PullImage(ctx context.Context, r *api.ImagePullRequest) (response *api.ImagePullResponse, retErr error) {
	log := s.Logger.With(
		slog.String("image_url", r.ImageUrl),
		slog.String("bundle_path", r.BundlePath),
	)
	log.Info("Handling image pull request")

	defer func() {
		if retErr != nil {
			log.Error("Request failed", "err", retErr)
		}
	}()

	remoteImg, err := s.getAndVerifyImage(ctx, log, r.ImageUrl)
	if err != nil {
		return nil, fmt.Errorf("obtaining and verifying image: %w", err)
	}
	log.Info("Validated image")

	layers, err := remoteImg.Layers()
	if err != nil {
		return nil, fmt.Errorf("obtaining the image layers: %w", err)
	}

	manifest, err := remoteImg.Manifest()
	if err != nil {
		return nil, fmt.Errorf("obtaining image manifest: %w", err)
	}

	previousLayer := ""
	for idx, layer := range layers {
		rc, err := layer.Compressed()
		if err != nil {
			return nil, fmt.Errorf("reading layer %d: %w", idx, err)
		}

		// Wrap in func to close each rc immediately
		err = func() error {
			putLayer, n, err := s.Store.PutLayer(
				"",            // empty ID -> let store decide
				previousLayer, // parent is previous layer
				nil,           // empty parent chain -> let store decide
				"",            // mount label
				false,         // readonly
				nil,           // mount options
				rc,            // tar stream
			)
			if err != nil {
				return errors.Join(err, rc.Close())
			}
			if err := rc.Close(); err != nil {
				return fmt.Errorf("closing the layer reader: %w", err)
			}

			ldManifest := manifest.Layers[idx].Digest
			ld, err := layer.Digest()
			if err != nil {
				return fmt.Errorf("obtaining digest: %w", err)
			}
			if ldManifest != ld {
				return fmt.Errorf("validation failed, expected digest '%s' but got digest '%s'", ldManifest, ld)
			}

			log.Info("Applied and validated layer", "id", putLayer.ID, "size", n, "digest", ld)
			previousLayer = putLayer.ID
			return nil
		}()
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("pull aborted while applying layer (deadline exceeded): %w", err)
		} else if err != nil {
			return nil, fmt.Errorf("applying layer: %w", err)
		}
	}

	newImg, err := s.Store.CreateImage("", nil, previousLayer, "", nil)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}
	log.Info("Created image", "id", newImg.ID)

	container, err := s.Store.CreateContainer("", nil, newImg.ID, "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("creating container: %w", err)
	}
	log.Info("Created container", "id", container.ID)

	mountPoint, err := s.Store.Mount(container.ID, "")
	if err != nil {
		return nil, fmt.Errorf("mounting container: %w", err)
	}
	log.Debug("Mounted in tmpdir", "mountPoint", mountPoint)

	rootfs := filepath.Join(r.BundlePath, "rootfs")
	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		return nil, fmt.Errorf("creating bundle path: %w", err)
	}

	if err := unix.Mount(mountPoint, rootfs, "", unix.MS_BIND, ""); err != nil {
		return nil, fmt.Errorf("binding mount %s to %s: %w", mountPoint, rootfs, err)
	}
	log.Info("Pulled and mounted image", "mount_path", rootfs)

	return &api.ImagePullResponse{}, nil
}
