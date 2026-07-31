// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/imagepuller/internal/auth"
	"github.com/edgelesssys/contrast/imagepuller/internal/remote"
	"github.com/edgelesssys/contrast/imagepuller/internal/test/registry"
	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	gcrremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.podman.io/storage"
)

var zeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

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
			stubRemote: stubRemote{errDescriptor: assert.AnError},
			wantErr:    assert.AnError,
		},
		"remote image failure": {
			digest:     fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef:   "busybox:v0.0.1",
			stubRemote: stubRemote{errRemoteImage: assert.AnError, mediaType: types.DockerManifestSchema2},
			wantErr:    assert.AnError,
		},
		"remote index failure": {
			digest:     fmt.Sprintf("sha256:%s", zeroDigest),
			imageRef:   "busybox:v0.0.1",
			stubRemote: stubRemote{errRemoteIndex: assert.AnError, mediaType: types.DockerManifestList},
			wantErr:    assert.AnError,
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
				mediaType: types.DockerManifestList,
				imageIndex: stubImageIndex{
					errImage: assert.AnError,
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
			wantErr: assert.AnError,
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
			_, err := s.getAndVerifyImage(
				context.Background(),
				log,
				fmt.Sprintf(
					"%s@%s",
					tc.imageRef,
					tc.digest,
				),
			)

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
			stubImg: stubImage{layersErr: assert.AnError},
			wantErr: assert.AnError,
		},
		"manifest fails": {
			stubImg: stubImage{manifestErr: assert.AnError},
			wantErr: assert.AnError,
		},
		"compressed fails": {
			stubImg: stubImage{layersLayers: []gcr.Layer{stubLayer{compressedErr: assert.AnError}}},
			wantErr: assert.AnError,
		},
		"put layer fails": {
			stubImg: stubImage{},
			wantErr: assert.AnError,
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
			realImg, err := s.getAndVerifyImage(
				t.Context(),
				log,
				fmt.Sprintf(
					"%s/busybox:v0.0.1@%s",
					server.Listener.Addr().String(),
					registry.ManifestDigest(),
				),
			)
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
		wantErr string
	}{
		"correct manifest digest, wrong layer digest is caught": {
			digest:  registry.ManifestForWrongBlobDigest(),
			wantErr: "error verifying sha256 checksum",
		},
		"correct manifest digest, wrong layer digest for second layer is caught": {
			digest:  registry.ManifestForWrongBlobDigestTwoLayers(),
			wantErr: "error verifying sha256 checksum",
		},
		"correct index digest, correct manifest digest, wrong layer digest is caught": {
			digest:  registry.IndexForManifestForWrongBlobDigest(),
			wantErr: "error verifying sha256 checksum",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			log := slog.Default()
			server := httptest.NewServer(registry.New())
			t.Cleanup(server.Close)

			store := &fakeStore{}
			s := ImagePullerService{
				Logger: log,
				Store:  store,
				Remote: remote.DefaultRemote{},
			}
			remoteImg, err := s.getAndVerifyImage(
				t.Context(),
				log,
				fmt.Sprintf(
					"%s/busybox:v0.0.1@%s",
					server.Listener.Addr().String(),
					tc.digest,
				),
			)
			require.NoError(err)

			_, err = s.storeAndVerifyLayers(log, remoteImg)

			assert.ErrorContains(err, tc.wantErr)

			// Ensure artifacts were cleaned up after failure.
			if tc.wantErr != "" {
				assert.Empty(store.layers)
			}
		})
	}
}

type fakeStore struct {
	layers map[string]struct{}

	storage.Store
}

func (s *fakeStore) PutLayer(requestedID, _ string, _ []string, _ string, _ bool, _ *storage.LayerOptions, r io.Reader) (*storage.Layer, int64, error) {
	if s.layers == nil {
		s.layers = make(map[string]struct{})
	}
	hash := sha256.New()
	n, err := io.Copy(hash, r)
	if err != nil {
		return nil, n, err
	}
	id := fmt.Sprintf("%x", hash.Sum(nil))
	if requestedID != "" {
		id = requestedID
	}
	s.layers[id] = struct{}{}
	return &storage.Layer{ID: id, CompressedDigest: digest.NewDigest(digest.SHA256, hash)}, n, nil
}

func (s *fakeStore) DeleteLayer(id string) error {
	if _, ok := s.layers[id]; !ok {
		return fmt.Errorf("DeleteLayer for non-existent id: %q", id)
	}
	delete(s.layers, id)
	return nil
}

// TestRetry ensures that dial timeouts are retried.
func TestRetry(t *testing.T) {
	require := require.New(t)
	ctx := t.Context()
	log := slog.New(slog.DiscardHandler)

	// 192.0.2.1 is not routable, packets should be dropped.
	// https://datatracker.ietf.org/doc/html/rfc5737
	img := "192.0.2.1/foo:latest"
	ref, err := name.NewTag(img)
	require.NoError(err)

	// Now configure the gcr method as close to the ImagePullerService as possible.
	// First, the HTTP transport, but with a dialer that returns a timeout on first try.
	authConfig := &auth.Config{
		Registries: map[string]auth.Registry{
			".": {InsecureSkipVerify: true},
		},
	}
	_, rt, err := authConfig.AuthTransportFor(img, log)
	require.NoError(err)

	tr, ok := rt.(*http.Transport)
	require.True(ok)
	originalDialer := tr.DialContext
	var failed bool
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if failed {
			// Marker error to check whether dialing was retried.
			return nil, assert.AnError
		}
		failed = true
		// Simulate dial timeout by setting a deadline in the past.
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Hour))
		defer cancel()
		return originalDialer(ctx, network, addr)
	}

	// Second, the backoff strategy. We don't want to wait forever in the unit test, so just use a
	// short non-exponential backoff with two attempts.
	backOff := transport.Backoff{
		Duration: time.Microsecond,
		Steps:    2,
	}

	// Third, the retry predicate without any modifications.
	s := &ImagePullerService{
		Logger: log,
	}
	retryTransport := transport.NewRetry(tr, transport.WithRetryBackoff(backOff), transport.WithRetryPredicate(s.retryPredicate))

	_, err = gcrremote.Head(ref, gcrremote.WithContext(ctx), gcrremote.WithTransport(retryTransport))
	require.ErrorIs(err, assert.AnError)
}

func TestPredicate(t *testing.T) {
	for name, tc := range map[string]struct {
		err       error
		wantRetry bool
	}{
		"no error":          {},
		"deadline exceeded": {err: context.DeadlineExceeded},
		"i/o timeout":       {err: &timeoutError{}, wantRetry: true},
		"unrelated":         {err: assert.AnError},
		"temporary":         {err: &net.DNSError{IsTemporary: true}, wantRetry: true},
	} {
		t.Run(name, func(t *testing.T) {
			s := &ImagePullerService{
				Logger: slog.New(slog.DiscardHandler),
			}

			got := s.retryPredicate(tc.err)
			require.Equal(t, tc.wantRetry, got)
		})
	}
}

// From Go src/net/net.go, since not exported.
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func (e *timeoutError) Is(err error) bool {
	return err == context.DeadlineExceeded
}
