// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"github.com/edgelesssys/contrast/imagepuller/internal/api"
	"github.com/edgelesssys/contrast/imagepuller/internal/auth"
	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"go.podman.io/storage"
	"go.podman.io/storage/types"
)

// Remote allows stubbing remote calls.
type Remote interface {
	Head(ref name.Reference, options ...gcrRemote.Option) (*gcr.Descriptor, error)
	Image(ref name.Reference, opts ...gcrRemote.Option) (gcr.Image, error)
	Index(ref name.Reference, opts ...gcrRemote.Option) (gcr.ImageIndex, error)
}

// ImagePullerService is the struct for which the PullImage ttRPC service is implemented.
type ImagePullerService struct {
	Logger            *slog.Logger
	Store             storage.Store
	StorePathOverride string
	Remote            Remote
	AuthConfig        auth.Config
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

	var storePath string
	if s.StorePathOverride != "" {
		storePath = s.StorePathOverride
	} else if _, err := os.Stat(api.StorePathStorage); err == nil {
		storePath = api.StorePathStorage
	} else {
		storePath = api.StorePathMemory
	}
	store, err := storage.GetStore(types.StoreOptions{
		TransientStore:  true,
		DisableVolatile: false,
		RunRoot:         filepath.Join(storePath, "run"),
		GraphRoot:       filepath.Join(storePath, "graph"),
	})
	log.Info("Found or created store", "storage_dir", storePath)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}
	s.Store = store

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

	requiredStorage, err := s.minimumRequiredStorage(remoteImg)
	if err != nil {
		return nil, fmt.Errorf("determining required storage: %w", err)
	}
	availableStorage, err := s.availableStorage()
	if err != nil {
		return nil, fmt.Errorf("determining available storage: %w", err)
	}
	if availableStorage < requiredStorage {
		return nil, fmt.Errorf("insufficient storage: pulling %q would require at least %s, but only %s are currently available. Increase the memory limit or image store size",
			r.ImageUrl,
			formatBytes(requiredStorage),
			formatBytes(availableStorage),
		)
	}

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

func (s *ImagePullerService) minimumRequiredStorage(remoteImg gcr.Image) (uint64, error) {
	manifest, err := remoteImg.Manifest()
	if err != nil {
		return 0, err
	}

	total := uint64(0)
	for _, layer := range manifest.Layers {
		total += uint64(layer.Size)
	}

	return total, nil
}

func (s *ImagePullerService) availableStorage() (uint64, error) {
	path := s.Store.GraphRoot()

	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}

	// Available blocks * size per block = available bytes
	return stat.Bavail * uint64(stat.Bsize), nil
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	const unitNames = "kMGTPE"
	x, exp := float64(bytes)/unit, 0
	for x > unit && exp < len(unitNames) {
		x /= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", x, unitNames[exp])
}
