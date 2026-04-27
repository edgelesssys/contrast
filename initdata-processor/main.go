// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/initdata-processor/policy"
	"github.com/edgelesssys/contrast/initdata-processor/validator"
	"github.com/edgelesssys/contrast/internal/attestation/insecure"
	"github.com/edgelesssys/contrast/internal/initdata"
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

	var hostdata []byte
	var insecurePlatform bool

	// Handle initdata.
	if err := os.MkdirAll(measuredConfigPath, 0o755); err != nil {
		failf("Could not create directory %q: %v", measuredConfigPath, err)
		return
	}
	device, err := checkDeviceAvailability("initdata")
	if err != nil {
		failf("no initdata device found")
		return
	}
	doc, err := initdata.FromDevice(device, "initdata")
	if err != nil {
		failf("%s is not an initdata device: %v", device, err)
		return
	}
	hostdata, insecurePlatform, err = handleInitdata(doc)
	if err != nil {
		failf("handling initdata: %v", err)
		return
	}
	log.Printf("Processed initdata from %q ", device)

	// Handle imagepuller auth config.
	if err := os.MkdirAll(insecureConfigPath, 0o755); err != nil {
		failf("Could not create directory %q: %v", insecureConfigPath, err)
		return
	}
	device, err = checkDeviceAvailability("imagepuller")
	if err != nil {
		log.Println("No imagepuller auth config found, only unauthenticated pulls will be available")
	} else {
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

	// Signal systemd that initdata processing is complete.
	sdNotifyReady()

	// On insecure platforms, serve the hostdata digest via HTTP so that
	// the insecure aTLS issuer (running inside containers) can fetch it.
	if insecurePlatform {
		log.Printf("Starting insecure hostdata server on %s", insecure.HostdataAddr)
		if err := serveHostdata(hostdata); err != nil {
			log.Printf("Hostdata server error: %v", err)
		}
	}
}

func handleInitdata(doc initdata.Raw) (hostdata []byte, insecurePlatform bool, retErr error) {
	digest, err := doc.Digest()
	if err != nil {
		return nil, false, fmt.Errorf("computing initdata digest: %w", err)
	}

	v, verr := validator.New()
	if errors.Is(verr, validator.ErrNoPlatform) {
		log.Print("WARNING: No TEE platform detected, skipping initdata digest validation. This is expected on insecure platforms.")
		insecurePlatform = true
	} else if verr != nil {
		return nil, false, fmt.Errorf("creating validator: %w", verr)
	} else if err := v.ValidateDigest(digest); err != nil {
		return nil, false, fmt.Errorf("validating initdata digest: %w", err)
	}

	data, err := doc.Parse()
	if err != nil {
		return nil, false, fmt.Errorf("parsing initdata: %w", err)
	}
	for name, content := range data.Data {
		name = filepath.Clean(name)
		path := filepath.Join(measuredConfigPath, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, false, fmt.Errorf("writing file %q: %w", path, err)
		}
	}
	return digest, insecurePlatform, nil
}

// serveHostdata starts an HTTP server that serves the hostdata digest.
func serveHostdata(hostdata []byte) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hostdata", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		if _, err := w.Write(hostdata); err != nil {
			log.Printf("hostdata write error: %v", err)
		}
	})
	return http.ListenAndServe(insecure.HostdataAddr, mux)
}

// sdNotifyReady signals systemd that the service is ready.
func sdNotifyReady() {
	addr := os.Getenv("NOTIFY_SOCKET")
	if addr == "" {
		return
	}
	conn, err := (&net.Dialer{}).DialContext(context.Background(), "unixgram", addr)
	if err != nil {
		log.Printf("sd_notify: dial: %v", err)
		return
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("READY=1")); err != nil {
		log.Printf("sd_notify: write: %v", err)
	}
}

func handleImagepullerAuthConfig(doc initdata.Raw) error {
	configLocation := "/run/insecure-cfg/imagepuller.toml"
	return os.WriteFile(configLocation, doc, 0o644)
}

func checkDeviceAvailability(id string) (string, error) {
	device := fmt.Sprintf("/dev/disk/by-id/virtio-%s", id)
	info, err := os.Stat(device)
	if err != nil {
		return "", fmt.Errorf("could not open %s: %w", device, err)
	}

	if info.Mode()&(fs.ModeDevice|fs.ModeCharDevice) != fs.ModeDevice {
		return "", fmt.Errorf("%s is not a block device (mode: %s)", device, info.Mode())
	}

	return device, nil
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
