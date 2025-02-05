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
	"os"

	"github.com/spf13/afero"
)

const (
	// HashSize is the number of octets in hashes used by this package.
	HashSize = sha256.Size

	histPath = "/mnt/state/history"
)

// History is the history of the Coordinator.
type History struct {
	store   Store
	hashFun func() hash.Hash
}

// New creates a new History backed by the default filesystem store.
func New() (*History, error) {
	osFS := afero.NewOsFs()
	if err := osFS.MkdirAll(histPath, 0o755); err != nil {
		return nil, fmt.Errorf("creating history directory: %w", err)
	}
	store := NewAferoStore(&afero.Afero{Fs: afero.NewBasePathFs(osFS, histPath)})
	return NewWithStore(store), nil
}

// NewWithStore creates a new History with the given storage backend.
func NewWithStore(store Store) *History {
	h := &History{
		store:   store,
		hashFun: sha256.New,
	}
	if HashSize != h.hashFun().Size() {
		panic("mismatch between hashSize and hash function size")
	}
	return h
}

// GetManifest returns the manifest for the given hash.
func (h *History) GetManifest(hash [HashSize]byte) ([]byte, error) {
	return h.getContentaddressed("manifests/%s", hash)
}

// SetManifest sets the manifest and returns its hash.
func (h *History) SetManifest(manifest []byte) ([HashSize]byte, error) {
	return h.setContentaddressed("manifests/%s", manifest)
}

// GetPolicy returns the policy for the given hash.
func (h *History) GetPolicy(hash [HashSize]byte) ([]byte, error) {
	return h.getContentaddressed("policies/%s", hash)
}

// SetPolicy sets the policy and returns its hash.
func (h *History) SetPolicy(policy []byte) ([HashSize]byte, error) {
	return h.setContentaddressed("policies/%s", policy)
}

// GetTransition returns the transition for the given hash.
func (h *History) GetTransition(hash [HashSize]byte) (*Transition, error) {
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
func (h *History) SetTransition(transition *Transition) ([HashSize]byte, error) {
	return h.setContentaddressed("transitions/%s", transition.marshalBinary())
}

// GetLatest verifies the latest transition with the given public key and returns it.
func (h *History) GetLatest(pubKey *ecdsa.PublicKey) (*LatestTransition, error) {
	transitionBytes, err := h.store.Get("transitions/latest")
	if err != nil {
		return nil, fmt.Errorf("getting latest transition: %w", err)
	}
	var latestTransition LatestTransition
	if err := latestTransition.unmarshalBinary(transitionBytes); err != nil {
		return nil, fmt.Errorf("unmarshaling latest transition: %w", err)
	}
	if err := latestTransition.verify(pubKey); err != nil {
		return nil, fmt.Errorf("verifying latest transition: %w", err)
	}
	return &latestTransition, nil
}

// HasLatest returns true if there exist a latest transaction. It does not
// verify the transaction signature or return the transaction.
func (h *History) HasLatest() (bool, error) {
	if _, err := h.store.Get("transitions/latest"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SetLatest signs and sets the latest transition if the current latest is equal to oldT.
func (h *History) SetLatest(oldT, newT *LatestTransition, signingKey *ecdsa.PrivateKey) error {
	if err := newT.sign(signingKey); err != nil {
		return fmt.Errorf("signing latest transition: %w", err)
	}
	if err := h.store.CompareAndSwap("transitions/latest", oldT.marshalBinary(), newT.marshalBinary()); err != nil {
		return fmt.Errorf("setting latest transition: %w", err)
	}
	return nil
}

func (h *History) getContentaddressed(pathFmt string, hash [HashSize]byte) ([]byte, error) {
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

func (h *History) setContentaddressed(pathFmt string, data []byte) ([HashSize]byte, error) {
	hash := h.hash(data)
	hashStr := hex.EncodeToString(hash[:])
	if err := h.store.Set(fmt.Sprintf(pathFmt, hashStr), data); err != nil {
		return [HashSize]byte{}, err
	}
	return hash, nil
}

func (h *History) hash(in []byte) [HashSize]byte {
	hf := h.hashFun()
	_, _ = hf.Write(in) // Hash.Write never returns an error.
	sum := hf.Sum(nil)
	var hash [HashSize]byte
	copy(hash[:], sum) // Correct len of sum enforced in constructor.
	return hash
}

// Transition is a transition between two manifests.
type Transition struct {
	ManifestHash           [HashSize]byte
	PreviousTransitionHash [HashSize]byte
}

func (t *Transition) unmarshalBinary(data []byte) error {
	if len(data) != 2*HashSize {
		return fmt.Errorf("transition has invalid length %d, expected %d", len(data), 2*HashSize)
	}
	copy(t.ManifestHash[:], data[:HashSize])
	copy(t.PreviousTransitionHash[:], data[HashSize:])
	return nil
}

func (t *Transition) marshalBinary() []byte {
	data := make([]byte, 2*HashSize)
	copy(data[:HashSize], t.ManifestHash[:])
	copy(data[HashSize:], t.PreviousTransitionHash[:])
	return data
}

// LatestTransition is the latest transition signed by the Coordinator.
type LatestTransition struct {
	TransitionHash [HashSize]byte
	signature      []byte
}

func (l *LatestTransition) unmarshalBinary(data []byte) error {
	if len(data) <= HashSize {
		return errors.New("latest transition has invalid length")
	}
	sigLen := len(data) - HashSize
	l.signature = make([]byte, sigLen)
	copy(l.TransitionHash[:], data[:HashSize])
	copy(l.signature, data[HashSize:])
	return nil
}

func (l *LatestTransition) marshalBinary() []byte {
	if l == nil {
		return []byte{}
	}
	data := make([]byte, HashSize+len(l.signature))
	copy(data[:HashSize], l.TransitionHash[:])
	copy(data[HashSize:], l.signature)
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

// Store defines the Key-Value store interface used by History.
//
// In addition to the documented behavior below, History expects all functions to be thread-safe
// and the Store to be globally consistent.
type Store interface {
	// Get the value for key.
	//
	// If the key is not found, an error wrapping os.ErrNotExist must be returned.
	Get(key string) ([]byte, error)

	// Set the value for key.
	Set(key string, value []byte) error

	// CompareAndSwap sets key to newVal if, and only if, key is currently oldVal.
	//
	// If the current value is not equal to oldVal, an error must be returned. The comparison must
	// treat a nil slice the same as an empty slice.
	CompareAndSwap(key string, oldVal, newVal []byte) error
}
