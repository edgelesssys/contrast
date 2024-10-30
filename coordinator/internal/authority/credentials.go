// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	atlsinsecure "github.com/edgelesssys/contrast/internal/attestation/insecure"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/credentials"
	"k8s.io/utils/clock"
)

// Credentials are gRPC transport credentials that dynamically update with the Coordinator state.
type Credentials struct {
	issuer   atls.Issuer
	getState func() (*State, error)

	logger                     *slog.Logger
	attestationFailuresCounter prometheus.Counter
	kdsGetter                  *certcache.CachedHTTPSGetter
}

// Credentials creates new transport credentials that validate peers according to the latest manifest.
func (a *Authority) Credentials(reg *prometheus.Registry, issuer atls.Issuer) (*Credentials, func()) {
	ticker := clock.RealClock{}.NewTicker(24 * time.Hour)
	kdsGetter := certcache.NewCachedHTTPSGetter(memstore.New[string, []byte](), ticker, logger.NewNamed(a.logger, "kds-getter"))
	attestationFailuresCounter := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Subsystem: "contrast_meshapi",
		Name:      "attestation_failures_total",
		Help:      "Number of attestation failures from workloads to the Coordinator.",
	})

	return &Credentials{
		issuer: issuer,
		getState: func() (*State, error) {
			if err := a.syncState(); err != nil {
				return nil, fmt.Errorf("syncing state: %w", err)
			}
			state := a.state.Load()
			if state == nil {
				return nil, errors.New("coordinator is not initialized")
			}
			return state, nil
		},
		logger:                     a.logger,
		attestationFailuresCounter: attestationFailuresCounter,
		kdsGetter:                  kdsGetter,
	}, ticker.Stop
}

// ServerHandshake implements an aTLS handshake for the latest state.
//
// If successful, the state will be passed to gRPC as [AuthInfo].
func (c *Credentials) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	c.logger.Debug("ServerHandshake started")

	state, err := c.getState()
	if err != nil {
		c.logger.Error("getting state", "err", err)
		return nil, nil, fmt.Errorf("getting state: %w", err)
	}

	authInfo := AuthInfo{
		State: state,
	}

	var validators []atls.Validator

	opts, err := state.Manifest.SNPValidateOpts(c.kdsGetter)
	if err != nil {
		c.logger.Error("generating SNP validation options", "err", err)
		return nil, nil, fmt.Errorf("generating SNP validation options: %w", err)
	}

	for _, opt := range opts {
		validator := snp.NewValidatorWithReportSetter(opt.VerifyOpts, opt.ValidateOpts,
			logger.NewWithAttrs(logger.NewNamed(c.logger, "validator"), map[string]string{"tee-type": "snp"}),
			&authInfo)
		validators = append(validators, validator)
	}

	tdxOpts, err := state.Manifest.TDXValidateOpts()
	if err != nil {
		c.logger.Error("generating TDX validation options", "err", err)
		return nil, nil, fmt.Errorf("generating TDX validation options: %w", err)
	}
	for _, opt := range tdxOpts {
		validators = append(validators, tdx.NewValidatorWithReportSetter(&tdx.StaticValidateOptsGenerator{Opts: opt},
			logger.NewWithAttrs(logger.NewNamed(c.logger, "validator"), map[string]string{"tee-type": "tdx"}), &authInfo))
	}

	// TODO(@3u13r): Don't add the insecure validator for all manifests.
	validators = append(validators, atlsinsecure.NewValidatorWithReportSetter(c.logger, &authInfo))

	serverCfg, err := atls.CreateAttestationServerTLSConfig(c.issuer, validators, c.attestationFailuresCounter)
	if err != nil {
		c.logger.Error("creating server TLS config", "err", err)
		return nil, nil, err
	}

	conn, info, err := credentials.NewTLS(serverCfg).ServerHandshake(rawConn)
	if err != nil {
		c.logger.Error("credentials.NewTLS.ServerHandshake", "err", err)
		return nil, nil, err
	}
	tlsInfo, ok := info.(credentials.TLSInfo)
	if ok {
		authInfo.TLSInfo = tlsInfo
	} else {
		c.logger.Error("credentials.NewTLS returned unexpected AuthInfo", "obj", info)
	}

	c.logger.Debug("ServerHandshake completed", "peer", tlsInfo.State.PeerCertificates[0].Subject.CommonName)

	return conn, authInfo, nil
}

// Info provides information about the protocol.
func (c *Credentials) Info() credentials.ProtocolInfo {
	return credentials.NewTLS(nil).Info()
}

// Clone is only necessary for clients and thus not implemented.
func (c *Credentials) Clone() credentials.TransportCredentials {
	panic("authority.Credentials does not implement Clone()")
}

// OverrideServerName is not implemented.
func (c *Credentials) OverrideServerName(_ string) error {
	return errors.New("OverrideServerName is not implemented")
}

// ClientHandshake is not implemented.
func (c *Credentials) ClientHandshake(_ context.Context, _ string, _ net.Conn) (net.Conn, credentials.AuthInfo, error) {
	return nil, nil, errors.New("ClientHandshake is not implemented")
}

// AuthInfo is used to pass channel authentication information and state to gRPC handlers.
//
// It implements [snp.validateCallbacker] to capture report data from the TLS handshake.
type AuthInfo struct {
	// TLSInfo holds details from the TLS handshake.
	credentials.TLSInfo
	// State is the coordinator state at the time of the TLS handshake.
	State *State
	// Report is the attestation report sent by the peer.
	Report attestation.Report
}

// SetReport takes the validated report and attaches it to the [AuthInfo].
func (a *AuthInfo) SetReport(report attestation.Report) {
	a.Report = report
}
