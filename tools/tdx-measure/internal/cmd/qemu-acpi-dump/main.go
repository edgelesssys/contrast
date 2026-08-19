// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("qemu-acpi-dump", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	outputDir := flags.String("output", "", "directory to write the ACPI blobs into")
	qemuBinary := flags.String("qemu", "", "QEMU system binary")
	metadataJSON := flags.String("metadata-json", "{}", "JSON object to merge into manifest.json")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *outputDir == "" {
		return errors.New("--output is required")
	}
	if *qemuBinary == "" {
		return errors.New("--qemu is required")
	}

	metadata := map[string]any{}
	if err := json.Unmarshal([]byte(*metadataJSON), &metadata); err != nil {
		return fmt.Errorf("parsing --metadata-json: %w", err)
	}
	if metadata == nil {
		return errors.New("--metadata-json must be a JSON object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := dump(ctx, dumpConfig{
		OutputDir:  *outputDir,
		QEMUBinary: *qemuBinary,
		QEMUArgs:   flags.Args(),
	}); err != nil {
		return err
	}
	version, err := qemuVersion(ctx, *qemuBinary)
	if err != nil {
		return err
	}

	metadata["qemuVersion"] = version
	manifest, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	manifest = append(manifest, '\n')
	if err := os.WriteFile(filepath.Join(*outputDir, "manifest.json"), manifest, 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return nil
}
