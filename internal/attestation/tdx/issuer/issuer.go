// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package issuer

import (
	"context"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/edgelesssys/contrast/internal/attestation/tdx/quote"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-tdx-guest/abi"
	"github.com/google/go-tdx-guest/client"
	"github.com/google/go-tdx-guest/proto/tdx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Issuer issues attestation statements.
type Issuer struct {
	logger *slog.Logger
}

// New returns a new Issuer.
func New(log *slog.Logger) *Issuer {
	return &Issuer{
		logger: log,
	}
}

// OID returns the OID of the issuer.
func (i *Issuer) OID() asn1.ObjectIdentifier {
	return oid.RawTDXReport
}

// Issue the attestation document.
func (i *Issuer) Issue(ctx context.Context, reportData [64]byte) (res []byte, err error) {
	i.logger.Info("Issue called")
	defer func() {
		if err != nil {
			i.logger.Error("Failed to issue attestation statement", "err", err)
		}
	}()

	// Get TD quote
	quoteProvider, err := client.GetQuoteProvider()
	if err != nil {
		return nil, fmt.Errorf("issuer: getting quote provider: %w", err)
	}

	quoteRaw, err := quoteProvider.GetRawQuote(reportData)
	if err != nil {
		return nil, fmt.Errorf("issuer: getting raw quote: %w", err)
	}
	i.logger.Debug("Retrieved quote", "quoteRaw", hex.EncodeToString(quoteRaw))

	quoteProto, err := abi.QuoteToProto(quoteRaw)
	if err != nil {
		return nil, fmt.Errorf("issuer: parsing quote: %w", err)
	}

	quotev4, ok := quoteProto.(*tdx.QuoteV4)
	if !ok {
		return nil, fmt.Errorf("issuer: unexpected quote type: %T", quoteProto)
	}
	i.logger.Debug("Parsed quote", "quote", protojson.MarshalOptions{Multiline: false}.Format(quotev4))

	// Make sure we don't time out while fetching optional data.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := quote.AddExtensions(ctx, quotev4); err != nil {
		// Extensions are optional, don't fail Issue because they are not available.
		i.logger.Warn("Failed to obtain quote extensions", "error", err)
	}

	quoteBytes, err := proto.Marshal(quotev4)
	if err != nil {
		return nil, fmt.Errorf("issuer: marshaling quote: %w", err)
	}

	i.logger.Info("Successfully issued attestation statement")
	return quoteBytes, nil
}
