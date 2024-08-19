// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/validate"
)

// Manifest is the Coordinator manifest and contains the reference values of the deployment.
type Manifest struct {
	// policyHash/HOSTDATA -> commonName
	Policies map[HexString]PolicyEntry
	// ReferenceValues specifies the allowed TEE configurations in the deployment. If ANY
	// of the reference values validates the attestation report of the workload,
	// the workload is considered valid.
	ReferenceValues         ReferenceValues
	WorkloadOwnerKeyDigests []HexString
	SeedshareOwnerPubKeys   []HexString
}

// PolicyEntry is a policy entry in the manifest. It contains further information the user wants to associate with the policy.
type PolicyEntry struct {
	SANs             []string
	WorkloadSecretID string `json:",omitempty"`
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
	for _, v := range r.SNP {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("validating SNP reference values: %w", err)
		}
	}
	for _, v := range r.TDX {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("validating TDX reference values: %w", err)
		}
	}

	if len(r.SNP)+len(r.TDX) == 0 {
		return fmt.Errorf("reference values in manifest cannot be empty. Is the chosen platform supported?")
	}

	return nil
}

// Validate checks the validity of all fields in the AKS reference values.
func (r SNPReferenceValues) Validate() error {
	if r.MinimumTCB.BootloaderVersion == nil {
		return fmt.Errorf("field BootloaderVersion in manifest cannot be empty")
	} else if r.MinimumTCB.TEEVersion == nil {
		return fmt.Errorf("field TEEVersion in manifest cannot be empty")
	} else if r.MinimumTCB.SNPVersion == nil {
		return fmt.Errorf("field SNPVersion in manifest cannot be empty")
	} else if r.MinimumTCB.MicrocodeVersion == nil {
		return fmt.Errorf("field MicrocodeVersion in manifest cannot be empty")
	}

	switch r.ProductName {
	case Milan, Genoa:
	default:
		return fmt.Errorf("unknown product name: %s", r.ProductName)
	}

	if len(r.TrustedMeasurement) != abi.MeasurementSize*2 {
		return fmt.Errorf("trusted measurement has invalid length: %d (expected %d)", len(r.TrustedMeasurement), abi.MeasurementSize*2)
	}

	return nil
}

// Validate checks the validity of all fields in the bare metal TDX reference values.
func (r TDXReferenceValues) Validate() error {
	if r.TrustedMeasurement == "" {
		return fmt.Errorf("field TrustedMeasurement in manifest cannot be empty")
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

// TODO(msanft): add generic validation interface for other attestation types.

// SNPValidateOpts returns validate options generators populated with the manifest's
// SNP reference values and trusted measurement for the given runtime.
func (m *Manifest) SNPValidateOpts() ([]*validate.Options, error) {
	if len(m.ReferenceValues.SNP) == 0 {
		return nil, errors.New("reference values cannot be empty")
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}

	var out []*validate.Options
	for _, refVal := range m.ReferenceValues.SNP {
		if len(refVal.TrustedMeasurement) == 0 {
			return nil, errors.New("trusted measurement cannot be empty")
		}

		trustedMeasurement, err := refVal.TrustedMeasurement.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert TrustedMeasurement from manifest to byte slices: %w", err)
		}

		out = append(out, &validate.Options{
			Measurement: trustedMeasurement,
			GuestPolicy: abi.SnpPolicy{
				Debug: false,
				SMT:   true,
			},
			VMPL: new(int), // VMPL0
			MinimumTCB: kds.TCBParts{
				BlSpl:    refVal.MinimumTCB.BootloaderVersion.UInt8(),
				TeeSpl:   refVal.MinimumTCB.TEEVersion.UInt8(),
				SnpSpl:   refVal.MinimumTCB.SNPVersion.UInt8(),
				UcodeSpl: refVal.MinimumTCB.MicrocodeVersion.UInt8(),
			},
			MinimumLaunchTCB: kds.TCBParts{
				BlSpl:    refVal.MinimumTCB.BootloaderVersion.UInt8(),
				TeeSpl:   refVal.MinimumTCB.TEEVersion.UInt8(),
				SnpSpl:   refVal.MinimumTCB.SNPVersion.UInt8(),
				UcodeSpl: refVal.MinimumTCB.MicrocodeVersion.UInt8(),
			},
			PermitProvisionalFirmware: true,
		})
	}

	if len(out) == 0 {
		return nil, errors.New("no SNP reference values found in manifest")
	}

	return out, nil
}
