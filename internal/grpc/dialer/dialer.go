// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package dialer provides a grpc dialer that can be used to create grpc client connections with different levels of ATLS encryption / verification.
package dialer

import (
	"context"
	"crypto"
	"log/slog"
	"net"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/grpc/atlscredentials"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dialer can open grpc client connections with different levels of ATLS encryption / verification.
type Dialer struct {
	issuer              atls.Issuer
	validators          []atls.Validator
	attestationFailures prometheus.Counter
	netDialer           NetDialer
	privKey             crypto.PrivateKey
	logger              *slog.Logger
}

// New creates a new Dialer.
func New(issuer atls.Issuer, validators []atls.Validator, attestationFailures prometheus.Counter, netDialer NetDialer, log *slog.Logger) *Dialer {
	return &Dialer{
		issuer:              issuer,
		validators:          validators,
		attestationFailures: attestationFailures,
		netDialer:           netDialer,
		logger:              log,
	}
}

// NewWithKey creates a new Dialer with the given private key.
func NewWithKey(issuer atls.Issuer, validators []atls.Validator, attestationFailures prometheus.Counter, netDialer NetDialer, privKey crypto.PrivateKey, log *slog.Logger) *Dialer {
	return &Dialer{
		issuer:              issuer,
		validators:          validators,
		attestationFailures: attestationFailures,
		netDialer:           netDialer,
		privKey:             privKey,
		logger:              log,
	}
}

// Dial creates a new grpc client connection to the given target using the atls validator.
func (d *Dialer) Dial(_ context.Context, target string) (*grpc.ClientConn, error) {
	credentials := atlscredentials.NewWithKey(d.issuer, d.validators, d.attestationFailures, d.privKey, logger.NewNamed(d.logger, "atlscredentials"))

	return grpc.NewClient(target,
		d.grpcWithDialer(),
		grpc.WithTransportCredentials(credentials),
		grpc.WithConnectParams(grpc.ConnectParams{
			// We need a high initial timeout, because otherwise the client will get stuck in a reconnect loop
			// where the timeout is too low to get a full handshake done.
			MinConnectTimeout: constants.ATLSClientTimeout,
		}),
	)
}

// DialInsecure creates a new grpc client connection to the given target without using encryption or verification.
// Only use this method when using another kind of encryption / verification (VPN, etc).
func (d *Dialer) DialInsecure(_ context.Context, target string) (*grpc.ClientConn, error) {
	return grpc.NewClient(target,
		d.grpcWithDialer(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// DialNoVerify creates a new grpc client connection to the given target without verifying the server's attestation.
func (d *Dialer) DialNoVerify(_ context.Context, target string) (*grpc.ClientConn, error) {
	credentials := atlscredentials.New(atls.NoIssuer, atls.NoValidators, atls.NoMetrics, logger.NewNamed(d.logger, "atlscredentials"))

	return grpc.NewClient(target,
		d.grpcWithDialer(),
		grpc.WithTransportCredentials(credentials),
	)
}

func (d *Dialer) grpcWithDialer() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return d.netDialer.DialContext(ctx, "tcp", addr)
	})
}

// NetDialer implements the net Dialer interface.
type NetDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
