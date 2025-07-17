// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

func (s *ImagePullerService) getAndVerifyImage(ctx context.Context, log *slog.Logger, imageURL string) (v1.Image, error) {
	ref, err := name.NewDigest(imageURL)
	if err != nil {
		return nil, fmt.Errorf("parsing image digest: %w", err)
	}

	tr := transport.NewRetry(remote.DefaultTransport)

	remoteImgIndex, err := remote.Index(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	if isDigestMismatch(err) {
		return nil, fmt.Errorf("validating image ref: %w", err)
	}

	var remoteImg v1.Image
	var imgErr error
	if err == nil {
		log.Info("Received manifest list")

		manifest, err := remoteImgIndex.IndexManifest()
		if err != nil {
			return nil, fmt.Errorf("obtaining index manifest: %w", err)
		}

		var digestFound v1.Hash
		for _, m := range manifest.Manifests {
			if m.Platform.String() == "linux/amd64" {
				digestFound = m.Digest
				break
			}
		}

		if digestFound.String() == "" {
			return nil, fmt.Errorf("obtaining image digest for linux/amd64: platform missing from image index")
		}
		log.Info("Obtained actual image digest", "image_digest_linux", digestFound.String())

		remoteImg, imgErr = remoteImgIndex.Image(digestFound)
	} else {
		remoteImg, imgErr = remote.Image(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	}

	if isDigestMismatch(imgErr) {
		return nil, fmt.Errorf("validating image: %w", imgErr)
	} else if errors.Is(imgErr, context.DeadlineExceeded) {
		return nil, fmt.Errorf("pull aborted (deadline exceeded): %w", imgErr)
	} else if imgErr != nil {
		return nil, fmt.Errorf("accessing the remote image URL: %w", imgErr)
	}

	return remoteImg, nil
}

func (s *ImagePullerService) storeAndVerifyLayers(log *slog.Logger, remoteImg v1.Image) (string, error) {
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
		if err := rc.Close(); err != nil {
			return "", fmt.Errorf("closing layer reader: %w", err)
		}

		ldManifest := manifest.Layers[idx].Digest.String()
		ld := putLayer.CompressedDigest.String()
		if ldManifest != ld {
			return "", fmt.Errorf("validating layer: expected digest '%s' but got digest '%s'", ldManifest, ld)
		}

		log.Info("Applied and validated layer", "id", putLayer.ID, "size", n, "digest", ld)
		previousLayer = putLayer.ID
	}

	return previousLayer, nil
}

func isDigestMismatch(err error) bool {
	return err != nil && strings.Contains(err.Error(), "does not match requested digest")
}
