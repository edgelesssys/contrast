// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package oid

import "encoding/asn1"

// RawSNPReport is the root OID for the raw SNP report extensions
// used by the aTLS issuer and validator.
var RawSNPReport = asn1.ObjectIdentifier{1, 3, 9901, 2, 1}

// RawTDXReport is the root OID for the raw TDX report extensions
// used by the aTLS issuer and validator.
var RawTDXReport = asn1.ObjectIdentifier{1, 3, 9901, 2, 2}

// RawInsecureReport is the root OID for the raw insecure report extensions
// used by the aTLS issuer and validator.
var RawInsecureReport = asn1.ObjectIdentifier{1, 3, 9901, 2, 3}

// IsAttestationDocumentExtension checks whether the given OID corresponds to an attestation document extension
// supported by Contrast (i.e. TDX or SNP).
func IsAttestationDocumentExtension(oid asn1.ObjectIdentifier) bool {
	return oid.Equal(RawTDXReport) || oid.Equal(RawSNPReport) || oid.Equal(RawInsecureReport)
}
