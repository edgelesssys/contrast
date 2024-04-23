// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"

	"github.com/edgelesssys/contrast/internal/appendable"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
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
		mSGetter         *stubManifestSetGetter
		caGetter         *stubCertChainGetter
		workloadOwnerKey *ecdsa.PrivateKey
		wantErr          bool
	}{
		"empty request": {
			req:      &userapi.SetManifestRequest{},
			mSGetter: &stubManifestSetGetter{},
			wantErr:  true,
		},
		"manifest without policies": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = nil
				}),
			},
			mSGetter: &stubManifestSetGetter{},
			wantErr:  true,
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
			mSGetter: &stubManifestSetGetter{},
			wantErr:  true,
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
			mSGetter: &stubManifestSetGetter{},
			wantErr:  true,
		},
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
			mSGetter: &stubManifestSetGetter{},
			caGetter: &stubCertChainGetter{},
		},
		"valid manifest but error when setting it": {
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
			mSGetter: &stubManifestSetGetter{setManifestErr: assert.AnError},
			caGetter: &stubCertChainGetter{},
			wantErr:  true,
		},
		"workload owner key match": {
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
			mSGetter: &stubManifestSetGetter{
				getManifestResp: []*manifest.Manifest{manifestWithTrustedKey},
			},
			caGetter:         &stubCertChainGetter{},
			workloadOwnerKey: trustedKey,
		},
		"workload owner key mismatch": {
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
			mSGetter: &stubManifestSetGetter{
				getManifestResp: []*manifest.Manifest{manifestWithTrustedKey},
			},
			caGetter:         &stubCertChainGetter{},
			workloadOwnerKey: untrustedKey,
			wantErr:          true,
		},
		"workload owner key missing": {
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
			mSGetter: &stubManifestSetGetter{
				getManifestResp: []*manifest.Manifest{manifestWithTrustedKey},
			},
			caGetter: &stubCertChainGetter{},
			wantErr:  true,
		},
		"manifest not updatable": {
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
			mSGetter: &stubManifestSetGetter{
				getManifestResp: []*manifest.Manifest{manifestWithoutTrustedKey},
			},
			caGetter:         &stubCertChainGetter{},
			workloadOwnerKey: trustedKey,
			wantErr:          true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			coordinator := userAPIServer{
				manifSetGetter:  tc.mSGetter,
				caChainGetter:   tc.caGetter,
				policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
				logger:          slog.Default(),
			}

			ctx := rpcContext(tc.workloadOwnerKey)
			resp, err := coordinator.SetManifest(ctx, tc.req)

			if tc.wantErr {
				assert.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal([]byte("root"), resp.CoordinatorRoot)
			assert.Equal([]byte("mesh"), resp.MeshRoot)
			assert.Equal(1, tc.mSGetter.setManifestCount)
		})
	}
}

func TestGetManifests(t *testing.T) {
	testCases := map[string]struct {
		mSGetter           *stubManifestSetGetter
		caGetter           *stubCertChainGetter
		policyStoreContent map[manifest.HexString]manifest.Policy
		wantErr            bool
	}{
		"no manifest set": {
			mSGetter: &stubManifestSetGetter{},
			caGetter: &stubCertChainGetter{},
			wantErr:  true,
		},
		"no policy in store": {
			mSGetter: &stubManifestSetGetter{
				getManifestResp: []*manifest.Manifest{
					toPtr(manifest.Default()),
					toPtr(manifest.Default()),
				},
			},
			wantErr: true,
		},
		"one manifest set": {
			mSGetter: &stubManifestSetGetter{
				getManifestResp: []*manifest.Manifest{
					toPtr(manifest.Default()),
					toPtr(manifest.Default()),
				},
			},
			policyStoreContent: map[manifest.HexString]manifest.Policy{
				manifest.HexString("a"): {},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			policyStore := memstore.New[manifest.HexString, manifest.Policy]()
			for k, v := range tc.policyStoreContent {
				policyStore.Set(k, v)
			}

			coordinator := userAPIServer{
				manifSetGetter:  tc.mSGetter,
				caChainGetter:   tc.caGetter,
				policyTextStore: policyStore,
				logger:          slog.Default(),
			}

			ctx := context.Background()
			resp, err := coordinator.GetManifests(ctx, &userapi.GetManifestsRequest{})

			if tc.wantErr {
				assert.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal([]byte("root"), resp.CoordinatorRoot)
			assert.Equal([]byte("mesh"), resp.MeshRoot)
			assert.Len(resp.Policies, len(tc.policyStoreContent))
		})
	}
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

	coordinator := userAPIServer{
		manifSetGetter:  &stubManifestSetGetter{},
		caChainGetter:   &stubCertChainGetter{},
		policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
		logger:          slog.Default(),
	}
	setReq := &userapi.SetManifestRequest{
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

type stubManifestSetGetter struct {
	mux              sync.RWMutex
	setManifestCount int
	setManifestErr   error
	getManifestResp  []*manifest.Manifest
}

func (s *stubManifestSetGetter) SetManifest(*manifest.Manifest) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.setManifestCount++
	return s.setManifestErr
}

func (s *stubManifestSetGetter) GetManifests() []*manifest.Manifest {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.getManifestResp
}

func (s *stubManifestSetGetter) LatestManifest() (*manifest.Manifest, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	if len(s.getManifestResp) == 0 {
		return nil, appendable.ErrIsEmpty
	}
	return s.getManifestResp[len(s.getManifestResp)-1], nil
}

type stubCertChainGetter struct{}

func (s *stubCertChainGetter) GetCoordinatorRootCert() []byte { return []byte("root") }
func (s *stubCertChainGetter) GetMeshRootCert() []byte        { return []byte("mesh") }
func (s *stubCertChainGetter) GetIntermCert() []byte          { return []byte("inter") }

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

func toPtr[T any](t T) *T {
	return &t
}
