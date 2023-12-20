package appendable

import (
	"errors"
	"fmt"
	"sync"
)

type Appendable[T any] struct {
	list []*T
	mux  sync.RWMutex
}

func (a *Appendable[T]) Append(value *T) error {
	if value == nil {
		return fmt.Errorf("nil value of type %T cannot be appended", value)
	}
	a.mux.Lock()
	defer a.mux.Unlock()
	a.list = append(a.list, value)
	return nil
}

func (a *Appendable[T]) All() []*T {
	a.mux.RLock()
	defer a.mux.RUnlock()
	return a.list
}

func (a *Appendable[T]) Latest() (*T, error) {
	a.mux.RLock()
	defer a.mux.RUnlock()

	if len(a.list) == 0 {
		return nil, errors.New("appendable is empty")
	}

	return a.list[len(a.list)-1], nil
}
