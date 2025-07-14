// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	digestExpected := ref.DigestStr()

	tr := transport.NewRetry(remote.DefaultTransport)

	var remoteImg v1.Image
	var imgErr error
	remoteImgIndex, err := remote.Index(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	if err == nil {
		log.Info("Received manifest list")

		digestGot, err := remoteImgIndex.Digest()
		if err != nil {
			return nil, fmt.Errorf("obtaining actual image index digest: %w", err)
		}
		if digestGot.String() != digestExpected {
			return nil, fmt.Errorf("validating image index: expected digest '%s', got digest '%s'", digestExpected, digestGot)
		}
		log.Info("Validated image index digest")

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

		digestExpected = digestFound.String()
		log.Info("Obtained actual image digest", "image_digest_linux", digestExpected)

		remoteImg, imgErr = remoteImgIndex.Image(digestFound)
	} else {
		remoteImg, imgErr = remote.Image(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	}

	if errors.Is(imgErr, context.DeadlineExceeded) {
		return nil, fmt.Errorf("pull aborted (deadline exceeded): %w", imgErr)
	} else if imgErr != nil {
		return nil, fmt.Errorf("accessing the remote image URL: %w", imgErr)
	}

	digestGot, err := remoteImg.Digest()
	if err != nil {
		return nil, fmt.Errorf("obtaining actual image digest: %w", err)
	}
	if digestGot.String() != digestExpected {
		return nil, fmt.Errorf("validating image: expected digest '%s', got digest '%s'", digestExpected, digestGot)
	}

	return remoteImg, nil
}
