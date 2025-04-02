// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/verify/trust"
)

// embeddedReferenceValuesJSON contains the embedded reference values in JSON format.
//
//go:embed assets/reference-values.json
var embeddedReferenceValuesJSON []byte

// EmbeddedReferenceValues is a map of runtime handler names to a list of reference values
// for the runtime handler, as embedded in the binary.
type EmbeddedReferenceValues map[string]ReferenceValues

// GetEmbeddedReferenceValues returns the reference values embedded in the binary.
func GetEmbeddedReferenceValues() (EmbeddedReferenceValues, error) {
	var mapping EmbeddedReferenceValues
	if err := json.Unmarshal(embeddedReferenceValuesJSON, &mapping); err != nil {
		return nil, fmt.Errorf("unmarshal embedded reference values mapping: %w", err)
	}
	return mapping, nil
}

// ForPlatform returns the reference values for the given platform.
func (e *EmbeddedReferenceValues) ForPlatform(platform platforms.Platform) (*ReferenceValues, error) {
	mapping, err := GetEmbeddedReferenceValues()
	if err != nil {
		return nil, err
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

// ReferenceValues contains the workload-independent reference values for each TEE type.
type ReferenceValues struct {
	// SNP holds the reference values for SNP.
	SNP []SNPReferenceValues `json:"snp,omitempty"`
	// TDX holds the reference values for TDX.
	TDX []TDXReferenceValues `json:"tdx,omitempty"`
}

// Validate checks the validity of all fields in the reference values.
func (r ReferenceValues) Validate() error {
	var errs []error
	for i, v := range r.SNP {
		if err := v.Validate(); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("snp[%d]", i), err))
		}
	}
	for i, v := range r.TDX {
		if err := v.Validate(); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("tdx[%d]", i), err))
		}
	}

	if len(r.SNP)+len(r.TDX) == 0 {
		errs = append(errs, fmt.Errorf("reference values in manifest cannot be empty. Is the chosen platform supported?"))
	}

	return errors.Join(errs...)
}

// SNPReferenceValues contains reference values for SEV-SNP.
type SNPReferenceValues struct {
	MinimumTCB         SNPTCB
	ProductName        ProductName
	TrustedMeasurement HexString
	GuestPolicy        abi.SnpPolicy
}

// Validate checks the validity of all fields in the AKS reference values.
func (r SNPReferenceValues) Validate() error {
	var minTCBErrs []error
	if r.MinimumTCB.BootloaderVersion == nil {
		minTCBErrs = append(minTCBErrs, newValidationError("BootloaderVersion", fmt.Errorf("field cannot be empty")))
	}
	if r.MinimumTCB.TEEVersion == nil {
		minTCBErrs = append(minTCBErrs, newValidationError("TEEVersion", fmt.Errorf("field cannot be empty")))
	}
	if r.MinimumTCB.SNPVersion == nil {
		minTCBErrs = append(minTCBErrs, newValidationError("SNPVersion", fmt.Errorf("field cannot be empty")))
	}
	if r.MinimumTCB.MicrocodeVersion == nil {
		minTCBErrs = append(minTCBErrs, newValidationError("MicrocodeVersion", fmt.Errorf("field cannot be empty")))
	}

	errs := []error{newValidationError("MinimumTCB", minTCBErrs...)}

	switch r.ProductName {
	case Milan, Genoa:
		// These are valid. We don't need to report an error.
	default:
		errs = append(errs, newValidationError("ProductName", fmt.Errorf("unknown product name: %s", r.ProductName)))
	}

	if err := validateHexString(r.TrustedMeasurement, abi.MeasurementSize); err != nil {
		errs = append(errs, newValidationError("TrustedMeasurement", err))
	}

	return errors.Join(errs...)
}

// SNPTCB represents a set of SEV-SNP TCB values.
type SNPTCB struct {
	BootloaderVersion *SVN
	TEEVersion        *SVN
	SNPVersion        *SVN
	MicrocodeVersion  *SVN
}

// ProductName is the name mentioned in the VCEK/ASK/ARK.
type ProductName string

const (
	// Milan is the product name for 3rd generation EPYC CPUs.
	Milan ProductName = "Milan"
	// Genoa is the product name for 4th generation EPYC CPUs.
	Genoa ProductName = "Genoa"
)

var (
	// source: https://kdsintf.amd.com/vcek/v1/Milan/cert_chain
	//go:embed Milan.pem
	askArkMilanVcekBytes []byte
	// source: https://kdsintf.amd.com/vcek/v1/Genoa/cert_chain
	//go:embed Genoa.pem
	askArkGenoaVcekBytes []byte
)

func amdTrustedRootCerts(productName ProductName) (map[string][]*trust.AMDRootCerts, error) {
	trustedRoots := make(map[string][]*trust.AMDRootCerts)

	switch productName {
	case Milan:
		milanCerts := trust.AMDRootCertsProduct("Milan")
		if err := milanCerts.FromKDSCertBytes(askArkMilanVcekBytes); err != nil {
			panic(fmt.Errorf("failed to parse cert: %w", err))
		}
		trustedRoots["Milan"] = []*trust.AMDRootCerts{milanCerts}
	case Genoa:
		genoaCerts := trust.AMDRootCertsProduct("Genoa")
		if err := genoaCerts.FromKDSCertBytes(askArkGenoaVcekBytes); err != nil {
			panic(fmt.Errorf("failed to parse cert: %w", err))
		}
		trustedRoots["Genoa"] = []*trust.AMDRootCerts{genoaCerts}
	default:
		return nil, fmt.Errorf("unknown product name: %s", productName)
	}

	return trustedRoots, nil
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

// Validate checks the validity of all fields in the bare metal TDX reference values.
func (r TDXReferenceValues) Validate() error {
	var errs []error
	if err := validateHexString(r.MrTd, 48); err != nil {
		errs = append(errs, newValidationError("MrTd", err))
	}
	if r.MinimumQeSvn == nil {
		errs = append(errs, newValidationError("MinimumQeSvn", fmt.Errorf("field cannot be empty")))
	}
	if r.MinimumPceSvn == nil {
		errs = append(errs, newValidationError("MinimumPceSvn", fmt.Errorf("field cannot be empty")))
	}
	if err := validateHexString(r.MinimumTeeTcbSvn, 16); err != nil {
		errs = append(errs, newValidationError("MinimumTeeTcbSvn", err))
	}
	if err := validateHexString(r.MrSeam, 48); err != nil {
		errs = append(errs, newValidationError("MrSeam", err))
	}
	if err := validateHexString(r.TdAttributes, 8); err != nil {
		errs = append(errs, newValidationError("TdAttributes", err))
	}
	if err := validateHexString(r.Xfam, 8); err != nil {
		errs = append(errs, newValidationError("Xfam", err))
	}
	for i, rtmr := range r.Rtrms {
		if err := validateHexString(rtmr, 48); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("Rtrms[%d]", i), err))
		}
	}
	return errors.Join(errs...)
}

// The QE Vendor ID used by Intel.
var intelQeVendorID = []byte{0x93, 0x9a, 0x72, 0x33, 0xf7, 0x9c, 0x4c, 0xa9, 0x94, 0x0a, 0x0d, 0xb3, 0x95, 0x7f, 0x06, 0x07}

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
