// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package memstore

import "sync"

// Store is a thread-safe map.
type Store[keyT comparable, valueT any] struct {
	m   map[keyT]valueT
	mux sync.RWMutex
}

// New returns a new Store.
func New[keyT comparable, valueT any]() *Store[keyT, valueT] {
	return &Store[keyT, valueT]{
		m: make(map[keyT]valueT),
	}
}

// Get returns the value for the given key.
func (s *Store[keyT, valueT]) Get(key keyT) (valueT, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	v, ok := s.m[key]
	return v, ok
}

// Set sets the value for the given key.
func (s *Store[keyT, valueT]) Set(key keyT, value valueT) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.m[key] = value
}

// GetAll returns all values in the store.
func (s *Store[keyT, valueT]) GetAll() []valueT {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var values []valueT
	for _, v := range s.m {
		values = append(values, v)
	}
	return values
}

// Clear clears all values from store.
func (s *Store[keyT, valueT]) Clear() {
	s.mux.Lock()
	defer s.mux.Unlock()
	clear(s.m)
}
