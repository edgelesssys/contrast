// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"

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

// retryBackoff is chosen such that the pull can succeed even if dialing times out for up to a minute.
// TODO(burgerdev): figure out why the network is slow to begin with.
var retryBackoff = transport.Backoff{
	Duration: 5 * time.Second,
	Jitter:   0.1,
	Steps:    12,
}

// retryPredicate fixes the default retry predicate of the transport package for dial timeouts.
//
// It returns true if an error looks transient and could be retried, such as dial timeouts. It
// never returns true for the proper context.DeadlineExceeded error, since it comes from the
// caller and should be respected.
func (s *ImagePullerService) retryPredicate(err error) (ret bool) {
	if err == nil {
		return false
	}
	defer func() {
		s.Logger.Warn("Failed remote call", "error", err, "retriable", ret)
	}()

	if errors.Is(err, context.DeadlineExceeded) {
		// There are at least two errors that enter this block, context.DeadlineExceeded and
		// net.errTimeout. The latter is returned when net.Dial-like functions time out, for
		// historical reasons. To make it compatible with the context package, errTimeout
		// implements Is(DeadlineExceeded), which forces us to look closer into this error and
		// decide whether it's an external deadline, or the internal dial deadline.
		return strings.Contains(err.Error(), "i/o timeout")
	}

	if te, ok := err.(interface{ Temporary() bool }); ok && te.Temporary() {
		return true
	}

	return false
}

func (s *ImagePullerService) getAndVerifyImage(ctx context.Context, log *slog.Logger, imageURL string) (gcr.Image, error) {
	ref, err := name.NewDigest(imageURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParseDigest, err)
	}

	authenticator, transportConfig, err := s.AuthConfig.AuthTransportFor(imageURL, log)
	if err != nil {
		return nil, fmt.Errorf("obtaining authenticator and transport for %s: %w", imageURL, err)
	}

	tr := transport.NewRetry(transportConfig, transport.WithRetryBackoff(retryBackoff), transport.WithRetryPredicate(s.retryPredicate))

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

func (s *ImagePullerService) storeAndVerifyLayers(log *slog.Logger, remoteImg gcr.Image) (id string, retErr error) {
	layers, err := remoteImg.Layers()
	if err != nil {
		return "", fmt.Errorf("obtaining the image layers: %w", err)
	}

	pulledLayers := make([]string, 0, len(layers))
	defer func() {
		if retErr == nil {
			return
		}
		// Clean up before returning an error. The layers need to be removed in reverse order,
		// because later layers are children of earlier layers.
		slices.Reverse(pulledLayers)
		for _, id := range pulledLayers {
			if err := s.Store.DeleteLayer(id); err != nil {
				s.Logger.Error("cleaning layer failed", "id", id, "err", err)
			}
		}
	}()

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
		// Save pulled ID for removal in case of failure.
		pulledLayers = append(pulledLayers, putLayer.ID)

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
