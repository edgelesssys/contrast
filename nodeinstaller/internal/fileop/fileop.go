// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package fileop

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"slices"
	"syscall"
)

// OS provides file operations on the real filesystem.
type OS struct {
	newHash func() hash.Hash
}

// NewDefault creates a new file operation helper with a default hash function.
func NewDefault() *OS {
	return New(sha512.New)
}

// New creates a new file operation helper.
func New(newHash func() hash.Hash) *OS {
	return &OS{newHash: newHash}
}

// CopyOnDiff copies a file from src to dst.
// It will only modify the destination file if the contents are different.
// It returns true if the file was modified, or an error if any occurred.
func (o *OS) CopyOnDiff(src, dst string) (bool, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return false, fmt.Errorf("opening source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// open readonly first to prevent ETXTBSY
	dstFile, err := os.OpenFile(dst, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return false, fmt.Errorf("opening destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// check if the file already exists and has the correct hash
	identical, err := o.identical(srcFile, dstFile)
	if err != nil {
		return false, err
	}
	if identical {
		return false, nil
	}

	// reopen the file for writing
	if err := dstFile.Close(); err != nil {
		return false, fmt.Errorf("closing destination file: %w", err)
	}
	dstFile, err = o.openWritableWithForce(dst)
	if err != nil {
		return false, fmt.Errorf("opening destination file %s: %w", dst, err)
	}

	if err := o.copyAt(srcFile, dstFile); err != nil {
		return false, fmt.Errorf("copying file: %w", err)
	}
	return true, nil
}

// Copy copies a file from src to dst.
// It will overwrite the destination file in any case.
func (o *OS) Copy(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file %s: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := o.openWritableWithForce(dst)
	if err != nil {
		return fmt.Errorf("opening destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	if err := o.copyAt(srcFile, dstFile); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}
	return nil
}

// Move moves a file from src to dst.
func (o *OS) Move(src, dst string) error {
	switch err := os.Rename(src, dst); {
	case err == nil:
		return nil
	case errors.Is(err, syscall.EXDEV):
		// rename across devices is not supported, so we need to copy and remove
	default:
		return fmt.Errorf("moving file: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file %s: %w", src, err)
	}
	defer srcFile.Close()
	dstFile, err := o.openWritableWithForce(dst)
	if err != nil {
		return fmt.Errorf("opening destination file %s: %w", dst, err)
	}
	defer dstFile.Close()
	if err := o.copyAt(srcFile, dstFile); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("removing source file: %w", err)
	}
	return nil
}

// identical returns true if the contents of the two files are identical.
// The files are rewound to their original positions after the comparison.
func (o *OS) identical(src, dst *os.File) (bool, error) {
	srcHash, err := o.checksumAt(src)
	if err != nil {
		return false, fmt.Errorf("getting source file checksum: %w", err)
	}
	dstHash, err := o.checksumAt(dst)
	if err != nil {
		return false, fmt.Errorf("getting destination file checksum: %w", err)
	}
	return slices.Equal(srcHash, dstHash), nil
}

// openWritableWithForce opens a file for writing.
// If the existing file cannot be opened for writing,
// it will be removed and recreated.
func (o *OS) openWritableWithForce(path string) (*os.File, error) {
	// try to open the file for writing
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err == nil {
		return file, nil
	}
	switch {
	case errors.Is(err, syscall.ETXTBSY):
		// file is currently being executed. need to unlink it first
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("removing file %s: %w", path, err)
		}
	default:
		return nil, err
	}
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
}

// checksumAt returns the checksum of the file.
// It will temporarily move the file offset but restore it to the previous position.
func (o *OS) checksumAt(file *os.File) ([]byte, error) {
	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("getting file position: %w", err)
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("rewinding file: %w", err)
	}
	hasher := o.newHash()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("hashing file: %w", err)
	}
	_, err = file.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("restoring file position: %w", err)
	}
	return hasher.Sum(nil), nil
}

// copyTraditional copies the contents of src to dst.
// This is the slow path, used when reflink is not supported.
func (o *OS) copyTraditional(src, dst *os.File) error {
	_, err := src.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = dst.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	if err = dst.Truncate(0); err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}
