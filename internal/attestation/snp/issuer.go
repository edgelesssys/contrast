/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package snp

import (
	"context"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/client"
)

// Issuer issues attestation statements.
type Issuer struct {
	logger *slog.Logger
}

// NewIssuer returns a new Issuer.
func NewIssuer(log *slog.Logger) *Issuer {
	return &Issuer{logger: log}
}

// OID returns the OID of the issuer.
func (i *Issuer) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Issue the attestation document.
func (i *Issuer) Issue(_ context.Context, ownPublicKey []byte, nonce []byte) (res []byte, err error) {
	i.logger.Info("Issue called")
	defer func() {
		if err != nil {
			i.logger.Error("Failed to issue attestation statement", "err", err)
		}
	}()

	snpGuestDevice, err := client.OpenDevice()
	if err != nil {
		return nil, fmt.Errorf("issuer: opening device: %w", err)
	}
	defer snpGuestDevice.Close()

	reportData := constructReportData(ownPublicKey, nonce)

	quoteProvider, err := client.GetQuoteProvider()
	if err != nil {
		return nil, fmt.Errorf("issuer: getting quote provider: %w", err)
	}
	reportRaw, err := quoteProvider.GetRawQuote(reportData)
	if err != nil {
		return nil, fmt.Errorf("issuer: getting raw report: %w", err)
	}
	i.logger.Info("Retrieved report", "reportRaw", hex.EncodeToString(reportRaw))

	reportB64 := make([]byte, base64.StdEncoding.EncodedLen(len(reportRaw)))
	base64.StdEncoding.Encode(reportB64, reportRaw)

	i.logger.Info("Successfully issued attestation statement")
	return reportB64, nil
}
