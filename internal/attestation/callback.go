// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package attestation

import (
	"crypto/x509/pkix"
)

// Report is a verified and validates TEE attestation report.
type Report interface {
	HostData() []byte
	ClaimsToCertExtension() ([]pkix.Extension, error)
}

// ReportSetter is called by a validator after it verified and validated an attestation report.
type ReportSetter interface {
	SetReport(report Report)
}
