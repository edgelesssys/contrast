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

// WorkloadSecretOID is the root OID for the workloadSecretID report
// extension, added to the mesh certificates to allow verification
// and authorization based on the workloadSecretID.
var WorkloadSecretOID = asn1.ObjectIdentifier{1, 3, 9901, 3, 1}
