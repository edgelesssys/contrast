// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package fileop

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// copyAt copies the contents of src to dst.
func (o *OS) copyAt(src, dst *os.File) error {
	// try to copy the file using the copy_file_range syscall
	if _, err := copyReflink(src, dst); err == nil {
		return nil
	}

	// fallback to regular copy if the reflink is not supported
	return o.copyTraditional(src, dst)
}

// copyReflink copies the contents of src to dst using the copy_file_range syscall.
// This is more efficient than a regular copy, as it can share the backing storage (copy-on-write).
func copyReflink(src, dst *os.File) (int, error) {
	size, err := src.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if err := dst.Truncate(size); err != nil {
		return 0, err
	}
	offIn := int64(0)
	offOut := int64(0)
	written, err := unix.CopyFileRange(
		int(src.Fd()), &offIn, // copy from the start of src
		int(dst.Fd()), &offOut, // to the start of dst
		int(size), // copy the entire file
		0,         // no flags
	)
	if err != nil {
		return written, err
	}
	if written != int(size) {
		return written, io.ErrShortWrite
	}
	return written, nil
}
