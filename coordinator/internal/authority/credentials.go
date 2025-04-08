// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/google/go-sev-guest/verify/trust"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/credentials"
)

// Credentials are gRPC transport credentials that dynamically update with the Coordinator state.
type Credentials struct {
	issuer   atls.Issuer
	getState func() (*State, error)

	logger                     *slog.Logger
	attestationFailuresCounter prometheus.Counter
	kdsGetter                  trust.HTTPSGetter
}

// Credentials creates new transport credentials that validate peers according to the latest manifest.
func (a *Authority) Credentials(reg *prometheus.Registry, issuer atls.Issuer, httpsGetter trust.HTTPSGetter) *Credentials {
	attestationFailuresCounter := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Subsystem: "contrast_meshapi",
		Name:      "attestation_failures_total",
		Help:      "Number of attestation failures from workloads to the Coordinator.",
	})

	return &Credentials{
		issuer:                     issuer,
		getState:                   a.GetState,
		logger:                     a.logger,
		attestationFailuresCounter: attestationFailuresCounter,
		kdsGetter:                  httpsGetter,
	}
}

// ServerHandshake implements an aTLS handshake for the latest state.
//
// If successful, the state will be passed to gRPC as [AuthInfo].
func (c *Credentials) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	log := c.logger.With("peer", rawConn.RemoteAddr())
	state, err := c.getState()
	if err != nil {
		log.Warn("Could not get manifest state to validate peer", "error", err)
		return nil, nil, fmt.Errorf("getting state: %w", err)
	}

	authInfo := AuthInfo{
		State: state,
	}

	var validators []atls.Validator

	opts, err := state.Manifest().SNPValidateOpts(c.kdsGetter)
	if err != nil {
		log.Error("Could not generate SNP validation options", "error", err)
		return nil, nil, fmt.Errorf("generating SNP validation options: %w", err)
	}

	for i, opt := range opts {
		name := fmt.Sprintf("snp-%d-%s", i, strings.TrimPrefix(opt.VerifyOpts.Product.Name.String(), "SEV_PRODUCT_"))
		validator := snp.NewValidatorWithReportSetter(opt.VerifyOpts, opt.ValidateOpts,
			logger.NewWithAttrs(logger.NewNamed(c.logger, "validator"), map[string]string{"reference-values": name}),
			&authInfo, name)
		validators = append(validators, validator)
	}

	tdxOpts, err := state.Manifest().TDXValidateOpts()
	if err != nil {
		log.Error("Could not generate TDX validation options", "error", err)
		return nil, nil, fmt.Errorf("generating TDX validation options: %w", err)
	}
	for i, opt := range tdxOpts {
		name := fmt.Sprintf("tdx-%d", i)
		validators = append(validators, tdx.NewValidatorWithReportSetter(&tdx.StaticValidateOptsGenerator{Opts: opt},
			logger.NewWithAttrs(logger.NewNamed(c.logger, "validator"), map[string]string{"reference-values": name}), &authInfo, name))
	}

	serverCfg, err := atls.CreateAttestationServerTLSConfig(c.issuer, validators, c.attestationFailuresCounter)
	if err != nil {
		log.Error("Could not create TLS config", "error", err)
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.ATLSServerTimeout)
	defer cancel()

	conn := tls.Server(rawConn, serverCfg)
	if err := conn.HandshakeContext(ctx); err != nil {
		return nil, nil, fmt.Errorf("handshake error: %w", err)
	}

	authInfo.TLSInfo = credentials.TLSInfo{
		State: conn.ConnectionState(),
		CommonAuthInfo: credentials.CommonAuthInfo{
			SecurityLevel: credentials.PrivacyAndIntegrity,
		},
	}

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
