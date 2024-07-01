// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/validate"
)

// Manifest is the Coordinator manifest and contains the reference values of the deployment.
type Manifest struct {
	// policyHash/HOSTDATA -> commonName
	Policies                map[HexString][]string
	ReferenceValues         ReferenceValues
	WorkloadOwnerKeyDigests []HexString
	SeedshareOwnerPubKeys   []HexString
}

// ReferenceValues contains the workload independent reference values.
type ReferenceValues struct {
	SNP SNPReferenceValues
	// TrustedMeasurement is the hash of the trusted launch digest.
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

// HexStrings is a slice of HexString.
type HexStrings []HexString

// ByteSlices returns the byte slice representation of the HexStrings.
func (l *HexStrings) ByteSlices() ([][]byte, error) {
	var res [][]byte
	for _, s := range *l {
		b, err := s.Bytes()
		if err != nil {
			return nil, err
		}
		res = append(res, b)
	}
	return res, nil
}

// Policy is a CocCo execution policy.
type Policy []byte

// NewPolicyFromAnnotation parses a base64 encoded policy from an annotation.
func NewPolicyFromAnnotation(annotation []byte) (Policy, error) {
	return base64.StdEncoding.DecodeString(string(annotation))
}

// Bytes returns the policy as byte slice.
func (p Policy) Bytes() []byte {
	return []byte(p)
}

// Hash returns the hash of the policy.
func (p Policy) Hash() HexString {
	hashBytes := sha256.Sum256(p)
	return NewHexString(hashBytes[:])
}

// Validate checks the validity of all fields in the reference values.
func (r ReferenceValues) Validate() error {
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

// Validate checks the validity of all fields in the manifest.
func (m *Manifest) Validate() error {
	for policyHash := range m.Policies {
		if _, err := policyHash.Bytes(); err != nil {
			return fmt.Errorf("decoding policy hash %s: %w", policyHash, err)
		} else if len(policyHash) != sha256.Size*2 {
			return fmt.Errorf("policy hash %s has invalid length: %d (expected %d)", policyHash, len(policyHash), sha256.Size*2)
		}
	}

	if err := m.ReferenceValues.Validate(); err != nil {
		return fmt.Errorf("validating reference values: %w", err)
	}

	for _, keyDigest := range m.WorkloadOwnerKeyDigests {
		if _, err := keyDigest.Bytes(); err != nil {
			return fmt.Errorf("decoding key digest %s: %w", keyDigest, err)
		} else if len(keyDigest) != sha256.Size*2 {
			return fmt.Errorf("workload owner key digest %s has invalid length: %d (expected %d)", keyDigest, len(keyDigest), sha256.Size*2)
		}
	}

	for _, key := range m.SeedshareOwnerPubKeys {
		if _, err := ParseSeedShareOwnerKey(key); err != nil {
			return fmt.Errorf("invalid seed share owner public key %s: %w", key, err)
		}
	}
	return nil
}

// SNPValidateOpts returns validate options populated with the manifest's
// SNP reference values and trusted measurement.
func (m *Manifest) SNPValidateOpts() (*validate.Options, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}
	trustedMeasurement, err := m.ReferenceValues.TrustedMeasurement.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to convert TrustedMeasurement from manifest to byte slices: %w", err)
	}

	return &validate.Options{
		Measurement: trustedMeasurement,
		GuestPolicy: abi.SnpPolicy{
			Debug: false,
			SMT:   true,
		},
		VMPL: new(int), // VMPL0
		MinimumTCB: kds.TCBParts{
			BlSpl:    m.ReferenceValues.SNP.MinimumTCB.BootloaderVersion.UInt8(),
			TeeSpl:   m.ReferenceValues.SNP.MinimumTCB.TEEVersion.UInt8(),
			SnpSpl:   m.ReferenceValues.SNP.MinimumTCB.SNPVersion.UInt8(),
			UcodeSpl: m.ReferenceValues.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
		},
		MinimumLaunchTCB: kds.TCBParts{
			BlSpl:    m.ReferenceValues.SNP.MinimumTCB.BootloaderVersion.UInt8(),
			TeeSpl:   m.ReferenceValues.SNP.MinimumTCB.TEEVersion.UInt8(),
			SnpSpl:   m.ReferenceValues.SNP.MinimumTCB.SNPVersion.UInt8(),
			UcodeSpl: m.ReferenceValues.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
		},
		PermitProvisionalFirmware: true,
	}, nil
}
