// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/imagepuller/internal/api"
	"github.com/edgelesssys/contrast/imagepuller/internal/auth"
	"github.com/edgelesssys/contrast/imagepuller/internal/store"
	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/opencontainers/umoci/oci/layer"
)

// Remote allows stubbing remote calls.
type Remote interface {
	Head(ref name.Reference, options ...gcrRemote.Option) (*gcr.Descriptor, error)
	Image(ref name.Reference, opts ...gcrRemote.Option) (gcr.Image, error)
	Index(ref name.Reference, opts ...gcrRemote.Option) (gcr.ImageIndex, error)
}

type Store interface {
	Mount(targetPath string, layerDigests ...string) error
	PutLayer(gcr.Layer) (retErr error)
}

// ImagePullerService is the struct for which the PullImage ttRPC service is implemented.
type ImagePullerService struct {
	Logger            *slog.Logger
	Store             Store
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
	store := &store.Store{
		Unpacker: &unpacker{},
		Root:     storePath,
	}
	log.Info("Created store", "storage_dir", storePath)
	s.Store = store

	// TODO(burgerdev): cache manifests

	remoteImg, err := s.getAndVerifyImage(ctx, log, r.ImageUrl)
	if err != nil {
		return nil, fmt.Errorf("obtaining and verifying image: %w", err)
	}
	log.Info("Validated image")

	if err := s.storeAndVerifyLayers(log, remoteImg); err != nil {
		return nil, fmt.Errorf("verifying and putting layers in store: %w", err)
	}
	log.Info("Verified and put in store layers")

	var layers []string
	remoteLayers, err := remoteImg.Layers()
	if err != nil {
		return nil, fmt.Errorf("listing layers: %w", err)
	}
	for i, l := range remoteLayers {
		digest, err := l.Digest()
		if err != nil {
			return nil, fmt.Errorf("getting digest of layer %d: %w", i, err)
		}
		layers = append(layers, digest.String())
	}

	rootfs := filepath.Join(r.BundlePath, "rootfs")
	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		return nil, fmt.Errorf("creating directory %q: %w", rootfs, err)
	}
	if err := s.Store.Mount(rootfs, layers...); err != nil {
		return nil, fmt.Errorf("mounting container: %w", err)
	}
	log.Info("Pulled and mounted image", "mount_path", rootfs)

	return &api.ImagePullResponse{}, nil
}

type unpacker struct{}

func (*unpacker) UnpackLayer(root string, content io.Reader, opt *layer.UnpackOptions) error {
	return layer.UnpackLayer(root, content, opt)
}
