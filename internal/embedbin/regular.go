// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package embedbin

import (
	"github.com/spf13/afero"
)

// RegularInstaller uses a regular file to install an embedded binary.
type RegularInstaller struct {
	fs afero.Fs
}

// Install creates a regular file and writes the contents to it.
// prefix is an optional prefix for the temporary file.
// If prefix is empty, a temporary directory will be used.
func (r *RegularInstaller) Install(prefix string, contents []byte) (Installed, error) {
	if prefix != "" {
		if err := r.fs.MkdirAll(prefix, 0o777); err != nil {
			return nil, err
		}
	}
	file, err := afero.TempFile(r.fs, prefix, "contrast-embedded-binary-*")
	if err != nil {
		return nil, err
	}
	if err := r.fs.Chmod(file.Name(), 0o755); err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	if _, err := file.Write(contents); err != nil {
		return nil, err
	}
	return &RegularInstall{
		fs:   r.fs,
		path: file.Name(),
	}, nil
}

// RegularInstall uses a regular file to install an embedded binary.
type RegularInstall struct {
	fs   afero.Fs
	path string
}

// Path returns the path to the regular file.
func (r *RegularInstall) Path() string {
	return r.path
}

// IsRegular returns true for regularInstall.
// It is always backed by a regular file.
func (r *RegularInstall) IsRegular() bool {
	return true
}

// Uninstall removes the regular file.
func (r *RegularInstall) Uninstall() error {
	return r.fs.Remove(r.path)
}
