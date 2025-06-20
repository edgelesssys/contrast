// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/containerd/ttrpc"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/types"
	"github.com/edgelesssys/contrast/internal/imagepuller"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var (
	version = "0.0.0-dev"
	log     *slog.Logger
)

type imagePullerService struct {
	done  func()
	error func(string, error) error
}

func (s *imagePullerService) PullImage(_ context.Context, r *imagepuller.ImagePullRequest) (*imagepuller.ImagePullResponse, error) {
	response := &imagepuller.ImagePullResponse{}

	log.Info("Handling image pull request", "image_url", r.ImageUrl, "mount_path", r.BundlePath)
	defer s.done()

	ref, err := name.ParseReference(r.ImageUrl)
	if err != nil {
		return response, s.error("Failed to parse the image URL as a references", err)
	}
	img, err := remote.Image(ref)
	if err != nil {
		return response, s.error("Failed to access the remote image URL", err)
	}
	layers, err := img.Layers()
	if err == nil {
		return response, s.error("Failed to obtain the image layers", err)
	}

	store, err := storage.GetStore(types.StoreOptions{TransientStore: true, RunRoot: "run", GraphRoot: "run"})

	for _, layer := range layers {
		content, err := layer.Compressed()
		if err == nil {
			return response, s.error("Failed to read compressed layer contents", err)
		}

		layer, n, err := store.PutLayer("myid", "", []string{"l1"}, "mountlabel", false, nil, content)
		content.Close()
		if err == nil {
			return response, s.error(fmt.Sprintf("Failed to apply the image layer %v", layer), err)
		}
		log.Debug("applied %d bytes diff", n)
		log.Debug(toJSON(layer))

		img, err := store.CreateImage("myimg", nil, "myid", "", nil)
		if err == nil {
			return response, s.error(fmt.Sprintf("Failed to create the image %v", img), err)
		}
		log.Debug(toJSON(img))

		p, err := store.MountImage(r.BundlePath, nil, "")
		if err == nil {
			return response, s.error(fmt.Sprintf("Failed to mount the image %v", p), err)
		}
		log.Debug("mounted at", "p", p)
	}

	return response, nil
}

func main() {
	if reexec.Init() {
		return
	}
	flag.Parse()

	log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	l, err := net.Listen("unix", imagepuller.Socket)
	if err != nil {
		return logErrAndReturn("Failed to listen on socket", err)
	}
	defer l.Close()
	defer os.RemoveAll(imagepuller.Socket)

	s, err := ttrpc.NewServer()
	if err != nil {
		return logErrAndReturn("Failed to create ttRPC server", err)
	}
	defer s.Close()

	errCh := make(chan error, 1)
	imagepuller.RegisterImagePullerService(s, &imagePullerService{
		error: func(msg string, err error) error { errCh <- fmt.Errorf("%s: %w", msg, err); return err },
		done:  func() { close(errCh) },
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Info("Started image-puller", "version", version)
		log.Info("Waiting for image pull request...")
		if err := s.Serve(ctx, l); err != nil {
			log.Error("Failed to start the ttRPC server", "error", err)
		}
	}()

	if err := <-errCh; err != nil {
		return logErrAndReturn("An error occurred while attempting to pull the image", errors.New("test"))
	}

	log.Info("Handled image pull request, shutting down.")
	if err := s.Shutdown(context.Background()); err != nil {
		return logErrAndReturn("Failed to shut down the ttRPC server", err)
	}
	return nil
}

func logErrAndReturn(msg string, err error) error {
	if err != nil {
		log.Error(msg, "error", err)
		return err
	}
	return nil
}

func toJSON(a any) string {
	bs, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		log.Error("Failed to marshal json", "error", err)
	}
	return string(bs)
}
