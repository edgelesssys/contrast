// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

// package issuer provides functions to create an aTLS issuer.
package issuer

import (
	"context"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/abi"
	snpabi "github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/client"
	spb "github.com/google/go-sev-guest/proto/sevsnp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Issuer issues attestation statements.
type Issuer struct {
	thimGetter *THIMGetter
	logger     *slog.Logger
}

// New returns a new Issuer.
func New(log *slog.Logger) *Issuer {
	return &Issuer{
		thimGetter: NewTHIMGetter(http.DefaultClient),
		logger:     log,
	}
}

// OID returns the OID of the issuer.
func (i *Issuer) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Issue the attestation document.
func (i *Issuer) Issue(ctx context.Context, ownPublicKey []byte, nonce []byte) (res []byte, err error) {
	i.logger.Info("Issue called")
	defer func() {
		if err != nil {
			i.logger.Error("Failed to issue attestation statement", "err", err)
		}
	}()

	reportData := reportdata.Construct(ownPublicKey, nonce)

	// Get quote from SNP device
	quoteProvider, err := getQuoteProvider(i.logger)
	if err != nil {
		return nil, fmt.Errorf("issuer: getting quote provider: %w", err)
	}
	reportRaw, err := quoteProvider.GetRawQuote(reportData)
	if err != nil {
		return nil, fmt.Errorf("issuer: getting raw report: %w", err)
	}
	report, err := snpabi.ReportToProto(reportRaw)
	if err != nil {
		return nil, fmt.Errorf("issuer: parsing report: %w", err)
	}
	i.logger.Info("Retrieved report", "reportRaw", hex.EncodeToString(reportRaw))

	// Get cert chain from THIM
	var certChain *spb.CertificateChain
	thimRaw, err := i.thimGetter.GetCertification(ctx)
	if err != nil {
		i.logger.Info("Could not retrieve THIM certification", "error", err)
	} else {
		i.logger.Info("Retrieved THIM certification", "thim", thimRaw)
		certChain, err = thimRaw.Proto()
		if err != nil {
			return nil, fmt.Errorf("issuer: converting THIM cert chain: %w", err)
		}
	}

	// Get SNP product info from cpuid
	product := abi.SevProduct()
	i.logger.Info("cpuid product info", "name", product.GetName(), "machineStepping", product.GetMachineStepping().Value)
	// Host cpuid can result in incorrect stepping: https://github.com/google/go-sev-guest/issues/115
	product.MachineStepping = &wrapperspb.UInt32Value{Value: 0}
	i.logger.Info("patched product info", "name", product.GetName(), "machineStepping", product.GetMachineStepping().Value)

	att := &spb.Attestation{
		Report:           report,
		CertificateChain: certChain,
		Product:          product,
	}

	attRaw, err := proto.Marshal(att)
	if err != nil {
		return nil, fmt.Errorf("issuer: marshaling attestation: %w", err)
	}

	i.logger.Info("Successfully issued attestation statement")
	return attRaw, nil
}

// getQuoteProvider returns the first supported quote provider.
// This is the improved version of https://pkg.go.dev/github.com/google/go-sev-guest@v0.11.1/client#GetQuoteProvider,
// with a check for ioctl support and logging.
func getQuoteProvider(logger *slog.Logger) (client.QuoteProvider, error) {
	var provider client.QuoteProvider
	provider = &client.LinuxConfigFsQuoteProvider{}
	if provider.IsSupported() {
		logger.Debug("Using LinuxConfigFsQuoteProvider")
		return provider, nil
	}
	provider = &client.LinuxIoctlQuoteProvider{}
	if provider.IsSupported() {
		logger.Debug("Using LinuxIoctlQuoteProvider")
		return provider, nil
	}
	return nil, fmt.Errorf("no supported quote provider found")
}
