// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cryptsetup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Device struct {
	devicePath string
	headerPath string
	keyPath    string
}

func NewDevice(devicePath, headerPath, keyPath string) *Device {
	return &Device{
		devicePath: devicePath,
		headerPath: headerPath,
		keyPath:    keyPath,
	}
}

// IsLuks wraps the cryptsetup isLuks command and returns a bool reflecting if the device is formatted as LUKS.
func (d *Device) IsLuks(ctx context.Context) (bool, error) {
	var exitErr *exec.ExitError
	out, err := runCryptsetupCmd(ctx, "isLuks", "--verbose", d.devicePath)
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// isLuks exits 1 if the device is not a LUKS device.
		return false, nil
	} else if errors.As(err, &exitErr) {
		return false, fmt.Errorf("checking if device %s is LUKS encrypted: %w, stdout: %s, stderr: %s", d.devicePath, err, out, exitErr.Stderr)
	} else if err != nil {
		return false, fmt.Errorf("checking if device %s is LUKS encrypted: %w", d.devicePath, err)
	}
	return true, nil
}

// Format wraps the luksFormat command.
func (d *Device) Format(ctx context.Context) error {
	if err := ensureBase(d.headerPath); err != nil {
		return err
	}
	args := []string{
		"luksFormat",
		"--type=luks2",                          // Use LUKS2 header format.
		"--cipher=aes-xts-plain64",              // Use AES-XTS cipher.
		"--pbkdf=argon2id",                      // Use Argon2id as the key derivation function.
		"--pbkdf-memory=10240",                  // Memory usage for Argon2i, limit to 10 MiB so it won't fail in low-memory pods.
		"--batch-mode",                          // Suppresses all confirmation questions.
		fmt.Sprintf("--key-file=%s", d.keyPath), // Path to the key file.
		d.devicePath,
	}
	_, err := runCryptsetupCmd(ctx, args...)
	return err
}

// Open wraps the luksOpen command to open a LUKS device with a detached header.
func (d *Device) Open(ctx context.Context, mappingName string) error {
	if err := d.headerBackup(ctx); err != nil {
		return fmt.Errorf("backing up LUKS header from device %s to %s: %w", d.devicePath, d.headerPath, err)
	}
	header, err := d.readHeader(ctx)
	if err != nil {
		return fmt.Errorf("reading LUKS header from %s: %w", d.headerPath, err)
	}
	if err := d.verifyHeader(header); err != nil {
		return fmt.Errorf("verifying LUKS header from %s: %w", d.headerPath, err)
	}

	args := []string{
		"luksOpen",
		fmt.Sprintf("--header=%s", d.headerPath), // Use the detached header.
		fmt.Sprintf("--key-file=%s", d.keyPath),  // Path to the key file.
		d.devicePath,                             // The device to open.
		mappingName,                              // The name for the mapping.
	}
	if _, err = runCryptsetupCmd(ctx, args...); err != nil {
		return fmt.Errorf("opening LUKS device %s with mapping name %s: %w", d.devicePath, mappingName, err)
	}
	return nil
}

func (d *Device) headerBackup(ctx context.Context) error {
	if err := ensureBase(d.headerPath); err != nil {
		return err
	}
	_, err := runCryptsetupCmd(ctx, "luksHeaderBackup", d.devicePath, "--header-backup-file", d.headerPath)
	return err
}

func (d *Device) readHeader(ctx context.Context) (cryptsetupMetadata, error) {
	args := []string{
		"luksDump",
		fmt.Sprintf("--header=%s", d.headerPath), // Read the detached header.
		"--dump-json-metadata",                   // Get metadata as JSON.
		"/dev/null",                              // The device isn't used but required as parameter.
	}
	output, err := runCryptsetupCmd(ctx, args...)
	if err != nil {
		return cryptsetupMetadata{}, fmt.Errorf("dumping LUKS header from %s: %w", d.headerPath, err)
	}
	var metadata cryptsetupMetadata
	decoder := json.NewDecoder(strings.NewReader(output))
	decoder.DisallowUnknownFields() // Ensure no unknown fields are present in the JSON.
	if err := decoder.Decode(&metadata); err != nil {
		return cryptsetupMetadata{}, fmt.Errorf("decoding LUKS header JSON from %s: %w", d.headerPath, err)
	}
	return metadata, nil
}

func (d *Device) verifyHeader(header cryptsetupMetadata) error {
	if len(header.KeySlots) != 1 {
		return fmt.Errorf("expected exactly one keyslot, got %d", len(header.KeySlots))
	}
	key, ok := header.KeySlots["0"]
	if !ok {
		return fmt.Errorf("keyslot '0' not found in header")
	}
	if key.Type != "luks2" {
		return fmt.Errorf("expected keyslot type 'luks2', got '%s'", key.Type)
	}
	if key.KeySize != 64 {
		return fmt.Errorf("expected key size 64, got %d", key.KeySize)
	}
	if key.Area.Type != "raw" {
		return fmt.Errorf("expected area type 'raw', got '%s'", key.Area.Type)
	}
	if key.Area.Encryption != "aes-xts-plain64" {
		return fmt.Errorf("expected area encryption 'aes-xts-plain64', got '%s'", key.Area.Encryption)
	}
	if key.Area.KeySize != 64 {
		return fmt.Errorf("expected area key size 64, got %d", key.Area.KeySize)
	}
	if key.AntiForensicSplitter.Type != "luks1" {
		return fmt.Errorf("expected anti-forensic splitter type 'luks1', got '%s'", key.AntiForensicSplitter.Type)
	}
	if key.AntiForensicSplitter.Stripes != 4000 {
		return fmt.Errorf("expected anti-forensic splitter stripes 4000, got %d", key.AntiForensicSplitter.Stripes)
	}
	if key.AntiForensicSplitter.Hash != "sha256" {
		return fmt.Errorf("expected anti-forensic splitter hash 'sha256', got '%s'", key.AntiForensicSplitter.Hash)
	}
	if key.KDF.Type != "argon2id" {
		return fmt.Errorf("expected KDF type 'argon2id', got '%s'", key.KDF.Type)
	}
	if key.KDF.Salt == "" {
		return fmt.Errorf("expected KDF salt to be non-empty")
	}
	if len(header.Segments) != 1 {
		return fmt.Errorf("expected exactly one segment, got %d", len(header.Segments))
	}
	segment, ok := header.Segments["0"]
	if !ok {
		return fmt.Errorf("segment '0' not found in header")
	}
	if segment.Type != "crypt" {
		return fmt.Errorf("expected segment type 'crypt', got '%s'", segment.Type)
	}
	if len(segment.Flags) != 0 {
		return fmt.Errorf("expected no segment flags, got %d", len(segment.Flags))
	}
	if segment.IVTweak != "0" {
		return fmt.Errorf("expected segment IV tweak '0', got '%s'", segment.IVTweak)
	}
	if segment.Encryption != "aes-xts-plain64" {
		return fmt.Errorf("expected segment encryption 'aes-xts-plain64', got '%s'", segment.Encryption)
	}
	if len(header.Digests) != 1 {
		return fmt.Errorf("expected exactly one digest, got %d", len(header.Digests))
	}
	digest, ok := header.Digests["0"]
	if !ok {
		return fmt.Errorf("digest '0' not found in header")
	}
	if digest.Type != "pbkdf2" {
		return fmt.Errorf("expected digest type 'pbkdf2', got '%s'", digest.Type)
	}
	if len(digest.Keyslots) != 1 || digest.Keyslots[0] != "0" {
		return fmt.Errorf("expected digest to reference keyslot '0', got %s", digest.Keyslots)
	}
	if len(digest.Segments) != 1 || digest.Segments[0] != "0" {
		return fmt.Errorf("expected digest to reference segment '0', got %s", digest.Segments)
	}
	if digest.Hash != "sha256" {
		return fmt.Errorf("expected digest hash 'sha256', got '%s'", digest.Hash)
	}
	if digest.Salt == "" {
		return fmt.Errorf("expected digest salt to be non-empty")
	}
	if digest.Digest == "" {
		return fmt.Errorf("expected digest to be non-empty")
	}
	if len(header.Tokens) != 0 {
		return fmt.Errorf("expected no tokens, got %d", len(header.Tokens))
	}
	return nil
}

type cryptsetupMetadata struct {
	KeySlots map[string]struct {
		Type                 string `json:"type"`
		KeySize              int    `json:"key_size"`
		AntiForensicSplitter struct {
			Type    string `json:"type"`
			Stripes int    `json:"stripes"`
			Hash    string `json:"hash"`
		} `json:"af"`
		Area struct {
			Type       string `json:"type"`
			Offset     string `json:"offset"`
			Size       string `json:"size"`
			Encryption string `json:"encryption"`
			KeySize    int    `json:"key_size"`
		} `json:"area"`
		KDF struct {
			Type   string `json:"type"`
			Time   int    `json:"time"`
			Memory int    `json:"memory"`
			CPUs   int    `json:"cpus"`
			Salt   string `json:"salt"`
		} `json:"kdf"`
	} `json:"keyslots"`
	Tokens   map[string]struct{} `json:"tokens"`
	Segments map[string]struct {
		Type       string   `json:"type"`
		Offset     string   `json:"offset"`
		Size       string   `json:"size"`
		Flags      []string `json:"flags,omitempty"`
		IVTweak    string   `json:"iv_tweak"`
		Encryption string   `json:"encryption"`
		SectorSize int      `json:"sector_size"`
	} `json:"segments"`
	Digests map[string]struct {
		Type       string   `json:"type"`
		Keyslots   []string `json:"keyslots"`
		Segments   []string `json:"segments"`
		Hash       string   `json:"hash"`
		Iterations int      `json:"iterations"`
		Salt       string   `json:"salt"`
		Digest     string   `json:"digest"`
	} `json:"digests"`
	Config struct {
		JSONSize     string `json:"json_size"`
		KeyslotsSize string `json:"keyslots_size"`
	}
}

func runCryptsetupCmd(ctx context.Context, args ...string) (string, error) {
	if err := os.MkdirAll("/run/cryptsetup", 0o755); err != nil {
		return "", fmt.Errorf("creating directory /run/cryptsetup, which is required for locking: %w", err)
	}
	cmd := exec.CommandContext(ctx, "cryptsetup", args...)
	output, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "", fmt.Errorf("executing '%s': %w, stderr: %s", cmd.String(), err, exitErr.Stderr)
	} else if err != nil {
		return "", fmt.Errorf("executing '%s': %w", cmd.String(), err)
	}
	return string(output), nil
}

func ensureBase(p string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("ensuring base directory %s: %w", filepath.Dir(p), err)
	}
	return nil
}
