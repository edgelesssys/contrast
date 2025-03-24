// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"crypto/sha256"
	"crypto/sha512"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/idblock"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	snpvalidate "github.com/google/go-sev-guest/validate"
	"github.com/google/go-sev-guest/verify"
	"github.com/google/go-sev-guest/verify/trust"
	tdxvalidate "github.com/google/go-tdx-guest/validate"
)

// Manifest is the Coordinator manifest and contains the reference values of the deployment.
type Manifest struct {
	// Policies is a map from policy hash (HOSTDATA) to policy entry.
	Policies map[HexString]PolicyEntry
	// ReferenceValues specifies the allowed TEE configurations in the deployment. If ANY
	// of the reference values validates the attestation report of the workload,
	// the workload is considered valid.
	ReferenceValues ReferenceValues
	// WorkloadOwnerKeyDigests is a list of ECDSA public keys in PKIX DER format, hashed with SHA256 and hex-encoded.
	WorkloadOwnerKeyDigests []HexString
	// SeedshareOwnerPubKeys is a list of RSA public keys in PKCS1 DER format, hex-encoded.
	SeedshareOwnerPubKeys []HexString
}

// PolicyEntry is a policy entry in the manifest. It contains further information the user wants to associate with the policy.
type PolicyEntry struct {
	SANs             []string
	WorkloadSecretID string `json:",omitempty"`
	Role             Role   `json:",omitempty"`
}

// Validate checks the validity of a policy entry given its policy hash.
func (e PolicyEntry) Validate(policyHash HexString) error {
	var errs []error
	if _, err := policyHash.Bytes(); err != nil {
		errs = append(errs, fmt.Errorf("decoding policy hash %q: %w", policyHash, err))
	} else if len(policyHash) != hex.EncodedLen(sha256.Size) {
		errs = append(errs, fmt.Errorf("invalid policy hash length: %d (expected %d)", len(policyHash), hex.EncodedLen(sha256.Size)))
	}

	if err := e.Role.Validate(); err != nil {
		errs = append(errs, newValidationError("Role", err))
	}

	return errors.Join(errs...)
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

// Role is the role of the workload identified by the policy hash.
type Role string

const (
	// RoleNone means the workload has no specific role.
	RoleNone Role = ""
	// RoleCoordinator is the coordinator role.
	RoleCoordinator Role = "coordinator"
)

// Validate checks the validity of the role.
func (r Role) Validate() error {
	switch r {
	case RoleNone, RoleCoordinator:
		return nil
	default:
		return fmt.Errorf("unknown role: %s", r)
	}
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

func validateHexString(value HexString, expectedNumBytes int) error {
	if len(value) != expectedNumBytes*2 {
		return fmt.Errorf("invalid length: %d (expected %d)", len(value), expectedNumBytes*2)
	}
	_, err := value.Bytes()
	return err
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

// Validate checks the validity of all fields in the manifest.
func (m *Manifest) Validate() error {
	var errs []error
	for policyHash, policy := range m.Policies {
		if err := policy.Validate(policyHash); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("Policies[%q]", policyHash), err))
		}
	}

	var coordinatorCount int
	for _, policy := range m.Policies {
		if policy.Role == RoleCoordinator {
			coordinatorCount++
		}
	}
	if coordinatorCount != 1 {
		return fmt.Errorf("expected exactly 1 policy with role 'coordinator', got %d", coordinatorCount)
	}

	if err := m.ReferenceValues.Validate(); err != nil {
		errs = append(errs, newValidationError("ReferenceValues", err))
	}

	for i, keyDigest := range m.WorkloadOwnerKeyDigests {
		if _, err := keyDigest.Bytes(); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("WorkloadOwnerKeyDigests[%d]", i), err))
		} else if len(keyDigest) != hex.EncodedLen(sha256.Size) {
			errs = append(errs, newValidationError(fmt.Sprintf("WorkloadOwnerKeyDigests[%d]", i), fmt.Errorf("invalid length: %d (expected %d)", len(keyDigest), hex.EncodedLen(sha256.Size))))
		}
	}

	for i, key := range m.SeedshareOwnerPubKeys {
		if _, err := ParseSeedShareOwnerKey(key); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("SeedshareOwnerPubKeys[%d]", i), err))
		}
	}
	return errors.Join(errs...)
}

// validationError contains a JSON path and a list of errors.
// Nested validation errors are printed on newlines with the full path.
type validationError struct {
	path string
	errs []error
}

func newValidationError(path string, errs ...error) error {
	e := &validationError{
		path: path,
		errs: make([]error, 0, len(errs)),
	}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, flatten(err)...)
		}
	}
	if len(e.errs) == 0 {
		return nil
	}
	return e
}

func (e *validationError) Error() string {
	return e.formatError(e.path)
}

func (e *validationError) Unwrap() []error {
	return e.errs
}

func (e *validationError) formatError(path string) string {
	var sb strings.Builder
	for i, err := range e.errs {
		var ve *validationError
		if errors.As(err, &ve) {
			sb.WriteString(ve.formatError(path + "." + ve.path))
		} else {
			sb.WriteString(path)
			sb.WriteString(": ")
			sb.WriteString(err.Error())
		}
		if i < len(e.errs)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func flatten(err error) (errs []error) {
	if ve, ok := err.(*validationError); ok { //nolint:errorlint // check for exact type
		return []error{ve}
	}
	if wrapped, ok := err.(interface{ Unwrap() []error }); ok {
		for _, err := range wrapped.Unwrap() {
			errs = append(errs, flatten(err)...)
		}
		return errs
	}
	return []error{err}
}

// TODO(msanft): add generic validation interface for other attestation types.

// ValidatorOptions contains the verification and validation options to be used
// by a Validator.
type ValidatorOptions struct {
	VerifyOpts   *verify.Options
	ValidateOpts *snpvalidate.Options
}

// SNPValidateOpts returns validate options generators populated with the manifest's
// SNP reference values and trusted measurement for the given runtime.
func (m *Manifest) SNPValidateOpts(kdsGetter trust.HTTPSGetter) ([]ValidatorOptions, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}

	var out []ValidatorOptions
	for _, refVal := range m.ReferenceValues.SNP {
		if len(refVal.TrustedMeasurement) == 0 {
			return nil, errors.New("trusted measurement cannot be empty")
		}

		trustedMeasurement, err := refVal.TrustedMeasurement.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert TrustedMeasurement from manifest to byte slices: %w", err)
		}

		verifyOpts := verify.DefaultOptions()
		// Setting the productLine explicitly, because of full dependence of trustedMeasurements and derivation of trustedRoots on productLine.
		verifyOpts.Product, err = kds.ParseProductLine(string(refVal.ProductName))
		if err != nil {
			return nil, fmt.Errorf("SNP reference values: %w", err)
		}
		verifyOpts.TrustedRoots, err = trustedRoots(refVal.ProductName)
		if err != nil {
			return nil, fmt.Errorf("determine trusted roots: %w", err)
		}
		verifyOpts.CheckRevocations = true
		verifyOpts.Getter = kdsGetter

		// Generate static public IDKey based on the launch digest and guest policy.
		_, authBlk, err := idblock.IDBlocksFromLaunchDigest([48]byte(trustedMeasurement), refVal.GuestPolicy)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID blocks: %w", err)
		}
		idKeyBytes, err := authBlk.IDKey.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal IDKey: %w", err)
		}
		idKeyHash := sha512.Sum384(idKeyBytes)

		validateOpts := snpvalidate.Options{
			Measurement: trustedMeasurement,
			GuestPolicy: refVal.GuestPolicy,
			VMPL:        new(int), // VMPL0
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
			RequireIDBlock:            true,
			TrustedIDKeyHashes:        [][]byte{idKeyHash[:]},
		}

		out = append(out, ValidatorOptions{VerifyOpts: verifyOpts, ValidateOpts: &validateOpts})
	}

	return out, nil
}

var (
	// source: https://kdsintf.amd.com/vcek/v1/Milan/cert_chain
	//go:embed Milan.pem
	askArkMilanVcekBytes []byte
	// source: https://kdsintf.amd.com/vcek/v1/Genoa/cert_chain
	//go:embed Genoa.pem
	askArkGenoaVcekBytes []byte
)

func trustedRoots(productName ProductName) (map[string][]*trust.AMDRootCerts, error) {
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

// The QE Vendor ID used by Intel.
var intelQeVendorID = []byte{0x93, 0x9a, 0x72, 0x33, 0xf7, 0x9c, 0x4c, 0xa9, 0x94, 0x0a, 0x0d, 0xb3, 0x95, 0x7f, 0x06, 0x07}

// TDXValidateOpts returns validate options generators populated with the manifest's
// TDX reference values and trusted measurement for the given runtime.
func (m *Manifest) TDXValidateOpts() ([]*tdxvalidate.Options, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}

	var out []*tdxvalidate.Options
	for _, refVal := range m.ReferenceValues.TDX {
		mrTd, err := refVal.MrTd.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert MrTd from manifest to byte slices: %w", err)
		}

		minimumTeeTcbSvn, err := refVal.MinimumTeeTcbSvn.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert MinimumTeeTcbSvn from manifest to byte slices: %w", err)
		}

		mrSeam, err := refVal.MrSeam.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert MrSeam from manifest to byte slices: %w", err)
		}

		tdAttributes, err := refVal.TdAttributes.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert TdAttributes from manifest to byte slices: %w", err)
		}

		xfam, err := refVal.Xfam.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert Xfam from manifest to byte slices: %w", err)
		}

		var rtmrs [4][]byte
		for i, rtmr := range refVal.Rtrms {
			bytes, err := rtmr.Bytes()
			if err != nil {
				return nil, fmt.Errorf("failed to convert Rtmr[%d] from manifest to byte slices: %w", i, err)
			}
			rtmrs[i] = bytes
		}

		out = append(out, &tdxvalidate.Options{
			HeaderOptions: tdxvalidate.HeaderOptions{
				MinimumQeSvn:  *refVal.MinimumQeSvn,
				MinimumPceSvn: *refVal.MinimumPceSvn,
				QeVendorID:    intelQeVendorID,
			},
			TdQuoteBodyOptions: tdxvalidate.TdQuoteBodyOptions{
				MinimumTeeTcbSvn: minimumTeeTcbSvn,
				MrSeam:           mrSeam,
				TdAttributes:     tdAttributes,
				Xfam:             xfam,
				MrTd:             mrTd,
				Rtmrs:            rtmrs[:],
			},
		})
		_ = mrTd
		_ = rtmrs
	}

	return out, nil
}

// CoordinatorPolicyHash returns the hash of the coordinator policy.
func (m *Manifest) CoordinatorPolicyHash() (HexString, error) {
	for policyHash, policy := range m.Policies {
		if policy.Role == RoleCoordinator {
			return policyHash, nil
		}
	}
	return "", errors.New("no coordinator found in manifest")
}
