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
	"encoding/hex"
	"errors"
	"fmt"
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

	// Attempt to calculate the Digest of an empty array.
	// Fails with an err if the Initdata was created with an unsupported algorithm.
	if _, err := i.digest([]byte{}); err != nil {
		return err
	}

	return nil
}

// DecodeKataAnnotation decodes a Kata annotation into Initdata.
func DecodeKataAnnotation(annotation string) (*Initdata, error) {
	decoded, err := base64.StdEncoding.DecodeString(annotation)
	if err != nil {
		return nil, err
	}

	i, _, err := parseCompressed(decoded)

	return i, err
}

// EncodeKataAnnotation encodes Initdata into a Kata annotation and the corresponding digest.
func (i *Initdata) EncodeKataAnnotation() (string, string, error) {
	// Since API users can change version/algorithm, check that they are valid.
	if err := i.Validate(); err != nil {
		return "", "", err
	}

	marshalled, err := toml.Marshal(i)
	if err != nil {
		return "", "", err
	}

	compressed, err := compress(marshalled)
	if err != nil {
		return "", "", err
	}

	encoded := base64.StdEncoding.EncodeToString(compressed)

	digest, err := i.digest(marshalled)
	if err != nil {
		return "", "", err
	}

	// TODO(burgerdev): why is the digest a hex string?
	return encoded, hex.EncodeToString(digest), nil
}

// FromDevice reads initdata and its digest from a block device prepared by the Kata runtime.
func FromDevice(devicePath string) (*Initdata, []byte, error) {
	f, err := os.Open(devicePath)
	if err != nil {
		return nil, nil, err
	}
	buf := make([]byte, 8)
	_, err = f.ReadAt(buf, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("reading magic number: %w", err)
	}
	const magic = "initdata"
	if slices.Compare(buf, []byte(magic)) != 0 {
		return nil, nil, fmt.Errorf("%w: expected %x, got %x", errWrongMagic, magic, buf)
	}
	buf = make([]byte, 8)
	_, err = f.ReadAt(buf, 8)
	if err != nil {
		return nil, nil, fmt.Errorf("reading magic number: %w", err)
	}
	size := binary.LittleEndian.Uint64(buf)
	const maxSize = 128 * 1024 * 1024 * 1024
	if size > maxSize {
		return nil, nil, fmt.Errorf("%w: expected at most 128MiB, got %d byte", errTooLarge, size)
	}
	buf = make([]byte, size)

	_, err = f.ReadAt(buf, 16)
	if err != nil {
		return nil, nil, fmt.Errorf("reading initdata blob: %w", err)
	}

	return parseCompressed(buf)
}

func (i *Initdata) digest(digestible []byte) ([]byte, error) {
	var digest []byte
	switch i.Algorithm {
	case "sha256":
		bytes := sha256.Sum256(digestible)
		digest = bytes[:]
	case "sha384":
		bytes := sha512.Sum384(digestible)
		digest = bytes[:]
	case "sha512":
		bytes := sha512.Sum512(digestible)
		digest = bytes[:]
	default:
		return nil, fmt.Errorf(
			"%w: algorithm %q is not supported. Supported are only sha256, sha384, sha512",
			errAlgorithmUnknown,
			i.Algorithm,
		)
	}

	return digest, nil
}

func parseCompressed(zipped []byte) (*Initdata, []byte, error) {
	decompressed, err := decompress(zipped)
	if err != nil {
		return nil, nil, fmt.Errorf("unzipping initdata: %w", err)
	}

	var i Initdata
	if err := toml.Unmarshal(decompressed, &i); err != nil {
		return nil, nil, fmt.Errorf("parsing TOML: %w", err)
	}

	// "shaXXX" and "sha-XXX" are used inconsistently upstream
	i.Algorithm = strings.Replace(i.Algorithm, "-", "", 1)

	if err := i.Validate(); err != nil {
		return nil, nil, fmt.Errorf("validating parsed initdata: %w", err)
	}

	digest, err := i.digest(decompressed)
	if err != nil {
		return nil, nil, fmt.Errorf("calculating digest: %w", err)
	}

	return &i, digest, nil
}

func decompress(compressed []byte) ([]byte, error) {
	gzreader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return []byte{}, err
	}

	readBytes, err := io.ReadAll(gzreader)
	if err != nil {
		return []byte{}, err
	}

	var decompressed bytes.Buffer
	decompressed.Write(readBytes)

	return decompressed.Bytes(), nil
}

func compress(decompressed []byte) ([]byte, error) {
	var compressed bytes.Buffer
	gzwriter := gzip.NewWriter(&compressed)
	if _, err := gzwriter.Write(decompressed); err != nil {
		return []byte{}, err
	}
	if err := gzwriter.Close(); err != nil {
		return []byte{}, err
	}

	return compressed.Bytes(), nil
}
