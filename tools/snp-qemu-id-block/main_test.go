package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"slices"
	"testing"

	"github.com/google/go-sev-guest/tools/lib/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoverPublicKey(t *testing.T) {
	empty := [0x48]byte{}
	// A valid R must be an x-coordinate on the curve
	validR := [0x48]byte(append(empty[1:], 0x02))
	// Any S is valid
	validS := [0x48]byte(append(empty[1:], 0x01))

	// Create true ecdsa key
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Sign message
	hash := sha256.Sum256([]byte("message"))
	signedR, signedS, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	testCases := []struct {
		name      string
		r, s      []byte
		pubKey    *ecdsa.PublicKey
		message   string
		expectErr bool
	}{
		{
			name:    "Valid Signature",
			r:       signedR.Bytes(),
			s:       signedS.Bytes(),
			pubKey:  &privateKey.PublicKey,
			message: "message",
		},
		{
			name:    "Valid constant Signature",
			r:       validR[:],
			s:       validS[:],
			message: "message",
		},
		{
			name:      "Invalid Signature",
			r:         []byte{0x00},
			s:         []byte{0x00},
			message:   "message",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			curve := elliptic.P384()
			bigR := new(big.Int).SetBytes(tc.r)
			bigS := new(big.Int).SetBytes(tc.s)
			hash := sha256.Sum256([]byte(tc.message))
			z := new(big.Int).SetBytes(hash[:])

			publicKeys, err := RecoverPublicKey(curve, bigR, bigS, z)
			if tc.expectErr {
				assert.Error(err)
				return
			}

			require.NoError(err)

			require.Len(publicKeys, 2)

			for _, pubKey := range publicKeys {
				assert.True(ecdsa.Verify(&pubKey, hash[:], bigR, bigS))
			}

			if tc.pubKey != nil {
				slices.ContainsFunc(publicKeys, func(pk ecdsa.PublicKey) bool {
					return pk.X.Cmp(tc.pubKey.X) == 0 && pk.Y.Cmp(tc.pubKey.Y) == 0
				})
			}
		})
	}
}

func TestAbc(t *testing.T) {
	reportAsHex := "020000000000000000000300000000000000000000000000000000000000000000000000000000000000000000000000010000000100000008000000000013480100000000000000000000000000000012bac10ef52375c722c12e8538134a2c6bed16f6376b5634e1575a81fe694255bcd646ecc6f9a8b6251424bc1ec18f45ab8d269c865e0be61e6a68d0610db8e0eb3f808c49ba01fccaa7178a7237a76487791a202585fcf7b2b40375f0c446def2ce0813bf4d3a1ffb0c7f21e5f75f680000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000d6ec0cc500f86415b35059d50446e5cf0af7fed921a4edb1440e8cbc5aaa1c16ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0800000000001348000000000000000000000000000000000000000000000000051391e917130d370868bb82336a2fa7e0e07fd6067e596e032bc7f62e1fd4489c7edd946f064d1860b1a1a6b276222c42a8f0838f4f706790e8025a5c95ee85080000000000134820370100203701000800000000001348000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000cd772f872f1a31ee54a7011c9f6ad9978c053507be98ab4e5d39a07d4e5f046303d5a07f046e003def59e79721677098000000000000000000000000000000000000000000000000a6d5b57e3aa4cb1d84a83d498c1a9dbca10b62576edadcff9d44e97b8062577a3618c54d2a8a82c22af6a694e97e7f220000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	// parse and print report
	// Parse the SEV-SNP report
	attestation, err := report.ParseAttestation(must(hex.DecodeString(reportAsHex)), "bin")
	if err != nil {
		log.Fatalf("Failed to parse SEV-SNP report: %v", err)
	}

	// Print individual fields of the SEV-SNP report
	fmt.Println("SEV-SNP Report Fields:")
	fmt.Printf("  Version: %d\n", attestation.Report.Version)
	fmt.Printf("  Guest SVN: %d\n", attestation.Report.GuestSvn)
	fmt.Printf("  Policy: %d\n", attestation.Report.Policy)
	fmt.Printf("  Family ID: %x\n", attestation.Report.FamilyId)
	fmt.Printf("  Image ID: %x\n", attestation.Report.ImageId)
	fmt.Printf("  VMPL: %d\n", attestation.Report.Vmpl)
	fmt.Printf("  Signature: %x\n", attestation.Report.Signature)
	fmt.Printf("  Report Data: %x\n", attestation.Report.ReportData)
	fmt.Printf("  Measurement: %x\n", attestation.Report.Measurement)

	// If desired, output the entire parsed report as JSON for easier visualization
	parsedReportJSON, err := json.MarshalIndent(attestation, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal report to JSON: %v", err)
	}

	fmt.Println("\nFull Parsed Report (JSON):")
	fmt.Println(string(parsedReportJSON))
}
