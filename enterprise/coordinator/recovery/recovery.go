// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build enterprise

package recovery

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/seedengine"
	"github.com/google/go-sev-guest/verify/trust"
	"k8s.io/utils/clock"
)

const peerRecoveryInterval = 15 * time.Second

// Recoverer can recover a Coordinator from a peer.
type Recoverer struct {
	authority   Authority
	peerGetter  PeerGetter
	hist        *history.History
	issuer      atls.Issuer
	httpsGetter trust.HTTPSGetter
	logger      *slog.Logger
}

// Authority can be recovered with a pre-fetched state.
//
// This interface is only expected to be implemented by Authority.Authority and tests.
type Authority interface {
	NeedsRecovery() bool
	RecoverWith(*seedengine.SeedEngine, *history.LatestTransition, *ecdsa.PrivateKey) error
}

type PeerGetter interface {
	GetPeers(context.Context) ([]string, error)
}

// New creates a new Recoverer.
func New(authority Authority, peerGetter PeerGetter, hist *history.History, issuer atls.Issuer, httpsGetter trust.HTTPSGetter, logger *slog.Logger) *Recoverer {
	return &Recoverer{
		authority:   authority,
		peerGetter:  peerGetter,
		hist:        hist,
		issuer:      issuer,
		httpsGetter: httpsGetter,
		logger:      logger,
	}
}

func (r *Recoverer) RunRecovery(ctx context.Context, clock clock.WithTicker) error {
	t := clock.NewTicker(peerRecoveryInterval)
	defer t.Stop()
	for {
		if r.authority.NeedsRecovery() {
			if err := r.recover(ctx); err != nil {
				r.logger.Warn("Could not recover from any peer.", "err", err)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C():
		}
	}
}

var ErrNoPeers = errors.New("no peers found")

func (r *Recoverer) recover(ctx context.Context) error {
	peers, err := r.peerGetter.GetPeers(ctx)
	if err != nil {
		return fmt.Errorf("getting peers: %w", err)
	}
	if len(peers) == 0 {
		return ErrNoPeers
	}
	var errs []error
	for _, peer := range peers {
		err := r.recoverFromPeer(ctx, net.JoinHostPort(peer, "7777"))
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// recoverFromPeer sends a recovery request to the peer coordinator and recovers the state.
func (r *Recoverer) recoverFromPeer(ctx context.Context, addr string) error {
	insecureLatest, err := r.hist.GetLatestInsecure()
	if err != nil {
		return fmt.Errorf("getting latest transition: %w", err)
	}
	transition, err := r.hist.GetTransition(insecureLatest.TransitionHash)
	if err != nil {
		return fmt.Errorf("getting transition: %w", err)
	}
	mnfstBytes, err := r.hist.GetManifest(transition.ManifestHash)
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}
	mnfst := &manifest.Manifest{}
	if err := json.Unmarshal(mnfstBytes, mnfst); err != nil {
		return fmt.Errorf("unmarshaling manifest: %w", err)
	}

	validators, err := r.validatorsFromManifest(mnfst)
	if err != nil {
		return fmt.Errorf("generating validators: %w", err)
	}
	dial := dialer.New(r.issuer, validators, atls.NoMetrics, nil, r.logger)
	conn, err := dial.Dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("dialing coordinator: %w", err)
	}
	defer conn.Close()

	client := meshapi.NewMeshAPIClient(conn)

	resp, err := client.Recover(ctx, &meshapi.RecoverRequest{})
	if err != nil {
		return fmt.Errorf("calling Recover: %w", err)
	}

	if !bytes.Equal(mnfstBytes, resp.LatestManifest) {
		return fmt.Errorf("recovered manifest does not match expected manifest")
	}

	se, err := seedengine.New(resp.Seed, resp.Salt)
	if err != nil {
		return fmt.Errorf("creating seed engine: %w", err)
	}

	latest, err := r.hist.GetLatest(&se.TransactionSigningKey().PublicKey)
	if err != nil {
		return fmt.Errorf("getting signed latest transition: %w", err)
	}
	if latest.TransitionHash != insecureLatest.TransitionHash {
		return fmt.Errorf("latest transition changed: from %x to %x", insecureLatest.TransitionHash, latest.TransitionHash)
	}

	block, _ := pem.Decode(resp.MeshCAKey)
	meshCAKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parsing mesh CA key: %w", err)
	}

	return r.authority.RecoverWith(se, latest, meshCAKey)
}

func (r *Recoverer) validatorsFromManifest(mnfst *manifest.Manifest) ([]atls.Validator, error) {
	// TODO(burgerdev): consolidate with authority.validatorsFromManifest.
	coordPolicyHash, err := mnfst.CoordinatorPolicyHash()
	if err != nil {
		r.logger.Warn("Failed to get coordinator policy hash", "error", err)
		return nil, err
	}
	coordPolicyHashBytes, err := coordPolicyHash.Bytes()
	if err != nil {
		r.logger.Warn("Failed to convert coordinator policy hash to bytes", "error", err)
		return nil, err
	}
	var validators []atls.Validator

	opts, err := mnfst.SNPValidateOpts(r.httpsGetter)
	if err != nil {
		r.logger.Error("Could not generate SNP validation options", "error", err)
		return nil, fmt.Errorf("generating SNP validation options: %w", err)
	}
	for i, opt := range opts {
		opt.ValidateOpts.HostData = coordPolicyHashBytes
		name := fmt.Sprintf("snp-%d-%s", i, strings.TrimPrefix(opt.VerifyOpts.Product.Name.String(), "SEV_PRODUCT_"))
		validators = append(validators, snp.NewValidator(opt.VerifyOpts, opt.ValidateOpts,
			logger.NewWithAttrs(logger.NewNamed(r.logger, "validator"), map[string]string{"reference-values": name}), name,
		))
	}

	tdxOpts, err := mnfst.TDXValidateOpts()
	if err != nil {
		r.logger.Error("Could not generate TDX validation options", "error", err)
		return nil, fmt.Errorf("generating TDX validation options: %w", err)
	}
	var mrConfigID [48]byte
	copy(mrConfigID[:], coordPolicyHashBytes)
	for i, opt := range tdxOpts {
		name := fmt.Sprintf("tdx-%d", i)
		opt.TdQuoteBodyOptions.MrConfigID = mrConfigID[:]
		logger := logger.NewNamed(r.logger, "validator").With("reference-values", name)
		validators = append(validators, tdx.NewValidator(&tdx.StaticValidateOptsGenerator{Opts: opt}, logger, name))
	}

	return validators, nil
}
