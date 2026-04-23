// Translated from the Python sev-snp-measure tool:
// Copyright 2022- IBM Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0
// Source: https://github.com/virtee/sev-snp-measure/blob/46664d0347fb07c5ac2cb8ab5bf5aebc09fc67ab/sevsnpmeasure/vcpu_types.py

package snp

import "fmt"

// CPUSig computes the 32-bit CPUID signature from family, model, and stepping.
// See AMD CPUID Specification, publication #25481, section CPUID Fn0000_0001_EAX.
func CPUSig(family, model, stepping int) uint32 {
	var familyLow, familyHigh uint32
	if family > 0xf {
		familyLow = 0xf
		familyHigh = uint32(family-0x0f) & 0xff
	} else {
		familyLow = uint32(family)
		familyHigh = 0
	}
	modelLow := uint32(model) & 0xf
	modelHigh := (uint32(model) >> 4) & 0xf
	steppingLow := uint32(stepping) & 0xf

	return (familyHigh << 20) | (modelHigh << 16) | (familyLow << 8) | (modelLow << 4) | steppingLow
}

// CPUSigs maps CPU type names (as used in QEMU) to their CPUID signatures.
var CPUSigs = map[string]uint32{
	"EPYC":          CPUSig(23, 1, 2),
	"EPYC-v1":       CPUSig(23, 1, 2),
	"EPYC-v2":       CPUSig(23, 1, 2),
	"EPYC-IBPB":     CPUSig(23, 1, 2),
	"EPYC-v3":       CPUSig(23, 1, 2),
	"EPYC-v4":       CPUSig(23, 1, 2),
	"EPYC-Rome":     CPUSig(23, 49, 0),
	"EPYC-Rome-v1":  CPUSig(23, 49, 0),
	"EPYC-Rome-v2":  CPUSig(23, 49, 0),
	"EPYC-Rome-v3":  CPUSig(23, 49, 0),
	"EPYC-Milan":    CPUSig(25, 1, 1),
	"EPYC-Milan-v1": CPUSig(25, 1, 1),
	"EPYC-Milan-v2": CPUSig(25, 1, 1),
	"EPYC-Genoa":    CPUSig(25, 17, 0),
	"EPYC-Genoa-v1": CPUSig(25, 17, 0),
}

// LookupCPUSig returns the CPUID signature for the named CPU type, or an error.
func LookupCPUSig(cpuType string) (uint32, error) {
	sig, ok := CPUSigs[cpuType]
	if !ok {
		return 0, fmt.Errorf("unknown CPU type %q", cpuType)
	}
	return sig, nil
}

// CPUSigForProduct returns the CPUID signature for the given AMD SEV-SNP product name
// ("Milan" → EPYC-Milan, "Genoa" → EPYC-Genoa).
func CPUSigForProduct(productName string) (uint32, error) {
	switch productName {
	case "Milan":
		return LookupCPUSig("EPYC-Milan")
	case "Genoa":
		return LookupCPUSig("EPYC-Genoa")
	default:
		return 0, fmt.Errorf("unknown SNP product name %q", productName)
	}
}
