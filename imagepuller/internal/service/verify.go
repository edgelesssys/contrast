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
	ErrParseDigest         = errors.New("parsing image digest")
	ErrDescriptor          = errors.New("obtaining descriptor")
	ErrUnexpectedMediaType = errors.New("unexpected media type")
	ErrRemoteImage         = errors.New("obtaining remote image")
	ErrRemoteIndex         = errors.New("obtaining remote image index")
	ErrMissingPlatform     = errors.New("obtaining image digest for linux/amd64: platform missing from image index")

	ErrObtainingLayers   = errors.New("obtaining the image layers")
	ErrObtainingManifest = errors.New("obtaining image manifest")
	ErrValidateLayer     = errors.New("validating layer")
	ErrReadLayer         = errors.New("reading layer")
	ErrPutLayer          = errors.New("putting layer to store")
)

// Remote allows stubbing remote calls.
type Remote interface {
	Head(ref name.Reference, options ...remote.Option) (*gcr.Descriptor, error)
	Image(ref name.Reference, opts ...remote.Option) (gcr.Image, error)
	Index(ref name.Reference, opts ...remote.Option) (gcr.ImageIndex, error)
}

// DefaultRemote implements Remote, passing all function calls to the remote module.
type DefaultRemote struct{}

// Head returns a gcr.Descriptor for the given reference by issuing a HEAD request.
func (DefaultRemote) Head(ref name.Reference, opts ...remote.Option) (*gcr.Descriptor, error) {
	return remote.Head(ref, opts...)
}

// Image provides access to a remote image reference.
func (DefaultRemote) Image(ref name.Reference, opts ...remote.Option) (gcr.Image, error) {
	return remote.Image(ref, opts...)
}

// Index provides access to a remote index reference.
func (DefaultRemote) Index(ref name.Reference, opts ...remote.Option) (gcr.ImageIndex, error) {
	return remote.Index(ref, opts...)
}

func (s *ImagePullerService) getAndVerifyImage(ctx context.Context, log *slog.Logger, imageURL string) (gcr.Image, error) {
	ref, err := name.NewDigest(imageURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseDigest, err)
	}

	tr := transport.NewRetry(remote.DefaultTransport)

	desc, err := s.Remote.Head(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDescriptor, err)
	}

	var remoteImg gcr.Image
	var imgErr error
	switch {
	case desc.MediaType.IsIndex():
		log.Info("Received manifest list")

		remoteImgIndex, err := s.Remote.Index(ref, remote.WithContext(ctx), remote.WithTransport(tr))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrRemoteIndex, err)
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
			return nil, ErrMissingPlatform
		}
		log.Info("Obtained actual image digest", "image_digest_linux", digestFound.String())

		remoteImg, imgErr = remoteImgIndex.Image(*digestFound)
	case desc.MediaType.IsImage():
		remoteImg, imgErr = s.Remote.Image(ref, remote.WithContext(ctx), remote.WithTransport(tr))
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnexpectedMediaType, desc.MediaType)
	}

	if imgErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrRemoteImage, imgErr)
	}

	return remoteImg, nil
}

func (s *ImagePullerService) storeAndVerifyLayers(log *slog.Logger, remoteImg gcr.Image) (string, error) {
	layers, err := remoteImg.Layers()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrObtainingLayers, err)
	}

	manifest, err := remoteImg.Manifest()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrObtainingManifest, err)
	}

	previousLayer := ""
	for idx, layer := range layers {
		rc, err := layer.Compressed()
		if err != nil {
			return "", fmt.Errorf("%w %d: %w", ErrReadLayer, idx, err)
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
				fmt.Errorf("%w: %w", ErrPutLayer, err),
				fmt.Errorf("closing layer reader: %w", rc.Close()),
			)
		}
		if err := rc.Close(); err != nil {
			return "", fmt.Errorf("closing layer reader: %w", err)
		}

		ldManifest := manifest.Layers[idx].Digest.String()
		ld := putLayer.CompressedDigest.String()
		if ldManifest != ld {
			return "", fmt.Errorf("%w: expected digest '%s' but got digest '%s'", ErrValidateLayer, ldManifest, ld)
		}

		log.Info("Applied and validated layer", "id", putLayer.ID, "size", n, "digest", ld)
		previousLayer = putLayer.ID
	}

	return previousLayer, nil
}
