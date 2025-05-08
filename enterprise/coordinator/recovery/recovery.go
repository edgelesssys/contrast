// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package recovery

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/edgelesssys/contrast/coordinator/stateguard"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/seedengine"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/google/go-sev-guest/verify/trust"
	"k8s.io/utils/clock"
)

const peerRecoveryInterval = 15 * time.Second

// Recoverer can recover a Coordinator from a peer.
type Recoverer struct {
	guard       guard
	peerGetter  peerGetter
	issuer      atls.Issuer
	httpsGetter trust.HTTPSGetter
	logger      *slog.Logger

	clock clock.WithTicker
	dial  func(context.Context, atls.Issuer, []atls.Validator, *slog.Logger, string) (meshapi.MeshAPIClient, func() error, error)
}

// guard is the public API of stateguard.Guard used by Recoverer.
type guard interface {
	// GetState returns the current state. If the error is nil, the state must be set.
	GetState() (*stateguard.State, error)
	// ResetState recovers to the latest persisted state, authorizing the recovery seed with the passed func.
	ResetState(oldState *stateguard.State, a stateguard.SecretSourceAuthorizer) (newState *stateguard.State, err error)
}

type peerGetter interface {
	GetPeers(context.Context) ([]string, error)
}

// New creates a new Recoverer.
func New(guard guard, peerGetter peerGetter, issuer atls.Issuer, httpsGetter trust.HTTPSGetter, logger *slog.Logger) *Recoverer {
	return &Recoverer{
		guard:       guard,
		peerGetter:  peerGetter,
		issuer:      issuer,
		httpsGetter: httpsGetter,
		logger:      logger,

		clock: clock.RealClock{},
		dial:  dial,
	}
}

// RunRecovery periodically checks whether recovery is needed and runs recover if yes.
//
// The function returns only when the context expires, with the error returned from the context.
func (r *Recoverer) RunRecovery(ctx context.Context) error {
	return periodically(ctx, r.clock, peerRecoveryInterval, func() {
		if err := r.recoverOnce(ctx); err != nil {
			r.logger.Warn("Could not recover from any peer.", "err", err)
		}
	})
}

var errNoPeers = errors.New("no peers found")

// recoverOnce performs one round of recovery attempts over all discovered peers.
func (r *Recoverer) recoverOnce(ctx context.Context) error {
	oldState, err := r.guard.GetState()
	if !errors.Is(err, stateguard.ErrStaleState) {
		return nil
	}
	r.logger.Info("Stale state observed, attempting recovery")
	peers, err := r.peerGetter.GetPeers(ctx)
	if err != nil {
		return fmt.Errorf("getting peers: %w", err)
	}
	if len(peers) == 0 {
		r.logger.Info("No peers to recover from")
		return errNoPeers
	}
	var errs []error
	for _, peer := range peers {
		err := r.recoverFromPeer(ctx, oldState, net.JoinHostPort(peer, "7777"))
		if err == nil {
			return nil
		}
		r.logger.Warn("recovery attempt failed", "peer", peer, "err", err)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// recoverFromPeer sends a recovery request to the peer coordinator and recovers the state.
func (r *Recoverer) recoverFromPeer(ctx context.Context, oldState *stateguard.State, peer string) error {
	r.logger.Info("attempting recovery", "peer", peer)
	_, err := r.guard.ResetState(oldState, authorizerFunc(func(mnfst *manifest.Manifest) (*seedengine.SeedEngine, *ecdsa.PrivateKey, error) {
		validators, err := sdk.ValidatorsFromManifest(r.httpsGetter, mnfst, r.logger)
		if err != nil {
			return nil, nil, fmt.Errorf("generating validators: %w", err)
		}

		client, closeConn, err := r.dial(ctx, r.issuer, validators, r.logger, peer)
		if err != nil {
			return nil, nil, fmt.Errorf("dialing coordinator: %w", err)
		}
		defer func() {
			if err := closeConn(); err != nil {
				r.logger.Warn("Could not close connection", "err", err)
			}
		}()

		resp, err := client.Recover(ctx, &meshapi.RecoverRequest{})
		if err != nil {
			return nil, nil, fmt.Errorf("calling Recover: %w", err)
		}

		se, err := seedengine.New(resp.Seed, resp.Salt)
		if err != nil {
			return nil, nil, fmt.Errorf("creating seed engine: %w", err)
		}

		block, _ := pem.Decode(resp.MeshCAKey)
		meshCAKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing mesh CA key: %w", err)
		}

		return se, meshCAKey, nil
	}))
	if err != nil {
		return fmt.Errorf("resetting state: %w", err)
	}
	return nil
}

func periodically(ctx context.Context, clock clock.WithTicker, interval time.Duration, f func()) error {
	t := clock.NewTicker(interval)
	defer t.Stop()
	for {
		f()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C():
		}
	}
}

type authorizerFunc func(mnfst *manifest.Manifest) (*seedengine.SeedEngine, *ecdsa.PrivateKey, error)

func (f authorizerFunc) AuthorizeByManifest(mnfst *manifest.Manifest) (*seedengine.SeedEngine, *ecdsa.PrivateKey, error) {
	return f(mnfst)
}

func dial(ctx context.Context, issuer atls.Issuer, validators []atls.Validator, logger *slog.Logger, addr string) (meshapi.MeshAPIClient, func() error, error) {
	dial := dialer.New(issuer, validators, atls.NoMetrics, nil, logger)
	conn, err := dial.Dial(ctx, addr)
	if err != nil {
		return nil, nil, fmt.Errorf("dialing coordinator: %w", err)
	}

	client := meshapi.NewMeshAPIClient(conn)
	return client, conn.Close, nil
}
