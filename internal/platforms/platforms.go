// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// The platforms package provides a constant interface to the different deployment platforms
// of Contrast.
package platforms

import (
	"fmt"
	"strings"
)

// Platform is a type that represents a deployment platform of Contrast.
type Platform int

const (
	// Unknown is the default value for the platform type.
	Unknown Platform = iota
	// AKSCloudHypervisorSNP represents a deployment with Cloud-Hypervisor on SEV-SNP AKS.
	AKSCloudHypervisorSNP
	// K3sQEMUTDX represents a deployment with QEMU on bare-metal TDX K3s.
	K3sQEMUTDX
	// K3sQEMUSNP represents a deployment with QEMU on bare-metal SNP K3s.
	K3sQEMUSNP
	// RKE2QEMUTDX represents a deployment with QEMU on bare-metal TDX RKE2.
	RKE2QEMUTDX
	// MetalQEMUSNP is the generic platform for bare-metal SNP deployments.
	MetalQEMUSNP
	// MetalQEMUTDX is the generic platform for bare-metal TDX deployments.
	MetalQEMUTDX
	// K3sQEMUSNPGPU represents a deployment with QEMU on bare-metal SNP K3s with GPU passthrough.
	K3sQEMUSNPGPU
	// MetalQEMUSNPGPU is the generic platform for bare-metal SNP deployments with GPU passthrough.
	MetalQEMUSNPGPU
)

// All returns a list of all available platforms.
func All() []Platform {
	return []Platform{AKSCloudHypervisorSNP, K3sQEMUTDX, K3sQEMUSNP, RKE2QEMUTDX, MetalQEMUSNP, MetalQEMUTDX, K3sQEMUSNPGPU, MetalQEMUSNPGPU}
}

// AllStrings returns a list of all available platforms as strings.
func AllStrings() []string {
	platformStrings := make([]string, 0, len(All()))
	for _, p := range All() {
		platformStrings = append(platformStrings, p.String())
	}
	return platformStrings
}

// String returns the string representation of the Platform type.
func (p Platform) String() string {
	switch p {
	case AKSCloudHypervisorSNP:
		return "AKS-CLH-SNP"
	case K3sQEMUTDX:
		return "K3s-QEMU-TDX"
	case K3sQEMUSNP:
		return "K3s-QEMU-SNP"
	case K3sQEMUSNPGPU:
		return "K3s-QEMU-SNP-GPU"
	case RKE2QEMUTDX:
		return "RKE2-QEMU-TDX"
	case MetalQEMUSNP:
		return "Metal-QEMU-SNP"
	case MetalQEMUSNPGPU:
		return "Metal-QEMU-SNP-GPU"
	case MetalQEMUTDX:
		return "Metal-QEMU-TDX"
	default:
		return "Unknown"
	}
}

// FromString returns the Platform type corresponding to the given string.
func FromString(s string) (Platform, error) {
	switch strings.ToLower(s) {
	case "aks-clh-snp":
		return AKSCloudHypervisorSNP, nil
	case "k3s-qemu-tdx":
		return K3sQEMUTDX, nil
	case "k3s-qemu-snp":
		return K3sQEMUSNP, nil
	case "k3s-qemu-snp-gpu":
		return K3sQEMUSNPGPU, nil
	case "rke2-qemu-tdx":
		return RKE2QEMUTDX, nil
	case "metal-qemu-snp":
		return MetalQEMUSNP, nil
	case "metal-qemu-snp-gpu":
		return MetalQEMUSNPGPU, nil
	case "metal-qemu-tdx":
		return MetalQEMUTDX, nil
	default:
		return Unknown, fmt.Errorf("unknown platform: %s", s)
	}
}
