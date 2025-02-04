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

	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/google/go-sev-guest/abi"
)

// https://github.com/microsoft/igvm-tooling/blob/main/src/igvm/structure/igvmfileformat.py#L453
// https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/specifications/56860.pdf
// All number fields are little-endian encoded.
type idBlock struct {
	// The expected launch digest of the guest.
	ld [0x30]byte
	// FAMILY_ID. Family ID of the guest, provided by the guest owner and uninterpreted by the firmware. Not checked by us
	familyID [0x10]byte
	// IMAGE_ID. Image ID of the guest, provided by the guest owner and uninterpreted by the firmware. Not checked by us
	imageID [0x10]byte
	// VERSION. Version of the ID block format. Must be 1h for this version of the ABI.
	version uint32
	// GUEST_SVN. Default 2 in https://github.com/microsoft/igvm-tooling/blob/main/src/igvm/igvmfile.py#L178C18-L178C27
	guestSVN uint32
	// POLICY. The policy of the guest.
	policy uint64
}

type idAuthentication struct {
	// ID_KEY_ALGO. The algorithm of the ID Key. 0x1 for ECDSA P-384 with SHA-384.
	idKeyAlgo uint32
	// AUTH_KEY_ALGO. The algorithm of the Author Key. 0x1 for ECDSA P-384 with SHA-384.
	authKeyAlgo uint32
	// RESERVED. Must be 0.
	reserved0 [0x38]byte
	// ID_BLOCK_SIG. The signature of all bytes of the ID block. Consists of r,s.
	idBlockSig [0x200]byte
	// ID_KEY. The public component of the ID key. Consists of Curve, Reserved, Qx, Qy.
	idKey [0x404]byte
	// RESERVED. Must be 0.
	reserved1 [0x3c]byte
	// ID_KEY_SIG. The signature of the ID_KEY. Consists of r,s.
	idKeySig [0x200]byte
	// AUTH_KEY. The public component of the Author key. Consists of Curve, Reserved, Qx, Qy.
	// Ignored if AUTHOR_KEY_EN is 0.
	authKey [0x404]byte
	// RESERVED. Must be 0.
	reserved2 [0x37c]byte // 0
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
func IDBlocksFromLaunchDigest(launchDigest [48]byte) ([0x60]byte, [0x1000]byte, error) {
	idBlock := idBlock{
		ld:       launchDigest,
		version:  0x1,
		guestSVN: 0x2,
		policy:   abi.SnpPolicyToBytes(constants.SNPPolicy),
	}

	idBlockBytes := encodeIDBlock(idBlock)
	hash := sha512.Sum384(idBlockBytes[:])

	// https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/specifications/56860.pdf
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
		return [0x60]byte{}, [0x1000]byte{}, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Always choose the same recovered public key
	pubKey := pubKeys[0]
	pubKeyBytes := encodeP384PublicKey(pubKey)

	idAuth := idAuthentication{
		idKeyAlgo:   0x1,
		authKeyAlgo: 0x1,
		idBlockSig:  signatureBytes,
		idKey:       pubKeyBytes,
	}

	idAuthBytes := encodeIDAuthentication(idAuth)

	return idBlockBytes, idAuthBytes, nil
}

func encodeP384PublicKey(pubKey ecdsa.PublicKey) [0x404]byte {
	buffer := make([]byte, 0x404)
	offset := 0

	// Curve
	// 2h indicates P-384.
	binary.LittleEndian.PutUint32(buffer[offset:offset+4], 2)
	offset += 4

	xLittleEndian := pubKey.X.Bytes()
	slices.Reverse(xLittleEndian)
	copy(buffer[offset:offset+0x48], xLittleEndian)
	offset += 0x48

	yLittleEndian := pubKey.Y.Bytes()
	slices.Reverse(yLittleEndian)
	copy(buffer[offset:offset+0x48], yLittleEndian)

	return [0x404]byte(buffer)
}

func encodeIDBlock(idBlock idBlock) [0x60]byte {
	// Create a buffer to hold all bytes (total size = 48 + 16 + 16 + 4 + 4 + 8 = 96)
	buffer := make([]byte, 0x60)
	offset := 0

	copy(buffer[offset:offset+0x30], idBlock.ld[:])
	offset += 0x30

	copy(buffer[offset:offset+0x10], idBlock.familyID[:])
	offset += 0x10

	copy(buffer[offset:offset+0x10], idBlock.imageID[:])
	offset += 0x10

	binary.LittleEndian.PutUint32(buffer[offset:offset+4], idBlock.version)
	offset += 4

	binary.LittleEndian.PutUint32(buffer[offset:offset+4], idBlock.guestSVN)
	offset += 4

	binary.LittleEndian.PutUint64(buffer[offset:offset+8], idBlock.policy)

	return [0x60]byte(buffer)
}

func encodeIDAuthentication(idAuth idAuthentication) [0x1000]byte {
	// Create a buffer to hold all bytes (total size = 0x1000)
	buffer := make([]byte, 0x1000)
	offset := 0

	binary.LittleEndian.PutUint32(buffer[offset:offset+4], idAuth.idKeyAlgo)
	offset += 4

	binary.LittleEndian.PutUint32(buffer[offset:offset+4], idAuth.authKeyAlgo)
	offset += 4

	copy(buffer[offset:offset+0x38], idAuth.reserved0[:])
	offset += 0x38

	copy(buffer[offset:offset+0x200], idAuth.idBlockSig[:])
	offset += 0x200

	copy(buffer[offset:offset+0x404], idAuth.idKey[:])
	offset += 0x404

	copy(buffer[offset:offset+0x3c], idAuth.reserved1[:])
	offset += 0x3c

	copy(buffer[offset:offset+0x200], idAuth.idKeySig[:])
	offset += 0x200

	copy(buffer[offset:offset+0x404], idAuth.authKey[:])
	offset += 0x404

	copy(buffer[offset:offset+0x37c], idAuth.reserved2[:])

	return [0x1000]byte(buffer)
}
