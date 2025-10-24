// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	gcr "github.com/google/go-containerregistry/pkg/v1"
	"golang.org/x/sys/unix"
)

// Mount a list of layers to the specified mount path.
//
// The layer digests must be in the order they appear in the manifest.
func (s *Store) Mount(where string, layerDigests ...gcr.Hash) error {
	overlayDir := filepath.Join(s.Root, "overlay")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		return fmt.Errorf("creating overlay dir %q: %w", overlayDir, err)
	}
	upper, err := os.MkdirTemp(overlayDir, "upper-*")
	if err != nil {
		return fmt.Errorf("creating upper dir: %w", err)
	}
	work, err := os.MkdirTemp(overlayDir, "work-*")
	if err != nil {
		return fmt.Errorf("creating work dir: %w", err)
	}
	lower, err := s.getLayerDirs(layerDigests...)
	if err != nil {
		return fmt.Errorf("collecting overlay directories: %w", err)
	}
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)

	if err := unix.Mount("none", where, "overlay", 0, opts); err != nil {
		return fmt.Errorf("`mount -t overlay none %q -o %q` failed: %w", where, opts, err)
	}
	return nil
}

// getLayerDirs creates a string of colon-separated layer dirs which is suitable for putting into
// the lowerdir mount option for an overlayfs.
func (s *Store) getLayerDirs(layerDigests ...gcr.Hash) (string, error) {
	var dirs []string

	// In the mount syscall options, the lowest directory is the rightmost one.
	// https://www.kernel.org/doc/html/latest/filesystems/overlayfs.html#multiple-lower-layers
	slices.Reverse(layerDigests)
	for _, l := range layerDigests {
		p := filepath.Join(s.Root, l.Algorithm, l.Hex)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("problem with layer %q: %w", l, err)
		}
		dirs = append(dirs, p)
	}
	return strings.Join(dirs, ":"), nil
}
