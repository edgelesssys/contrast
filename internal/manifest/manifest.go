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
	snpmeasure "github.com/edgelesssys/contrast/internal/snp"
	"github.com/google/go-sev-guest/kds"
	snpvalidate "github.com/google/go-sev-guest/validate"
	snpverify "github.com/google/go-sev-guest/verify"
	"github.com/google/go-tdx-guest/pcs"
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
	// WorkloadOwnerPubKeys is a list of ECDSA public keys in PKIX DER format, hex-encoded.
	WorkloadOwnerPubKeys []HexString
	// SeedshareOwnerPubKeys is a list of RSA public keys in PKCS1 DER format, hex-encoded.
	SeedshareOwnerPubKeys []HexString
}

// Default returns a default manifest with reference values for the given platform.
func Default(platforms []platforms.Platform) (*Manifest, error) {
	embeddedRefValues, err := GetEmbeddedReferenceValues()
	if err != nil {
		return nil, fmt.Errorf("get embedded reference values: %w", err)
	}

	var merged ReferenceValues

	for _, platform := range platforms {
		refValues, err := embeddedRefValues.ForPlatform(platform)
		if err != nil {
			return nil, fmt.Errorf(
				"get reference values for platform %s: %w",
				platform,
				err,
			)
		}

		// Add the platform as a marker.
		// Used later for patching-in reference values.
		for i := range refValues.SNP {
			refValues.SNP[i].Platform = platform.String()
		}
		for i := range refValues.TDX {
			refValues.TDX[i].Platform = platform.String()
		}

		merged.SNP = append(merged.SNP, refValues.SNP...)
		merged.TDX = append(merged.TDX, refValues.TDX...)
	}
	return &Manifest{ReferenceValues: merged}, nil
}

// Validate checks the validity of all fields in the manifest.
func (m *Manifest) Validate() error {
	var errs []error
	for policyHash, policy := range m.Policies {
		if err := policy.Validate(policyHash); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("Policies[%q]", policyHash), err))
		}
	}

	// Implicitly checks Coordinator count.
	if _, err := m.CoordinatorPolicyHashes(); err != nil {
		return err
	}

	if err := m.ReferenceValues.Validate(); err != nil {
		errs = append(errs, newValidationError("ReferenceValues", err))
	}

	if m.HasInsecurePlatforms() && m.HasSecurePlatforms() {
		errs = append(errs, newValidationError("ReferenceValues", errors.New("manifest must not mix secure and insecure platforms")))
	}

	for i, key := range m.WorkloadOwnerPubKeys {
		if _, err := ParseWorkloadOwnerPublicKey(key); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("WorkloadOwnerPubKeys[%d]", i), err))
		}
	}

	for i, key := range m.SeedshareOwnerPubKeys {
		if _, err := ParseSeedShareOwnerKey(key); err != nil {
			errs = append(errs, newValidationError(fmt.Sprintf("SeedshareOwnerPubKeys[%d]", i), err))
		}
	}
	return errors.Join(errs...)
}

// CoordinatorPolicyHashes returns policy hashes for all workloads with role Coordinator.
func (m *Manifest) CoordinatorPolicyHashes() ([]HexString, error) {
	var all []HexString
	for policyHash, policy := range m.Policies {
		if policy.Role == RoleCoordinator {
			all = append(all, policyHash)
		}
	}
	if len(all) == 0 {
		return nil, ErrMissingCoordinator
	}
	return all, nil
}

// HasInsecurePlatforms returns true if the manifest contains a reference value for a recognized
// insecure platform.
func (m *Manifest) HasInsecurePlatforms() bool {
	return m.anyReferenceValue(func(p platforms.Platform, ok bool) bool {
		return ok && platforms.IsInsecure(p)
	})
}

// HasSecurePlatforms returns true if the manifest contains a reference value that does not target a
// recognized insecure platform. Unknown or unset platforms are treated as secure (fail-closed), so
// a malformed secure reference value mixed with an insecure one is still detected as a mix.
func (m *Manifest) HasSecurePlatforms() bool {
	return m.anyReferenceValue(func(p platforms.Platform, ok bool) bool {
		return !ok || !platforms.IsInsecure(p)
	})
}

// anyReferenceValue reports whether any SNP or TDX reference value's platform satisfies pred. pred
// receives the parsed platform and whether parsing succeeded.
func (m *Manifest) anyReferenceValue(pred func(p platforms.Platform, ok bool) bool) bool {
	for _, v := range m.ReferenceValues.SNP {
		p, err := platforms.FromString(v.Platform)
		if pred(p, err == nil) {
			return true
		}
	}
	for _, v := range m.ReferenceValues.TDX {
		p, err := platforms.FromString(v.Platform)
		if pred(p, err == nil) {
			return true
		}
	}
	return false
}

// SNPValidateOpts returns validate options generators populated with the manifest's
// SNP reference values and trusted measurement for the given runtime.
func (m *Manifest) SNPValidateOpts(kdsGetter *certcache.CachedHTTPSGetter) ([]SNPValidatorOptions, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}

	var out []SNPValidatorOptions
	for _, refVal := range m.ReferenceValues.SNP {
		if p, err := platforms.FromString(refVal.Platform); err == nil && platforms.IsInsecure(p) {
			continue
		}
		if len(refVal.TrustedMeasurement) == 0 {
			return nil, errors.New("trusted measurement cannot be empty")
		}

		seed, err := refVal.TrustedMeasurement.Bytes()
		if err != nil {
			return nil, fmt.Errorf("failed to decode TrustedMeasurement: %w", err)
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

		var allowedChipIDs [][]byte
		for _, chipIDHex := range refVal.AllowedChipIDs {
			chipID, err := chipIDHex.Bytes()
			if err != nil {
				return nil, fmt.Errorf("failed to convert AllowedChipID from manifest to byte slices: %w", err)
			}
			allowedChipIDs = append(allowedChipIDs, chipID)
		}

		validateOpts := &snpvalidate.Options{
			// Measurement holds the 1-vCPU seed. When APEIP is set the IterativeValidator
			// reads it as the seed and overrides it per vCPU count; when APEIP is absent it
			// is treated as the exact expected measurement (backwards compatibility).
			Measurement:  seed,
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
			PermitProvisionalFirmware:      true,
			RequireIDBlock:                 true,
			MinimumLaunchMitigationVector:  refVal.MinimumMitigationVector,
			MinimumCurrentMitigationVector: refVal.MinimumMitigationVector,
		}

		vcpuSig, err := snpmeasure.CPUSigForProduct(string(refVal.ProductName))
		if err != nil {
			return nil, fmt.Errorf("looking up CPU signature for product %q: %w", refVal.ProductName, err)
		}

		opt := SNPValidatorOptions{
			VerifyOpts:     verifyOpts,
			ValidateOpts:   validateOpts,
			VCPUSig:        vcpuSig,
			AllowedChipIDs: allowedChipIDs,
		}

		if refVal.APEIP != "" {
			// APEIP present: IterativeValidator will expand the seed to vCPU counts 1–220
			// at verify time and compute the IDKey hash per iteration.
			apEIPBytes, err := refVal.APEIP.Bytes()
			if err != nil {
				return nil, fmt.Errorf("failed to decode APEIP: %w", err)
			}
			opt.APEIP = apEIPBytes
		} else {
			// Backwards compatibility: treat TrustedMeasurement as an exact match and
			// compute the IDKey hash for it now.
			_, authBlk, err := idblock.IDBlocksFromLaunchDigest([48]byte(seed), refVal.GuestPolicy)
			if err != nil {
				return nil, fmt.Errorf("failed to generate ID blocks: %w", err)
			}
			idKeyBytes, err := authBlk.IDKey.MarshalBinary()
			if err != nil {
				return nil, fmt.Errorf("failed to marshal IDKey: %w", err)
			}
			idKeyHash := sha512.Sum384(idKeyBytes)
			validateOpts.TrustedIDKeyHashes = [][]byte{idKeyHash[:]}
		}

		out = append(out, opt)
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
		if p, err := platforms.FromString(refVal.Platform); err == nil && platforms.IsInsecure(p) {
			continue
		}
		verifyOpts := tdxverify.DefaultOptions()

		var err error
		verifyOpts.TrustedRoots, err = tdxTrustedRootCerts()
		if err != nil {
			return nil, fmt.Errorf("getting trusted roots: %w", err)
		}

		verifyOpts.CheckRevocations = true
		verifyOpts.GetCollateral = true
		verifyOpts.Getter = kdsGetter

		if refVal.MinTCBEvaluationDataNumber > 0 {
			verifyOpts.EvaluationDataNumber = refVal.MinTCBEvaluationDataNumber
		}

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

		pckOptions := tdxvalidate.PCKOptions{}
		if refVal.MemoryIntegrity {
			pckOptions.SgxType = toPtr(pcs.SGXTypeScalableWithIntegrity)
		}
		if refVal.SMTDisabled {
			pckOptions.SMTEnabled = toPtr(false)
		}
		if refVal.StaticPlatform {
			pckOptions.DynamicPlatform = toPtr(false)
		}

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
			PCKOptions: pckOptions,
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
	VerifyOpts   *snpverify.Options
	ValidateOpts *snpvalidate.Options
	// APEIP, when set (4 bytes), signals that ValidateOpts.Measurement is the 1-vCPU seed
	// and that the caller should use an IterativeValidator to try vCPU counts 1–220.
	// When nil, ValidateOpts.Measurement is an exact expected measurement.
	APEIP []byte
	// VCPUSig is the CPUID signature for the vCPU type (e.g. EPYC-Milan, EPYC-Genoa).
	// Required when APEIP is set so that AP VMSA pages are built with the correct rdx value.
	VCPUSig        uint32
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
	// RoleNodeInstaller is the node installer DaemonSet.
	RoleNodeInstaller Role = "contrast-node-installer"
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
	return fmt.Sprintf("Please fill in all reference values for %s", e.Err.Error())
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

// ErrMissingCoordinator is returned when the manifest does not contain at least one policy for a Coordinator.
var ErrMissingCoordinator = errors.New("expected at least 1 policy with role 'coordinator'")

func toPtr[T any](v T) *T {
	return &v
}
