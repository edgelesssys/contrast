// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package platforms

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/go-sev-guest/abi"
)

// EmbeddedReferenceValuesJSON contains the embedded reference values in JSON format.
// At startup, they are unmarshaled into a globally-shared ReferenceValues struct.
//
//go:embed assets/reference-values.json
var EmbeddedReferenceValuesJSON []byte

// EmbeddedReferenceValues returns the reference values embedded in the binary.
func EmbeddedReferenceValues() ReferenceValues {
	var embeddedReferenceValues *ReferenceValues

	if err := json.Unmarshal(EmbeddedReferenceValuesJSON, &embeddedReferenceValues); err != nil {
		// As this relies on a constant, predictable value (i.e. the embedded JSON), which -- in a correctly built binary -- should
		// unmarshal safely into the [ReferenceValues], it's acceptable to panic here.
		panic(fmt.Errorf("failed to unmarshal embedded reference values: %w", err))
	}

	return *embeddedReferenceValues
}

// ReferenceValues contains the workload-independent reference values for each platform.
type ReferenceValues struct {
	// AKS holds the reference values for AKS.
	AKS *AKSReferenceValues `json:"aks,omitempty"`
	// BareMetalTDX holds the reference values for TDX on bare metal.
	BareMetalTDX *BareMetalTDXReferenceValues `json:"bareMetalTDX,omitempty"`
}

// AKSReferenceValues contains reference values for AKS.
type AKSReferenceValues struct {
	SNP                SNPReferenceValues
	TrustedMeasurement HexString
}

// BareMetalTDXReferenceValues contains reference values for BareMetalTDX.
type BareMetalTDXReferenceValues struct {
	TrustedMeasurement HexString
}

// SNPReferenceValues contains reference values for the SNP report.
type SNPReferenceValues struct {
	MinimumTCB SNPTCB
}

// SNPTCB represents a set of SNP TCB values.
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

// Validate checks the validity of all fields in the reference values.
func (r ReferenceValues) Validate() error {
	if r.AKS != nil {
		if err := r.AKS.Validate(); err != nil {
			return fmt.Errorf("validating AKS reference values: %w", err)
		}
	}
	if r.BareMetalTDX != nil {
		if err := r.BareMetalTDX.Validate(); err != nil {
			return fmt.Errorf("validating bare metal TDX reference values: %w", err)
		}
	}

	if r.BareMetalTDX == nil && r.AKS == nil {
		return fmt.Errorf("reference values in manifest cannot be empty. Is the chosen platform supported?")
	}

	return nil
}

// Validate checks the validity of all fields in the AKS reference values.
func (r AKSReferenceValues) Validate() error {
	if r.SNP.MinimumTCB.BootloaderVersion == nil {
		return fmt.Errorf("field BootloaderVersion in manifest cannot be empty")
	} else if r.SNP.MinimumTCB.TEEVersion == nil {
		return fmt.Errorf("field TEEVersion in manifest cannot be empty")
	} else if r.SNP.MinimumTCB.SNPVersion == nil {
		return fmt.Errorf("field SNPVersion in manifest cannot be empty")
	} else if r.SNP.MinimumTCB.MicrocodeVersion == nil {
		return fmt.Errorf("field MicrocodeVersion in manifest cannot be empty")
	}

	if len(r.TrustedMeasurement) != abi.MeasurementSize*2 {
		return fmt.Errorf("trusted measurement has invalid length: %d (expected %d)", len(r.TrustedMeasurement), abi.MeasurementSize*2)
	}

	return nil
}

// Validate checks the validity of all fields in the bare metal TDX reference values.
func (r BareMetalTDXReferenceValues) Validate() error {
	if r.TrustedMeasurement == "" {
		return fmt.Errorf("field TrustedMeasurement in manifest cannot be empty")
	}
	return nil
}
