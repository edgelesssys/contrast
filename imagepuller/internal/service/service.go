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
	"strings"

	"github.com/containers/storage"
	"github.com/containers/storage/types"
	"github.com/edgelesssys/contrast/imagepuller/internal/api"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/opencontainers/go-digest"
	"golang.org/x/sys/unix"
)

// ImagePullerService is the struct for which the PullImage ttRPC service is implemented.
type ImagePullerService struct {
	Logger *slog.Logger
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

	ref, err := name.ParseReference(r.ImageUrl)
	if err != nil {
		return nil, fmt.Errorf("parsing the image URL as a references: %w", err)
	}

	tr := transport.NewRetry(remote.DefaultTransport)
	remoteImg, err := remote.Image(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("pull aborted (deadline exceeded): %w", err)
	} else if err != nil {
		return nil, fmt.Errorf("accessing the remote image URL: %w", err)
	}

	digestExpected, err := newDigest(r.ImageUrl)
	if err != nil {
		return nil, fmt.Errorf("parsing image digest: %w", err)
	}
	digestGot, err := remoteImg.Digest()
	if err != nil {
		return nil, fmt.Errorf("obtaining actual image digest: %w", err)
	}
	if digestGot.String() != digestExpected {
		return nil, fmt.Errorf("validating image: expected digest '%s', got digest '%s'", digestExpected, digestGot)
	}
	log.Info("Validated image digest")

	layers, err := remoteImg.Layers()
	if err != nil {
		return nil, fmt.Errorf("obtaining the image layers: %w", err)
	}

	store, err := storage.GetStore(types.StoreOptions{
		TransientStore: true,
		RunRoot:        filepath.Join(os.TempDir(), "run"),
		GraphRoot:      filepath.Join(os.TempDir(), "graph"),
	})
	if err != nil {
		return nil, fmt.Errorf("opening storage: %w", err)
	}

	manifest, err := remoteImg.Manifest()
	if err != nil {
		return nil, fmt.Errorf("obtaining image manifest: %w", err)
	}

	var parentIDs []string
	for idx, layer := range layers {
		rc, err := layer.Compressed()
		if err != nil {
			return nil, fmt.Errorf("reading layer %d: %w", idx, err)
		}

		// Wrap in func to close each rc immediately
		err = func() error {
			putLayer, n, err := store.PutLayer(
				"",        // empty ID -> let store decide
				"",        // empty parent ID -> let store decide
				parentIDs, // full parent chain
				"",        // mount label
				false,     // readonly
				nil,       // mount options
				rc,        // tar stream
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
			parentIDs = append(parentIDs, putLayer.ID)
			return nil
		}()
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("pull aborted while applying layer (deadline exceeded): %w", err)
		} else if err != nil {
			return nil, fmt.Errorf("applying layer: %w", err)
		}
	}

	newImg, err := store.CreateImage("", nil, parentIDs[len(parentIDs)-1], "", nil)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}
	log.Info("Created image", "id", newImg.ID)

	container, err := store.CreateContainer("", nil, newImg.ID, "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("creating container: %w", err)
	}
	log.Info("Created container", "id", container.ID)

	mountPoint, err := store.Mount(container.ID, "")
	if err != nil {
		return nil, fmt.Errorf("mounting container: %w", err)
	}
	log.Debug("Mounted in tmpdir", "mountPoint", mountPoint)

	imagePath := filepath.Join(r.BundlePath, "rootfs")
	if err := os.MkdirAll(imagePath, 0o755); err != nil {
		return nil, fmt.Errorf("creating bundle path: %w", err)
	}

	if err := unix.Mount(mountPoint, imagePath, "", unix.MS_BIND, ""); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("binding mount %s to %s", mountPoint, imagePath), err)
	}
	log.Info("Pulled and mounted image", "mount_path", imagePath)

	return &api.ImagePullResponse{}, nil
}

func newDigest(name string) (string, error) {
	parts := strings.Split(name, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("a digest must contain exactly one '@' separator (e.g. registry/repository@digest) saw: %s", name)
	}
	dig := parts[1]
	prefix := digest.Canonical.String() + ":"
	if !strings.HasPrefix(dig, prefix) {
		return "", fmt.Errorf("unsupported digest algorithm: %s", dig)
	}
	hex := strings.TrimPrefix(dig, prefix)
	if err := digest.Canonical.Validate(hex); err != nil {
		return "", err
	}

	return dig, nil
}
