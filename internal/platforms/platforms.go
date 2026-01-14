// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package platforms provides a constant interface to the different deployment platforms
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
	// MetalQEMUSNP is the generic platform for bare-metal SNP deployments.
	MetalQEMUSNP
	// MetalQEMUTDX is the generic platform for bare-metal TDX deployments.
	MetalQEMUTDX
	// MetalQEMUSNPGPU is the generic platform for bare-metal SNP deployments with GPU passthrough.
	MetalQEMUSNPGPU
	// MetalQEMUTDXGPU is the generic platform for bare-metal TDX deployments with GPU passthrough.
	MetalQEMUTDXGPU
)

// All returns a list of all available platforms.
func All() []Platform {
	return []Platform{MetalQEMUSNP, MetalQEMUTDX, MetalQEMUSNPGPU, MetalQEMUTDXGPU}
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
	case MetalQEMUSNP:
		return "Metal-QEMU-SNP"
	case MetalQEMUSNPGPU:
		return "Metal-QEMU-SNP-GPU"
	case MetalQEMUTDX:
		return "Metal-QEMU-TDX"
	case MetalQEMUTDXGPU:
		return "Metal-QEMU-TDX-GPU"
	default:
		return "Unknown"
	}
}

// MarshalJSON marshals a Platform type to a JSON string.
func (p Platform) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `"%s"`, p.String()), nil
}

// UnmarshalJSON unmarshals a JSON string to a Platform type.
func (p *Platform) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	parsed, err := FromString(s)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

// MarshalText marshals a Platform type to a text string.
func (p Platform) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// UnmarshalText unmarshals a text string to a Platform type.
func (p *Platform) UnmarshalText(data []byte) error {
	s := string(data)
	parsed, err := FromString(s)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

// FromString returns the Platform type corresponding to the given string.
func FromString(s string) (Platform, error) {
	switch strings.ToLower(s) {
	case "metal-qemu-snp":
		return MetalQEMUSNP, nil
	case "metal-qemu-snp-gpu":
		return MetalQEMUSNPGPU, nil
	case "metal-qemu-tdx":
		return MetalQEMUTDX, nil
	case "metal-qemu-tdx-gpu":
		return MetalQEMUTDXGPU, nil
	default:
		return Unknown, fmt.Errorf("unknown platform: %s", s)
	}
}

// FromRuntimeClassString returns the Platform type corresponding to the given runtime class string,
// possibly suffixed with a hash.
func FromRuntimeClassString(s string) (Platform, error) {
	s = strings.ToLower(s)
	switch {
	case strings.HasPrefix(s, "contrast-cc-metal-qemu-snp-gpu"):
		return MetalQEMUSNPGPU, nil
	case strings.HasPrefix(s, "contrast-cc-metal-qemu-snp"):
		return MetalQEMUSNP, nil
	case strings.HasPrefix(s, "contrast-cc-metal-qemu-tdx-gpu"):
		return MetalQEMUTDXGPU, nil
	case strings.HasPrefix(s, "contrast-cc-metal-qemu-tdx"):
		return MetalQEMUTDX, nil
	default:
		return Unknown, fmt.Errorf("unknown platform: %s", s)
	}
}

// DefaultMemoryInMebiBytes returns the desired VM overhead for the given platform.
func DefaultMemoryInMebiBytes(p Platform) int {
	switch p {
	case MetalQEMUSNPGPU, MetalQEMUTDXGPU:
		// Guest components contribute around 600MiB with GPU enabled.
		return 1024
	default:
		// There are no additional guest components compared to AKS, but since the images are being
		// pulled in the guest we leave a little bit of extra room to accommodate for our init
		// containers.
		return 512
	}
}

// IsSNP returns true if the platform is a SEV-SNP platform.
func IsSNP(p Platform) bool {
	switch p {
	case MetalQEMUSNP, MetalQEMUSNPGPU:
		return true
	default:
		return false
	}
}

// IsTDX returns true if the platform is a TDX platform.
func IsTDX(p Platform) bool {
	switch p {
	case MetalQEMUTDX, MetalQEMUTDXGPU:
		return true
	default:
		return false
	}
}

// IsGPU returns true if the platform supports GPUs.
func IsGPU(p Platform) bool {
	switch p {
	case MetalQEMUSNPGPU, MetalQEMUTDXGPU:
		return true
	default:
		return false
	}
}

// IsQEMU returns true if the platform uses QEMU as the hypervisor.
func IsQEMU(p Platform) bool {
	switch p {
	case MetalQEMUSNP, MetalQEMUSNPGPU, MetalQEMUTDX, MetalQEMUTDXGPU:
		return true
	default:
		return false
	}
}
