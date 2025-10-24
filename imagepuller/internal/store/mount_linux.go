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

	"golang.org/x/sys/unix"
)

func (s *Store) Mount(where string, layerDigests ...string) error {
	upper, err := os.MkdirTemp(s.Staging, "upper-*")
	if err != nil {
		return fmt.Errorf("creating upper dir: %w", err)
	}
	work, err := os.MkdirTemp(s.Staging, "work-*")
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

func (s *Store) getLayerDirs(layerDigests ...string) (string, error) {
	var dirs []string

	// In the mount syscall options, the lowest directory is the rightmost one.
	// https://www.kernel.org/doc/html/latest/filesystems/overlayfs.html#multiple-lower-layers
	slices.Reverse(layerDigests)
	for _, l := range layerDigests {
		algo, digest, ok := strings.Cut(l, ":")
		if !ok {
			return "", fmt.Errorf("digest should contain colon: %q", l)
		}
		p := filepath.Join(s.Root, algo, digest)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("problem with layer %q: %w", l, err)
		}
		dirs = append(dirs, p)
	}
	return strings.Join(dirs, ":"), nil
}
