package manifest

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
)

// Manifest is the Coordinator manifest and contains the reference values of the deployment.
type Manifest struct {
	// policyHash/HOSTDATA -> commonName
	Policies                map[HexString][]string
	ReferenceValues         ReferenceValues
	WorkloadOwnerKeyDigests []HexString
}

// ReferenceValues contains the workload independent reference values.
type ReferenceValues struct {
	SNP SNPReferenceValues
}

// SNPReferenceValues contains reference values for the SNP report.
type SNPReferenceValues struct {
	MinimumTCB         SNPTCB
	TrustedIDKeyHashes HexStrings // 0356215882a825279a85b300b0b742931d113bf7e32dde2e50ffde7ec743ca491ecdd7f336dc28a6e0b2bb57af7a44a3
}

// SNPTCB represents a set of SNP TCB values.
type SNPTCB struct {
	BootloaderVersion SVN
	TEEVersion        SVN
	SNPVersion        SVN
	MicrocodeVersion  SVN
}

// SVN is a SNP secure version number.
type SVN uint8

// UInt8 returns the uint8 value of the SVN.
func (s SVN) UInt8() uint8 {
	return uint8(s)
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
