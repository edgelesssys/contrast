package memstore

import "sync"

type Store[keyT comparable, valueT any] struct {
	m   map[keyT]valueT
	mux sync.RWMutex
}

func New[keyT comparable, valueT any]() *Store[keyT, valueT] {
	return &Store[keyT, valueT]{
		m: make(map[keyT]valueT),
	}
}

func (s *Store[keyT, valueT]) Get(key keyT) (valueT, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	v, ok := s.m[key]
	return v, ok
}

func (s *Store[keyT, valueT]) Set(key keyT, value valueT) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.m[key] = value
}

func (s *Store[keyT, valueT]) GetAll() []valueT {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var values []valueT
	for _, v := range s.m {
		values = append(values, v)
	}
	return values
}
