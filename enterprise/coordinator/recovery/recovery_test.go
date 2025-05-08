// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package recovery

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/coordinator/stateguard"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/google/go-sev-guest/abi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	testingclock "k8s.io/utils/clock/testing"
)

func TestPeriodically(t *testing.T) {
	require := require.New(t)
	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 1)

	clock := testingclock.NewFakeClock(time.Now())
	interval := time.Second
	calls := make(chan struct{}, 15)

	go func() {
		errs <- periodically(ctx, clock, interval, func() {
			calls <- struct{}{}
		})
	}()

	for range 12 {
		// The function should be called once per clock step.
		receiveEventually(t, time.Millisecond, calls)
		clock.Step(interval)
	}

	cancel()
	err := receiveEventually(t, time.Second, errs)
	require.ErrorIs(err, context.Canceled)
}

func receiveEventually[A any](t *testing.T, d time.Duration, ch chan A) A {
	// t.FailNow, require.Fail etc. are not considered function returns, so the compiler complains
	// about a missing return after the select. Wrapping the select in a for loop makes the error
	// disappear.
	for {
		select {
		case <-time.After(d):
			t.Fatalf("no object received within %s", d.String())
		case v := <-ch:
			return v
		}
	}
}

func TestRecoverOnce(t *testing.T) {
	logger := slog.Default()
	ctx := t.Context()

	for name, tc := range map[string]struct {
		peerGetter peerGetter
		guard      guard
		dial       func(_ context.Context, issuer atls.Issuer, validators []atls.Validator, _ *slog.Logger, addr string) (meshapi.MeshAPIClient, func() error, error)
		wantErr    error
	}{
		"no peers": {
			peerGetter: &fakePeerGetter{nil, nil},
			guard:      newStaleGuard(t),
			dial: func(context.Context, atls.Issuer, []atls.Validator, *slog.Logger, string) (meshapi.MeshAPIClient, func() error, error) {
				return newFakeClient(t), func() error { return nil }, nil
			},
			wantErr: errNoPeers,
		},
		"bad peerGetter": {
			peerGetter: &fakePeerGetter{nil, assert.AnError},
			guard:      newStaleGuard(t),
			dial: func(context.Context, atls.Issuer, []atls.Validator, *slog.Logger, string) (meshapi.MeshAPIClient, func() error, error) {
				return newFakeClient(t), func() error { return nil }, nil
			},
			wantErr: assert.AnError,
		},
		"bad dial": {
			peerGetter: &fakePeerGetter{[]string{"foo"}, nil},
			guard:      newStaleGuard(t),
			dial: func(context.Context, atls.Issuer, []atls.Validator, *slog.Logger, string) (meshapi.MeshAPIClient, func() error, error) {
				return nil, nil, assert.AnError
			},
			wantErr: assert.AnError,
		},
		"one bad peer": {
			peerGetter: &fakePeerGetter{[]string{"a", "b"}, nil},
			guard:      newStaleGuard(t),
			dial: func(_ context.Context, _ atls.Issuer, _ []atls.Validator, _ *slog.Logger, addr string) (meshapi.MeshAPIClient, func() error, error) {
				if addr == "b:7777" {
					return newFakeClient(t), func() error { return nil }, nil
				}
				return nil, nil, assert.AnError
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			r := &Recoverer{
				guard:      tc.guard,
				peerGetter: tc.peerGetter,
				issuer:     &atls.FakeIssuer{},
				dial:       tc.dial,
				logger:     logger,
			}
			err := r.recoverOnce(ctx)
			require.ErrorIs(err, tc.wantErr)
		})
	}
}

func TestRecoverFromPeer(t *testing.T) {
	require := require.New(t)
	ctx := t.Context()
	logger := slog.Default()

	dialCalled := false
	expectedAddr := "127.1.2.3:7777"
	r := &Recoverer{
		guard:  newStaleGuard(t),
		issuer: &atls.FakeIssuer{},
		dial: func(_ context.Context, issuer atls.Issuer, validators []atls.Validator, _ *slog.Logger, addr string) (meshapi.MeshAPIClient, func() error, error) {
			dialCalled = true
			require.Equal(expectedAddr, addr)
			require.Len(validators, 1)
			require.IsType(&snp.Validator{}, validators[0])
			require.NotNil(issuer)
			return newFakeClient(t), func() error { return nil }, nil
		},
		logger: logger,
	}

	require.NoError(r.recoverFromPeer(ctx, nil, expectedAddr))
	require.True(dialCalled)
}

type fakeGuard struct {
	getState func() (*stateguard.State, error)
	manifest *manifest.Manifest
}

func newStaleGuard(t *testing.T) *fakeGuard {
	mnfst, _ := newManifest(t)
	return &fakeGuard{
		getState: func() (*stateguard.State, error) {
			return nil, stateguard.ErrStaleState
		},
		manifest: mnfst,
	}
}

// GetState returns the current state. If the error is nil, the state must be set.
func (g *fakeGuard) GetState() (*stateguard.State, error) {
	return g.getState()
}

// ResetState recovers to the latest persisted state, authorizing the recovery seed with the passed func.
func (g *fakeGuard) ResetState(_ *stateguard.State, a stateguard.SecretSourceAuthorizer) (*stateguard.State, error) {
	if g.manifest == nil {
		return nil, assert.AnError
	}
	mnfstBytes, err := json.Marshal(g.manifest)
	if err != nil {
		return nil, err
	}
	se, meshCAKey, err := a.AuthorizeByManifest(g.manifest)
	if err != nil {
		return nil, err
	}
	ca, err := ca.New(se.RootCAKey(), meshCAKey)
	if err != nil {
		return nil, err
	}
	return stateguard.NewStateForTest(se, g.manifest, mnfstBytes, ca), nil
}

type fakeClient struct {
	meshapi.MeshAPIClient
	meshapi.RecoverResponse
}

func newFakeClient(t *testing.T) *fakeClient {
	require := require.New(t)

	seed := [32]byte{1}
	salt := [32]byte{2}

	meshCAKey := testkeys.ECDSA(t)
	meshCAKeyDER, err := x509.MarshalECPrivateKey(meshCAKey)
	require.NoError(err)
	meshCAKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: meshCAKeyDER,
	})
	return &fakeClient{
		RecoverResponse: meshapi.RecoverResponse{
			Seed:      seed[:],
			Salt:      salt[:],
			MeshCAKey: meshCAKeyPEM,
		},
	}
}

func (c *fakeClient) Recover(context.Context, *meshapi.RecoverRequest, ...grpc.CallOption) (*meshapi.RecoverResponse, error) {
	return &c.RecoverResponse, nil
}

type fakePeerGetter struct {
	peers []string
	err   error
}

func (pg *fakePeerGetter) GetPeers(context.Context) ([]string, error) {
	return pg.peers, pg.err
}

func newManifest(t *testing.T) (*manifest.Manifest, []byte) {
	t.Helper()
	policy := []byte("=== SOME REGO HERE ===")
	policyHash := sha256.Sum256(policy)
	policyHashHex := manifest.NewHexString(policyHash[:])

	mnfst := &manifest.Manifest{}
	mnfst.Policies = map[manifest.HexString]manifest.PolicyEntry{
		policyHashHex: {
			SANs:             []string{"test"},
			WorkloadSecretID: "test2",
			Role:             manifest.RoleCoordinator,
		},
	}
	svn0 := manifest.SVN(0)
	measurement := [48]byte{}
	mnfst.ReferenceValues.SNP = []manifest.SNPReferenceValues{{
		ProductName: "Milan",
		MinimumTCB: manifest.SNPTCB{
			BootloaderVersion: &svn0,
			TEEVersion:        &svn0,
			SNPVersion:        &svn0,
			MicrocodeVersion:  &svn0,
		},
		TrustedMeasurement: manifest.NewHexString(measurement[:]),
		GuestPolicy: abi.SnpPolicy{
			SMT: true,
		},
	}}
	mnfstBytes, err := json.Marshal(mnfst)
	require.NoError(t, err)
	return mnfst, mnfstBytes
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
