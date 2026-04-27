// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package insecure

import (
	"bytes"
	"context"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/oid"
)

// Validator validates fake attestation documents from insecure (non-CC) platforms.
type Validator struct {
	reportSetter attestation.ReportSetter
	logger       *slog.Logger
	name         string
}

// NewValidator creates a new insecure validator.
func NewValidator(log *slog.Logger, name string) *Validator {
	return &Validator{logger: log, name: name}
}

// NewValidatorWithReportSetter creates a new insecure validator with a report setter callback.
func NewValidatorWithReportSetter(log *slog.Logger, reportSetter attestation.ReportSetter, name string) *Validator {
	return &Validator{reportSetter: reportSetter, logger: log, name: name}
}

// OID returns the OID for the insecure attestation.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawInsecureReport
}

// Validate verifies the fake attestation document and extracts the host data.
func (v *Validator) Validate(_ context.Context, attDocRaw []byte, reportData []byte) error {
	var doc attestationDoc
	if err := json.Unmarshal(attDocRaw, &doc); err != nil {
		return fmt.Errorf("unmarshaling insecure attestation: %w", err)
	}
	if !bytes.Equal(doc.ReportData, reportData) {
		return fmt.Errorf("reportData mismatch: expected %x, got %x", reportData, doc.ReportData)
	}
	if v.reportSetter != nil {
		v.reportSetter.SetReport(report{hostData: doc.HostData})
	}
	return nil
}

// String returns the validator's name.
func (v *Validator) String() string {
	return v.name
}

// report implements the [attestation.Report] interface for insecure platforms.
type report struct {
	hostData []byte
}

// HostData returns the initdata digest.
func (r report) HostData() []byte {
	return r.hostData
}

// ClaimsToCertExtension returns no extensions for insecure platforms.
func (r report) ClaimsToCertExtension() ([]pkix.Extension, error) {
	return nil, nil
}
