// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/google/go-sev-guest/abi"
)

// EmbeddedReferenceValuesJSON contains the embedded reference values in JSON format.
//
//go:embed assets/reference-values.json
var EmbeddedReferenceValuesJSON []byte

// ReferenceValues contains the workload-independent reference values for each TEE type.
type ReferenceValues struct {
	// SNP holds the reference values for SNP.
	SNP []SNPReferenceValues `json:"snp,omitempty"`
	// TDX holds the reference values for TDX.
	TDX []TDXReferenceValues `json:"tdx,omitempty"`
}

// EmbeddedReferenceValues is a map of runtime handler names to a list of reference values
// for the runtime handler, as embedded in the binary.
type EmbeddedReferenceValues map[string]ReferenceValues

// SNPReferenceValues contains reference values for SEV-SNP.
type SNPReferenceValues struct {
	MinimumTCB         SNPTCB
	ProductName        ProductName
	TrustedMeasurement HexString
	GuestPolicy        abi.SnpPolicy
}

// TDXReferenceValues contains reference values for TDX.
type TDXReferenceValues struct {
	MrTd             HexString
	Rtrms            [4]HexString
	MinimumQeSvn     *uint16
	MinimumPceSvn    *uint16
	MinimumTeeTcbSvn HexString
	MrSeam           HexString
	TdAttributes     HexString
	Xfam             HexString
}

// SNPTCB represents a set of SEV-SNP TCB values.
type SNPTCB struct {
	BootloaderVersion *SVN
	TEEVersion        *SVN
	SNPVersion        *SVN
	MicrocodeVersion  *SVN
}

// SVN is a SNP secure version number.
type SVN uint8

// UInt8 returns the uint8 value of the SVN.
func (s *SVN) UInt8() uint8 {
	return uint8(*s)
}

// MarshalJSON marshals the SVN to JSON.
func (s SVN) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(s))), nil
}

// UnmarshalJSON unmarshals the SVN from a JSON.
func (s *SVN) UnmarshalJSON(data []byte) error {
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	if value < 0 || value > 255 { // Ensure the value fits into uint8 range
		return fmt.Errorf("value out of range for uint8")
	}

	*s = SVN(value)
	return nil
}

// ProductName is the name mentioned in the VCEK/ASK/ARK.
type ProductName string

const (
	// Milan is the product name for 3rd generation EPYC CPUs.
	Milan ProductName = "Milan"
	// Genoa is the product name for 4th generation EPYC CPUs.
	Genoa ProductName = "Genoa"
)

// HexString is a hex encoded string.
type HexString string

// NewHexString creates a new HexString from a byte slice.
func NewHexString(b []byte) HexString {
	return HexString(hex.EncodeToString(b))
}

// String returns the string representation of the HexString.
func (h HexString) String() string {
	return string(h)
}

// Bytes returns the byte slice representation of the HexString.
func (h HexString) Bytes() ([]byte, error) {
	return hex.DecodeString(string(h))
}

// ForPlatform returns the reference values for the given platform.
func (e *EmbeddedReferenceValues) ForPlatform(platform platforms.Platform) (*ReferenceValues, error) {
	var mapping EmbeddedReferenceValues
	if err := json.Unmarshal(EmbeddedReferenceValuesJSON, &mapping); err != nil {
		return nil, fmt.Errorf("unmarshal embedded reference values mapping: %w", err)
	}

	for handler, referenceValues := range mapping {
		p, err := platformFromHandler(handler)
		if err != nil {
			return nil, fmt.Errorf("invalid handler name: %w", err)
		}

		if p == platform {
			return &referenceValues, nil
		}
	}

	return nil, fmt.Errorf("no embedded reference values found for platform: %s", platform)
}

// platformFromHandler extracts the platform from the runtime handler name.
func platformFromHandler(handler string) (platforms.Platform, error) {
	rest, found := strings.CutPrefix(handler, "contrast-cc-")
	if !found {
		return platforms.Unknown, fmt.Errorf("invalid handler name: %s", handler)
	}

	parts := strings.Split(rest, "-")
	if len(parts) != 4 && len(parts) != 5 {
		return platforms.Unknown, fmt.Errorf("invalid handler name: %s", handler)
	}

	rawPlatform := strings.Join(parts[:len(parts)-1], "-")

	platform, err := platforms.FromString(rawPlatform)
	if err != nil {
		return platforms.Unknown, fmt.Errorf("invalid platform in handler name: %w", err)
	}

	return platform, nil
}
