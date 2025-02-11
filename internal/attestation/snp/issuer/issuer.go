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
	"time"

	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/abi"
	snpabi "github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/client"
	"github.com/google/go-sev-guest/kds"
	spb "github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/verify"
	"github.com/google/go-sev-guest/verify/trust"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/clock"
)

// Issuer issues attestation statements.
type Issuer struct {
	thimGetter *THIMGetter
	logger     *slog.Logger
	kdsGetter  *certcache.CachedHTTPSGetter
}

// New returns a new Issuer.
func New(log *slog.Logger) *Issuer {
	month := 30 * 24 * time.Hour
	ticker := clock.RealClock{}.NewTicker(9 * month)
	return &Issuer{
		thimGetter: NewTHIMGetter(http.DefaultClient),
		logger:     log,
		kdsGetter:  certcache.NewCachedHTTPSGetter(memstore.New[string, []byte](), ticker, logger.NewNamed(log, "kds-getter-issuer")),
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

	// Get SNP product info from cpuid
	product := abi.SevProduct()
	i.logger.Info("cpuid product info", "name", product.GetName(), "machineStepping", product.GetMachineStepping().Value)
	// Host cpuid can result in incorrect stepping: https://github.com/google/go-sev-guest/issues/115
	product.MachineStepping = &wrapperspb.UInt32Value{Value: 0}
	i.logger.Info("patched product info", "name", product.GetName(), "machineStepping", product.GetMachineStepping().Value)

	// Get cert chain from THIM
	var att *spb.Attestation
	thimRaw, err := i.thimGetter.GetCertification(ctx)
	if err != nil {
		i.logger.Info("Could not retrieve THIM certification", "err", err)
		// Get cert chain including ARK, ASK and VCEK (part of spb.Attestation).
		att, err = verify.GetAttestationFromReport(report, &verify.Options{Getter: i.kdsGetter, Product: product})
		if err != nil {
			// Continue without VCEK, the client can still try to request it from KDS on their side.
			i.logger.Error("could not get attestation from report", "err", err)
		}
	} else {
		i.logger.Info("Retrieved THIM certification", "thim", thimRaw)
		certChain, err := thimRaw.Proto()
		if err != nil {
			return nil, fmt.Errorf("issuer: converting THIM cert chain: %w", err)
		}
		att = &spb.Attestation{
			Report:           report,
			CertificateChain: certChain,
			Product:          product,
		}
	}

	// Get the CRL.
	var productLine string
	if fms := att.GetReport().GetCpuid1EaxFms(); fms != 0 {
		productLine = kds.ProductLineFromFms(fms)
	}
	root := trust.AMDRootCertsProduct(productLine) // Create AMDRootCerts for product line.
	chain := att.GetCertificateChain()             // Get ARK, ASK and VCEK from attestation.
	// Decode ASK/ARK into root.
	if err := root.Decode(chain.GetAskCert(), chain.GetArkCert()); err != nil {
		return nil, err
	}
	crl, err := verify.GetCrlAndCheckRoot(root, &verify.Options{Getter: i.kdsGetter})
	if err == nil {
		// Add CRL as CertificateChain.Extras to the attestation, so it can be used by a validator.
		if att.CertificateChain.Extras == nil {
			att.CertificateChain.Extras = make(map[string][]byte)
		}
		att.CertificateChain.Extras[constants.SNPCertChainExtrasCRLKey] = crl.Raw
	} else {
		// Continue, as the client can still try to request the CRL from KDS on their side.
		i.logger.Error("could not get CRL", "err", err)
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
