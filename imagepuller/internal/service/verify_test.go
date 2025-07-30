// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/containers/storage"
	"github.com/edgelesssys/contrast/imagepuller/internal/test/registry"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type StubStore struct {
	putLayerLayer digest.Digest
	storage.Store
}

func (s *StubStore) PutLayer(_, _ string, _ []string, _ string, _ bool, _ *storage.LayerOptions, _ io.Reader) (*storage.Layer, int64, error) {
	return &storage.Layer{CompressedDigest: s.putLayerLayer}, 0, nil
}

func TestGetAndVerifyImage_EvilRegistry(t *testing.T) {
	tests := map[string]struct {
		digest  string
		wantErr error
	}{
		"missing digest is rejected": {
			digest:  "",
			wantErr: ErrParseDigest,
		},
		"wrong manifest digest is caught": {
			digest:  registry.WrongManifestDigest(),
			wantErr: ErrRemoteImage,
		},
		"wrong index digest is caught": {
			digest:  registry.WrongIndexDigest(),
			wantErr: ErrRemoteIndex,
		},
		"correct index digest, wrong manifest digest is caught": {
			digest:  registry.IndexForWrongManifestDigest(),
			wantErr: ErrRemoteImage,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			server := httptest.NewServer(registry.New())
			t.Cleanup(server.Close)

			var s ImagePullerService
			_, err := s.getAndVerifyImage(
				t.Context(),
				slog.Default(),
				fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), tc.digest),
			)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}

func TestStoreAndVerifyLayers_EvilRegistry(t *testing.T) {
	tests := map[string]struct {
		digest  string
		wantErr error
	}{
		"correct manifest digest, wrong layer digest is caught": {
			digest:  registry.ManifestForWrongBlobDigest(),
			wantErr: ErrValidateLayer,
		},
		"correct index digest, correct manifest digest, wrong layer digest is caught": {
			digest:  registry.IndexForManifestForWrongBlobDigest(),
			wantErr: ErrValidateLayer,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			log := slog.Default()
			server := httptest.NewServer(registry.New())
			t.Cleanup(server.Close)

			s := ImagePullerService{Logger: log, Store: &StubStore{
				putLayerLayer: digest.NewDigestFromBytes(digest.SHA256, []byte{}),
			}}
			remoteImg, err := s.getAndVerifyImage(t.Context(), log, fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), tc.digest))
			require.NoError(err)

			_, err = s.storeAndVerifyLayers(log, remoteImg)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}
