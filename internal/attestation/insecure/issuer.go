// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package atlsinsecure

import (
	"context"
	"encoding/asn1"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/oid"
)

// Issuer issues attestation statements.
type Issuer struct {
	logger *slog.Logger
}

// NewIssuer returns a new Issuer.
func NewIssuer(log *slog.Logger) *Issuer {
	return &Issuer{
		logger: log,
	}
}

// OID returns the OID of the issuer.
func (i *Issuer) OID() asn1.ObjectIdentifier {
	return oid.RawInsecureReport
}

// Issue the attestation document.
func (i *Issuer) Issue(_ context.Context, ownPublicKey []byte, nonce []byte) (res []byte, err error) {
	i.logger.Info("Issue called")
	defer func() {
		if err != nil {
			i.logger.Error("Failed to issue attestation statement", "err", err)
		}
	}()

	reportData := reportdata.Construct(ownPublicKey, nonce)

	i.logger.Info("Successfully issued attestation statement")
	return reportData[:], nil
}
