package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"slices"

	"github.com/google/go-sev-guest/abi"
)

type nistCurve[Point nistPoint[Point]] struct {
	newPoint func() Point
	params   *elliptic.CurveParams
}

// nistPoint is a generic constraint for the nistec Point types.
type nistPoint[T any] interface {
	Bytes() []byte
	SetBytes([]byte) (T, error)
	Add(T, T) T
	Double(T) T
	ScalarMult(T, []byte) (T, error)
	ScalarBaseMult([]byte) (T, error)
}

// https://github.com/microsoft/igvm-tooling/blob/main/src/igvm/structure/igvmfileformat.py#L453
type IDBlock struct {
	LD        [0x30]byte // digest //current: 679eb9ca1e96782ef4510caebb198e62805467ce6756a66f4b91280416f3c648abb7eee9b102217c2531bdfe937a6083
	FAMILY_ID [0x10]byte // Not checked by us
	IMAGE_ID  [0x10]byte // Not checked by us
	VERSION   uint32     // const 1h
	GUEST_SNV uint32     // Default 2 in https://github.com/microsoft/igvm-tooling/blob/main/src/igvm/igvmfile.py#L178C18-L178C27
	POLICY    uint64     // done
}

type IDAuthentication struct {
	ID_KEY_ALGO   uint32      // const 1
	AUTH_KEY_ALGO uint32      // const 0
	RESERVED_0    [0x38]byte  // 0
	ID_BLOCK_SIG  [0x200]byte // consists of r,s
	ID_KEY        [0x404]byte // consists of Curve, Reserved, Qx, Qy
	RESERVED_1    [0x3c]byte  // 0
	ID_KEY_SIG    [0x200]byte // 0
	AUTH_KEY      [0x404]byte // 0
	RESERVED_2    [0x37c]byte // 0
}

// RecoverPublicKey attempts to recover the public key from a given ECDSA signature
// and message hash.
func RecoverPublicKey(curve elliptic.Curve, r, s, z *big.Int) ([]ecdsa.PublicKey, error) {
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

func main() {
	abc()
}

func abc() {
	launchDigest := must(hex.DecodeString("679eb9ca1e96782ef4510caebb198e62805467ce6756a66f4b91280416f3c648abb7eee9b102217c2531bdfe937a6083"))
	pol := abi.SnpPolicy{
		SMT:   true,
		Debug: false,
	}
	idBlock := IDBlock{
		LD:        [48]byte(launchDigest),
		VERSION:   0x1,
		GUEST_SNV: 0x2,
		POLICY:    abi.SnpPolicyToBytes(pol),
	}

	idBlockBytes := encodeIDBlock(idBlock)
	fmt.Printf("%x\n", idBlockBytes)

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

	fmt.Printf("signatureBytes: %x\n", signatureBytes) // 0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001

	validRBigEndian := validR
	slices.Reverse(validRBigEndian[:])
	validSBigEndian := validS
	slices.Reverse(validSBigEndian[:])
	// recover public key
	r := new(big.Int).SetBytes(validRBigEndian[:])
	s := new(big.Int).SetBytes(validSBigEndian[:])
	z := new(big.Int).SetBytes(hash[:])

	pubKeys, err := RecoverPublicKey(elliptic.P384(), r, s, z)
	if err != nil {
		log.Fatal(err)
	}

	pubKey := pubKeys[0]
	pubKeyBytes := encodeP384PublicKey(pubKey)
	fmt.Printf("pubKeyBytes: %x\n", pubKeyBytes)

	idAuth := IDAuthentication{
		ID_KEY_ALGO:   0x1,
		AUTH_KEY_ALGO: 0x1,
		ID_BLOCK_SIG:  signatureBytes,
		ID_KEY:        pubKeyBytes,
	}

	idAuthBytes := encodeIDAuthentication(idAuth)
	fmt.Printf("idAuthBytes: %x\n", idAuthBytes)

	// print base 64 of id block and id auth block
	fmt.Printf("idBlock: %s\n", base64.StdEncoding.EncodeToString(idBlockBytes[:]))
	fmt.Printf("idAuth: %s\n", base64.StdEncoding.EncodeToString(idAuthBytes[:]))
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

func encodeIDBlock(idBlock IDBlock) [60]byte {
	// Create a buffer to hold all bytes (total size = 48 + 16 + 16 + 4 + 4 + 8 = 96)
	buffer := make([]byte, 0x60)
	offset := 0

	copy(buffer[offset:offset+0x30], idBlock.LD[:])
	offset += 0x30

	copy(buffer[offset:offset+0x10], idBlock.FAMILY_ID[:])
	offset += 0x10

	copy(buffer[offset:offset+0x10], idBlock.IMAGE_ID[:])
	offset += 0x10

	binary.LittleEndian.PutUint32(buffer[offset:offset+4], idBlock.VERSION)
	offset += 4

	binary.LittleEndian.PutUint32(buffer[offset:offset+4], idBlock.GUEST_SNV)
	offset += 4

	binary.LittleEndian.PutUint64(buffer[offset:offset+8], idBlock.POLICY)

	return [60]byte(buffer)
}

func encodeIDAuthentication(idAuth IDAuthentication) [0x1000]byte {
	// Create a buffer to hold all bytes (total size = 0x1000)
	buffer := make([]byte, 0x1000)
	offset := 0

	binary.LittleEndian.PutUint32(buffer[offset:offset+4], idAuth.ID_KEY_ALGO)
	offset += 4

	binary.LittleEndian.PutUint32(buffer[offset:offset+4], idAuth.AUTH_KEY_ALGO)
	offset += 4

	copy(buffer[offset:offset+0x38], idAuth.RESERVED_0[:])
	offset += 0x38

	copy(buffer[offset:offset+0x200], idAuth.ID_BLOCK_SIG[:])
	offset += 0x200

	copy(buffer[offset:offset+0x404], idAuth.ID_KEY[:])
	offset += 0x404

	copy(buffer[offset:offset+0x3c], idAuth.RESERVED_1[:])
	offset += 0x3c

	copy(buffer[offset:offset+0x200], idAuth.ID_KEY_SIG[:])
	offset += 0x200

	copy(buffer[offset:offset+0x404], idAuth.AUTH_KEY[:])
	offset += 0x404

	copy(buffer[offset:offset+0x37c], idAuth.RESERVED_2[:])

	return [0x1000]byte(buffer)
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

/*
   def gen_id_block(self, digest: bytes) -> IGVM_VHS_SNP_ID_BLOCK:
       x = self.sign_key.verifying_key.pubkey.point.x()
       y = self.sign_key.verifying_key.pubkey.point.y()

       block = SNP_ID_BLOCK((c_uint8 * 48)(*digest),
                            self.config.family_id,
                            self.config.image_id,
                            self.config.version,
                            self.config.guest_svn,
                            self.config.policy)
       signer = self.sign_key.sign_deterministic if self.sign_deterministic else self.sign_key.sign
       r, s = signer(
           bytearray(block), sigencode=lambda r, s, o: (r, s))

       signature = IGVM_VHS_SNP_ID_BLOCK_SIGNATURE(
           (c_uint8 * 72)(*list(r.to_bytes(48, 'little'))),
           (c_uint8 * 72)(*list(s.to_bytes(48, 'little'))))
       public_key = IGVM_VHS_SNP_ID_BLOCK_PUBLIC_KEY(
           2, 0, (c_uint8 * 72)(*list(x.to_bytes(48, 'little'))),
           (c_uint8 * 72)(*list(y.to_bytes(48, 'little'))))
       id_block = IGVM_VHS_SNP_ID_BLOCK(
           1, 0, (c_uint8 * 3)(),
           block.Ld, block.FamilyId, block.ImageId, block.Version, block.
           GuestSvn, 1, 0, signature, public_key)
       return id_block
*/
