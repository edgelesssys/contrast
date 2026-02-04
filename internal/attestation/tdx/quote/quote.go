// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package quote

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/edgelesssys/contrast/internal/attestation/tdx/qgs"
	"github.com/google/go-tdx-guest/pcs"
	"github.com/google/go-tdx-guest/proto/tdx"
	"github.com/google/go-tdx-guest/verify"
	"github.com/mdlayher/vsock"
)

// GetPCKCertificate extracts the PCK certificate from the embedded certificate chain.
func GetPCKCertificate(quote *tdx.QuoteV4) (*x509.Certificate, error) {
	pckCertChain := quote.GetSignedData().GetCertificationData().GetQeReportCertificationData().GetPckCertificateChainData().GetPckCertChain()

	// TODO(burgerdev): we should be checking the CertificateDataType.

	// The certChain input is a concatenated list of PEM-encoded X.509 certificates.
	// https://download.01.org/intel-sgx/latest/dcap-latest/linux/docs/Intel_TDX_DCAP_Quoting_Library_API.pdf, A.3.9

	var pckBlock *pem.Block
	var pck *x509.Certificate
	for len(pckCertChain) > 0 {
		pckBlock, pckCertChain = pem.Decode(pckCertChain)
		if pckBlock == nil {
			// no more PEM data
			break
		}
		candidate, err := x509.ParseCertificate(pckBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errParseCertificate, err)
		}

		// PCK certificates have specified static names.
		// https://api.trustedservices.intel.com/documents/Intel_SGX_PCK_Certificate_CRL_Spec-1.5.pdf, 1.3.5
		if candidate.Subject.CommonName == "Intel SGX PCK Certificate" {
			pck = candidate
			break
		}
	}
	if pck == nil {
		return nil, errNoPCKCertificate
	}
	return pck, nil
}

// Extensions hold optional supplemental resources that can be used by the validator.
//
// If the issuer finds relevant resources, it adds them to an Extensions struct, serializes the
// struct as JSON and attaches it to the quote's ExtraBytes field (which is otherwise unused).
//
// These resources are not covered by any signature and thus need to be verified independently!
type Extensions struct {
	// NOTE: the JSON-serialization of this struct should stay backwards-compatible!

	// Collateral is a collection of additional resources required for quote verification.
	// We include the struct as a quote extension so that the verifier does not need to fetch it
	// from PCS.
	Collateral *verify.Collateral
}

// AddExtensions prepares additional data for the verifier and stores it in quotev4.ExtraBytes.
//
// See the Extensions struct above for more details.
func AddExtensions(ctx context.Context, quotev4 *tdx.QuoteV4) error {
	pck, err := GetPCKCertificate(quotev4)
	if err != nil {
		return fmt.Errorf("extracting PCK certificate: %w", err)
	}

	extensions, err := pcs.PckCertificateExtensions(pck)
	if err != nil {
		return fmt.Errorf("extracting PCK certificate extensions: %w", err)
	}

	conn, err := vsock.Dial(vsock.Host, 4050, nil)
	if err != nil {
		return fmt.Errorf("dialing QGS vsock: %w", err)
	}
	client := qgs.NewClient(conn)
	defer client.Close()

	req := &qgs.GetCollateralRequest{
		CAType: qgs.CATypePlatform,
	}
	fmspc, err := hex.DecodeString(extensions.FMSPC)
	if err != nil {
		return fmt.Errorf("decoding FMSPC: %w", err)
	}
	copy(req.FMSPC[:], fmspc)

	resp, err := client.GetCollateral(ctx, req)
	if err != nil {
		return fmt.Errorf("getting collateral from QGS: %w", err)
	}

	collateral, err := resp.ToTDXGuest()
	if err != nil {
		return fmt.Errorf("converting collateral: %w", err)
	}

	extraBytes, err := json.Marshal(&Extensions{Collateral: collateral})
	if err != nil {
		return fmt.Errorf("marshalling extensions: %w", err)
	}
	quotev4.ExtraBytes = extraBytes
	return nil
}

// GetExtensions obtains an extensions struct added to the quote by AddExtensions.
func GetExtensions(quote *tdx.QuoteV4) (*Extensions, error) {
	var extensions Extensions
	if err := json.Unmarshal(quote.ExtraBytes, &extensions); err != nil {
		return nil, fmt.Errorf("unmarshalling extensions: %w", err)
	}
	return &extensions, nil
}

var (
	errNoPCKCertificate = errors.New("no PCK certificate found in TDX quote")
	errParseCertificate = errors.New("parsing PCK certificate")
)
