package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"

	"github.com/edgelesssys/nunki/internal/coordapi"
	"github.com/edgelesssys/nunki/internal/manifest"
	"github.com/edgelesssys/nunki/internal/memstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	testCases := map[string]struct {
		req      *coordapi.SetManifestRequest
		mSGetter *stubManifestSetGetter
		caGetter *stubCertChainGetter
		wantErr  bool
	}{
		"empty request": {
			req:     &coordapi.SetManifestRequest{},
			wantErr: true,
		},
		"manifest without policies": {
			req: &coordapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = nil
				}),
			},
			wantErr: true,
		},
		"request without policies": {
			req: &coordapi.SetManifestRequest{
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
			req: &coordapi.SetManifestRequest{
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
		"valid manifest": {
			req: &coordapi.SetManifestRequest{
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
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			coordinator := coordAPIServer{
				manifSetGetter:  tc.mSGetter,
				caChainGetter:   tc.caGetter,
				policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
				logger:          slog.Default(),
			}

			ctx := context.Background()
			resp, err := coordinator.SetManifest(ctx, tc.req)

			if tc.wantErr {
				assert.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal([]byte("root"), resp.CACert)
			assert.Equal([]byte("inter"), resp.IntermCert)
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

			coordinator := coordAPIServer{
				manifSetGetter:  tc.mSGetter,
				caChainGetter:   tc.caGetter,
				policyTextStore: policyStore,
				logger:          slog.Default(),
			}

			ctx := context.Background()
			resp, err := coordinator.GetManifests(ctx, &coordapi.GetManifestsRequest{})

			if tc.wantErr {
				assert.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal([]byte("root"), resp.CACert)
			assert.Equal([]byte("inter"), resp.IntermCert)
			assert.Len(resp.Policies, len(tc.policyStoreContent))
		})
	}
}

// TestCoordAPIConcurrent tests potential synchronization problems between the different
// gRPCs of the server.
func TestCoordAPIConcurrent(t *testing.T) {
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

	coordinator := coordAPIServer{
		manifSetGetter:  &stubManifestSetGetter{},
		caChainGetter:   &stubCertChainGetter{},
		policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
		logger:          slog.Default(),
	}
	setReq := &coordapi.SetManifestRequest{
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
		_, _ = coordinator.GetManifests(ctx, &coordapi.GetManifestsRequest{})
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
	getManifestResp  []*manifest.Manifest
}

func (s *stubManifestSetGetter) SetManifest(*manifest.Manifest) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.setManifestCount++
}

func (s *stubManifestSetGetter) GetManifests() []*manifest.Manifest {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.getManifestResp
}

type stubCertChainGetter struct{}

func (s *stubCertChainGetter) GetRootCACert() []byte { return []byte("root") }
func (s *stubCertChainGetter) GetMeshCACert() []byte { return []byte("mesh") }
func (s *stubCertChainGetter) GetIntermCert() []byte { return []byte("inter") }

func toPtr[T any](t T) *T {
	return &t
}
