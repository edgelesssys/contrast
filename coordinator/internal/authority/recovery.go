// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"strings"

	"github.com/edgelesssys/contrast/coordinator/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/meshapi"
)

// RecoverFromPeer sends a recovery request to the peer coordinator and recovers the state.
func (m *Authority) RecoverFromPeer(ctx context.Context, addr string) error {
	latest, err := m.hist.GetLatestInsecure()
	if err != nil {
		return fmt.Errorf("getting latest transition: %w", err)
	}
	transition, err := m.hist.GetTransition(latest.TransitionHash)
	if err != nil {
		return fmt.Errorf("getting transition: %w", err)
	}
	mnfstBytes, err := m.hist.GetManifest(transition.ManifestHash)
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}
	mnfst := &manifest.Manifest{}
	if err := json.Unmarshal(mnfstBytes, mnfst); err != nil {
		return fmt.Errorf("unmarshaling manifest: %w", err)
	}

	validators, err := m.validatorsFromManifest(mnfst, m.kdsGetter)
	if err != nil {
		return fmt.Errorf("generating validators: %w", err)
	}
	dial := dialer.New(m.issuer, validators, atls.NoMetrics, &net.Dialer{}, m.logger)
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

	newState, err := m.fetchState(se)
	if err != nil {
		return fmt.Errorf("fetching state: %w", err)
	}

	if !bytes.Equal(newState.ManifestBytes(), resp.LatestManifest) {
		return fmt.Errorf("recovered manifest does not match expected manifest")
	}

	block, _ := pem.Decode(resp.RootCAKey)
	rootCAKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parsing root CA key: %w", err)
	}
	block, _ = pem.Decode(resp.MeshCAKey)
	meshCAKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parsing mesh CA key: %w", err)
	}

	ca, err := ca.New(rootCAKey, meshCAKey)
	if err != nil {
		return fmt.Errorf("creating CA: %w", err)
	}
	newState.ca = ca

	if m.state.CompareAndSwap(nil, newState) {
		m.logger.Info("Successfully recovered from peer", "addr", addr)
	}

	return nil
}

func (m *Authority) validatorsFromManifest(mnfst *manifest.Manifest, kdsGetter *certcache.CachedHTTPSGetter) ([]atls.Validator, error) {
	coordPolicyHash, err := mnfst.CoordinatorPolicyHash()
	if err != nil {
		m.logger.Warn("Failed to get coordinator policy hash", "error", err)
		return nil, err
	}
	coordPolicyHashBytes, err := coordPolicyHash.Bytes()
	if err != nil {
		m.logger.Warn("Failed to convert coordinator policy hash to bytes", "error", err)
		return nil, err
	}
	var validators []atls.Validator

	opts, err := mnfst.SNPValidateOpts(kdsGetter)
	if err != nil {
		m.logger.Error("Could not generate SNP validation options", "error", err)
		return nil, fmt.Errorf("generating SNP validation options: %w", err)
	}
	for i, opt := range opts {
		opt.ValidateOpts.HostData = coordPolicyHashBytes
		name := fmt.Sprintf("snp-%d-%s", i, strings.TrimPrefix(opt.VerifyOpts.Product.Name.String(), "SEV_PRODUCT_"))
		validators = append(validators, snp.NewValidator(opt.VerifyOpts, opt.ValidateOpts,
			logger.NewWithAttrs(logger.NewNamed(m.logger, "validator"), map[string]string{"reference-values": name}), name,
		))
	}

	tdxOpts, err := mnfst.TDXValidateOpts()
	if err != nil {
		m.logger.Error("Could not generate TDX validation options", "error", err)
		return nil, fmt.Errorf("generating TDX validation options: %w", err)
	}
	var mrConfigID [48]byte
	copy(mrConfigID[:], coordPolicyHashBytes)
	for i, opt := range tdxOpts {
		name := fmt.Sprintf("tdx-%d", i)
		opt.TdQuoteBodyOptions.MrConfigID = mrConfigID[:]
		validators = append(validators, tdx.NewValidator(&tdx.StaticValidateOptsGenerator{Opts: opt}, logger.NewWithAttrs(logger.NewNamed(m.logger, "validator"), map[string]string{"reference-values": name}), name))
	}

	return validators, nil
}
