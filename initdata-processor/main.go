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

func main() {
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
		doc, digest, err := initdata.FromDevice(path)
		if err != nil {
			log.Printf("%s is not an initdata device: %v", path, err)
			continue
		}
		validator := validator.New()
		if err := validator.ValidateDigest(digest); err != nil {
			failf("Validating initdata digest: %v", err)
		}
		if err := handleInitdata(doc); err != nil {
			failf("Handling initdata: %v", err)
		}
		log.Printf("Processed initdata from %q ", path)
	}
}

func handleInitdata(data *initdata.Initdata) error {
	const targetPath = "/run/measured-cfg"
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return err
	}

	for name, content := range data.Data {
		// TODO(burgerdev): clean name
		path := filepath.Join(targetPath, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing file %q: %w", path, err)
		}
	}
	return nil
}

func failf(format string, v ...any) {
	log.Printf(format, v...)
	os.Exit(0)
}
