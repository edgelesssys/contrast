// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/containers/storage"
	"github.com/edgelesssys/contrast/imagepuller/internal/api"
)

// ContainersStorageStore defines the functions used from the storage.Store interface to allow for easier testing.
type ContainersStorageStore interface {
	AddNames(id string, names []string) error
	CreateContainer(id string, names []string, image, layer, metadata string, options *storage.ContainerOptions) (*storage.Container, error)
	CreateImage(id string, names []string, layer, metadata string, options *storage.ImageOptions) (*storage.Image, error)
	Lookup(name string) (string, error)
	Mount(id, mountLabel string) (string, error)
	PutLayer(id, parent string, names []string, mountLabel string, writeable bool, options *storage.LayerOptions, diff io.Reader) (*storage.Layer, int64, error)
	RemoveNames(id string, names []string) error
}

// ImagePullerService is the struct for which the PullImage ttRPC service is implemented.
type ImagePullerService struct {
	Logger *slog.Logger
	Store  ContainersStorageStore
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

	cachedID, err := s.Store.Lookup(r.ImageUrl)
	if err == nil {
		rootfs, err := s.createAndMountContainer(log, cachedID, r.BundlePath)
		if err != nil {
			return nil, fmt.Errorf("mounting container from cached image: %w", err)
		}
		log.Info("Mounted container from cached image", "mount_path", rootfs)
		return &api.ImagePullResponse{}, nil
	}

	remoteImg, err := s.getAndVerifyImage(ctx, log, r.ImageUrl)
	if err != nil {
		return nil, fmt.Errorf("obtaining and verifying image: %w", err)
	}
	log.Info("Validated image")

	finalLayer, err := s.storeAndVerifyLayers(log, remoteImg)
	if err != nil {
		return nil, fmt.Errorf("verifying and putting layers in store: %w", err)
	}
	log.Info("Verified and put in store layers")

	newImg, err := s.Store.CreateImage("", nil, finalLayer, "", nil)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}
	log.Info("Created image", "id", newImg.ID)

	if err := s.Store.RemoveNames(newImg.ID, newImg.Names); err != nil {
		return nil, fmt.Errorf("removing pre-existing image names: %w", err)
	}
	if err := s.Store.AddNames(newImg.ID, []string{r.ImageUrl}); err != nil {
		return nil, fmt.Errorf("adding image url as image name: %w", err)
	}

	rootfs, err := s.createAndMountContainer(log, newImg.ID, r.BundlePath)
	if err != nil {
		return nil, fmt.Errorf("mounting container: %w", err)
	}
	log.Info("Pulled and mounted image", "mount_path", rootfs)

	return &api.ImagePullResponse{}, nil
}
