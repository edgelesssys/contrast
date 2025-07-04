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
	"github.com/containers/storage/types"
	"github.com/edgelesssys/contrast/imagepuller/internal/api"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"golang.org/x/sys/unix"
)

// ImagePullerService is the struct for which the PullImage ttRPC service is implemented.
type ImagePullerService struct {
	Logger *slog.Logger
}

// PullImage is a ttRPC service which pulls and mounts docker images.
func (s *ImagePullerService) PullImage(ctx context.Context, r *api.ImagePullRequest) (_ *api.ImagePullResponse, retErr error) {
	log := s.Logger.With(
		slog.String("image_url", r.ImageUrl),
		slog.String("bundle_path", r.BundlePath),
	)
	log.Info("Handling image pull request")

	defer func() {
		if retErr != nil {
			log.Error("request failed", "err", retErr)
		}
	}()

	ref, err := name.ParseReference(r.ImageUrl)
	if err != nil {
		return nil, fmt.Errorf("parsing the image URL as a references: %w", err)
	}

	tr := transport.NewRetry(remote.DefaultTransport)
	remoteImg, err := remote.Image(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return nil, fmt.Errorf("pull aborted (context cancelled): %w, %w", ctx.Err(), err)
	} else if err != nil {
		return nil, fmt.Errorf("accessing the remote image URL: %w", err)
	}

	layers, err := remoteImg.Layers()
	if err != nil {
		return nil, fmt.Errorf("obtaining the image layers: %w", err)
	}

	store, err := storage.GetStore(types.StoreOptions{
		TransientStore: true,
		RunRoot:        filepath.Join(os.TempDir(), "run"),
		GraphRoot:      filepath.Join(os.TempDir(), "graph"),
	})
	defer store.Shutdown(true)
	if err != nil {
		return nil, fmt.Errorf("opening storage: %w", err)
	}

	var parentIDs []string
	for idx, layer := range layers {
		rc, err := layer.Compressed() // Uncompressed?
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("reading layer %d", idx), err)
		}

		// Wrap in func to close each rc immediately
		err = func() error {
			defer rc.Close()
			putLayer, n, err := store.PutLayer(
				"",        // empty ID -> let store decide
				"",        // parent == top of parentIDs (empty for first)
				parentIDs, // full parent chain
				"",        // mount label
				false,     // readonly
				nil,       // mount options
				rc,        // tar stream
			)
			if err != nil {
				return err
			}
			log.Info("Applied layer", "id", putLayer.ID, "bytes", n)
			parentIDs = append(parentIDs, putLayer.ID)
			return nil
		}()
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return nil, fmt.Errorf("pull aborted while applying layer (context cancelled)", "ctx_err", ctx.Err(), err)
		} else if err != nil {
			return nil, fmt.Errorf("applying layer: %w", err)
		}
	}

	if len(parentIDs) == 0 {
		return nil, fmt.Errorf("no layers found (empty image?)", errors.New("no layers found"))
	}

	newImg, err := store.CreateImage("", nil, parentIDs[len(parentIDs)-1], "", nil)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}
	log.Info("Created image", "id", newImg.ID)

	mountPoint, err := store.MountImage(newImg.ID, nil, "")
	if err != nil {
		return nil, fmt.Errorf("mounting image: %w", err)
	}
	log.Debug("Mounted in tmpdir", "mountPoint", mountPoint)

	imagePath := filepath.Join(r.BundlePath, "rootfs")
	if err := os.MkdirAll(imagePath, 0755); err != nil {
		return nil, fmt.Errorf("creating bundle path: %w", err)
	}
	if err := unix.Unmount(imagePath, 0); err == nil {
		log.Warn("Unmounted existing mount")
	} else if !errors.Is(err, unix.EINVAL) {
		return nil, fmt.Errorf("unmounting existing mount: %w", err)
	}

	lower, upper, work := mountPoint, filepath.Join(imagePath, "upper"), filepath.Join(imagePath, "work")
	data := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)
	if err := os.MkdirAll(upper, 0755); err != nil {
		return nil, fmt.Errorf("creating upper path: %w", err)
	}
	if err := os.MkdirAll(work, 0755); err != nil {
		return nil, fmt.Errorf("creating workdir path: %w", err)
	}

	if err := unix.Mount(mountPoint, imagePath, "overlay", 0, data); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("moving mount %s to %s", mountPoint, imagePath), err)
	}
	log.Info("Pulled and mounted image", "mount_path", imagePath)

	return &api.ImagePullResponse{}, nil
}
