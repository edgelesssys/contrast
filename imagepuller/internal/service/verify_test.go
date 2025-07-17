// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/burgerdev/evil-registry/registry"
	"github.com/containers/storage"
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
		wantErr string
	}{
		"missing digest is rejected": {
			digest:  "",
			wantErr: "parsing image digest",
		},
		"wrong manifest digest is caught": {
			// the evil registry responds to unknown digests with a default manifest
			digest:  "sha256:6ad6bbb5735b84b10af42d2441e8d686b1d9a6cbf096b53842711ef5ddabd28d",
			wantErr: "validating image ref:",
		},
		"wrong index digest is caught": {
			digest:  registry.WrongIndexDigest,
			wantErr: "validating image ref:",
		},
		"correct index digest, wrong manifest digest is caught": {
			digest:  registry.IndexForEvilManifestDigest,
			wantErr: "validating image:",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mux := http.NewServeMux()
			server := httptest.NewUnstartedServer(mux)
			go registry.Run(server.Listener, mux)
			server.Start()
			defer server.Close()

			var s ImagePullerService
			_, err := s.getAndVerifyImage(
				context.Background(),
				slog.Default(),
				fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), tc.digest),
			)

			assert.ErrorContains(err, tc.wantErr)
		})
	}
}

func TestStoreAndVerifyLayers_EvilRegistry(t *testing.T) {
	tests := map[string]struct {
		digest  string
		wantErr string
	}{
		"correct manifest digest, wrong layer digest is caught": {
			digest:  registry.ManifestForEvilBlobDigest,
			wantErr: "validating layer:",
		},
		"correct index digest, correct manifest digest, wrong layer digest is caught": {
			digest:  registry.IndexForManifestForEvilBlobDigest,
			wantErr: "validating layer:",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			log := slog.Default()
			mux := http.NewServeMux()
			server := httptest.NewUnstartedServer(mux)
			go registry.Run(server.Listener, mux)
			server.Start()
			defer server.Close()

			s := ImagePullerService{Logger: log, Store: &StubStore{
				putLayerLayer: digest.NewDigestFromBytes(digest.SHA256, []byte{}),
			}}
			remoteImg, err := s.getAndVerifyImage(context.Background(), log, fmt.Sprintf("%s/busybox:v0.0.1@%s", server.URL[7:], tc.digest))
			require.NoError(err)

			_, err = s.storeAndVerifyLayers(log, remoteImg)

			assert.ErrorContains(err, tc.wantErr)
		})
	}
}
