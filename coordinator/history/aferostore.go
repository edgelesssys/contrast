// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package history

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/spf13/afero"
)

// AferoStore is a Store implementation backed by an Afero filesystem.
type AferoStore struct {
	fs  *afero.Afero
	mux sync.RWMutex
}

// NewAferoStore creates a new instance backed by the given fs.
func NewAferoStore(fs *afero.Afero) *AferoStore {
	return &AferoStore{fs: fs}
}

// Get the value for key.
func (s *AferoStore) Get(key string) ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.fs.ReadFile(key)
}

// Set the value for key.
func (s *AferoStore) Set(key string, value []byte) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	if err := s.fs.MkdirAll(filepath.Dir(key), 0o755); err != nil {
		return fmt.Errorf("creating directory for %q: %w", key, err)
	}
	return s.fs.WriteFile(key, value, 0o644)
}

// CompareAndSwap updates the key to newVal if its current value is oldVal.
func (s *AferoStore) CompareAndSwap(key string, oldVal, newVal []byte) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	current, err := s.fs.ReadFile(key)
	// Treat non-existing file as empty to allow initial set.
	if err != nil && !(errors.Is(err, fs.ErrNotExist) && len(oldVal) == 0) {
		return err
	}
	if !bytes.Equal(current, oldVal) {
		return fmt.Errorf("object %q has changed since last read", key)
	}
	if err := s.fs.MkdirAll(filepath.Dir(key), 0o755); err != nil {
		return fmt.Errorf("creating directory for %q: %w", key, err)
	}
	return s.fs.WriteFile(key, newVal, 0o644)
}

// Watch watches for changes to the value of key.
//
// Not implemented for AferoStore.
func (s *AferoStore) Watch(_ string) (<-chan []byte, func(), error) {
	return nil, func() {}, nil
}
