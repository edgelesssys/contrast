// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/idblock"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/google/go-sev-guest/kds"
	snpvalidate "github.com/google/go-sev-guest/validate"
	snpverify "github.com/google/go-sev-guest/verify"
	tdxvalidate "github.com/google/go-tdx-guest/validate"
	tdxverify "github.com/google/go-tdx-guest/verify"
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

// Default returns a default manifest with reference values for the given platform.
func Default(platform platforms.Platform) (*Manifest, error) {
	embeddedRefValues, err := GetEmbeddedReferenceValues()
	if err != nil {
		return nil, fmt.Errorf("get embedded reference values: %w", err)
	}

	refValues, err := embeddedRefValues.ForPlatform(platform)
	if err != nil {
		return nil, fmt.Errorf("get reference values for platform %s: %w", platform, err)
	}

	return &Manifest{ReferenceValues: *refValues}, nil
}

// Validate checks the validity of all fields in the manifest.
func (m *Manifest) Validate() error {
	var errs []error
	for policyHash, policy := range m.Policies {
		if err := policy.Validate(policyHash); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("Policies[%q]", policyHash), err))
		}
	}

	var coordinatorCount uint
	for _, policy := range m.Policies {
		if policy.Role == RoleCoordinator {
			coordinatorCount++
		}
	}
	if coordinatorCount != 1 {
		return &CoordinatorCountError{Count: coordinatorCount}
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

// CoordinatorPolicyHash returns the hash of the coordinator policy.
func (m *Manifest) CoordinatorPolicyHash() (HexString, error) {
	for policyHash, policy := range m.Policies {
		if policy.Role == RoleCoordinator {
			return policyHash, nil
		}
	}
	return "", errors.New("no coordinator found in manifest")
}

// SNPValidateOpts returns validate options generators populated with the manifest's
// SNP reference values and trusted measurement for the given runtime.
func (m *Manifest) SNPValidateOpts(kdsGetter *certcache.CachedHTTPSGetter) ([]SNPValidatorOptions, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}

	var out []SNPValidatorOptions
	for _, refVal := range m.ReferenceValues.SNP {
		if len(refVal.TrustedMeasurement) == 0 {
			return nil, errors.New("trusted measurement cannot be empty")
		}

		trustedMeasurement, err := refVal.TrustedMeasurement.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert TrustedMeasurement from manifest to byte slices: %w", err)
		}

		verifyOpts := snpverify.DefaultOptions()
		// Setting the productLine explicitly, because of full dependence of trustedMeasurements and derivation of trustedRoots on productLine.
		verifyOpts.Product, err = kds.ParseProductLine(string(refVal.ProductName))
		if err != nil {
			return nil, fmt.Errorf("SNP reference values: %w", err)
		}
		verifyOpts.TrustedRoots, err = amdTrustedRootCerts(refVal.ProductName)
		if err != nil {
			return nil, fmt.Errorf("determine trusted roots: %w", err)
		}
		verifyOpts.CheckRevocations = true
		verifyOpts.Getter = kdsGetter.SNPGetter()

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

		validateOpts := &snpvalidate.Options{
			Measurement:  trustedMeasurement,
			PlatformInfo: &refVal.PlatformInfo,
			GuestPolicy:  refVal.GuestPolicy,
			VMPL:         new(int), // VMPL0
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

		var allowedChipIDs [][]byte
		for _, chipIDHex := range refVal.AllowedChipIDs {
			chipID, err := chipIDHex.Bytes()
			if err != nil {
				return nil, fmt.Errorf("failed to convert AllowedChipID from manifest to byte slices: %w", err)
			}
			allowedChipIDs = append(allowedChipIDs, chipID)
		}

		out = append(out, SNPValidatorOptions{
			VerifyOpts:     verifyOpts,
			ValidateOpts:   validateOpts,
			AllowedChipIDs: allowedChipIDs,
		})
	}

	return out, nil
}

// TDXValidateOpts returns validate options generators populated with the manifest's
// TDX reference values and trusted measurement for the given runtime.
func (m *Manifest) TDXValidateOpts(kdsGetter *certcache.CachedHTTPSGetter) ([]TDXValidatorOptions, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}

	var out []TDXValidatorOptions
	for _, refVal := range m.ReferenceValues.TDX {
		verifyOpts := tdxverify.DefaultOptions()

		var err error
		verifyOpts.TrustedRoots, err = tdxTrustedRootCerts()
		if err != nil {
			return nil, fmt.Errorf("getting trusted roots: %w", err)
		}

		verifyOpts.CheckRevocations = true
		verifyOpts.GetCollateral = true
		verifyOpts.Getter = kdsGetter

		mrTd, err := refVal.MrTd.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert MrTd from manifest to byte slices: %w", err)
		}
		mrSeam, err := refVal.MrSeam.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert MrSeam from manifest to byte slices: %w", err)
		}
		xfam, err := refVal.Xfam.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert Xfam from manifest to byte slices: %w", err)
		}
		var rtmrs [4][]byte
		for i, rtmr := range refVal.Rtmrs {
			bytes, err := rtmr.Bytes()
			if err != nil {
				return nil, fmt.Errorf("failed to convert Rtmr[%d] from manifest to byte slices: %w", i, err)
			}
			rtmrs[i] = bytes
		}

		// TdAttributes is configured by Kata/QEMU, with only the SEPT_VE_DISABLE bit (28) set.
		// See https://download.01.org/intel-sgx/sgx-dcap/1.24/linux/docs/Intel_TDX_DCAP_Quoting_Library_API.pdf, A.3.4.
		tdAttributes := binary.LittleEndian.AppendUint64(nil, 1<<28)

		validateOptions := &tdxvalidate.Options{
			HeaderOptions: tdxvalidate.HeaderOptions{
				QeVendorID: intelQeVendorID,
			},
			TdQuoteBodyOptions: tdxvalidate.TdQuoteBodyOptions{
				MrSeam:       mrSeam,
				TdAttributes: tdAttributes,
				Xfam:         xfam,
				MrTd:         mrTd,
				Rtmrs:        rtmrs[:],
			},
		}

		var allowedPIIDs [][]byte
		for _, piidHex := range refVal.AllowedPIIDs {
			piid, err := piidHex.Bytes()
			if err != nil {
				return nil, fmt.Errorf("failed to convert AllowedPIID from manifest to byte slices: %w", err)
			}
			allowedPIIDs = append(allowedPIIDs, piid)
		}

		out = append(out, TDXValidatorOptions{
			VerifyOpts:   verifyOpts,
			ValidateOpts: validateOptions,
			AllowedPIIDs: allowedPIIDs,
		})
	}

	return out, nil
}

// SNPValidatorOptions contains the verification and validation options to be used
// by an SNP Validator.
//
// TODO(msanft): add generic validation interface for other attestation types.
type SNPValidatorOptions struct {
	VerifyOpts     *snpverify.Options
	ValidateOpts   *snpvalidate.Options
	AllowedChipIDs [][]byte
}

// TDXValidatorOptions contains the verification and validation options to be used
// by a TDX Validator.
type TDXValidatorOptions struct {
	VerifyOpts   *tdxverify.Options
	ValidateOpts *tdxvalidate.Options
	AllowedPIIDs [][]byte
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

func validateHexString(value HexString, expectedNumBytes int) error {
	if len(value) != expectedNumBytes*2 {
		return fmt.Errorf("invalid length: %d (expected %d)", len(value), expectedNumBytes*2)
	}
	_, err := value.Bytes()
	return err
}

// ValidationError contains a JSON path and a list of errors.
// Nested validation errors are printed on newlines with the full path.
type ValidationError struct {
	path string
	errs []error
}

func newValidationError(path string, errs ...error) error {
	e := &ValidationError{
		path: path,
		errs: make([]error, 0, len(errs)),
	}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, flattenValidationError(err)...)
		}
	}
	if len(e.errs) == 0 {
		return nil
	}
	return e
}

func (e *ValidationError) Error() string {
	return e.formatError(e.path)
}

func (e *ValidationError) Unwrap() []error {
	return e.errs
}

func (e *ValidationError) formatError(path string) string {
	var sb strings.Builder
	for i, err := range e.errs {
		var ve *ValidationError
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

func flattenValidationError(err error) (errs []error) {
	if ve, ok := err.(*ValidationError); ok { //nolint:errorlint // check for exact type
		return []error{ve}
	}
	if wrapped, ok := err.(interface{ Unwrap() []error }); ok {
		for _, err := range wrapped.Unwrap() {
			errs = append(errs, flattenValidationError(err)...)
		}
		return errs
	}
	return []error{err}
}

// ExpectedMissingReferenceValueError should wrap ValidationErrors which are expected, depending on the host platform.
type ExpectedMissingReferenceValueError struct {
	Err error
}

func (e ExpectedMissingReferenceValueError) Error() string {
	return e.Err.Error()
}

// OnlyExpectedMissingReferenceValues checks if all nested ValidationErrors stem from expected missing reference values.
func (e *ValidationError) OnlyExpectedMissingReferenceValues() bool {
	for _, err := range e.errs {
		var ve *ValidationError
		if errors.As(err, &ve) && !ve.OnlyExpectedMissingReferenceValues() {
			return false
		}

		var emrve ExpectedMissingReferenceValueError
		if !errors.As(err, &emrve) {
			return false
		}
	}

	return true
}

// CoordinatorCountError occurs during manifest validation when either zero, or more than one coordinators have been defined.
type CoordinatorCountError struct{ Count uint }

func (e *CoordinatorCountError) Error() string {
	return fmt.Sprintf("expected exactly 1 policy with role 'coordinator', got %d", e.Count)
}
