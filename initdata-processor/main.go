package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/initdata"
)

func main() {
	entries, err := os.ReadDir("/dev")
	if err != nil {
		log.Fatalf("Listing devices: %v", err)
	}

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
		if err := handleInitdata(doc); err != nil {
			log.Fatalf("Handling initdata: %v", err)
		}
		log.Printf("Processed initdata from %q ", path)
	}
}

func handleInitdata(data *initdata.Initdata) error {
	// TODO(burgerdev): validate!

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
