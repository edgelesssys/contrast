// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package embedbin

import (
	"fmt"

	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

// MemfdInstaller installs embedded binaries.
type MemfdInstaller struct {
	fallback *RegularInstaller
}

// New returns a new installer.
func New() *MemfdInstaller {
	return &MemfdInstaller{
		fallback: &RegularInstaller{fs: afero.NewOsFs()},
	}
}

// Install creates a memfd and writes the contents to it.
// the first argument is ignored on Linux (would be the prefix on other implementations).
func (i *MemfdInstaller) Install(_ string, contents []byte) (Installed, error) {
	// Try to install using memfd.
	install, err := New().installMemfd(contents)
	if err == nil {
		return install, nil
	}

	// Fallback to regular installer.
	return i.fallback.Install("", contents)
}

// installMemfd creates a memfd and writes the contents to it.
func (*MemfdInstaller) installMemfd(contents []byte) (*MemfdInstall, error) {
	// Create a memfd.
	fd, err := unix.MemfdCreate("embedded-binary", 0)
	if err != nil {
		return nil, fmt.Errorf("installer memfd_create: %w", err)
	}
	if err := unix.Ftruncate(fd, int64(len(contents))); err != nil {
		return nil, fmt.Errorf("memfd installer ftruncate: %w", err)
	}

	// Map the memfd into memory and copy the contents.
	data, err := unix.Mmap(fd, 0, len(contents), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("memfd installer mmap: %w", err)
	}
	copy(data, contents)
	if err := unix.Munmap(data); err != nil {
		return nil, fmt.Errorf("memfd installer munmap: %w", err)
	}

	// memfd is ready to use.
	return &MemfdInstall{fd: fd}, nil
}

// MemfdInstall uses memfd to temporarily get and fd / path for
// the embedded binary.
type MemfdInstall struct {
	fd int
}

// Path returns the path to the memfd.
func (m *MemfdInstall) Path() string {
	return fmt.Sprintf("/proc/self/fd/%d", m.fd)
}

// IsRegular returns false for memfdInstall.
// The file is a memfd, not a regular file.
func (m *MemfdInstall) IsRegular() bool {
	return false
}

// Uninstall closes the memfd.
// The file is automatically removed when the last process closes it.
func (m *MemfdInstall) Uninstall() error {
	return unix.Close(m.fd)
}
