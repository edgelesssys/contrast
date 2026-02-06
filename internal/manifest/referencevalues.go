// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"bytes"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/edgelesssys/contrast/internal/platforms"
	jsonpatch "github.com/evanphx/json-patch/v5"
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
	decoder := json.NewDecoder(bytes.NewReader(embeddedReferenceValuesJSON))
	decoder.DisallowUnknownFields()
	var mapping EmbeddedReferenceValues
	if err := decoder.Decode(&mapping); err != nil {
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
		var ve *ValidationError
		err := v.Validate()
		if errors.As(err, &ve) && ve.OnlyExpectedMissingReferenceValues() {
			errs = append(errs, ExpectedMissingReferenceValueError{Err: fmt.Errorf("snp[%d]", i)})
		} else if err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("snp[%d]", i), err))
		}
	}
	for i, v := range r.TDX {
		var ve *ValidationError
		err := v.Validate()
		if errors.As(err, &ve) && ve.OnlyExpectedMissingReferenceValues() {
			errs = append(errs, ExpectedMissingReferenceValueError{Err: fmt.Errorf("tdx[%d]", i)})
		} else if err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("tdx[%d]", i), err))
		}
	}

	if len(r.SNP)+len(r.TDX) == 0 {
		errs = append(errs, fmt.Errorf("reference values in manifest cannot be empty. Is the chosen platform supported?"))
	}

	return errors.Join(errs...)
}

// PurgeEmpty modifies r in-place to remove empty/not-filled-in reference values.
// Emptyness is decided by checking if validation only returns ExpectedMissingReferenceValueError errors.
func (r *ReferenceValues) PurgeEmpty() {
	snp := []SNPReferenceValues{}
	for _, v := range r.SNP {
		var ve *ValidationError
		err := v.Validate()
		if errors.As(err, &ve) && ve.OnlyExpectedMissingReferenceValues() {
			continue
		}
		snp = append(snp, v)
	}
	r.SNP = snp

	tdx := []TDXReferenceValues{}
	for _, v := range r.TDX {
		var ve *ValidationError
		err := v.Validate()
		if errors.As(err, &ve) && ve.OnlyExpectedMissingReferenceValues() {
			continue
		}
		tdx = append(tdx, v)
	}
	r.TDX = tdx
}

// ReferenceValuePatches is a slice of json-patch patchsets.
type ReferenceValuePatches []json.RawMessage

// PatchesFromFiles takes a slice of file paths and returns a patchset.
func PatchesFromFiles(files []string) (ReferenceValuePatches, error) {
	if len(files) == 0 {
		return nil, nil
	}

	patches := ReferenceValuePatches{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file, err)
		}
		var p ReferenceValuePatches
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON in file %s: %w", file, err)
		}
		patches = append(patches, p...)
	}
	return patches, nil
}

// Patch applies each patch in the given ReferenceValuePatches to r.
// The json-patches can contain "test"-ops, most commonly for the Plaftorm and/or ProductName.
// If application of a patch fails due to one of the "test"-ops failing, we interpret this
// as the patch not being intended for the ReferenceValues under consideration.
func (r *ReferenceValues) Patch(patches ReferenceValuePatches) error {
	applyPatch := func(refVals any, patch jsonpatch.Patch) error {
		refValBytes, err := json.Marshal(refVals)
		if err != nil {
			return fmt.Errorf("marshaling reference values for patching: %w", err)
		}
		modifiedrefValBytes, err := patch.Apply(refValBytes)
		if errors.Is(err, jsonpatch.ErrTestFailed) {
			// The patch was not meant for these refVals.
			return nil
		} else if err != nil {
			return fmt.Errorf("applying specs patch to reference values: %w", err)
		}
		if err := json.Unmarshal(modifiedrefValBytes, refVals); err != nil {
			return fmt.Errorf("unmarshaling patched reference values: %w", err)
		}
		return nil
	}

	for i, refValPatch := range patches {
		patch, err := jsonpatch.DecodePatch(refValPatch)
		if err != nil {
			return fmt.Errorf("decoding specs patch[%d]: %w", i, err)
		}

		for j := range r.SNP {
			if err := applyPatch(&r.SNP[j], patch); err != nil {
				return fmt.Errorf("applying patch[%d] to SNP reference values: %w", i, err)
			}
		}

		for j := range r.TDX {
			if err := applyPatch(&r.TDX[j], patch); err != nil {
				return fmt.Errorf("applying patch[%d] to TDX reference values: %w", i, err)
			}
		}
	}
	return nil
}

// SNPReferenceValues contains reference values for SEV-SNP.
type SNPReferenceValues struct {
	Platform                string
	ProductName             ProductName
	TrustedMeasurement      HexString
	MinimumTCB              SNPTCB
	GuestPolicy             abi.SnpPolicy
	PlatformInfo            abi.SnpPlatformInfo
	MinimumMitigationVector uint64
	AllowedChipIDs          []HexString
}

// Validate checks the validity of all fields in the AKS reference values.
func (r SNPReferenceValues) Validate() error {
	var minTCBErrs []error
	if r.MinimumTCB.BootloaderVersion == nil {
		minTCBErrs = append(minTCBErrs, newValidationError("BootloaderVersion", ExpectedMissingReferenceValueError{Err: errors.New("field cannot be empty")}))
	}
	if r.MinimumTCB.TEEVersion == nil {
		minTCBErrs = append(minTCBErrs, newValidationError("TEEVersion", ExpectedMissingReferenceValueError{Err: errors.New("field cannot be empty")}))
	}
	if r.MinimumTCB.SNPVersion == nil {
		minTCBErrs = append(minTCBErrs, newValidationError("SNPVersion", ExpectedMissingReferenceValueError{Err: errors.New("field cannot be empty")}))
	}
	if r.MinimumTCB.MicrocodeVersion == nil {
		minTCBErrs = append(minTCBErrs, newValidationError("MicrocodeVersion", ExpectedMissingReferenceValueError{Err: errors.New("field cannot be empty")}))
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

	noModificationPermittedErr := errors.New("modifying this field is not permitted")
	var guestPolicyErrs []error
	if r.GuestPolicy.ABIMajor != 0 {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("ABIMajor", noModificationPermittedErr))
	}
	// AbiMinor is 0 on bare metal and 31 on AKS.
	if r.GuestPolicy.ABIMinor != 0 && r.GuestPolicy.ABIMinor != 31 {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("ABIMinor", noModificationPermittedErr))
	}
	if !r.GuestPolicy.SMT {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("SMT", noModificationPermittedErr))
	}
	if r.GuestPolicy.MigrateMA {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("MigrateMA", noModificationPermittedErr))
	}
	if r.GuestPolicy.Debug {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("Debug", noModificationPermittedErr))
	}
	if r.GuestPolicy.SingleSocket {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("SingleSocket", noModificationPermittedErr))
	}
	if r.GuestPolicy.CXLAllowed {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("CXLAllowed", noModificationPermittedErr))
	}
	if r.GuestPolicy.MemAES256XTS {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("MemAES256XTS", noModificationPermittedErr))
	}
	if r.GuestPolicy.RAPLDis {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("RAPLDis", noModificationPermittedErr))
	}
	if r.GuestPolicy.CipherTextHidingDRAM {
		guestPolicyErrs = append(guestPolicyErrs, newValidationError("CipherTextHidingDRAM", noModificationPermittedErr))
	}
	errs = append(errs, newValidationError("GuestPolicy", guestPolicyErrs...))

	for i, chipID := range r.AllowedChipIDs {
		if err := validateHexString(chipID, abi.ChipIDSize); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("AllowedChipIDs[%d]", i), err))
		}
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
	Platform                   string
	MinTCBEvaluationDataNumber int
	MrTd                       HexString
	MrSeam                     HexString
	Rtmrs                      [4]HexString
	Xfam                       HexString
	AllowedPIIDs               []HexString
	MemoryIntegrity            bool
	SMTDisabled                bool
	StaticPlatform             bool
}

// Validate checks the validity of all fields in the bare metal TDX reference values.
func (r TDXReferenceValues) Validate() error {
	var errs []error
	if err := validateHexString(r.MrTd, 48); err != nil {
		errs = append(errs, newValidationError("MrTd", err))
	}
	if err := validateHexString(r.MrSeam, 48); err != nil {
		errs = append(errs, newValidationError("MrSeam", ExpectedMissingReferenceValueError{Err: err}))
	}
	if err := validateHexString(r.Xfam, 8); err != nil {
		errs = append(errs, newValidationError("Xfam", err))
	}
	for i, rtmr := range r.Rtmrs {
		if err := validateHexString(rtmr, 48); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("RTMR[%d]", i+1), err))
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

// Even though the vendored file has "SGX" in its name, it is the general "Provisioning Certificate for ECDSA Attestation"
// from Intel and used for both SGX *and* TDX.
//
// See https://api.portal.trustedservices.intel.com/content/documentation.html#pcs for more information.
//
// File Source: https://certificates.trustedservices.intel.com/Intel_SGX_Provisioning_Certification_RootCA.pem
//
//go:embed Intel_SGX_Provisioning_Certification_RootCA.pem
var tdxRootCert []byte

func tdxTrustedRootCerts() (*x509.CertPool, error) {
	rootCerts := x509.NewCertPool()
	if ok := rootCerts.AppendCertsFromPEM(tdxRootCert); !ok {
		return nil, fmt.Errorf("failed to append root certificate")
	}
	return rootCerts, nil
}
