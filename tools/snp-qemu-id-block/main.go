package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/google/go-sev-guest/abi"
)

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
	ID_KEY_ALGO   [0x4]byte   // const 1
	AUTH_KEY_ALGO [0x4]byte   // const 0
	RESERVED_0    [0x38]byte  // 0
	ID_BLOCK_SIG  [0x200]byte // consists of r,s
	ID_KEY        [0x404]byte // consists of Curve, Reserved, Qx, Qy
	RESERVED_1    [0x3c]byte  // 0
	ID_KEY_SIG    [0x200]byte // 0
	AUTH_KEY      [0x404]byte // 0
	RESERVED_2    [0x37c]byte // 0
}

func main() {
	launchDigest := must(hex.DecodeString("679eb9ca1e96782ef4510caebb198e62805467ce6756a66f4b91280416f3c648abb7eee9b102217c2531bdfe937a6083"))
	pol := abi.SnpPolicy{
		SMT:   true,
		Debug: true,
	}
	idBlock := IDBlock{
		LD:        [48]byte(launchDigest),
		VERSION:   0x1,
		GUEST_SNV: 0x2,
		POLICY:    abi.SnpPolicyToBytes(pol),
	}

	// base 64 encode idBlock and print
	fmt.Printf("%s\n", EncodeIDBlock(idBlock))
}

func EncodeIDBlock(idBlock IDBlock) string {
	// Create a buffer to hold all bytes (total size = 48 + 16 + 16 + 4 + 4 + 8 = 96)
	buffer := make([]byte, 96)
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

	return base64.StdEncoding.EncodeToString(buffer)
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
