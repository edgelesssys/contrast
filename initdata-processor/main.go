// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/edgelesssys/contrast/initdata-processor/policy"
	"github.com/edgelesssys/contrast/initdata-processor/validator"
	"github.com/edgelesssys/contrast/internal/initdata"
)

const measuredConfigPath = "/run/measured-cfg"

var version = "0.0.0-dev"

func main() {
	log.Printf("Contrast initdata-processor %s", version)
	log.Print("Report issues at https://github.com/edgelesssys/contrast/issues")

	if err := os.MkdirAll(measuredConfigPath, 0o755); err != nil {
		failf("Could not create directory %q: %v", measuredConfigPath, err)
		return
	}

	entries, err := os.ReadDir("/dev")
	if err != nil {
		failf("Could not list devices: %v", err)
		return
	}
	// The initdata device is usually /dev/vdX so let's start from the back.
	slices.Reverse(entries)

	var deviceFound bool
	for _, entry := range entries {
		// We're only interested in block devices.
		if entry.Type()&(fs.ModeDevice|fs.ModeCharDevice) != fs.ModeDevice {
			continue
		}

		path := filepath.Join("/dev", entry.Name())
		doc, err := initdata.FromDevice(path)
		if err != nil {
			log.Printf("%s is not an initdata device: %v", path, err)
			continue
		}
		deviceFound = true
		if err := handleInitdata(doc); err != nil {
			failf("handling initdata: %v", err)
			break
		}
		log.Printf("Processed initdata from %q ", path)
		break
	}
	if !deviceFound {
		failf("no initdata device found")
	}
	// We always exit with status code 0 so that the Kata agent can start and propagate errors to
	// the runtime.
}

func handleInitdata(doc initdata.Raw) error {
	digest, err := doc.Digest()
	if err != nil {
		return fmt.Errorf("initdata validation failed: %w", err)
	}
	validator, err := validator.New()
	if err != nil {
		return fmt.Errorf("creating validator: %w", err)
	}
	if err := validator.ValidateDigest(digest); err != nil {
		return fmt.Errorf("validating initdata digest: %w", err)
	}
	data, err := doc.Parse()
	if err != nil {
		return fmt.Errorf("parsing initdata: %w", err)
	}
	for name, content := range data.Data {
		name = filepath.Clean(name)
		path := filepath.Join(measuredConfigPath, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing file %q: %w", path, err)
		}
	}
	return nil
}

func failf(format string, v ...any) {
	log.Printf(format, v...)

	// Create a policy with this message in order to propagate it to the Kata runtime.
	content := policy.DenyWithMessage(format, v...)

	// Write this policy to a temp file and then atomically place it under /run/measured-cfg.
	// We don't want half-written files under this directory!

	f, err := os.CreateTemp(filepath.Dir(measuredConfigPath), "error-policy.*.rego")
	if err != nil {
		log.Printf("Error creating policy file: %v", err)
		return
	}
	sourcePath := f.Name()

	if _, err := io.Copy(f, bytes.NewBuffer(content)); err != nil {
		log.Printf("Error writing policy file: %v", err)
		return
	}

	path := filepath.Join(measuredConfigPath, "policy.rego")
	if err := os.Rename(sourcePath, path); err != nil {
		log.Printf("Error moving file: %v", err)
		return
	}
}
