// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/imagepuller/internal/remote"
	"github.com/edgelesssys/contrast/imagepuller/internal/test/registry"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var zeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

var (
	errDescriptor        = errors.New("descriptor")
	errRemoteImage       = errors.New("remote image")
	errRemoteIndex       = errors.New("remote index")
	errCompressed        = errors.New("compressed layer")
	errObtainingLayers   = errors.New("remote index")
	errObtainingManifest = errors.New("compressed layer")
	errPutLayer          = errors.New("put layer")
)

// TestGetAndVerifyImage contains unit tests for the getAndVerifyImage function.
func TestGetAndVerifyImage(t *testing.T) {
	tests := map[string]struct {
		digest     string
		imageRef   string
		wantErr    error
		stubRemote stubRemote
	}{
		"digest missing, no tag": {
			imageRef: "busybox",
			wantErr:  errParseDigest,
		},
		"digest malformed, no tag": {
			digest:   "sha256:000",
			imageRef: "busybox",
			wantErr:  errParseDigest,
		},
		"digest missing algorithm, no tag": {
			digest:   zeroDigest,
			imageRef: "busybox",
			wantErr:  errParseDigest,
		},
		"digest missing, with tag": {
			imageRef: "busybox:v0.0.1",
			wantErr:  errParseDigest,
		},
		"digest malformed, with tag": {
			digest:   "sha256:000",
			imageRef: "busybox:v0.0.1",
			wantErr:  errParseDigest,
		},
		"digest missing algorithm, with tag": {
			digest:   zeroDigest,
			imageRef: "busybox:v0.0.1",
			wantErr:  errParseDigest,
		},
		"head request error": {
			digest:     fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef:   "busybox:v0.0.1",
			stubRemote: stubRemote{errDescriptor: errDescriptor},
			wantErr:    errDescriptor,
		},
		"remote image failure": {
			digest:     fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef:   "busybox:v0.0.1",
			stubRemote: stubRemote{errRemoteImage: errRemoteImage, mediaType: types.DockerManifestSchema2},
			wantErr:    errRemoteImage,
		},
		"remote index failure": {
			digest:     fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef:   "busybox:v0.0.1",
			stubRemote: stubRemote{errRemoteIndex: errRemoteIndex, mediaType: types.DockerManifestList},
			wantErr:    errRemoteIndex,
		},
		"unexpected media type": {
			digest:     fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef:   "busybox:v0.0.1",
			wantErr:    errUnexpectedMediaType,
			stubRemote: stubRemote{mediaType: types.DockerForeignLayer},
		},
		"index valid, linux/amd64 missing": {
			digest:     fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef:   "busybox:v0.0.1",
			wantErr:    errMissingPlatform,
			stubRemote: stubRemote{mediaType: types.DockerManifestList},
		},
		"index remote image failure": {
			digest:   fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef: "busybox:v0.0.1",
			stubRemote: stubRemote{
				errRemoteImage: errRemoteImage,
				mediaType:      types.DockerManifestList,
				imageIndex: stubImageIndex{
					errImage: errRemoteImage,
					indexManifestIndexManifest: &gcr.IndexManifest{
						Manifests: []gcr.Descriptor{{
							Platform: &gcr.Platform{
								OS:           "linux",
								Architecture: "amd64",
							},
							Digest: gcr.Hash{
								Algorithm: "sha256",
								Hex:       "",
							},
						}},
					},
				},
			},
			wantErr: errRemoteImage,
		},
		"success, simple manifest": {
			digest:     fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef:   "busybox:v0.0.1",
			stubRemote: stubRemote{mediaType: types.DockerManifestSchema2},
		},
		"success, index pointing to manifest": {
			digest:   fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef: "busybox:v0.0.1",
			stubRemote: stubRemote{
				mediaType: types.DockerManifestList,
				imageIndex: stubImageIndex{
					indexManifestIndexManifest: &gcr.IndexManifest{
						Manifests: []gcr.Descriptor{{
							Platform: &gcr.Platform{
								OS:           "linux",
								Architecture: "amd64",
							},
							Digest: gcr.Hash{
								Algorithm: "sha256",
								Hex:       "",
							},
						}},
					},
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			log := slog.Default()

			s := ImagePullerService{Remote: tc.stubRemote}
			_, err := s.getAndVerifyImage(context.Background(), log, fmt.Sprintf("%s@%s", tc.imageRef, tc.digest))

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}

// TestStoreAndVerifyLayers contains unit tests for the storeAndVerifyLayers function.
func TestStoreAndVerifyLayers(t *testing.T) {
	tests := map[string]struct {
		stubImg    stubImage
		stubRemote stubRemote
		wantErr    error
	}{
		"layers fails": {
			stubImg: stubImage{layersErr: errObtainingLayers},
			wantErr: errObtainingLayers,
		},
		"manifest fails": {
			stubImg: stubImage{manifestErr: errObtainingManifest},
			wantErr: errObtainingManifest,
		},
		"compressed fails": {
			stubImg: stubImage{layersLayers: []gcr.Layer{stubLayer{compressedErr: errCompressed}}},
			wantErr: errCompressed,
		},
		"put layer fails": {
			stubImg: stubImage{},
			wantErr: errPutLayer,
		},
		"success": {
			stubImg: stubImage{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			log := slog.Default()
			server := httptest.NewServer(registry.New())
			t.Cleanup(server.Close)

			t.Log(registry.BlobDigest())

			store := &stubStore{
				putLayerDigest: digest.NewDigestFromEncoded(digest.SHA256, registry.BlobDigest()[7:]),
				putLayerErr:    tc.wantErr,
			}
			s := ImagePullerService{Logger: log, Store: store, Remote: tc.stubRemote.DefaultRemote}
			realImg, err := s.getAndVerifyImage(t.Context(), log, fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), registry.ManifestDigest()))
			require.NoError(err)

			tc.stubImg.Image = realImg
			_, err = s.storeAndVerifyLayers(log, tc.stubImg)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}

// TestStoreAndVerifyLayers_EvilRegistry contains integration tests for the storeAndVerifyLayers function.
// Unlike the unittests for this function, responses of the evil registry depend on test parameters.
// This allows testing the behavior against arbitrary (evil) responses.
func TestStoreAndVerifyLayers_EvilRegistry(t *testing.T) {
	tests := map[string]struct {
		digest  string
		wantErr error
	}{
		"correct manifest digest, wrong layer digest is caught": {
			digest:  registry.ManifestForWrongBlobDigest(),
			wantErr: errValidateLayer,
		},
		"correct index digest, correct manifest digest, wrong layer digest is caught": {
			digest:  registry.IndexForManifestForWrongBlobDigest(),
			wantErr: errValidateLayer,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			log := slog.Default()
			server := httptest.NewServer(registry.New())
			t.Cleanup(server.Close)

			s := ImagePullerService{
				Logger: log,
				Store: &stubStore{
					putLayerDigest: digest.NewDigestFromBytes(digest.SHA256, []byte{}),
				},
				Remote: remote.DefaultRemote{},
			}
			remoteImg, err := s.getAndVerifyImage(t.Context(), log, fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), tc.digest))
			require.NoError(err)

			_, err = s.storeAndVerifyLayers(log, remoteImg)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}
