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
	"encoding/pem"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/edgelesssys/contrast/internal/appendable"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

const (
	manifestGenerationExpected = `
# HELP coordinator_manifest_generation Current manifest generation.
# TYPE coordinator_manifest_generation gauge
coordinator_manifest_generation %d
`
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
			wantErr: true,
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
			workloadOwnerKey: trustedKey,
			wantErr:          true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			manifestPrometheusGauge := prometheus.NewGauge(prometheus.GaugeOpts{
				Subsystem: "coordinator",
				Name:      "manifest_generation",
				Help:      "Current manifest generation.",
			})

			coordinator := userAPIServer{
				manifSetGetter:  tc.mSGetter,
				policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
				logger:          slog.Default(),
				metrics: userAPIMetrics{
					manifestGeneration: manifestPrometheusGauge,
				},
			}

			ctx := rpcContext(tc.workloadOwnerKey)
			resp, err := coordinator.SetManifest(ctx, tc.req)

			if tc.wantErr {
				assert.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal("system:coordinator:root", parsePEMCertificate(t, resp.RootCA).Subject.CommonName)
			assert.Equal("system:coordinator:intermediate", parsePEMCertificate(t, resp.MeshCA).Subject.CommonName)
			assert.Equal(1, tc.mSGetter.setManifestCount)

			expected := fmt.Sprintf(manifestGenerationExpected, 1)
			assert.NoError(testutil.CollectAndCompare(manifestPrometheusGauge, strings.NewReader(expected)))
		})
	}
}

func TestGetManifests(t *testing.T) {
	testCases := map[string]struct {
		mSGetter           *stubManifestSetGetter
		policyStoreContent map[manifest.HexString]manifest.Policy
		wantErr            bool
	}{
		"no manifest set": {
			mSGetter: &stubManifestSetGetter{},
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
			assert.Equal("system:coordinator:root", parsePEMCertificate(t, resp.RootCA).Subject.CommonName)
			assert.Equal("system:coordinator:intermediate", parsePEMCertificate(t, resp.MeshCA).Subject.CommonName)
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

	manifestPrometheusGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Subsystem: "coordinator",
		Name:      "manifest_generation",
		Help:      "Current manifest generation.",
	})

	coordinator := userAPIServer{
		manifSetGetter:  &stubManifestSetGetter{},
		policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
		logger:          slog.Default(),
		metrics: userAPIMetrics{
			manifestGeneration: manifestPrometheusGauge,
		},
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

	expected := fmt.Sprintf(manifestGenerationExpected, 6)
	assert.NoError(t, testutil.CollectAndCompare(manifestPrometheusGauge, strings.NewReader(expected)))
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

func (s *stubManifestSetGetter) GetManifestsAndLatestCA() ([]*manifest.Manifest, *ca.CA) {
	ca, err := ca.New()
	if err != nil {
		panic(err)
	}
	s.mux.RLock()
	defer s.mux.RUnlock()
	if s.getManifestResp == nil {
		return make([]*manifest.Manifest, s.setManifestCount), ca
	}
	return s.getManifestResp, ca
}

func (s *stubManifestSetGetter) LatestManifest() (*manifest.Manifest, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	if len(s.getManifestResp) == 0 {
		return nil, appendable.ErrIsEmpty
	}
	return s.getManifestResp[len(s.getManifestResp)-1], nil
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

func toPtr[T any](t T) *T {
	return &t
}
