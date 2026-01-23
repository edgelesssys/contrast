// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package quote

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/google/go-tdx-guest/proto/tdx"
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

var (
	errNoPCKCertificate = errors.New("no PCK certificate found in TDX quote")
	errParseCertificate = errors.New("parsing PCK certificate")
)
