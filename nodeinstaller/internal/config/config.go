// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"encoding/base64"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

// Config is the configuration for the node-installer.
type Config struct {
	// Files is a list of files to download.
	Files []File `json:"files"`
	// QemuExtraKernelParams is an additional set of kernel parameters to pass kernels started with QEMU.
	QemuExtraKernelParams string `json:"QemuExtraKernelParams"`
	// DebugRuntime enables the debug mode of the runtime.
	// This only works if the igvm file has shell access enabled
	// and has no effect on production images.
	DebugRuntime bool `json:"debugRuntime"`
}

// Validate validates the configuration.
func (c Config) Validate() error {
	for _, file := range c.Files {
		if err := file.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// File is a file to download.
type File struct {
	// URL is the URL to fetch the file from.
	URL string `json:"url"`
	// Path is the absolute path (on the host) to save the file to.
	Path string `json:"path"`
	// Executable is true if the file should be executable.
	Executable bool `json:"executable"`
	// Integrity is the content subresource integrity (expected hash) of the file. Required if the file is downloaded.
	// The format of a subresource integrity string is defined here:
	// https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
	Integrity string `json:"integrity"`
}

// Validate validates the file.
func (f File) Validate() error {
	if f.URL == "" {
		return errors.New("url is required")
	}
	uri, err := url.Parse(f.URL)
	if err != nil {
		return errors.New("url is not valid")
	}
	var needsSRI bool
	switch uri.Scheme {
	case "http", "https":
		needsSRI = true
	case "file":
		needsSRI = false
	default:
		return errors.New("url scheme must be http, https, or file")
	}
	if f.Path == "" {
		return errors.New("path is required")
	}
	effectivePath := strings.ReplaceAll(f.Path, "@@runtimeBase@@", "")
	if !filepath.IsAbs(effectivePath) {
		return errors.New("path must be absolute")
	}
	if f.Integrity == "" {
		if needsSRI {
			return errors.New("integrity is required for http/https URLs")
		}
		return nil
	}
	if f.Integrity[:7] != "sha256-" && f.Integrity[:7] != "sha384-" && f.Integrity[:7] != "sha512-" {
		return errors.New("integrity must use a valid content sri algorithm (sha256, sha384, sha512)")
	}
	if _, err := base64.StdEncoding.DecodeString(f.Integrity[7:]); err != nil {
		return errors.New("integrity value is not valid base64")
	}
	return nil
}
