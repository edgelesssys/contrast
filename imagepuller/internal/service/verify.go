// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
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

func (s *ImagePullerService) storeAndVerifyLayers(log *slog.Logger, remoteImg gcr.Image) (string, error) {
	layers, err := remoteImg.Layers()
	if err != nil {
		return "", fmt.Errorf("obtaining the image layers: %w", err)
	}

	previousLayer := ""
	for idx, layer := range layers {
		rc, err := layer.Compressed()
		if err != nil {
			return "", fmt.Errorf("reading layer %d: %w", idx, err)
		}

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
			return "", errors.Join(
				fmt.Errorf("putting layer to store: %w", err),
				fmt.Errorf("closing layer reader: %w", rc.Close()),
			)
		}
		// Consume any leftover bytes from the reader, mostly to trigger the built-in digest validation.
		if _, err := io.Copy(io.Discard, rc); err != nil {
			return "", errors.Join(
				fmt.Errorf("finalizing layer: %w", err),
				fmt.Errorf("closing layer reader: %w", rc.Close()),
			)
		}
		if err := rc.Close(); err != nil {
			return "", fmt.Errorf("closing layer reader: %w", err)
		}

		log.Info("Applied and validated layer", "id", putLayer.ID, "size", n, "digest", putLayer.CompressedDigest.String())
		previousLayer = putLayer.ID
	}

	return previousLayer, nil
}
