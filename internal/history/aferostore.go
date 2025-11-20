// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package history

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/spf13/afero"
)

var keyRe = regexp.MustCompile(`^[a-zA-Z0-9-]+/[a-zA-Z0-9-]+$`)

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
	if !keyRe.MatchString(key) {
		return nil, fmt.Errorf("invalid key %q", key)
	}
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.fs.ReadFile(key)
}

// Set the value for key.
func (s *AferoStore) Set(key string, value []byte) error {
	if !keyRe.MatchString(key) {
		return fmt.Errorf("invalid key %q", key)
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	if err := s.fs.MkdirAll(filepath.Dir(key), 0o755); err != nil {
		return fmt.Errorf("creating directory for %q: %w", key, err)
	}
	return s.fs.WriteFile(key, value, 0o644)
}

// Has returns true if the key exists.
func (s *AferoStore) Has(key string) (bool, error) {
	if !keyRe.MatchString(key) {
		return false, fmt.Errorf("invalid key %q", key)
	}
	s.mux.RLock()
	defer s.mux.RUnlock()
	_, err := s.fs.Stat(key)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// CompareAndSwap updates the key to newVal if its current value is oldVal.
func (s *AferoStore) CompareAndSwap(key string, oldVal, newVal []byte) error {
	if !keyRe.MatchString(key) {
		return fmt.Errorf("invalid key %q", key)
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	current, err := s.fs.ReadFile(key)
	// Treat non-existing file as empty to allow initial set.
	if err != nil && (!errors.Is(err, fs.ErrNotExist) || len(oldVal) != 0) {
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
func (s *AferoStore) Watch(key string) (<-chan []byte, func(), error) {
	if !keyRe.MatchString(key) {
		return nil, nil, fmt.Errorf("invalid key %q", key)
	}
	return nil, func() {}, nil
}
