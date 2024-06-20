// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// The flavours package provides a constant interface to the different deployment "flavours"
// of Contrast.
package flavours

import "fmt"

// Flavour is a type that represents a deployment flavour of Contrast.
type Flavour int

const (
	// Unknown is the default value for the Flavour type.
	Unknown Flavour = iota
	// AKSCLHSNP represents a deployment with Cloud-Hypervisor on SEV-SNP AKS.
	AKSCLHSNP
	// BareMetalQEMUTDX represents a deployment with QEMU on bare-metal TDX.
	BareMetalQEMUTDX
)

// String returns the string representation of the Flavour type.
func (f Flavour) String() string {
	switch f {
	case AKSCLHSNP:
		return "AKS-CLH-SNP"
	case BareMetalQEMUTDX:
		return "BareMetal-QEMU-TDX"
	default:
		return "Unknown"
	}
}

// FromString returns the Flavour type corresponding to the given string.
func FromString(s string) (Flavour, error) {
	switch s {
	case "AKS-CLH-SNP":
		return AKSCLHSNP, nil
	case "BareMetal-QEMU-TDX":
		return BareMetalQEMUTDX, nil
	default:
		return Unknown, fmt.Errorf("unknown flavour: %s", s)
	}
}
