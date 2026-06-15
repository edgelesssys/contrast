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
	"syscall"

	"github.com/edgelesssys/contrast/initdata-processor/policy"
	"github.com/edgelesssys/contrast/initdata-processor/validator"
	"github.com/edgelesssys/contrast/internal/initdata"
	"golang.org/x/sys/unix"
)

const (
	measuredConfigPath = "/run/measured-cfg"
	insecureConfigPath = "/run/insecure-cfg"
)

var version = "0.0.0-dev"

// We always exit with status code 0 so that the Kata agent can start and propagate errors to
// the runtime.
func main() {
	log.Printf("Contrast initdata-processor %s", version)
	log.Print("Report issues at https://github.com/edgelesssys/contrast/issues")

	// Handle initdata.
	if err := os.MkdirAll(measuredConfigPath, 0o755); err != nil {
		failf("Could not create directory %q: %v", measuredConfigPath, err)
		return
	}
	device, err := checkDeviceAvailability("initdata", []byte("initdata"))
	if err != nil {
		failf("no initdata device found: %v", err)
		return
	}
	doc, err := initdata.FromDevice(device, "initdata")
	if err != nil {
		failf("%s is not an initdata device: %v", device, err)
		return
	}
	if err := handleInitdata(doc); err != nil {
		failf("handling initdata: %v", err)
		return
	}
	log.Printf("Processed initdata from %q ", device)

	// Handle imagepuller auth config.
	if err := os.MkdirAll(insecureConfigPath, 0o755); err != nil {
		failf("Could not create directory %q: %v", insecureConfigPath, err)
		return
	}
	device, err = checkDeviceAvailability("imagepuller", []byte("imgpullr"))
	if err != nil {
		log.Printf("No imagepuller auth config found, only unauthenticated pulls will be available: %v", err)
		return
	}
	doc, err = initdata.FromDevice(device, "imgpullr")
	if err != nil {
		failf("%s is not an imagepuller config device: %v", device, err)
		return
	}
	if err := handleImagepullerAuthConfig(doc); err != nil {
		failf("handling imagepuller auth config: %v", err)
		return
	}
	log.Printf("Processed imagepuller auth config from %q ", device)
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

func handleImagepullerAuthConfig(doc initdata.Raw) error {
	configLocation := "/run/insecure-cfg/imagepuller.toml"
	return os.WriteFile(configLocation, doc, 0o644)
}

// checkDeviceAvailability tries to detect a virtio device by its id, falling back to scanning /dev.
//
// An initdata (or imagepuller) device must
//   - be a block device
//   - belong to a range of known major device numbers
//   - start with the given magic
//
// This behaviour is similar to what's being implemented at
// https://github.com/kata-containers/kata-containers/pull/11655.
func checkDeviceAvailability(id string, magic []byte) (string, error) {
	candidates := []string{fmt.Sprintf("/dev/disk/by-id/virtio-%s", id)}

	devices, err := os.ReadDir("/dev")
	if err != nil {
		return "", fmt.Errorf("listing /dev: %w", err)
	}
	for _, device := range devices {
		candidates = append(candidates, filepath.Join("/dev", device.Name()))
	}

	for _, device := range candidates {
		info, err := os.Stat(device)
		if err != nil {
			log.Printf("Could not stat device candidate %q: %v", device, err)
			continue
		}

		if info.Mode()&(fs.ModeDevice|fs.ModeCharDevice) != fs.ModeDevice {
			// not a block device
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return "", fmt.Errorf("unexpected type of device stat_t for %q: %T", device, info.Sys())
		}

		// Check whether the device major number indicates a disk type.
		// https://www.kernel.org/doc/html/latest/admin-guide/devices.html
		major := unix.Major(stat.Rdev)
		switch {
		case major == 8:
			// SCSI / SATA
		case major == 3:
			// IDE / PATA
		case 240 <= major && major <= 254:
			// Dynamic, probably virtio.
		default:
			log.Printf("Skipping device candidate %q, unknown major device number %d", device, major)
			continue
		}

		head := make([]byte, len(magic))
		f, err := os.Open(device)
		if err != nil {
			log.Printf("Could not open device candidate %q: %v", device, err)
			continue
		}
		if _, err := io.ReadFull(f, head); err != nil {
			f.Close()
			log.Printf("Reading device magic of candidate %q: %v", device, err)
			continue
		}
		f.Close()
		if bytes.Equal(magic, head) {
			return device, nil
		}
	}

	return "", fmt.Errorf("no suitable device found")
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
