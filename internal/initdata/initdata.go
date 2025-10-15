// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package initdata

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	// InitdataVersion is the sole supported version of Initdata.
	InitdataVersion = "0.1.0"

	// InitdataAnnotationKey as specified in: https://github.com/kata-containers/kata-containers/blob/f6ff9cf7176989d414bf3f45a5b0c0b9fdb1bf3a/src/libs/kata-types/src/annotations/mod.rs#L276
	InitdataAnnotationKey = "io.katacontainers.config.hypervisor.cc_init_data"
)

var (
	errVersionMismatch  = errors.New("unknown version")
	errAlgorithmUnknown = errors.New("unknown algorithm")
	errWrongMagic       = errors.New("wrong magic number")
	errTooLarge         = errors.New("initdata is too large")
)

// Raw is an initdata document in its serialized TOML form.
type Raw []byte

// DecodeKataAnnotation decodes a Kata annotation into Raw.
func DecodeKataAnnotation(annotation string) (Raw, error) {
	decoded, err := base64.StdEncoding.DecodeString(annotation)
	if err != nil {
		return nil, err
	}

	return decompress(decoded)
}

// Digest returns the initdata hash using the algorithm specified in the document.
//
// In order to calculate the digest, the initdata document is parsed and validated.
func (r Raw) Digest() ([]byte, error) {
	i, err := r.Parse()
	if err != nil {
		return nil, err
	}

	newHash, ok := hashAlgorithms[i.Algorithm]
	if !ok {
		return nil, fmt.Errorf(
			"%w: algorithm %q is not supported. Supported are only sha256, sha384, sha512",
			errAlgorithmUnknown,
			i.Algorithm,
		)
	}
	hash := newHash()
	// hash.Hash.Write never returns an error.
	_, _ = hash.Write(r)
	return hash.Sum(nil), nil
}

// Parse the raw TOML document into an Initdata struct.
//
// This function canonicalizes the hash algorithm identifier and validates the resulting Initdata struct.
// A roundtrip through Parse() and Encode() may alter the digest!
func (r Raw) Parse() (*Initdata, error) {
	var i Initdata
	if err := toml.Unmarshal(r, &i); err != nil {
		return nil, fmt.Errorf("parsing TOML: %w", err)
	}

	// "shaXXX" and "sha-XXX" are used inconsistently upstream
	i.Algorithm = strings.Replace(i.Algorithm, "-", "", 1)

	if err := i.Validate(); err != nil {
		return nil, fmt.Errorf("validating parsed initdata: %w", err)
	}

	return &i, nil
}

// EncodeKataAnnotation encodes Initdata into a Kata annotation and the corresponding digest.
func (r Raw) EncodeKataAnnotation() (string, error) {
	compressed, err := compress(r)
	if err != nil {
		return "", fmt.Errorf("zipping initdata: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(compressed)
	return encoded, nil
}

// Initdata follows the standardized structure format as described here:
// https://github.com/confidential-containers/trustee/blob/d9c0b6fa01a042052ba93c703b1ff6aab2a2b63f/kbs/docs/initdata.md?plain=1#L82-L88.
type Initdata struct {
	Version   string            `toml:"version"`
	Algorithm string            `toml:"algorithm"`
	Data      map[string]string `toml:"data"`
}

// New creates a new Initdata struct and validates it.
func New(algorithm string, data map[string]string) (*Initdata, error) {
	i := Initdata{
		Version:   InitdataVersion,
		Algorithm: algorithm,
		Data:      data,
	}

	if err := i.Validate(); err != nil {
		return nil, fmt.Errorf("validating initdata: %w", err)
	}

	return &i, nil
}

// Encode the initdata into a raw TOML document.
func (i *Initdata) Encode() (Raw, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return toml.Marshal(i)
}

// Validate the parsed initdata.
func (i *Initdata) Validate() error {
	if i.Version != InitdataVersion {
		return fmt.Errorf(
			"%w: specified initdata version is %s, but supported is only version %s",
			errVersionMismatch,
			i.Version,
			InitdataVersion,
		)
	}

	if _, ok := hashAlgorithms[i.Algorithm]; !ok {
		return fmt.Errorf(
			"%w: algorithm %q is not supported. Supported algorithms: %v",
			errAlgorithmUnknown,
			i.Algorithm,
			hashAlgorithms,
		)
	}

	return nil
}

// FromDevice reads the raw TOML document from a block device prepared by the Kata runtime.
func FromDevice(devicePath, magic string) (Raw, error) {
	if len(magic) != 8 {
		return nil, fmt.Errorf("invalid magic id: must be exactly 8 bytes, but %q is %d bytes", magic, len(magic))
	}

	f, err := os.Open(devicePath)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 8)
	if _, err := f.ReadAt(buf, 0); err != nil {
		return nil, fmt.Errorf("reading magic number: %w", err)
	}
	if slices.Compare(buf, []byte(magic)) != 0 {
		return nil, fmt.Errorf("%w: expected %x, got %x", errWrongMagic, magic, buf)
	}
	buf = make([]byte, 8)
	if _, err := f.ReadAt(buf, 8); err != nil {
		return nil, fmt.Errorf("reading magic number: %w", err)
	}
	size := binary.LittleEndian.Uint64(buf)
	const maxSize = 128 * 1024 * 1024 * 1024
	if size > maxSize {
		return nil, fmt.Errorf("%w: expected at most 128MiB, got %d byte", errTooLarge, size)
	}
	buf = make([]byte, size)
	if _, err := f.ReadAt(buf, 16); err != nil {
		return nil, fmt.Errorf("reading initdata blob: %w", err)
	}

	return decompress(buf)
}

var hashAlgorithms = map[string]func() hash.Hash{
	"sha256": sha256.New,
	"sha384": sha512.New384,
	"sha512": sha512.New,
}

func decompress(compressed []byte) ([]byte, error) {
	gzreader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}

	decrompressed, err := io.ReadAll(gzreader)
	if err != nil {
		return nil, err
	}

	return decrompressed, nil
}

func compress(decompressed []byte) ([]byte, error) {
	var compressed bytes.Buffer
	gzwriter := gzip.NewWriter(&compressed)
	if _, err := gzwriter.Write(decompressed); err != nil {
		return nil, err
	}
	if err := gzwriter.Close(); err != nil {
		return nil, err
	}

	return compressed.Bytes(), nil
}
