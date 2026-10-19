// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package apiv1

import (
	"context"
	"encoding/asn1"
)

var testManifest = []byte(`
{
  "Policies": {
    "ef27c1c91a0ce044c67f0ec10d7c66ea9f178453dc96a233e97f0675578042f2": {
      "SANs": ["coordinator"],
      "WorkloadSecretID": "apps/v1/StatefulSet/default/coordinator",
      "Role": "coordinator"
    }
  },
  "ReferenceValues": {
    "snp": [
      {
        "MinimumTCB": {
          "BootloaderVersion": 3,
          "TEEVersion": 0,
          "SNPVersion": 23,
          "MicrocodeVersion": 213
        },
        "ProductName": "Milan",
        "TrustedMeasurement": "05c504736ca974b9ac0c84b5099f957907507c09e4844bd0672d0b647205f35837bd479ae35567b22b69ce636666c286",
        "APEIP": "0080b004",
        "GuestPolicy": {
          "ABIMinor": 0,
          "ABIMajor": 0,
          "SMT": true,
          "MigrateMA": false,
          "Debug": false,
          "SingleSocket": false,
          "CXLAllowed": false,
          "MemAES256XTS": false,
          "RAPLDis": false,
          "CipherTextHidingDRAM": false,
          "PageSwapDisable": false
        },
        "PlatformInfo": {
          "SMTEnabled": true,
          "TSMEEnabled": false,
          "ECCEnabled": false,
          "RAPLDisabled": false,
          "CiphertextHidingDRAMEnabled": false,
          "AliasCheckComplete": false,
          "TIOEnabled": false
        }
      }
	]
  }
}
`)

type stubValidator struct {
	err           error
	gotReportData []byte
}

func (v *stubValidator) Validate(_ context.Context, _ asn1.ObjectIdentifier, _ []byte, reportData []byte) error {
	v.gotReportData = reportData
	return v.err
}

func (v *stubValidator) String() string {
	return "stubValidator"
}
