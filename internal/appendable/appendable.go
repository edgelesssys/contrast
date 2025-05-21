// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package appendable

import (
	"errors"
	"sync"
)

// Appendable is a thread-safe list that can be appended to.
type Appendable[T any] struct {
	list []T
	mux  sync.RWMutex
}

// Append adds a value to the list.
func (a *Appendable[T]) Append(value T) {
	a.mux.Lock()
	defer a.mux.Unlock()
	a.list = append(a.list, value)
}

// All returns all values in the list.
func (a *Appendable[T]) All() []T {
	a.mux.RLock()
	defer a.mux.RUnlock()
	return a.list
}

// Latest returns the latest value in the list.
func (a *Appendable[T]) Latest() (T, error) {
	a.mux.RLock()
	defer a.mux.RUnlock()

	if len(a.list) == 0 {
		return *new(T), ErrIsEmpty
	}

	return a.list[len(a.list)-1], nil
}

// ErrIsEmpty is returned when trying to get the latest value from an empty list.
var ErrIsEmpty = errors.New("appendable is empty")
