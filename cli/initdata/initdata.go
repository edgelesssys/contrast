// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package initdata

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2"
)

const (
	// InitdataVersion is the sole supported version of Initdata.
	InitdataVersion = "0.1.0"

	// InitdataAnnotationKey as specified in: https://github.com/kata-containers/kata-containers/blob/f6ff9cf7176989d414bf3f45a5b0c0b9fdb1bf3a/src/libs/kata-types/src/annotations/mod.rs#L276
	InitdataAnnotationKey = "io.katacontainers.config.hypervisor.cc_init_data"
)

// Initdata follows the standardized structure format as described here:
// https://github.com/confidential-containers/trustee/blob/d9c0b6fa01a042052ba93c703b1ff6aab2a2b63f/kbs/docs/initdata.md?plain=1#L82-L88.
type Initdata struct {
	version   string
	algorithm string
	data      map[string]string
}

// New creates and parses a new Initdata struct from the given base64-encoded, gzipped TOML.
func New(raw string) (*Initdata, error) {
	var i Initdata

	if err := i.FromString(raw); err != nil {
		return nil, fmt.Errorf("parsing initdata from string: %w", err)
	}

	if err := i.Validate(); err != nil {
		return nil, fmt.Errorf("validating initdata: %w", err)
	}

	return &i, nil
}

// Validate the parsed initdata.
func (i *Initdata) Validate() error {
	if i.version != InitdataVersion {
		return fmt.Errorf(
			"specified initdata version is %s, but supported is only version %s",
			i.version,
			InitdataVersion,
		)
	}

	if _, err := i.Digest([]byte{}); err != nil {
		return err
	}

	return nil
}

// Digest calculates the hash of the Initdata.
func (i *Initdata) Digest(digestible []byte) (string, error) {
	var digest string
	switch i.algorithm {
	case "sha256":
		bytes := sha256.Sum256(digestible)
		digest = string(bytes[:])
	case "sha384":
		bytes := sha512.Sum384(digestible)
		digest = string(bytes[:])
	case "sha512":
		bytes := sha512.Sum512(digestible)
		digest = string(bytes[:])
	default:
		return "", fmt.Errorf(
			"algorithm '%s' is not supported. Supported are only sha256, sha384, sha512",
			i.algorithm,
		)
	}

	return digest, nil
}

// Get returns the value associated with the given key, or an error.
// A "" zero value response with nil error indicates that the field had explicitly been set to "".
func (i *Initdata) Get(key string) (string, error) {
	value, present := i.data[key]
	if !present {
		return "", fmt.Errorf("data entry for key '%s' is empty", key)
	}
	return value, nil
}

// Set sets the given key to the given value, and returns the old value.
// If the key had not previously been set, the "" zero value is returned.
func (i *Initdata) Set(key, value string) string {
	old := i.data[key]
	i.data[key] = value
	return old
}

// Delete removes the given key and its value from the initdata.
// If the key had not previously been set, the "" zero value is returned.
func (i *Initdata) Delete(key string) string {
	old := i.data[key]
	delete(i.data, key)
	return old
}

// FromString parses Initdata from a base64-encoded, gzipped TOML string.
func (i *Initdata) FromString(raw string) error {
	decoded, err := i.Decode([]byte(raw))
	if err != nil {
		return err
	}

	decompressed, err := i.Decompress(decoded)
	if err != nil {
		return err
	}

	if err := i.Unmarshal(decompressed); err != nil {
		return err
	}

	return nil
}

// ToStringAndDigest marshals Initdata into TOML, gzips and base64-encodes it.
// It returns both the encoded data and its digest.
func (i *Initdata) ToStringAndDigest() (string, string, error) {
	marshalled, err := i.Marshal()
	if err != nil {
		return "", "", err
	}

	compressed, err := i.Compress(marshalled)
	if err != nil {
		return "", "", err
	}

	encoded, err := i.Encode(compressed)
	if err != nil {
		return "", "", err
	}

	digest, err := i.Digest(encoded)
	if err != nil {
		return "", "", err
	}

	return string(encoded), digest, nil
}

// Marshal marshals an Initdata struct to TOML.
func (i *Initdata) Marshal() ([]byte, error) {
	toml, err := toml.Marshal(i)
	if err != nil {
		return []byte{}, err
	}
	return toml, nil
}

// Unmarshal attempts to unmarshal TOML into an Initdata struct.
func (i *Initdata) Unmarshal(raw []byte) error {
	err := toml.Unmarshal(raw, &i)
	return err
}

// Decompress decompresses gzipped bytes.
func (i *Initdata) Decompress(compressed []byte) ([]byte, error) {
	gzreader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return []byte{}, err
	}

	decompressed, err := io.ReadAll(gzreader)
	if err != nil {
		return []byte{}, err
	}

	return decompressed, nil
}

// Compress gzips bytes.
func (i *Initdata) Compress(uncompressed []byte) ([]byte, error) {
	var writer bytes.Buffer
	gzwriter := gzip.NewWriter(&writer)
	if _, err := gzwriter.Write(uncompressed); err != nil {
		return []byte{}, err
	}
	if err := gzwriter.Close(); err != nil {
		return []byte{}, err
	}

	var compressed []byte
	if _, err := writer.Write(compressed); err != nil {
		return []byte{}, err
	}

	return compressed, nil
}

// Decode decodes base64-encoded bytes.
func (i *Initdata) Decode(encoded []byte) ([]byte, error) {
	var decoded []byte
	if _, err := base64.StdEncoding.Decode(decoded, encoded); err != nil {
		return []byte{}, err
	}

	return decoded, nil
}

// Encode encodes bytes to base64.
func (i *Initdata) Encode(decoded []byte) ([]byte, error) {
	var encoded []byte
	base64.StdEncoding.Encode(encoded, decoded)
	return encoded, nil
}
