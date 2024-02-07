package oid

import "encoding/asn1"

// RawSNPReport is the root OID for the raw SNP report extensions
// used by the aTLS issuer and validator.
var RawSNPReport = asn1.ObjectIdentifier{1, 3, 9901, 2, 1}
