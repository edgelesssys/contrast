// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package issuer

import (
	"context"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation/tdx/qgs"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-tdx-guest/abi"
	"github.com/google/go-tdx-guest/client"
	"github.com/google/go-tdx-guest/pcs"
	"github.com/google/go-tdx-guest/proto/tdx"
	"github.com/mdlayher/vsock"
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
	i.logger.Info("Retrieved quote", "quoteRaw", hex.EncodeToString(quoteRaw))

	quote, err := abi.QuoteToProto(quoteRaw)
	if err != nil {
		return nil, fmt.Errorf("issuer: parsing quote: %w", err)
	}
	i.logger.Info("Parsed quote", "quote", quote)

	// Marshal the quote
	quotev4, ok := quote.(*tdx.QuoteV4)
	if !ok {
		return nil, fmt.Errorf("issuer: unexpected quote type: %T", quote)
	}

	extensions, err := i.getExtensions(ctx, quotev4)
	if err != nil {
		// Extensions are optional, don't fail Issue because they are not available.
		i.logger.Warn("Failed to obtain quote extensions", "error", err)
	} else {
		quotev4.ExtraBytes = extensions
	}

	quoteBytes, err := proto.Marshal(quotev4)
	if err != nil {
		return nil, fmt.Errorf("issuer: marshaling quote: %w", err)
	}

	i.logger.Info("Successfully issued attestation statement")
	return quoteBytes, nil
}

func (i *Issuer) getExtensions(ctx context.Context, quotev4 *tdx.QuoteV4) ([]byte, error) {
	// TODO(burgerdev): vsock package is absolutely unnecessary.
	conn, err := vsock.Dial(vsock.Host, 4050, nil)
	if err != nil {
		return nil, fmt.Errorf("dialing QGS vsock: %w", err)
	}
	client := qgs.NewClient(conn)
	defer client.Close()

	// TODO(burgerdev): should be shared with the validator
	pckCertChain := quotev4.GetSignedData().GetCertificationData().GetQeReportCertificationData().GetPckCertificateChainData().PckCertChain

	// The certChain input is a concatenated list of PEM-encoded X.509 certificates.
	// https://download.01.org/intel-sgx/latest/dcap-latest/linux/docs/Intel_TDX_DCAP_Quoting_Library_API.pdf, A.3.9

	var pckBlock *pem.Block
	var pck *x509.Certificate
	for len(pckCertChain) > 0 {
		pckBlock, pckCertChain = pem.Decode(pckCertChain)
		if pckBlock == nil {
			break
		}
		candidate, err := x509.ParseCertificate(pckBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing PCK certificate: %w", err)
		}

		// PCK certificates have specified static names.
		// https://api.trustedservices.intel.com/documents/Intel_SGX_PCK_Certificate_CRL_Spec-1.5.pdf, 1.3.5
		if candidate.Subject.CommonName == "Intel SGX PCK Certificate" {
			pck = candidate
			break
		}
	}
	if pck == nil {
		return nil, fmt.Errorf("no PCK certificate found in TDX quote")
	}

	extensions, err := pcs.PckCertificateExtensions(pck)
	if err != nil {
		return nil, fmt.Errorf("extracting PCK certificate extensions: %w", err)
	}

	req := &qgs.GetCollateralRequest{
		Header: qgs.Header{
			MajorVersion: 1,
			MinorVersion: 1,
		},
		CAType: qgs.CATypePlatform,
	}
	fmspc, err := hex.DecodeString(extensions.FMSPC)
	if err != nil {
		return nil, fmt.Errorf("decoding FMSPC: %w", err)
	}
	copy(req.FMSPC[:], fmspc)

	resp, err := client.GetCollateral(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("getting collateral from QGS: %w", err)
	}

	collateral, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshalling collateral: %w", err)
	}

	extraBytes, err := json.Marshal(&Extensions{Collateral: collateral})
	if err != nil {
		return nil, fmt.Errorf("marshalling extensions: %w", err)
	}
	return extraBytes, nil
}

// Extensions hold optional supplemental resources that can be used by the validator.
//
// If the issuer finds relevant resources, it adds them to an Extensions struct, serializes the
// struct as JSON and attaches it to the quote's ExtraBytes field (which is otherwise unused).
//
// These resources are not covered by any signature and thus need to be verified independently!
type Extensions struct {
	// NOTE: the JSON-serialization of this struct should stay backwards-compatible!

	// Collateral is a collection of additional resources obtain from the Intel QGS.
	// These are included so that the validator does not need to fetch them from PCS.
	// The value is a JSON-serialized qgs.GetCollateralResponse.
	Collateral []byte
}
