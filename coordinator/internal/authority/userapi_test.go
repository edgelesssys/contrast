// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"sync"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestManifestSet(t *testing.T) {
	newBaseManifest := func() manifest.Manifest {
		return manifest.Default()
	}
	newManifestBytes := func(f func(*manifest.Manifest)) []byte {
		m := newBaseManifest()
		if f != nil {
			f(&m)
		}
		b, err := json.Marshal(m)
		require.NoError(t, err)
		return b
	}
	trustedKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	untrustedKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	manifestWithTrustedKey, err := manifestWithWorkloadOwnerKey(trustedKey)
	require.NoError(t, err)
	manifestWithoutTrustedKey, err := manifestWithWorkloadOwnerKey(nil)
	require.NoError(t, err)

	testCases := map[string]struct {
		req              *userapi.SetManifestRequest
		workloadOwnerKey *ecdsa.PrivateKey
		wantErr          bool
	}{
		"empty request": {
			req:     &userapi.SetManifestRequest{},
			wantErr: true,
		},
		"manifest without policies": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = nil
				}),
			},
			wantErr: false,
		},
		"request without policies": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString][]string{
						manifest.HexString("a"): {"a1", "a2"},
						manifest.HexString("b"): {"b1", "b2"},
					}
				}),
			},
			wantErr: true,
		},
		"policy not in manifest": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString][]string{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {"a1", "a2"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {"b1", "b2"},
					}
				}),
				Policies: [][]byte{
					[]byte("a"),
					[]byte("c"),
				},
			},
			wantErr: true,
		},
		// TODO(burgerdev): add test for dysfunctional history backend
		"valid manifest": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString][]string{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {"a1", "a2"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {"b1", "b2"},
					}
				}),
				Policies: [][]byte{
					[]byte("a"),
					[]byte("b"),
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			coordinator := newCoordinator()
			ctx := rpcContext(tc.workloadOwnerKey)
			resp, err := coordinator.SetManifest(ctx, tc.req)

			if tc.wantErr {
				assert.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal("system:coordinator:root", parsePEMCertificate(t, resp.RootCA).Subject.CommonName)
			assert.Equal("system:coordinator:intermediate", parsePEMCertificate(t, resp.MeshCA).Subject.CommonName)
		})
	}

	keyTestCases := map[string]struct {
		workloadOwnerKey *ecdsa.PrivateKey
		wantCode         codes.Code
	}{
		"workload owner key match": {
			workloadOwnerKey: trustedKey,
		},
		"workload owner key mismatch": {
			workloadOwnerKey: untrustedKey,
			wantCode:         codes.PermissionDenied,
		},
		"workload owner key missing": {
			wantCode: codes.PermissionDenied,
		},
	}
	for name, tc := range keyTestCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			coordinator := newCoordinator()
			ctx := rpcContext(tc.workloadOwnerKey)
			m, err := json.Marshal(manifestWithTrustedKey)
			require.NoError(err)
			_, err = coordinator.SetManifest(ctx, &userapi.SetManifestRequest{Manifest: m})
			require.NoError(err)

			req := &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString][]string{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {"a1", "a2"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {"b1", "b2"},
					}
				}),
				Policies: [][]byte{
					[]byte("a"),
					[]byte("b"),
				},
			}
			_, err = coordinator.SetManifest(ctx, req)
			require.Equal(tc.wantCode, status.Code(err))
		})
	}

	t.Run("no workload owner key in manifest", func(t *testing.T) {
		require := require.New(t)

		coordinator := newCoordinator()
		ctx := rpcContext(trustedKey)
		m, err := json.Marshal(manifestWithoutTrustedKey)
		require.NoError(err)
		req := &userapi.SetManifestRequest{Manifest: m}
		_, err = coordinator.SetManifest(ctx, req)
		require.NoError(err)
		_, err = coordinator.SetManifest(ctx, req)
		require.Error(err)
		require.Equal(codes.PermissionDenied, status.Code(err))
	})
}

func TestGetManifests(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	coordinator := newCoordinator()

	ctx := context.Background()
	resp, err := coordinator.GetManifests(ctx, &userapi.GetManifestsRequest{})
	require.Error(err)
	assert.Nil(resp)

	m := manifest.Default()
	m.Policies = map[manifest.HexString][]string{
		manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {"a1", "a2"},
		manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {"b1", "b2"},
	}
	manifestBytes, err := json.Marshal(m)
	require.NoError(err)

	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: [][]byte{
			[]byte("a"),
			[]byte("b"),
		},
	}
	setResp, err := coordinator.SetManifest(ctx, req)
	require.NoError(err)
	assert.NotNil(setResp)

	resp, err = coordinator.GetManifests(ctx, &userapi.GetManifestsRequest{})

	require.NoError(err)
	assert.Equal("system:coordinator:root", parsePEMCertificate(t, resp.RootCA).Subject.CommonName)
	assert.Equal("system:coordinator:intermediate", parsePEMCertificate(t, resp.MeshCA).Subject.CommonName)
	assert.Len(resp.Policies, len(m.Policies))
}

// TestUserAPIConcurrent tests potential synchronization problems between the different
// gRPCs of the server.
func TestUserAPIConcurrent(t *testing.T) {
	newBaseManifest := func() manifest.Manifest {
		return manifest.Default()
	}
	newManifestBytes := func(f func(*manifest.Manifest)) []byte {
		m := newBaseManifest()
		if f != nil {
			f(&m)
		}
		b, err := json.Marshal(m)
		require.NoError(t, err)
		return b
	}

	fs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(store)
	coordinator := New(hist, prometheus.NewRegistry(), slog.Default())

	setReq := &userapi.SetManifestRequest{
		Manifest: newManifestBytes(func(m *manifest.Manifest) {
			m.Policies = map[manifest.HexString][]string{
				manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {"a1", "a2"},
				manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {"b1", "b2"},
			}
		}),
		Policies: [][]byte{
			[]byte("a"),
			[]byte("b"),
		},
	}

	ctx := context.Background()
	wg := sync.WaitGroup{}

	set := func() {
		defer wg.Done()
		_, _ = coordinator.SetManifest(ctx, setReq)
	}
	get := func() {
		defer wg.Done()
		_, _ = coordinator.GetManifests(ctx, &userapi.GetManifestsRequest{})
	}

	wg.Add(12)
	go set()
	go set()
	go set()
	go get()
	go get()
	go get()
	go set()
	go set()
	go set()
	go get()
	go get()
	go get()
	wg.Wait()
}

func newCoordinator() *Authority {
	fs := afero.NewMemMapFs()
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(store)
	return New(hist, prometheus.NewRegistry(), slog.Default())
}

func rpcContext(key *ecdsa.PrivateKey) context.Context {
	var peerCertificates []*x509.Certificate
	if key != nil {
		peerCertificates = []*x509.Certificate{{
			PublicKey:          key.Public(),
			PublicKeyAlgorithm: x509.ECDSA,
		}}
	}
	return peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{
			PeerCertificates: peerCertificates,
		}},
	})
}

func manifestWithWorkloadOwnerKey(key *ecdsa.PrivateKey) (*manifest.Manifest, error) {
	m := manifest.Default()
	if key == nil {
		return &m, nil
	}
	pubKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	ownerKeyHash := sha256.Sum256(pubKey)
	ownerKeyHex := manifest.NewHexString(ownerKeyHash[:])
	m.WorkloadOwnerKeyDigests = []manifest.HexString{ownerKeyHex}
	return &m, nil
}

func parsePEMCertificate(t *testing.T, pemCert []byte) *x509.Certificate {
	t.Helper()

	block, _ := pem.Decode(pemCert)
	require.NotNil(t, block, "no pem-encoded certificate found")

	// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}
