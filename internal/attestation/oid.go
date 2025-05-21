// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package attestation

import (
	"encoding/asn1"

	oids "github.com/edgelesssys/contrast/internal/oid"
)

// IsAttestationDocumentExtension checks whether the given OID corresponds to an attestation document extension
// supported by Contrast (i.e. TDX or SNP).
func IsAttestationDocumentExtension(oid asn1.ObjectIdentifier) bool {
	return oid.Equal(oids.RawTDXReport) || oid.Equal(oids.RawSNPReport)
}
