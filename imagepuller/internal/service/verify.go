// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"golang.org/x/sync/errgroup"
)

// Error types which should be differentiable in tests.
var (
	errParseDigest         = errors.New("parsing image digest")
	errUnexpectedMediaType = errors.New("unexpected media type")
	errMissingPlatform     = errors.New("obtaining image digest for linux/amd64: platform missing from image index")
)

func (s *ImagePullerService) getAndVerifyImage(ctx context.Context, log *slog.Logger, imageURL string) (gcr.Image, error) {
	ref, err := name.NewDigest(imageURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParseDigest, err)
	}

	authenticator, transportConfig, err := s.AuthConfig.AuthTransportFor(imageURL)
	if err != nil {
		return nil, fmt.Errorf("obtaining authenticator and transport for %s: %w", imageURL, err)
	}

	tr := transport.NewRetry(transportConfig)

	desc, err := s.Remote.Head(ref, remote.WithContext(ctx), remote.WithTransport(tr), remote.WithAuth(*authenticator))
	if err != nil {
		return nil, fmt.Errorf("obtaining descriptor: %w", err)
	}

	var remoteImg gcr.Image
	var imgErr error
	switch {
	case desc.MediaType.IsIndex():
		log.Info("Received manifest list")

		remoteImgIndex, err := s.Remote.Index(ref, remote.WithContext(ctx), remote.WithTransport(tr), remote.WithAuth(*authenticator))
		if err != nil {
			return nil, fmt.Errorf("obtaining remote image index: %w", err)
		}

		manifest, err := remoteImgIndex.IndexManifest()
		if err != nil {
			return nil, fmt.Errorf("obtaining index manifest: %w", err)
		}

		var digestFound *gcr.Hash
		for _, m := range manifest.Manifests {
			log.Info("MANIFEST", "name", m.Platform.String())
			if m.Platform.String() == "linux/amd64" {
				digestFound = &m.Digest
				break
			}
		}
		if digestFound == nil {
			return nil, errMissingPlatform
		}
		log.Info("Obtained actual image digest", "image_digest_linux", digestFound.String())

		remoteImg, imgErr = remoteImgIndex.Image(*digestFound)
	case desc.MediaType.IsImage():
		remoteImg, imgErr = s.Remote.Image(ref, remote.WithContext(ctx), remote.WithTransport(tr), remote.WithAuth(*authenticator))
	default:
		return nil, fmt.Errorf("%w: %q", errUnexpectedMediaType, desc.MediaType)
	}

	if imgErr != nil {
		return nil, fmt.Errorf("obtaining remote image: %w", imgErr)
	}

	return remoteImg, nil
}

func (s *ImagePullerService) storeLayers(log *slog.Logger, remoteImg gcr.Image) ([]gcr.Hash, error) {
	layers, err := remoteImg.Layers()
	if err != nil {
		return nil, fmt.Errorf("obtaining the image layers: %w", err)
	}

	// TODO(burgerdev): pass a context here?
	g := &errgroup.Group{}
	numProcs := runtime.GOMAXPROCS(0)
	log.Info("storing layers", "num-layers", len(layers), "parallelism", numProcs)
	g.SetLimit(numProcs)

	digests := make([]gcr.Hash, len(layers))

	for idx, layer := range layers {
		g.Go(func() error {
			digest, err := s.Store.PutLayer(layer)
			if err != nil {
				return fmt.Errorf("putting layer to store: %w", err)
			}
			log.Info("Applied and validated layer", "digest", digest.String())
			digests[idx] = digest
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return digests, nil
}
