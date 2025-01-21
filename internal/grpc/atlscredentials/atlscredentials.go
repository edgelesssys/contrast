// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package atlscredentials handles creation of TLS credentials for attested TLS (ATLS).
package atlscredentials

import (
	"context"
	"crypto"
	"errors"
	"net"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/credentials"
)

// Credentials for attested TLS (ATLS).
type Credentials struct {
	issuer              atls.Issuer
	validators          []atls.Validator
	attestationFailures prometheus.Counter
	privKey             crypto.PrivateKey
}

// New creates new ATLS credentials.
func New(issuer atls.Issuer, validators []atls.Validator, attestationFailures prometheus.Counter) *Credentials {
	return &Credentials{
		issuer:              issuer,
		attestationFailures: attestationFailures,
		validators:          validators,
	}
}

// NewWithKey creates new ATLS credentials for the given key.
func NewWithKey(issuer atls.Issuer, validators []atls.Validator, attestationFailures prometheus.Counter, key crypto.PrivateKey) *Credentials {
	c := &Credentials{privKey: key}
	c.issuer = issuer
	c.validators = validators
	c.attestationFailures = attestationFailures
	return c
}

// ClientHandshake performs the client handshake.
func (c *Credentials) ClientHandshake(ctx context.Context, authority string, rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	clientCfg, err := atls.CreateAttestationClientTLSConfig(c.issuer, c.validators, c.privKey)
	if err != nil {
		return nil, nil, err
	}

	return credentials.NewTLS(clientCfg).ClientHandshake(ctx, authority, rawConn)
}

// ServerHandshake performs the server handshake.
func (c *Credentials) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	serverCfg, err := atls.CreateAttestationServerTLSConfig(c.issuer, c.validators, c.attestationFailures)
	if err != nil {
		return nil, nil, err
	}

	return credentials.NewTLS(serverCfg).ServerHandshake(rawConn)
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
