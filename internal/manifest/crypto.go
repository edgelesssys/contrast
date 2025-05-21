// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/edgelesssys/contrast/internal/userapi"
)

// NewSeedShareOwnerPrivateKey creates and PEM-encodes a new seed share private key.
func NewSeedShareOwnerPrivateKey() ([]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}), nil
}

// ExtractSeedshareOwnerPublicKey extracts the public key for a seedshare owner and returns it as serialized DER.
//
// This function supports PEM-encoded public and private keys.
func ExtractSeedshareOwnerPublicKey(keyData []byte) (HexString, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("decoding seedshare owner key: no key found")
	}
	switch block.Type {
	case "PUBLIC KEY":
		return NewHexString(block.Bytes), nil
	case "RSA PRIVATE KEY":
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("parsing seedshare owner key: %w", err)
		}
		publicKey := MarshalSeedShareOwnerKey(&privateKey.PublicKey)
		return publicKey, nil
	default:
		return "", fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}

// ParseSeedshareOwnerPrivateKey decodes a PEM-encoded seed share private key.
func ParseSeedshareOwnerPrivateKey(keyData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("decoding seedshare owner key: no key found")
	}
	if block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("decoding seedshare owner key: invalid key type %q", block.Type)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing seedshare owner key: %w", err)
	}
	return key, nil
}

// MarshalSeedShareOwnerKey converts a public key into the format for userapi.SetManifestRequest.
func MarshalSeedShareOwnerKey(pubKey *rsa.PublicKey) HexString {
	return NewHexString(x509.MarshalPKCS1PublicKey(pubKey))
}

// ParseSeedShareOwnerKey reads a public key embedded in a userapi.SetManifestRequest.
func ParseSeedShareOwnerKey(pubKeyHex HexString) (*rsa.PublicKey, error) {
	pubKeyBytes, err := pubKeyHex.Bytes()
	if err != nil {
		return nil, fmt.Errorf("parsing from hex: %w", err)
	}
	pubKey, err := x509.ParsePKCS1PublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing from PKCS1: %w", err)
	}
	return pubKey, nil
}

// EncryptSeedShares encrypts a seed for owners identified by their public keys and returns a SeedShare slice suitable for userapi.SetManifestResponse.
func EncryptSeedShares(seed []byte, ownerPubKeys []HexString) ([]*userapi.SeedShare, error) {
	var out []*userapi.SeedShare
	for _, pubKeyHex := range ownerPubKeys {
		pubKey, err := ParseSeedShareOwnerKey(pubKeyHex)
		if err != nil {
			return nil, fmt.Errorf("parsing seed share owner key: %w", err)
		}
		cipherText, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, seed, []byte("seedshare"))
		if err != nil {
			return nil, fmt.Errorf("encrypting seed share: %w", err)
		}
		seedShare := &userapi.SeedShare{
			EncryptedSeed: cipherText,
			PublicKey:     pubKeyHex.String(),
		}
		out = append(out, seedShare)
	}
	return out, nil
}

// DecryptSeedShare tries to decrypt a SeedShare with the given owner key.
func DecryptSeedShare(key *rsa.PrivateKey, seedShare *userapi.SeedShare) ([]byte, error) {
	// TODO(burgerdev): check seedShare.PublicKey?
	return rsa.DecryptOAEP(sha256.New(), nil, key, seedShare.GetEncryptedSeed(), []byte("seedshare"))
}

// NewWorkloadOwnerKey creates and marshals a private key.
func NewWorkloadOwnerKey() ([]byte, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}), nil
}

// ParseWorkloadOwnerPrivateKey parses a PEM-encoded private key.
func ParseWorkloadOwnerPrivateKey(keyBytes []byte) (*ecdsa.PrivateKey, error) {
	pemBlock, _ := pem.Decode(keyBytes)
	if pemBlock == nil {
		return nil, fmt.Errorf("decoding workload owner key: no key found")
	}
	if pemBlock.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("workload owner key is not an EC private key")
	}
	workloadOwnerKey, err := x509.ParseECPrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing workload owner key: %w", err)
	}
	return workloadOwnerKey, nil
}

// HashWorkloadOwnerKey converts a public key into the format for Manifest.WorkloadOwnerKeyDigests.
func HashWorkloadOwnerKey(pubKey *ecdsa.PublicKey) HexString {
	keyData, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		// According to the docs for MarshalPKIXPublicKey, an error should only
		// occur for unsupported key types. *ecdsa.PublicKey is a supported key
		// type.
		panic(fmt.Errorf("failed to marshal key: %w", err))
	}

	ownerKeyHash := sha256.Sum256(keyData)
	return NewHexString(ownerKeyHash[:])
}

// ExtractWorkloadOwnerPublicKey extracts the public key for a workload owner and returns it as serialized DER.
//
// This function supports PEM-encoded public and private keys.
func ExtractWorkloadOwnerPublicKey(keyData []byte) ([]byte, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	var publicKey []byte
	switch block.Type {
	case "PUBLIC KEY":
		return block.Bytes, nil
	case "EC PRIVATE KEY":
		privateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing EC private key: %w", err)
		}
		publicKey, err = x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("marshaling public key: %w", err)
		}
		return publicKey, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}
