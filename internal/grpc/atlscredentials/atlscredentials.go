// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package atlscredentials handles creation of TLS credentials for attested TLS (ATLS).
package atlscredentials

import (
	"context"
	"crypto"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/credentials"
)

// Credentials for attested TLS (ATLS).
type Credentials struct {
	issuer              atls.Issuer
	validators          []atls.Validator
	attestationFailures prometheus.Counter
	privKey             crypto.PrivateKey
	logger              *slog.Logger
}

// New creates new ATLS credentials.
func New(issuer atls.Issuer, validators []atls.Validator, attestationFailures prometheus.Counter, log *slog.Logger) *Credentials {
	return &Credentials{
		issuer:              issuer,
		attestationFailures: attestationFailures,
		validators:          validators,
		logger:              log,
	}
}

// NewWithKey creates new ATLS credentials for the given key.
func NewWithKey(issuer atls.Issuer, validators []atls.Validator, attestationFailures prometheus.Counter, key crypto.PrivateKey, log *slog.Logger) *Credentials {
	return &Credentials{
		privKey:             key,
		issuer:              issuer,
		validators:          validators,
		attestationFailures: attestationFailures,
		logger:              log,
	}
}

// ClientHandshake performs the client handshake.
func (c *Credentials) ClientHandshake(ctx context.Context, authority string, rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	c.logger.DebugContext(ctx, "ClientHandshake", "authority", authority)

	clientCfg, err := atls.CreateAttestationClientTLSConfig(ctx, c.issuer, c.validators, c.privKey)
	if err != nil {
		c.logger.ErrorContext(ctx, "Creating client TLS config failed", "error", err)
		return nil, nil, err
	}

	return credentials.NewTLS(clientCfg).ClientHandshake(ctx, authority, rawConn)
}

// ServerHandshake performs the server handshake.
func (c *Credentials) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	c.logger.Debug("ServerHandshake", "peer", rawConn.RemoteAddr())

	serverCfg, err := atls.CreateAttestationServerTLSConfig(c.issuer, c.validators, c.attestationFailures)
	if err != nil {
		c.logger.Error("Error creating server TLS config", "error", err)
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.ATLSServerTimeout)
	defer cancel()

	conn := tls.Server(rawConn, serverCfg)
	if err := conn.HandshakeContext(ctx); err != nil {
		c.logger.DebugContext(ctx, "Handshake error; recurring EOF errors are expected due to the readiness check", "error", err)
		return nil, nil, fmt.Errorf("handshake error: %w", err)
	}

	info := credentials.TLSInfo{
		State: conn.ConnectionState(),
		CommonAuthInfo: credentials.CommonAuthInfo{
			SecurityLevel: credentials.PrivacyAndIntegrity,
		},
	}

	return conn, info, nil
}

// Info provides information about the protocol.
func (c *Credentials) Info() credentials.ProtocolInfo {
	return credentials.NewTLS(nil).Info()
}

// Clone the credentials object.
func (c *Credentials) Clone() credentials.TransportCredentials {
	cloned := *c
	return &cloned
}

// OverrideServerName is not supported and will fail.
func (c *Credentials) OverrideServerName(_ string) error {
	return errors.New("cannot override server name")
}
