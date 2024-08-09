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
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
)

// Manifest is the Coordinator manifest and contains the reference values of the deployment.
type Manifest struct {
	// policyHash/HOSTDATA -> commonName
	Policies                map[HexString]PolicyEntry
	ReferenceValues         []ReferenceValues
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
	if r.SNP != nil {
		if err := r.SNP.Validate(); err != nil {
			return fmt.Errorf("validating AKS reference values: %w", err)
		}
	}
	if r.TDX != nil {
		if err := r.TDX.Validate(); err != nil {
			return fmt.Errorf("validating bare metal TDX reference values: %w", err)
		}
	}

	if r.TDX == nil && r.SNP == nil {
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

	for i, rv := range m.ReferenceValues {
		if err := rv.Validate(); err != nil {
			return fmt.Errorf("validating reference values [%d]: %w", i, err)
		}
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

// SNPValidateOptsGenerator generates SNP validation options and
// can be instantiated from a manifest only.
type SNPValidateOptsGenerator struct {
	opts        *validate.Options
	manifest    *Manifest
	extraChecks []func(report *sevsnp.Report) error // additional checks that need to pass for the validation to succeed.
}

// SNPValidateOpts returns the SNP validation options.
func (g *SNPValidateOptsGenerator) SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error) {
	for _, check := range g.extraChecks {
		if err := check(report); err != nil {
			return nil, fmt.Errorf("additional check failed: %w", err)
		}
	}
	return g.opts, nil
}

// TODO(msanft): add generic validation interface for other attestation types.

// SNPValidateOpts returns validate options generators populated with the manifest's
// SNP reference values and trusted measurement for the given runtime.
func (m *Manifest) SNPValidateOpts() ([]*SNPValidateOptsGenerator, error) {
	if m.ReferenceValues == nil {
		return nil, errors.New("reference values cannot be empty")
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}

	var out []*SNPValidateOptsGenerator
	for _, refVal := range m.ReferenceValues {
		trustedMeasurement, err := refVal.SNP.TrustedMeasurement.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert TrustedMeasurement from manifest to byte slices: %w", err)
		}

		out = append(out, &SNPValidateOptsGenerator{
			manifest: m,
			opts: &validate.Options{
				Measurement: trustedMeasurement,
				GuestPolicy: abi.SnpPolicy{
					Debug: false,
					SMT:   true,
				},
				VMPL: new(int), // VMPL0
				MinimumTCB: kds.TCBParts{
					BlSpl:    refVal.SNP.MinimumTCB.BootloaderVersion.UInt8(),
					TeeSpl:   refVal.SNP.MinimumTCB.TEEVersion.UInt8(),
					SnpSpl:   refVal.SNP.MinimumTCB.SNPVersion.UInt8(),
					UcodeSpl: refVal.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
				},
				MinimumLaunchTCB: kds.TCBParts{
					BlSpl:    refVal.SNP.MinimumTCB.BootloaderVersion.UInt8(),
					TeeSpl:   refVal.SNP.MinimumTCB.TEEVersion.UInt8(),
					SnpSpl:   refVal.SNP.MinimumTCB.SNPVersion.UInt8(),
					UcodeSpl: refVal.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
				},
				PermitProvisionalFirmware: true,
			},
		})
	}

	if len(out) == 0 {
		return nil, errors.New("no AKS reference values found in manifest")
	}

	return out, nil
}

// WithReportHostData augments the validate options generator with
// a check that verifies whether the policy hash in the report's
// HOSTDATA is allowed by the manifest.
func (g *SNPValidateOptsGenerator) WithReportHostData() *SNPValidateOptsGenerator {
	g.extraChecks = append(g.extraChecks, func(report *sevsnp.Report) error {
		hostData := NewHexString(report.HostData)
		if _, ok := g.manifest.Policies[hostData]; !ok {
			return fmt.Errorf("hostdata %s not found in manifest", hostData)
		}
		return nil
	})
	return g
}

// WithStaticHostData augments the validate options with
// the given host data.
func (g *SNPValidateOptsGenerator) WithStaticHostData(hostData []byte) *SNPValidateOptsGenerator {
	g.opts.HostData = hostData
	return g
}
