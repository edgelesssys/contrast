// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package idblock

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"math/big"
	"slices"

	"github.com/google/go-sev-guest/abi"
)

// IDBlock is the ID block.
// https://github.com/microsoft/igvm-tooling/blob/main/src/igvm/structure/igvmfileformat.py#L453
// https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/specifications/56860.pdf (revision 1.57)
// All number fields are little-endian encoded.
type IDBlock struct {
	// The expected launch digest of the guest.
	LD [0x30]byte
	// FAMILY_ID. Family ID of the guest, provided by the guest owner and uninterpreted by the firmware. Not checked by us
	FamilyID [0x10]byte
	// IMAGE_ID. Image ID of the guest, provided by the guest owner and uninterpreted by the firmware. Not checked by us
	ImageID [0x10]byte
	// VERSION. Version of the ID block format. Must be 1h for this version of the ABI.
	Version uint32
	// GUEST_SVN. Default 2 in https://github.com/microsoft/igvm-tooling/blob/main/src/igvm/igvmfile.py#L178C18-L178C27
	GuestSVN uint32
	// POLICY. The policy of the guest.
	Policy uint64
}

// MarshalBinary marshals the ID block to binary.
func (idBlock *IDBlock) MarshalBinary() ([]byte, error) {
	data := make([]byte, 0x60)
	copy(data[0x00:0x30], idBlock.LD[:])
	copy(data[0x30:0x40], idBlock.FamilyID[:])
	copy(data[0x40:0x50], idBlock.ImageID[:])
	binary.LittleEndian.PutUint32(data[0x50:0x54], idBlock.Version)
	binary.LittleEndian.PutUint32(data[0x54:0x58], idBlock.GuestSVN)
	binary.LittleEndian.PutUint64(data[0x58:0x60], idBlock.Policy)
	return data, nil
}

// UnmarshalBinary unmarshals the ID block from binary.
func (idBlock *IDBlock) UnmarshalBinary(data []byte) error {
	if len(data) != 0x60 {
		return fmt.Errorf("invalid ID block size: %d", len(data))
	}
	copy(idBlock.LD[:], data[0x00:0x30])
	copy(idBlock.FamilyID[:], data[0x30:0x40])
	copy(idBlock.ImageID[:], data[0x40:0x50])
	idBlock.Version = binary.LittleEndian.Uint32(data[0x50:0x54])
	idBlock.GuestSVN = binary.LittleEndian.Uint32(data[0x54:0x58])
	idBlock.Policy = binary.LittleEndian.Uint64(data[0x58:0x60])
	return nil
}

// IDAuthentication is the ID authentication block.
// https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/specifications/56860.pdf (revision 1.57)
type IDAuthentication struct {
	// ID_KEY_ALGO. The algorithm of the ID Key. 0x1 for ECDSA P-384 with SHA-384.
	IDKeyAlgo uint32
	// AUTH_KEY_ALGO. The algorithm of the Author Key. 0x1 for ECDSA P-384 with SHA-384.
	AuthKeyAlgo uint32
	// RESERVED. Must be 0.
	Reserved0 [0x38]byte
	// ID_BLOCK_SIG. The signature of all bytes of the ID block. Consists of r,s.
	IDBlockSig Ecdsa384Sha384Signature
	// ID_KEY. The public component of the ID key. Consists of Curve, Reserved, Qx, Qy.
	IDKey Ecdsa384PublicKey
	// RESERVED. Must be 0.
	Reserved1 [0x3c]byte
	// ID_KEY_SIG. The signature of the ID_KEY. Consists of r,s.
	IDKeySig Ecdsa384Sha384Signature
	// AUTH_KEY. The public component of the Author key. Consists of Curve, Reserved, Qx, Qy.
	// Ignored if AUTHOR_KEY_EN is 0.
	AuthKey Ecdsa384PublicKey
	// RESERVED. Must be 0.
	Reserved2 [0x37c]byte // 0
}

// MarshalBinary marshals the ID authentication block to binary.
func (idAuth *IDAuthentication) MarshalBinary() ([]byte, error) {
	data := make([]byte, 0x1000)
	binary.LittleEndian.PutUint32(data[0x00:0x04], idAuth.IDKeyAlgo)
	binary.LittleEndian.PutUint32(data[0x04:0x08], idAuth.AuthKeyAlgo)
	copy(data[0x08:0x40], idAuth.Reserved0[:])

	idBlockBytes, err := idAuth.IDBlockSig.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ID block signature: %w", err)
	}
	copy(data[0x40:0x240], idBlockBytes)

	idKeyBytes, err := idAuth.IDKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ID key: %w", err)
	}
	copy(data[0x240:0x644], idKeyBytes)

	copy(data[0x644:0x680], idAuth.Reserved1[:])

	idKeySigBytes, err := idAuth.IDKeySig.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ID key signature: %w", err)
	}
	copy(data[0x680:0x880], idKeySigBytes)

	authKeyBytes, err := idAuth.AuthKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal author key: %w", err)
	}
	copy(data[0x880:0xC84], authKeyBytes)

	copy(data[0xC84:0x1000], idAuth.Reserved2[:])
	return data, nil
}

// UnmarshalBinary unmarshals the ID authentication block from binary.
func (idAuth *IDAuthentication) UnmarshalBinary(data []byte) error {
	if len(data) != 0x1000 {
		return fmt.Errorf("invalid ID authentication block size: %d", len(data))
	}
	idAuth.IDKeyAlgo = binary.LittleEndian.Uint32(data[0x00:0x04])
	idAuth.AuthKeyAlgo = binary.LittleEndian.Uint32(data[0x04:0x08])
	copy(idAuth.Reserved0[:], data[0x08:0x40])

	if err := idAuth.IDBlockSig.UnmarshalBinary(data[0x40:0x240]); err != nil {
		return fmt.Errorf("failed to unmarshal ID block signature: %w", err)
	}

	if err := idAuth.IDKey.UnmarshalBinary(data[0x240:0x644]); err != nil {
		return fmt.Errorf("failed to unmarshal ID key: %w", err)
	}

	copy(idAuth.Reserved1[:], data[0x644:0x680])

	if err := idAuth.IDKeySig.UnmarshalBinary(data[0x680:0x880]); err != nil {
		return fmt.Errorf("failed to unmarshal ID key signature: %w", err)
	}

	if err := idAuth.AuthKey.UnmarshalBinary(data[0x880:0xC84]); err != nil {
		return fmt.Errorf("failed to unmarshal author key: %w", err)
	}

	copy(idAuth.Reserved2[:], data[0xC84:0x1000])
	return nil
}

// Ecdsa384Sha384Signature is the signature of an ECDSA P-384 with SHA-384 signature.
// https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/specifications/56860.pdf (revision 1.57)
type Ecdsa384Sha384Signature struct {
	R         [0x48]byte
	S         [0x48]byte
	Reserved1 [0x170]byte
}

// MarshalBinary marshals the ECDSA P-384 with SHA-384 signature to binary.
func (sig *Ecdsa384Sha384Signature) MarshalBinary() ([]byte, error) {
	data := make([]byte, 0x200)
	if sig == nil {
		return data, nil
	}
	copy(data[0x00:0x48], sig.R[:])
	copy(data[0x48:0x90], sig.S[:])
	copy(data[0x90:0x200], sig.Reserved1[:])
	return data, nil
}

// UnmarshalBinary unmarshals the ECDSA P-384 with SHA-384 signature from binary.
func (sig *Ecdsa384Sha384Signature) UnmarshalBinary(data []byte) error {
	if len(data) != 0x200 {
		return fmt.Errorf("invalid ECDSA P-384 with SHA-384 signature size: %d", len(data))
	}
	copy(sig.R[:], data[0x00:0x48])
	copy(sig.S[:], data[0x48:0x90])
	copy(sig.Reserved1[:], data[0x90:0x200])
	return nil
}

// Ecdsa384PublicKey is the public key of an ECDSA P-384 key.
// https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/specifications/56860.pdf (revision 1.57)
type Ecdsa384PublicKey struct {
	CurveID   uint32
	Qx        [0x48]byte
	Qy        [0x48]byte
	Reserved1 [0x370]byte
}

// MarshalBinary marshals the ECDSA P-384 public key to binary.
func (pubKey *Ecdsa384PublicKey) MarshalBinary() ([]byte, error) {
	data := make([]byte, 0x404)
	if pubKey == nil {
		return data, nil
	}
	binary.LittleEndian.PutUint32(data[0x00:0x04], pubKey.CurveID)
	copy(data[0x04:0x4C], pubKey.Qx[:])
	copy(data[0x4C:0x94], pubKey.Qy[:])
	copy(data[0x94:0x404], pubKey.Reserved1[:])
	return data, nil
}

// UnmarshalBinary unmarshals the ECDSA P-384 public key from binary.
func (pubKey *Ecdsa384PublicKey) UnmarshalBinary(data []byte) error {
	if len(data) != 0x404 {
		return fmt.Errorf("invalid ECDSA P-384 public key size: %d", len(data))
	}
	pubKey.CurveID = binary.LittleEndian.Uint32(data[0x00:0x04])
	copy(pubKey.Qx[:], data[0x04:0x4C])
	copy(pubKey.Qy[:], data[0x4C:0x94])
	copy(pubKey.Reserved1[:], data[0x94:0x404])
	return nil
}

// recoverPublicKey attempts to recover the public key from a given ECDSA signature
// and message hash.
// This algorithm is described in the SEC 1: Elliptic Curve Cryptography standard
// section 4.1.6, "Public Key Recovery Operation"
// https://www.secg.org/sec1-v2.pdf.
func recoverPublicKey(curve elliptic.Curve, r, s, z *big.Int) ([]ecdsa.PublicKey, error) {
	// taken from: https://cs.opensource.google/go/go/+/master:src/crypto/elliptic/params.go;l=36;drc=5c2b5e02c422ab3936645e2faa4489bf32fa8a57
	polynomial := func(curve *elliptic.CurveParams, x *big.Int) *big.Int {
		x3 := new(big.Int).Mul(x, x)
		x3.Mul(x3, x)

		threeX := new(big.Int).Lsh(x, 1)
		threeX.Add(threeX, x)

		x3.Sub(x3, threeX)
		x3.Add(x3, curve.B)
		x3.Mod(x3, curve.P)

		return x3
	}

	var publicKeys []ecdsa.PublicKey

	params := curve.Params()
	order := params.N

	// Compute r^-1 mod n (the modular multiplicative inverse of r)
	rInv := new(big.Int).ModInverse(r, order)
	if rInv == nil {
		return nil, fmt.Errorf("unable to compute modular inverse")
	}

	pointX := new(big.Int).Mod(r, params.P)

	ySquared2 := polynomial(params, pointX)

	pointY := new(big.Int).ModSqrt(ySquared2, params.P)
	if pointY == nil {
		return nil, fmt.Errorf("no square root")
	}

	// Try both y-coordinates for recovery
	for _, ySign := range []int64{1, -1} {
		if ySign == -1 {
			pointY = new(big.Int).Sub(params.P, pointY)
		}

		// Verify if the point is on the curve
		if !curve.IsOnCurve(pointX, pointY) {
			continue
		}

		u1 := new(big.Int).Mul(z, rInv)
		u1.Mod(u1, order)
		u1.Neg(u1)
		u1.Mod(u1, order)

		// Compute u1 and u2
		u2 := new(big.Int).Mul(s, rInv)
		u2.Mod(u2, order)

		// Compute the candidate public key as u1*G + u2*Q
		x1, y1 := curve.ScalarBaseMult(u1.Bytes())

		x2, y2 := curve.ScalarMult(pointX, pointY, u2.Bytes())

		finalX, finalY := curve.Add(x1, y1, x2, y2)

		publicKeys = append(publicKeys, ecdsa.PublicKey{
			Curve: curve,
			X:     finalX,
			Y:     finalY,
		})

	}

	// sort public keys
	slices.SortStableFunc(publicKeys, func(a, b ecdsa.PublicKey) int {
		cmpX := a.X.Cmp(b.X)
		if cmpX != 0 {
			return cmpX
		}
		return a.Y.Cmp(b.Y)
	})

	return publicKeys, nil
}

// IDBlocksFromLaunchDigest generates the ID block and ID authentication block from a given launch digest.
// The ID auth block contains a constant signature (2,1) which signs the ID block.
// The public key in the ID block is recovered from the signature.
func IDBlocksFromLaunchDigest(launchDigest [48]byte, guestPolicy abi.SnpPolicy) (*IDBlock, *IDAuthentication, error) {
	idBlk := &IDBlock{
		LD:       launchDigest,
		Version:  0x1,
		GuestSVN: 0x2,
		FamilyID: [0x10]byte{0x1},
		ImageID:  [0x10]byte{0x2},
		Policy:   abi.SnpPolicyToBytes(guestPolicy),
	}

	idBlockBytes, err := idBlk.MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal ID block: %w", err)
	}
	hash := sha512.Sum384(idBlockBytes[:])

	// https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/specifications/56860.pdf (revision 1.57)
	// Chapter 10 page 145 Table 136
	// both R and S are little-endian encoded
	// A valid R must be an x-coordinate on the curve
	validR := [0x48]byte{0x2}
	// Any S is valid
	validS := [0x48]byte{0x1}
	signature := append(validR[:], validS[:]...)
	signatureBytes := [0x200]byte{}
	copy(signatureBytes[:], signature)

	validRBigEndian := validR
	slices.Reverse(validRBigEndian[:])
	validSBigEndian := validS
	slices.Reverse(validSBigEndian[:])

	// recover public key
	r := new(big.Int).SetBytes(validRBigEndian[:])
	s := new(big.Int).SetBytes(validSBigEndian[:])
	z := new(big.Int).SetBytes(hash[:])

	pubKeys, err := recoverPublicKey(elliptic.P384(), r, s, z)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Always choose the same recovered public key
	pubKey := pubKeys[0]

	xLittleEndian := pubKey.X.Bytes()
	slices.Reverse(xLittleEndian)

	yLittleEndian := pubKey.Y.Bytes()
	slices.Reverse(yLittleEndian)

	idAuth := &IDAuthentication{
		IDKeyAlgo:   0x1,
		AuthKeyAlgo: 0x0,
		IDBlockSig: Ecdsa384Sha384Signature{
			R: validR,
			S: validS,
		},
		IDKey: Ecdsa384PublicKey{
			// Curve
			// 2h indicates P-384.
			CurveID: 2,
		},
	}
	// The {x,y}LittleEndian values are only 0x30 bytes ling, so according to the spec
	// it needs to be padded on the right with 0x00 bytes.
	copy(idAuth.IDKey.Qx[:], xLittleEndian)
	copy(idAuth.IDKey.Qy[:], yLittleEndian)

	return idBlk, idAuth, nil
}
