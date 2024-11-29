// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package atls

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/klauspost/cpuid/v2"
	"github.com/stretchr/testify/require"
)

func TestPlatformIssuer(t *testing.T) {
	d := t.TempDir()
	tpmFile := filepath.Join(d, "tpm0")
	f, err := os.Create(tpmFile)
	require.NoError(t, err)
	t.Cleanup(func() { f.Close() })

	snpCPU := cpuid.CPUInfo{}
	snpCPU.Enable(cpuid.SEV_SNP)

	tdxCPU := cpuid.CPUInfo{}
	tdxCPU.Enable(cpuid.TDX_GUEST)

	for _, tc := range []struct {
		name string
		tpm  string
		cpu  *cpuid.CPUInfo

		expectIssuerType any
		expectError      bool
	}{
		{
			name: "tpm-snp",
			tpm:  tpmFile,
			cpu:  &snpCPU,

			expectIssuerType: &vtpmIssuer{},
		},
		{
			name: "tpm-tdx",
			tpm:  tpmFile,
			cpu:  &tdxCPU,

			expectIssuerType: &vtpmIssuer{},
		},
		{
			name: "notpm-snp",
			tpm:  "/invalid",
			cpu:  &snpCPU,

			expectIssuerType: &snp.Issuer{},
		},
		{
			name: "notpm-tdx",
			tpm:  "/invalid",
			cpu:  &tdxCPU,

			expectIssuerType: &tdx.Issuer{},
		},
		{
			name: "notpm-notee",
			tpm:  "/invalid",
			cpu:  &cpuid.CPUInfo{},

			expectError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			savedCPU := cpuid.CPU
			t.Cleanup(func() {
				cpuid.CPU = savedCPU
			})
			if tc.cpu != nil {
				cpuid.CPU = *tc.cpu
			}

			savedTPMDevice := tpmDevice
			t.Cleanup(func() {
				tpmDevice = savedTPMDevice
			})
			if tc.tpm != "" {
				tpmDevice = tc.tpm
			}

			issuer, err := PlatformIssuer(slog.Default())
			if tc.expectError {
				require.Error(err)
				return
			}
			require.NoError(err)

			require.IsType(tc.expectIssuerType, issuer)
		})
	}
}
