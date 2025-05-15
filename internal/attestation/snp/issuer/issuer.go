// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

// Package issuer provides functions to create an aTLS issuer.
package issuer

import (
	"context"
	"crypto/x509"
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
	kdsGetter  trust.HTTPSGetter
}

// New returns a new Issuer.
func New(log *slog.Logger) *Issuer {
	month := 30 * 24 * time.Hour
	ticker := clock.RealClock{}.NewTicker(9 * month)
	return &Issuer{
		thimGetter: NewTHIMGetter(http.DefaultClient),
		logger:     log,
		kdsGetter:  certcache.NewCachedHTTPSGetter(memstore.New[string, []byte](), ticker, logger.NewNamed(log, "kds-getter-issuer")).SNPGetter(),
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

	//
	//	Checkout dev-docs/kds.md for overview over VCEK/CRL retrieval/caching.
	//

	// Get cert chain from THIM
	att := i.getAttestation(ctx, report)

	// Get the CRL.
	if crl, err := getCRLforAttestation(ctx, att, i.kdsGetter); err == nil {
		// Add CRL as CertificateChain.Extras to the attestation, so it can be used by a validator.
		if att.CertificateChain.Extras == nil {
			att.CertificateChain.Extras = make(map[string][]byte)
		}
		att.CertificateChain.Extras[constants.SNPCertChainExtrasCRLKey] = crl.Raw
	} else {
		// Continue, as the client can still try to request the CRL from KDS on their side.
		i.logger.Warn("could not get CRL", "err", err)
	}

	attRaw, err := proto.Marshal(att)
	if err != nil {
		return nil, fmt.Errorf("issuer: marshaling attestation: %w", err)
	}

	i.logger.Info("Successfully issued attestation statement")
	return attRaw, nil
}

func (i *Issuer) getAttestation(ctx context.Context, report *spb.Report) *spb.Attestation {
	var att *spb.Attestation
	var err error

	// THIM isn't rate-limited (IMDS API), so we try it first. It's only available on Microsoft Azure.
	if att, err = i.getAttestationFromTHIM(ctx, report); err == nil {
		return att
	}
	i.logger.Warn("Failed to get attestation from THIM", "err", err)

	// go-sev-guest will use VCEK from the report if it is an extended report.
	// Otherwise, it will try to get the VCEK from KDS.
	if att, err = i.getAttestationFromKdsOrExtendedReport(ctx, report); err == nil {
		return att
	}
	i.logger.Warn("Failed to get attestation from KDS or extended report", "err", err)

	// Fallback to attestation without cert chain. The client can still try to request it on their end.
	return i.getAttestationWithoutCertChain(report)
}

func (i *Issuer) getAttestationFromTHIM(ctx context.Context, report *spb.Report) (*spb.Attestation, error) {
	thimRaw, err := i.thimGetter.GetCertification(ctx)
	if err != nil {
		return nil, fmt.Errorf("requesting THIM certification: %w", err)
	}

	certChain, err := thimRaw.Proto()
	if err != nil {
		return nil, fmt.Errorf("parsing THIM certification: %w", err)
	}
	return &spb.Attestation{
		Report:           report,
		CertificateChain: certChain,
		Product:          i.getProduct(),
	}, nil
}

func (i *Issuer) getAttestationFromKdsOrExtendedReport(ctx context.Context, report *spb.Report) (*spb.Attestation, error) {
	att, err := verify.GetAttestationFromReportContext(ctx, report, &verify.Options{
		Getter: i.kdsGetter,
		// Add product since it is the only way to know which vcek endpoint to use
		// when report v2 is used. Report v3 already contains the product
		// information.
		Product: i.getProduct(),
	})
	if err != nil {
		return nil, fmt.Errorf("getting attestation from report: %w", err)
	}
	return att, nil
}

func (i *Issuer) getAttestationWithoutCertChain(report *spb.Report) *spb.Attestation {
	return &spb.Attestation{
		Report:  report,
		Product: i.getProduct(),
	}
}

func (i *Issuer) getProduct() *spb.SevProduct {
	product := snpabi.SevProduct()
	i.logger.Info("cpuid product info", "name", product.GetName(), "machineStepping", product.GetMachineStepping().Value)
	// Host cpuid can result in incorrect stepping: https://github.com/google/go-sev-guest/issues/115
	product.MachineStepping = &wrapperspb.UInt32Value{Value: 0}
	i.logger.Info("patched product info", "name", product.GetName(), "machineStepping", product.GetMachineStepping().Value)
	return product
}

func getCRLforAttestation(ctx context.Context, att *spb.Attestation, kdsGetter trust.HTTPSGetter) (*x509.RevocationList, error) {
	// Create AMDRootCerts for product line.
	root := trust.AMDRootCertsProduct(kds.ProductLine(att.GetProduct()))

	// Try to get ARK, ASK and VCEK from attestation.
	chain := att.GetCertificateChain()
	if chain == nil {
		return nil, fmt.Errorf("no certificate chain found in attestation")
	}

	// Decode ASK/ARK into root.
	if err := root.Decode(chain.GetAskCert(), chain.GetArkCert()); err != nil {
		return nil, fmt.Errorf("decoding ARK/ASK into root: %w", err)
	}

	return verify.GetCrlAndCheckRootContext(ctx, root, &verify.Options{Getter: kdsGetter})
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
