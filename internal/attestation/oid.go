// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package attestation

import (
	"encoding/asn1"

	oids "github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-tdx-guest/proto/tdx"
)

// IsAttestationDocumentExtension checks whether the given OID corresponds to an attestation document extension
// supported by Contrast (i.e. TDX or SNP).
func IsAttestationDocumentExtension(oid asn1.ObjectIdentifier) bool {
	return oid.Equal(oids.RawTDXReport) || oid.Equal(oids.RawSNPReport)
}

type Report struct {
	Snp *sevsnp.Report
	Tdx *tdx.QuoteV4
}
