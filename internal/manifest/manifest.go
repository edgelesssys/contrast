// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/edgelesssys/contrast/node-installer/platforms"
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

// AKSValidateOpts returns validate options populated with the manifest's
// AKS reference values and trusted measurement.
func (m *Manifest) AKSValidateOpts() (*validate.Options, error) {
	if m.ReferenceValues.AKS == nil {
		return nil, fmt.Errorf("no AKS reference values present in manifest")
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}
	trustedMeasurement, err := m.ReferenceValues.AKS.TrustedMeasurement.Bytes()
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
			BlSpl:    m.ReferenceValues.AKS.SNP.MinimumTCB.BootloaderVersion.UInt8(),
			TeeSpl:   m.ReferenceValues.AKS.SNP.MinimumTCB.TEEVersion.UInt8(),
			SnpSpl:   m.ReferenceValues.AKS.SNP.MinimumTCB.SNPVersion.UInt8(),
			UcodeSpl: m.ReferenceValues.AKS.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
		},
		MinimumLaunchTCB: kds.TCBParts{
			BlSpl:    m.ReferenceValues.AKS.SNP.MinimumTCB.BootloaderVersion.UInt8(),
			TeeSpl:   m.ReferenceValues.AKS.SNP.MinimumTCB.TEEVersion.UInt8(),
			SnpSpl:   m.ReferenceValues.AKS.SNP.MinimumTCB.SNPVersion.UInt8(),
			UcodeSpl: m.ReferenceValues.AKS.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
		},
		PermitProvisionalFirmware: true,
	}, nil
}

// RuntimeHandler returns the runtime handler for the given platform.
func (m *Manifest) RuntimeHandler(platform platforms.Platform) (string, error) {
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		return fmt.Sprintf("contrast-cc-%s", m.ReferenceValues.AKS.TrustedMeasurement[:32]), nil
	case platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		return fmt.Sprintf("contrast-cc-%s", m.ReferenceValues.BareMetalTDX.TrustedMeasurement[:32]), nil
	default:
		return "", fmt.Errorf("unsupported platform %s", platform)
	}
}
