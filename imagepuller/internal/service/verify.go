// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
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
	errValidateLayer       = errors.New("validating layer")
)

func (s *ImagePullerService) getAndVerifyImage(ctx context.Context, log *slog.Logger, imageURL string) (gcr.Image, error) {
	ref, err := name.NewDigest(imageURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParseDigest, err)
	}

	tr := transport.NewRetry(remote.DefaultTransport)

	desc, err := s.Remote.Head(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	if err != nil {
		return nil, fmt.Errorf("obtaining descriptor: %w", err)
	}

	var remoteImg gcr.Image
	var imgErr error
	switch {
	case desc.MediaType.IsIndex():
		log.Info("Received manifest list")

		remoteImgIndex, err := s.Remote.Index(ref, remote.WithContext(ctx), remote.WithTransport(tr))
		if err != nil {
			return nil, fmt.Errorf("obtaining remote image index: %w", err)
		}

		manifest, err := remoteImgIndex.IndexManifest()
		if err != nil {
			return nil, fmt.Errorf("obtaining index manifest: %w", err)
		}

		var digestFound *gcr.Hash
		for _, m := range manifest.Manifests {
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
		remoteImg, imgErr = s.Remote.Image(ref, remote.WithContext(ctx), remote.WithTransport(tr))
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

	manifest, err := remoteImg.Manifest()
	if err != nil {
		return "", fmt.Errorf("obtaining image manifest: %w", err)
	}

	previousLayer := ""
	for idx, layer := range layers {
		layerDigest := manifest.Layers[idx].Digest.String()

		cachedID, err := s.Store.Lookup(layerDigest)
		if err == nil {
			log.Info("Reusing cached layer", "cache_id", cachedID, "digest", layerDigest)
			previousLayer = cachedID
			continue
		}

		rc, err := layer.Compressed()
		if err != nil {
			return "", fmt.Errorf("reading layer %d: %w", idx, err)
		}

		putLayer, n, err := s.Store.PutLayer(
			"",                    // empty ID -> let store decide
			previousLayer,         // parent is previous layer
			[]string{layerDigest}, // set layer digest as name for cache retrieval
			"",                    // mount label
			false,                 // readonly
			nil,                   // mount options
			rc,                    // tar stream
		)
		if err != nil {
			return "", errors.Join(
				fmt.Errorf("putting layer to store: %w", err),
				fmt.Errorf("closing layer reader: %w", rc.Close()),
			)
		}
		if err := rc.Close(); err != nil {
			return "", fmt.Errorf("closing layer reader: %w", err)
		}

		ld := putLayer.CompressedDigest.String()
		if layerDigest != ld {
			return "", fmt.Errorf("%w: expected digest '%s' but got digest '%s'", errValidateLayer, layerDigest, ld)
		}

		log.Info("Applied and validated layer", "id", putLayer.ID, "size", n, "digest", ld)
		previousLayer = putLayer.ID
	}

	return previousLayer, nil
}
