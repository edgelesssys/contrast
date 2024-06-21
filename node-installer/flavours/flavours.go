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
	// K3sQEMUTDX represents a deployment with QEMU on bare-metal TDX K3s.
	K3sQEMUTDX
	// RKE2QEMUTDX represents a deployment with QEMU on bare-metal TDX RKE2.
	RKE2QEMUTDX
)

// String returns the string representation of the Flavour type.
func (f Flavour) String() string {
	switch f {
	case AKSCLHSNP:
		return "AKS-CLH-SNP"
	case K3sQEMUTDX:
		return "K3s-QEMU-TDX"
	case RKE2QEMUTDX:
		return "RKE2-QEMU-TDX"
	default:
		return "Unknown"
	}
}

// FromString returns the Flavour type corresponding to the given string.
func FromString(s string) (Flavour, error) {
	switch s {
	case "AKS-CLH-SNP":
		return AKSCLHSNP, nil
	case "K3s-QEMU-TDX":
		return K3sQEMUTDX, nil
	case "RKE2-QEMU-TDX":
		return RKE2QEMUTDX, nil
	default:
		return Unknown, fmt.Errorf("unknown flavour: %s", s)
	}
}
