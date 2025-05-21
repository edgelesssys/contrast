// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package fsstore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"log/slog"

	"github.com/spf13/afero"
)

// Store is a filesystem backed store.
// It is not thread-safe.
type Store struct {
	dir    string
	fs     *afero.Afero
	logger *slog.Logger
}

// New returns a new Store.
func New(dir string, log *slog.Logger) *Store {
	return &Store{
		dir:    dir,
		fs:     &afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), dir)},
		logger: log,
	}
}

// Get returns the value for the given key.
func (s *Store) Get(key string) ([]byte, bool) {
	val, err := s.fs.ReadFile(keyToFilename(key))
	if errors.Is(err, fs.ErrNotExist) {
		s.logger.Debug("file does not exist", "file", keyToFilename(key))
		return nil, false
	} else if err != nil {
		s.logger.Error("failed to open file", "file", keyToFilename(key), "err", err)
		return nil, false
	}
	return val, true
}

// Set sets the value for the given key.
func (s *Store) Set(key string, value []byte) {
	if err := s.fs.MkdirAll("/", 0o777); err != nil {
		s.logger.Error("failed to create dir")
		return
	}
	if err := s.fs.WriteFile(keyToFilename(key), value, 0o644); err != nil {
		s.logger.Error("failed to write file", "file", keyToFilename(key), "err", err)
		return
	}
}

// GetAll returns all values in the store.
func (s *Store) GetAll() [][]byte {
	var values [][]byte
	files, err := s.fs.ReadDir("/")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		s.logger.Debug("failed to read dir", "err", err)
		return nil
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		val, err := s.fs.ReadFile(f.Name())
		if err != nil {
			s.logger.Error("failed to open file", "file", f.Name(), "err", err)
			continue
		}
		values = append(values, val)
	}
	return values
}

// Clear clears all values from store.
func (s *Store) Clear() {
	if err := s.fs.RemoveAll("/"); err != nil {
		s.logger.Error("failed to remove all", "err", err)
	}
}

func keyToFilename(key string) string {
	hash := sha256.Sum256([]byte(key))
	return "sha256-" + hex.EncodeToString(hash[:])
}
