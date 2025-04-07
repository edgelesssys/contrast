// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package seedengine provides deterministic key derivation of ECDSA and symmetric keys
// from a secret seed.
package seedengine

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"

	"filippo.io/keygen"
	"golang.org/x/crypto/hkdf"
)

// SeedEngine provides deterministic key derivation of ECDSA and symmetric keys
// from a secret seed.
type SeedEngine struct {
	curve   func() elliptic.Curve
	hashFun func() hash.Hash
	seed    []byte
	salt    []byte

	podStateSeed []byte
	historySeed  []byte

	rootCAKey             *ecdsa.PrivateKey
	transactionSigningKey *ecdsa.PrivateKey
}

// New creates a new SeedEngine from a secret seed and a salt.
func New(secretSeed []byte, salt []byte) (*SeedEngine, error) {
	se := &SeedEngine{
		curve:   elliptic.P384,
		hashFun: sha256.New,
		seed:    secretSeed,
		salt:    salt,
	}

	// Recommended to use salt length equal to hash size, see RFC 5869, section 3.1.
	if len(salt) != se.hashFun().Size() {
		return nil, fmt.Errorf("salt must be %d bytes long", se.hashFun().Size())
	}
	if len(secretSeed) < se.hashFun().Size() {
		return nil, fmt.Errorf("secret seed must be at least %d bytes long", se.hashFun().Size())
	}

	var err error
	se.podStateSeed, err = se.hkdfDerive(secretSeed, "POD STATE SECRET")
	if err != nil {
		return nil, fmt.Errorf("deriving seed: %w", err)
	}
	se.historySeed, err = se.hkdfDerive(secretSeed, "HISTORY SECRET")
	if err != nil {
		return nil, fmt.Errorf("deriving seed: %w", err)
	}
	transactionSigningSeed, err := se.hkdfDerive(secretSeed, "TRANSACTION SIGNING SECRET")
	if err != nil {
		return nil, fmt.Errorf("deriving seed: %w", err)
	}
	rootCASeed, err := se.hkdfDerive(secretSeed, "ROOT CA SEED")
	if err != nil {
		return nil, fmt.Errorf("deriving seed: %w", err)
	}

	se.transactionSigningKey, err = se.generateECDSAPrivateKey(transactionSigningSeed)
	if err != nil {
		return nil, fmt.Errorf("generating ECDSA key: %w", err)
	}
	se.rootCAKey, err = se.generateECDSAPrivateKey(rootCASeed)
	if err != nil {
		return nil, fmt.Errorf("generating ECDSA key: %w", err)
	}

	return se, nil
}

// DeriveWorkloadSecret derives a secret for a workload from the workload name and the secret seed.
func (s *SeedEngine) DeriveWorkloadSecret(workloadSecretID string) ([]byte, error) {
	if workloadSecretID == "" {
		return nil, errors.New("workload secret ID must not be empty")
	}
	return s.hkdfDerive(s.podStateSeed, fmt.Sprintf("WORKLOAD SECRET ID: %s", workloadSecretID))
}

// GenerateMeshCAKey generates a new random key for the mesh authority.
func (s *SeedEngine) GenerateMeshCAKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
}

// RootCAKey returns the root CA key which is derived from the secret seed.
func (s *SeedEngine) RootCAKey() *ecdsa.PrivateKey {
	return s.rootCAKey
}

// TransactionSigningKey returns the transaction signing key which is derived from the secret seed.
func (s *SeedEngine) TransactionSigningKey() *ecdsa.PrivateKey {
	return s.transactionSigningKey
}

// Seed returns the secret seed.
func (s *SeedEngine) Seed() []byte {
	return s.seed
}

// Salt returns the salt.
func (s *SeedEngine) Salt() []byte {
	return s.salt
}

func (s *SeedEngine) hkdfDerive(secret []byte, info string) ([]byte, error) {
	hkdf := hkdf.New(s.hashFun, secret, s.salt, []byte(info))
	newSecret := make([]byte, len(secret))
	if _, err := io.ReadFull(hkdf, newSecret); err != nil {
		return nil, err
	}
	return newSecret, nil
}

func (s *SeedEngine) generateECDSAPrivateKey(secret []byte) (*ecdsa.PrivateKey, error) {
	return keygen.ECDSA(s.curve(), secret)
}
