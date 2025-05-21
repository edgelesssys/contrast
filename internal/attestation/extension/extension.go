// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package extension

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"math/big"

	"golang.org/x/exp/constraints"
)

// NewBigIntExtension returns a new extension containing an unsigned integer value.
func NewBigIntExtension[T constraints.Unsigned](oid asn1.ObjectIdentifier, value T) Extension {
	bigInt := &big.Int{}
	bigInt.SetUint64(uint64(value))
	return bigIntExtension{OID: oid, Val: bigInt}
}

type bigIntExtension struct {
	OID asn1.ObjectIdentifier
	Val *big.Int
}

func (b bigIntExtension) toExtension() (pkix.Extension, error) {
	bytes, err := asn1.Marshal(b.Val)
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("marshaling big int: %w", err)
	}
	return pkix.Extension{Id: b.OID, Value: bytes}, nil
}

// NewBytesExtension returns a new extension containing bytes.
func NewBytesExtension(oid asn1.ObjectIdentifier, val []byte) Extension {
	return bytesExtension{OID: oid, Val: val}
}

type bytesExtension struct {
	OID asn1.ObjectIdentifier
	Val []byte
}

func (b bytesExtension) toExtension() (pkix.Extension, error) {
	bytes, err := asn1.Marshal(b.Val)
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("marshaling bytes: %w", err)
	}
	return pkix.Extension{Id: b.OID, Value: bytes}, nil
}

// NewBoolExtension returns a new extension containing a boolean value.
func NewBoolExtension(oid asn1.ObjectIdentifier, val bool) Extension {
	return boolExtension{OID: oid, Val: val}
}

type boolExtension struct {
	OID asn1.ObjectIdentifier
	Val bool
}

func (b boolExtension) toExtension() (pkix.Extension, error) {
	bytes, err := asn1.Marshal(b.Val)
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("marshaling bool: %w", err)
	}
	return pkix.Extension{Id: b.OID, Value: bytes}, nil
}

// Extension is a yet-to-be-marshalled pkix extension.
type Extension interface {
	toExtension() (pkix.Extension, error)
}

// ConvertExtensions converts the extensions into pkix extensions.
func ConvertExtensions(extensions []Extension) ([]pkix.Extension, error) {
	var exts []pkix.Extension
	for _, extension := range extensions {
		ext, err := extension.toExtension()
		if err != nil {
			return nil, fmt.Errorf("converting extension to pkix: %w", err)
		}
		exts = append(exts, ext)
	}
	return exts, nil
}

// ConvertExtension converts a single extension into a pkix extension.
func ConvertExtension(extension Extension) (pkix.Extension, error) {
	return extension.toExtension()
}
