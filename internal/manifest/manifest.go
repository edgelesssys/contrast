package manifest

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
)

type Manifest struct {
	// policyHash/HOSTDATA -> commonName
	Policies        map[HexString]string
	ReferenceValues ReferenceValues
}

type ReferenceValues struct {
	SNP SNPReferenceValues
}

type SNPReferenceValues struct {
	MinimumTCB         SNPTCB
	TrustedIDKeyHashes HexStrings // 0356215882a825279a85b300b0b742931d113bf7e32dde2e50ffde7ec743ca491ecdd7f336dc28a6e0b2bb57af7a44a3
}

type SNPTCB struct {
	BootloaderVersion SVN
	TEEVersion        SVN
	SNPVersion        SVN
	MicrocodeVersion  SVN
}

type SVN uint8

func (s SVN) UInt8() uint8 {
	return uint8(s)
}

func (s SVN) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(s))), nil
}

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

type HexString string

func NewHexString(b []byte) HexString {
	return HexString(hex.EncodeToString(b))
}

func (h HexString) String() string {
	return string(h)
}

func (h HexString) Bytes() ([]byte, error) {
	return hex.DecodeString(string(h))
}

type HexStrings []HexString

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

type Policy []byte

func NewPolicyFromAnnotation(annotation []byte) (Policy, error) {
	return base64.StdEncoding.DecodeString(string(annotation))
}

func (p Policy) Bytes() []byte {
	return []byte(p)
}

func (p Policy) Hash() HexString {
	hashBytes := sha256.Sum256(p)
	return NewHexString(hashBytes[:])
}
