// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
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
	"github.com/edgelesssys/contrast/internal/platforms"
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
	newBaseManifest := func() *manifest.Manifest {
		mnf, err := manifest.Default(platforms.AKSCloudHypervisorSNP)
		require.NoError(t, err)
		return mnf
	}
	newManifestBytes := func(f func(*manifest.Manifest)) []byte {
		m := newBaseManifest()
		if f != nil {
			f(m)
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
					m.Policies = map[manifest.HexString]manifest.PolicyEntry{
						manifest.HexString("a"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
						manifest.HexString("b"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
					}
				}),
			},
			wantErr: true,
		},
		"policy not in manifest": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString]manifest.PolicyEntry{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
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
					m.Policies = map[manifest.HexString]manifest.PolicyEntry{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
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

			reg := prometheus.NewRegistry()
			coordinator := newCoordinatorWithRegistry(reg)
			ctx := rpcContext(tc.workloadOwnerKey)
			resp, err := coordinator.SetManifest(ctx, tc.req)

			if tc.wantErr {
				assert.Error(err)
				requireGauge(t, reg, 0)
				return
			}
			require.NoError(err)
			assert.Equal("system:coordinator:root", parsePEMCertificate(t, resp.RootCA).Subject.CommonName)
			assert.Equal("system:coordinator:intermediate", parsePEMCertificate(t, resp.MeshCA).Subject.CommonName)
			requireGauge(t, reg, 1)
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

			reg := prometheus.NewRegistry()
			coordinator := newCoordinatorWithRegistry(reg)
			ctx := rpcContext(tc.workloadOwnerKey)
			m, err := json.Marshal(manifestWithTrustedKey)
			require.NoError(err)
			_, err = coordinator.SetManifest(ctx, &userapi.SetManifestRequest{Manifest: m})
			require.NoError(err)

			req := &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString]manifest.PolicyEntry{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
					}
				}),
				Policies: [][]byte{
					[]byte("a"),
					[]byte("b"),
				},
			}
			_, err = coordinator.SetManifest(ctx, req)
			require.Equal(tc.wantCode, status.Code(err))
			if tc.wantCode == codes.OK {
				requireGauge(t, reg, 2)
			} else {
				requireGauge(t, reg, 1)
			}
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

	t.Run("broken manifest", func(t *testing.T) {
		require := require.New(t)

		coordinator := newCoordinator()
		req := &userapi.SetManifestRequest{Manifest: []byte(`{ "policies": 1 }`)}
		_, err = coordinator.SetManifest(context.Background(), req)
		require.Error(err)
		require.Equal(codes.InvalidArgument, status.Code(err))
	})
}

func TestGetManifests(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	coordinator := newCoordinator()

	ctx := context.Background()
	resp, err := coordinator.GetManifests(ctx, &userapi.GetManifestsRequest{})
	require.Equal(codes.FailedPrecondition, status.Code(err))
	assert.Nil(resp)

	m, err := manifest.Default(platforms.AKSCloudHypervisorSNP)
	require.NoError(err)
	m.Policies = map[manifest.HexString]manifest.PolicyEntry{
		manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
		manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
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

func TestRecovery(t *testing.T) {
	var seed [32]byte
	var salt [32]byte
	testCases := []struct {
		name     string
		seed     []byte
		salt     []byte
		wantCode codes.Code
	}{
		{
			name:     "empty seed",
			salt:     salt[:],
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "empty salt",
			seed:     seed[:],
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "short seed",
			seed:     seed[:16],
			salt:     salt[:],
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "short salt",
			seed:     seed[:],
			salt:     salt[:16],
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "normal values",
			wantCode: codes.OK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			a := newCoordinator()

			seedShareOwnerKey, err := rsa.GenerateKey(rand.Reader, 2048)
			require.NoError(err)
			seedShareOwnerKeyBytes := manifest.MarshalSeedShareOwnerKey(&seedShareOwnerKey.PublicKey)

			mnfst, _, policies := newManifest(t)
			mnfst.SeedshareOwnerPubKeys = []manifest.HexString{seedShareOwnerKeyBytes}
			manifestBytes, err := json.Marshal(mnfst)
			require.NoError(err)

			req := &userapi.SetManifestRequest{
				Manifest: manifestBytes,
				Policies: policies,
			}
			resp, err := a.SetManifest(context.Background(), req)
			require.NoError(err)
			require.Len(resp.SeedSharesDoc.SeedShares, 1)
			seed, err := manifest.DecryptSeedShare(seedShareOwnerKey, resp.SeedSharesDoc.SeedShares[0])
			require.NoError(err)

			a = New(a.hist, prometheus.NewRegistry(), slog.Default())

			recoverReq := &userapi.RecoverRequest{
				Seed: tc.seed,
				Salt: tc.salt,
			}
			if recoverReq.Seed == nil {
				recoverReq.Seed = seed
			}
			if recoverReq.Salt == nil {
				recoverReq.Salt = resp.SeedSharesDoc.Salt
			}
			_, err = a.Recover(context.Background(), recoverReq)

			require.Equal(tc.wantCode, status.Code(err), "actual error: %v", err)
		})
	}
}

// TestRecoveryFlow exercises the recovery flow's expected path.
func TestRecoveryFlow(t *testing.T) {
	require := require.New(t)

	// 1. A Coordinator is created from empty state.

	a := newCoordinator()

	// 2. A manifest is set and the returned seed is recorded.
	seedShareOwnerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(err)
	seedShareOwnerKeyBytes := manifest.MarshalSeedShareOwnerKey(&seedShareOwnerKey.PublicKey)

	mnfst, _, policies := newManifest(t)
	mnfst.SeedshareOwnerPubKeys = []manifest.HexString{seedShareOwnerKeyBytes}
	manifestBytes, err := json.Marshal(mnfst)
	require.NoError(err)

	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: policies,
	}
	resp1, err := a.SetManifest(context.Background(), req)
	require.NoError(err)
	require.NotNil(resp1)
	seedSharesDoc := resp1.GetSeedSharesDoc()
	require.NotNil(seedSharesDoc)
	seedShares := seedSharesDoc.GetSeedShares()
	require.Len(seedShares, 1)

	seed, err := manifest.DecryptSeedShare(seedShareOwnerKey, seedShares[0])
	require.NoError(err)

	recoverReq := &userapi.RecoverRequest{
		Seed: seed,
		Salt: seedSharesDoc.GetSalt(),
	}

	// Recovery on this Coordinator should fail now that a manifest is set.
	_, err = a.Recover(context.Background(), recoverReq)
	require.ErrorContains(err, ErrAlreadyRecovered.Error())

	// 3. A new Coordinator is created with the existing history.
	// GetManifests and SetManifest are expected to fail.

	a = New(a.hist, prometheus.NewRegistry(), slog.Default())
	_, err = a.SetManifest(context.Background(), req)
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	_, err = a.GetManifests(context.Background(), &userapi.GetManifestsRequest{})
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	// 4. Recovery is called.
	_, err = a.Recover(context.Background(), recoverReq)
	require.NoError(err)

	// 5. Coordinator should be operational and know about the latest manifest.
	resp, err := a.GetManifests(context.Background(), &userapi.GetManifestsRequest{})
	require.NoError(err)
	require.NotNil(resp)
	require.Len(resp.Manifests, 1)
	require.Equal([][]byte{manifestBytes}, resp.Manifests)

	// Recover on a recovered authority should fail.
	_, err = a.Recover(context.Background(), recoverReq)
	require.Error(err)
}

// TestUserAPIConcurrent tests potential synchronization problems between the different
// gRPCs of the server.
func TestUserAPIConcurrent(t *testing.T) {
	newBaseManifest := func() *manifest.Manifest {
		mnf, err := manifest.Default(platforms.AKSCloudHypervisorSNP)
		require.NoError(t, err)
		return mnf
	}
	newManifestBytes := func(f func(*manifest.Manifest)) []byte {
		m := newBaseManifest()
		if f != nil {
			f(m)
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
			m.Policies = map[manifest.HexString]manifest.PolicyEntry{
				manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
				manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
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
	return newCoordinatorWithRegistry(prometheus.NewRegistry())
}

func newCoordinatorWithRegistry(reg *prometheus.Registry) *Authority {
	fs := afero.NewMemMapFs()
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(store)
	return New(hist, reg, slog.Default())
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
	m, err := manifest.Default(platforms.AKSCloudHypervisorSNP)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return m, nil
	}
	pubKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	ownerKeyHash := sha256.Sum256(pubKey)
	ownerKeyHex := manifest.NewHexString(ownerKeyHash[:])
	m.WorkloadOwnerKeyDigests = []manifest.HexString{ownerKeyHex}
	return m, nil
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
