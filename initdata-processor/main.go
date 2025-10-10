// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/edgelesssys/contrast/initdata-processor/validator"
	"github.com/edgelesssys/contrast/internal/initdata"
)

var version = "0.0.0-dev"

func main() {
	log.Printf("Contrast service-mesh %s", version)
	log.Print("Report issues at https://github.com/edgelesssys/contrast/issues")

	entries, err := os.ReadDir("/dev")
	if err != nil {
		log.Fatalf("Listing devices: %v", err)
	}
	// The initdata device is usually /dev/vdX so let's start from the back.
	slices.Reverse(entries)
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
		digest, err := doc.Digest()
		if err != nil {
			failf("%s failed initdata validation: %v", path, err)
			break
		}
		validator, err := validator.New()
		if err != nil {
			failf("Creating validator: %v", err)
			break
		}
		if err := validator.ValidateDigest(digest); err != nil {
			failf("Validating initdata digest: %v", err)
			break
		}
		i, err := doc.Parse()
		if err != nil {
			failf("Parsing initdata: %v", err)
			break
		}
		if err := handleInitdata(i); err != nil {
			failf("Handling initdata: %v", err)
			break
		}
		log.Printf("Processed initdata from %q ", path)
		break
	}
	// We always exit with status code 0 so that the Kata agent can start and propagate errors to
	// the runtime.
}

func handleInitdata(data *initdata.Initdata) error {
	const targetPath = "/run/measured-cfg"
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return err
	}

	for name, content := range data.Data {
		name = filepath.Clean(name)
		path := filepath.Join(targetPath, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing file %q: %w", path, err)
		}
	}
	return nil
}

func failf(format string, v ...any) {
	// TODO(burgerdev): this error won't be visible for the runtime - propagate it to the Kata agent.
	log.Printf(format, v...)
}
