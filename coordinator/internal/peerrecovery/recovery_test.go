// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package peerrecovery

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
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
		errs <- periodically(ctx, clock, interval, func(context.Context) {
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
	select {
	case <-time.After(d):
		t.Fatalf("no object received within %s", d.String())
	case v := <-ch:
		return v
	}
	// This code can't be reached, but t.Fatalf does not count as a function return, so we need to
	// artificially return a value here.
	return *new(A)
}

func TestRecoverOnce(t *testing.T) {
	logger := slog.Default()
	ctx := t.Context()

	for name, tc := range map[string]struct {
		peerGetter   peerGetter
		guard        guard
		dialResponse map[string]meshapi.MeshAPIClient
		wantErr      error
	}{
		"no peers": {
			peerGetter: &stubPeerGetter{nil, nil},
			guard:      newFakeStaleGuard(t),
			wantErr:    errNoPeers,
		},
		"bad peerGetter": {
			peerGetter: &stubPeerGetter{nil, assert.AnError},
			guard:      newFakeStaleGuard(t),
			wantErr:    assert.AnError,
		},
		"bad dial": {
			peerGetter: &stubPeerGetter{[]string{"foo"}, nil},
			guard:      newFakeStaleGuard(t),
			wantErr:    assert.AnError,
		},
		"one bad peer": {
			peerGetter: &stubPeerGetter{[]string{"a", "b"}, nil},
			guard:      newFakeStaleGuard(t),
			dialResponse: map[string]meshapi.MeshAPIClient{
				"b:7777": newStubClient(t),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			r := &Recoverer{
				guard:      tc.guard,
				peerGetter: tc.peerGetter,
				issuer:     &atls.FakeIssuer{},
				dialer:     &stubDialer{responses: tc.dialResponse},
				logger:     logger,
			}
			err := r.recoverFromAvailablePeers(ctx)
			require.ErrorIs(err, tc.wantErr)
		})
	}
}

func TestRecoverFromPeer(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := t.Context()
	logger := slog.Default()

	expectedAddr := "127.1.2.3:7777"
	dialer := &stubDialer{
		responses: map[string]meshapi.MeshAPIClient{
			expectedAddr: newStubClient(t),
		},
	}
	r := &Recoverer{
		guard:  newFakeStaleGuard(t),
		issuer: &atls.FakeIssuer{},
		dialer: dialer,
		logger: logger,
	}

	require.NoError(r.recoverFromPeer(ctx, nil, expectedAddr))

	assert.Equal(expectedAddr, dialer.recordedAddress)
	assert.NotNil(dialer.recordedIssuer)
	assert.True(dialer.closeCalled)
	require.Len(dialer.recordedValidators, 1)
	assert.IsType(&snp.Validator{}, dialer.recordedValidators[0])
}

type fakeStaleGuard struct {
	manifest *manifest.Manifest
}

func newFakeStaleGuard(t *testing.T) *fakeStaleGuard {
	mnfst, _ := newManifest(t)
	return &fakeStaleGuard{
		manifest: mnfst,
	}
}

// GetState returns the current state. If the error is nil, the state must be set.
func (g *fakeStaleGuard) GetState(context.Context) (*stateguard.State, error) {
	return nil, stateguard.ErrStaleState
}

// ResetState recovers to the latest persisted state, authorizing the recovery seed with the passed func.
func (g *fakeStaleGuard) ResetState(ctx context.Context, oldState *stateguard.State, a stateguard.SecretSourceAuthorizer) (*stateguard.State, error) {
	if g.manifest == nil {
		return nil, assert.AnError
	}
	if oldState != nil {
		return nil, stateguard.ErrConcurrentUpdate
	}
	mnfstBytes, err := json.Marshal(g.manifest)
	if err != nil {
		return nil, err
	}
	se, meshCAKey, err := a.AuthorizeByManifest(ctx, g.manifest)
	if err != nil {
		return nil, err
	}
	ca, err := ca.New(se.RootCAKey(), meshCAKey)
	if err != nil {
		return nil, err
	}
	return stateguard.NewStateForTest(se, g.manifest, mnfstBytes, ca), nil
}

type stubClient struct {
	meshapi.MeshAPIClient
	meshapi.RecoverResponse
}

func newStubClient(t *testing.T) *stubClient {
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
	return &stubClient{
		RecoverResponse: meshapi.RecoverResponse{
			Seed:      seed[:],
			Salt:      salt[:],
			MeshCAKey: meshCAKeyPEM,
		},
	}
}

func (c *stubClient) Recover(context.Context, *meshapi.RecoverRequest, ...grpc.CallOption) (*meshapi.RecoverResponse, error) {
	return &c.RecoverResponse, nil
}

type stubPeerGetter struct {
	peers []string
	err   error
}

func (pg *stubPeerGetter) GetPeers(context.Context) ([]string, error) {
	return pg.peers, pg.err
}

type stubDialer struct {
	responses map[string]meshapi.MeshAPIClient

	recordedIssuer     atls.Issuer
	recordedValidators []atls.Validator
	recordedAddress    string
	closeCalled        bool
}

func (d *stubDialer) Dial(_ context.Context, issuer atls.Issuer, validators []atls.Validator, _ *slog.Logger, addr string) (meshapi.MeshAPIClient, func() error, error) {
	d.recordedAddress = addr
	d.recordedIssuer = issuer
	d.recordedValidators = validators
	cancel := func() error {
		d.closeCalled = true
		return nil
	}
	if c, ok := d.responses[addr]; ok {
		return c, cancel, nil
	}
	return nil, nil, assert.AnError
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
