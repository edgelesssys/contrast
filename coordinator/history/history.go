// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package history

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"

	"github.com/spf13/afero"
)

const (
	hashSize = sha256.Size // byte, History.hashFun().Size()
	histPath = "/mnt/state/history"
)

// History is the history of the Coordinator.
type History struct {
	store      store
	hashFun    func() hash.Hash
	signingKey *ecdsa.PrivateKey
}

// New creates a new History with the given signing key.
func New() (*History, error) {
	osFS := afero.NewOsFs()
	if err := osFS.MkdirAll(histPath, 0o755); err != nil {
		return nil, fmt.Errorf("creating history directory: %w", err)
	}
	h := &History{
		store:   newPVStore(&afero.Afero{Fs: afero.NewBasePathFs(osFS, histPath)}),
		hashFun: sha256.New,
	}
	if hashSize != h.hashFun().Size() {
		return nil, errors.New("mismatch between hashSize and hash function size")
	}
	return h, nil
}

// ConfigureSigningKey sets the signing key for validation and signing of the protected history parts.
func (h *History) ConfigureSigningKey(signingKey *ecdsa.PrivateKey) {
	h.signingKey = signingKey
}

// GetManifest returns the manifest for the given hash.
func (h *History) GetManifest(hash [hashSize]byte) ([]byte, error) {
	return h.getContentaddressed("manifests/%s", hash)
}

// SetManifest sets the manifest and returns its hash.
func (h *History) SetManifest(manifest []byte) ([hashSize]byte, error) {
	return h.setContentaddressed("manifests/%s", manifest)
}

// GetPolicy returns the policy for the given hash.
func (h *History) GetPolicy(hash [hashSize]byte) ([]byte, error) {
	return h.getContentaddressed("policies/%s", hash)
}

// SetPolicy sets the policy and returns its hash.
func (h *History) SetPolicy(policy []byte) ([hashSize]byte, error) {
	return h.setContentaddressed("policies/%s", policy)
}

// GetTransition returns the transition for the given hash.
func (h *History) GetTransition(hash [hashSize]byte) (*Transition, error) {
	transitionBytes, err := h.getContentaddressed("transitions/%s", hash)
	if err != nil {
		return nil, err
	}
	var transition Transition
	if err := transition.unmarshalBinary(transitionBytes); err != nil {
		return nil, fmt.Errorf("unmarshaling transition: %w", err)
	}
	return &transition, nil
}

// SetTransition sets the transition and returns its hash.
func (h *History) SetTransition(transition *Transition) ([hashSize]byte, error) {
	return h.setContentaddressed("transitions/%s", transition.marshalBinary())
}

// GetLatest returns the verified transition for the given hash.
func (h *History) GetLatest() (*LatestTransition, error) {
	if h.signingKey == nil {
		return nil, errors.New("signing key not configured")
	}
	transitionBytes, err := h.store.Get("transitions/latest")
	if err != nil {
		return nil, fmt.Errorf("getting latest transition: %w", err)
	}
	var latestTransition LatestTransition
	if err := latestTransition.unmarshalBinary(transitionBytes); err != nil {
		return nil, fmt.Errorf("unmarshaling latest transition: %w", err)
	}
	if err := latestTransition.verify(&h.signingKey.PublicKey); err != nil {
		return nil, fmt.Errorf("verifying latest transition: %w", err)
	}
	return &latestTransition, nil
}

// SetLatest signs and sets the latest transition if the current latest is equal to oldT.
func (h *History) SetLatest(oldT, newT *LatestTransition) error {
	if h.signingKey == nil {
		return errors.New("signing key not configured")
	}
	if err := newT.sign(h.signingKey); err != nil {
		return fmt.Errorf("signing latest transition: %w", err)
	}
	if err := h.store.CompareAndSwap("transitions/latest", oldT.marshalBinary(), newT.marshalBinary()); err != nil {
		return fmt.Errorf("setting latest transition: %w", err)
	}
	return nil
}

func (h *History) getContentaddressed(pathFmt string, hash [hashSize]byte) ([]byte, error) {
	hashStr := hex.EncodeToString(hash[:])
	data, err := h.store.Get(fmt.Sprintf(pathFmt, hashStr))
	if err != nil {
		return nil, err
	}
	dataHash := h.hash(data)
	if !bytes.Equal(hash[:], dataHash[:]) {
		return nil, HashMismatchError{Expected: hash[:], Actual: dataHash[:]}
	}
	return data, nil
}

func (h *History) setContentaddressed(pathFmt string, data []byte) ([hashSize]byte, error) {
	hash := h.hash(data)
	hashStr := hex.EncodeToString(hash[:])
	if err := h.store.Set(fmt.Sprintf(pathFmt, hashStr), data); err != nil {
		return [hashSize]byte{}, err
	}
	return hash, nil
}

func (h *History) hash(in []byte) [hashSize]byte {
	hf := h.hashFun()
	_, _ = hf.Write(in) // Hash.Write never returns an error.
	sum := hf.Sum(nil)
	var hash [hashSize]byte
	copy(hash[:], sum) // Correct len of sum enforced in constructor.
	return hash
}

// Transition is a transition between two manifests.
type Transition struct {
	ManifestHash           [hashSize]byte
	PreviousTransitionHash [hashSize]byte
}

func (t *Transition) unmarshalBinary(data []byte) error {
	if len(data) != 2*hashSize {
		return fmt.Errorf("transition has invalid length %d, expected %d", len(data), 2*hashSize)
	}
	copy(t.ManifestHash[:], data[:hashSize])
	copy(t.PreviousTransitionHash[:], data[hashSize:])
	return nil
}

func (t *Transition) marshalBinary() []byte {
	data := make([]byte, 2*hashSize)
	copy(data[:hashSize], t.ManifestHash[:])
	copy(data[hashSize:], t.PreviousTransitionHash[:])
	return data
}

// LatestTransition is the latest transition signed by the Coordinator.
type LatestTransition struct {
	TransitionHash [hashSize]byte
	signature      []byte
}

func (l *LatestTransition) unmarshalBinary(data []byte) error {
	if len(data) <= hashSize {
		return errors.New("latest transition has invalid length")
	}
	sigLen := len(data) - hashSize
	l.signature = make([]byte, sigLen)
	copy(l.TransitionHash[:], data[:hashSize])
	copy(l.signature, data[hashSize:])
	return nil
}

func (l *LatestTransition) marshalBinary() []byte {
	if l == nil {
		return []byte{}
	}
	data := make([]byte, hashSize+len(l.signature))
	copy(data[:hashSize], l.TransitionHash[:])
	copy(data[hashSize:], l.signature)
	return data
}

func (l *LatestTransition) sign(key *ecdsa.PrivateKey) error {
	var err error
	l.signature, err = ecdsa.SignASN1(rand.Reader, key, l.TransitionHash[:])
	return err
}

func (l *LatestTransition) verify(key *ecdsa.PublicKey) error {
	if !ecdsa.VerifyASN1(key, l.TransitionHash[:], l.signature) {
		return errors.New("latest transition signature is invalid")
	}
	return nil
}

// HashMismatchError is returned when a hash does not match the expected value.
// This can occur when content addressed storage has been corrupted.
type HashMismatchError struct {
	Expected []byte
	Actual   []byte
}

func (e HashMismatchError) Error() string {
	return fmt.Sprintf("hash mismatch: expected %x, got %x", e.Expected, e.Actual)
}

type store interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	CompareAndSwap(key string, oldVal, newVal []byte) error
}
