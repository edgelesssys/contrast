// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"encoding/json"
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
	"golang.org/x/sys/unix"
)

type ImagePullerService struct {
	Error  func(string, error) error
	Logger *slog.Logger
}

func (s *ImagePullerService) PullImage(ctx context.Context, r *api.ImagePullRequest) (*api.ImagePullResponse, error) {
	log := s.Logger.With(
		slog.String("image_url", r.ImageUrl),
		slog.String("mount_path", r.BundlePath),
	)
	log.Info("Handling image pull request")

	ref, err := name.ParseReference(r.ImageUrl)
	if err != nil {
		return nil, s.Error("Failed to parse the image URL as a references", err)
	}
	remoteImg, err := remote.Image(ref, remote.WithContext(ctx))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return nil, s.Error("pull aborted (context cancelled)", ctx.Err())
		}
		return nil, s.Error("Failed to access the remote image URL", err)
	}
	layers, err := remoteImg.Layers()
	if err != nil {
		fmt.Println(layers)
		return nil, s.Error("Failed to obtain the image layers", err)
	}

	store, err := storage.GetStore(types.StoreOptions{
		TransientStore: true,
		RunRoot:        filepath.Join(os.TempDir(), "run"),
		GraphRoot:      filepath.Join(os.TempDir(), "graph"),
	})
	defer store.Shutdown(true)
	if err != nil {
		return nil, s.Error("Failed to open storage", err)
	}

	var parentIDs []string
	for idx, layer := range layers {
		content, err := layer.Compressed() // Uncompressed?
		defer content.Close()
		if err != nil {
			return nil, s.Error(fmt.Sprintf("Failed to read layer %d", idx), err)
		}

		putLayer, n, err := store.PutLayer(
			"",        // empty ID -> let store decide
			"",        // parent == top of parentIDs (empty for first)
			parentIDs, // full parent chain
			"",        // mount label
			false,     // readonly
			nil,       // mount options
			content,   // tar stream
		)
		if err != nil {
			return nil, s.Error("Failed to apply layer", err)
		}
		log.Debug("applied %d bytes diff", n)
		log.Debug(toJSON(putLayer))
		log.Info("Applied layer", "id", putLayer.ID, "bytes", n)

		parentIDs = append(parentIDs, putLayer.ID)
	}

	if len(parentIDs) == 0 {
		return nil, s.Error("No layers found (empty image?)", errors.New("no layers found"))
	}

	newImg, err := store.CreateImage("", nil, parentIDs[len(parentIDs)-1], "", nil)
	if err != nil {
		return nil, s.Error("Failed to create image", err)
	}
	log.Debug(toJSON(newImg))
	log.Info("Created image", "id", newImg.ID)

	mountPoint, err := store.MountImage(newImg.ID, nil, "")
	if err != nil {
		return nil, s.Error("Failed to mount image", err)
	}
	log.Debug("mounted at", "mountPoint", mountPoint)

	if err := os.MkdirAll(r.BundlePath, 0755); err != nil {
		return nil, s.Error("Failed to create bundle path", err)
	}

	if err := unix.Mount(mountPoint, r.BundlePath, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return nil, s.Error(fmt.Sprintf("Failed to bind-mount image to %s", r.BundlePath), err)
	}

	log.Info("Pulled and mounted image", "image", r.ImageUrl)
	return &api.ImagePullResponse{}, nil
}

func toJSON(a any) string {
	bs, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal json: %s", err)
	}
	return string(bs)
}
