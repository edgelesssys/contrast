// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package apiv1

import (
	"context"
	"encoding/asn1"
)

// testPolicy is the policy referenced by testManifest.
var testPolicy = []byte("version = \"0.1.0\"\nalgorithm = \"sha256\"\n\n[data]\n\"policy.rego\" = \"package agent_policy\"\n")

// otherPolicy is a policy that testManifest doesn't reference.
var otherPolicy = []byte("version = \"0.1.0\"\nalgorithm = \"sha256\"\n\n[data]\n\"policy.rego\" = \"package other_policy\"\n")

var testManifest = []byte(`
{
  "Policies": {
    "ec432606698641f33d970e8ba0655ec98ee420d10078c18ba13c18c0753d4f5c": {
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
