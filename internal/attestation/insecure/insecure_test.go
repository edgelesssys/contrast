// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package insecure

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAndValidate(t *testing.T) {
	hostData := []byte("hostdata")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/hostdata", r.URL.Path)
		_, err := w.Write(hostData)
		require.NoError(t, err)
	}))
	defer server.Close()

	issuer := &Issuer{hostdataURL: server.URL + "/hostdata", client: server.Client()}
	var reportData [64]byte
	copy(reportData[:], []byte("report-data"))

	attDoc, err := issuer.Issue(context.Background(), reportData)
	require.NoError(t, err)

	setter := &stubReportSetter{}
	validator := NewValidatorWithReportSetter(slog.Default(), setter, "insecure")
	require.NoError(t, validator.Validate(context.Background(), attDoc, reportData[:]))
	require.NotNil(t, setter.report)
	assert.Equal(t, hostData, setter.report.HostData())
}

func TestValidateMismatchingReportData(t *testing.T) {
	validator := NewValidator(slog.Default(), "insecure")
	attDoc := []byte(`{"reportData":"AQ==","hostData":"Ag=="}`)

	err := validator.Validate(context.Background(), attDoc, []byte{0x02})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reportData mismatch")
}

func TestAttestationDocumentOID(t *testing.T) {
	assert.True(t, attestation.IsAttestationDocumentExtension(NewIssuer().OID()))
}

type stubReportSetter struct {
	report attestation.Report
}

func (s *stubReportSetter) SetReport(report attestation.Report) {
	s.report = report
}
