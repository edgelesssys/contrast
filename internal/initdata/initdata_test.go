// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package initdata

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTemplate(algorithm, version string, data bool) Raw {
	builder := bytes.Buffer{}

	if algorithm != "" {
		fmt.Fprintf(&builder, "algorithm = \"%s\"\n", algorithm)
	}
	if version != "" {
		fmt.Fprintf(&builder, "version = \"%s\"\n", version)
	}

	if data {
		builder.WriteString(`
[data]
somekey = "somevalue"
"imagepuller.toml" = '''
socket = 'unix:///run/guest-services/imagepuller.sock'
credentials = []
'''
`)
	}
	return builder.Bytes()
}

func TestParse(t *testing.T) {
	tests := map[string]struct {
		version   string
		algorithm string
		data      bool
		wantErr   error
	}{
		"success sha256":       {version: InitdataVersion, algorithm: "sha256", data: true},
		"success sha-256":      {version: InitdataVersion, algorithm: "sha256", data: true},
		"success sha384":       {version: InitdataVersion, algorithm: "sha384", data: true},
		"success sha-384":      {version: InitdataVersion, algorithm: "sha384", data: true},
		"success sha512":       {version: InitdataVersion, algorithm: "sha512", data: true},
		"success sha-512":      {version: InitdataVersion, algorithm: "sha512", data: true},
		"version missing":      {version: "", wantErr: errVersionMismatch},
		"version unknown":      {version: "1.1.0", wantErr: errVersionMismatch},
		"algorithm missing":    {version: InitdataVersion, wantErr: errAlgorithmUnknown},
		"algorithm unknown":    {version: InitdataVersion, algorithm: "sha123", wantErr: errAlgorithmUnknown},
		"success data missing": {version: InitdataVersion, algorithm: "sha256", data: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			raw := buildTemplate(tc.algorithm, tc.version, tc.data)
			i, err := raw.Parse()
			if tc.wantErr != nil {
				require.ErrorIs(err, tc.wantErr)
				return
			}

			require.NoError(err)
			assert.Equal(tc.algorithm, i.Algorithm)
			assert.Equal(InitdataVersion, i.Version)

			if tc.data {
				assert.Contains(i.Data, "somekey")
				assert.Contains(i.Data, "imagepuller.toml")
			}
		})
	}
}

func TestEncode(t *testing.T) {
	tests := map[string]struct {
		version   string
		algorithm string
		data      map[string]string
		wantErr   error
	}{
		"success sha256": {
			version:   InitdataVersion,
			algorithm: "sha256",
			data:      map[string]string{"key": "value"},
		},
		"success sha384": {
			version:   InitdataVersion,
			algorithm: "sha384",
			data:      map[string]string{"key": "value"},
		},
		"success sha512": {
			version:   InitdataVersion,
			algorithm: "sha512",
			data:      map[string]string{"key": "value"},
		},
		"success complex keys": {
			version:   InitdataVersion,
			algorithm: "sha256",
			data:      map[string]string{"key": "value", "imagepuller.toml": "something"},
		},
		"version missing": {
			wantErr: errVersionMismatch,
		},
		"version unsupported": {
			version: "1.1.0",
			wantErr: errVersionMismatch,
		},
		"algorithm not sanitized": {
			version:   InitdataVersion,
			algorithm: "sha-256",
			wantErr:   errAlgorithmUnknown,
		},
		"algorithm missing": {
			version: InitdataVersion,
			wantErr: errAlgorithmUnknown,
		},
		"algorithm unknown": {
			version:   InitdataVersion,
			algorithm: "sha123",
			wantErr:   errAlgorithmUnknown,
		},
		"success data missing": {
			version:   InitdataVersion,
			algorithm: "sha256",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			i := Initdata{
				Version:   tc.version,
				Algorithm: tc.algorithm,
				Data:      tc.data,
			}

			toml, err := i.Encode()
			if tc.wantErr != nil {
				require.ErrorIs(err, tc.wantErr)
				return
			}
			require.NoError(err)
			tomlString := string(toml)

			assert.Contains(tomlString, fmt.Sprintf("version = '%s'", tc.version))
			assert.Contains(tomlString, fmt.Sprintf("algorithm = '%s'", tc.algorithm))
			for key, value := range tc.data {
				if strings.Contains(key, ".") {
					key = fmt.Sprintf("'%s'", key)
				}
				assert.Contains(tomlString, fmt.Sprintf("%s = '%s'", key, value))
			}

			parsed, err := toml.Parse()
			require.NoError(err)
			assert.Equal(&i, parsed)

			// Round-trip to and from Kata annotation should not change the raw doc.
			anno, err := toml.EncodeKataAnnotation()
			require.NoError(err)
			toml2, err := DecodeKataAnnotation(anno)
			require.NoError(err)
			assert.Equal(toml, toml2)
		})
	}
}

func TestFromDevice(t *testing.T) {
	tmpDir := t.TempDir()

	expectedInitdata, err := New("sha256", map[string]string{"foo": "bar"})
	require.NoError(t, err)

	marshalled, err := toml.Marshal(expectedInitdata)
	require.NoError(t, err)
	expectedDigest := sha256.Sum256(marshalled)

	realContent := prepareInitdataImage(t, marshalled)

	for name, tc := range map[string]struct {
		deviceContent []byte
		wantErr       error
	}{
		"good": {
			deviceContent: realContent,
		},
		"less-padding": {
			deviceContent: realContent[:480],
		},
		"bad-magic": {
			deviceContent: []byte("LUKS\xba\xbefoobar"),
			wantErr:       errWrongMagic,
		},
		"short-magic": {
			deviceContent: []byte("in"),
			wantErr:       io.EOF,
		},
		"short-size": {
			deviceContent: []byte("initdata\x01\x00"),
			wantErr:       io.EOF,
		},
		"humonguous-allocation": {
			deviceContent: []byte("initdata\x00\x00\x00\x00\x00\x00\x00\x10"),
			wantErr:       errTooLarge,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			path := filepath.Join(tmpDir, name)
			require.NoError(os.WriteFile(path, tc.deviceContent, 0o755))

			raw, err := FromDevice(path)
			if tc.wantErr != nil {
				require.ErrorIs(err, tc.wantErr)
				return
			}
			require.NoError(err)

			initdata, err := raw.Parse()
			require.NoError(err)
			digest, err := raw.Digest()
			require.NoError(err)

			require.Equal(expectedInitdata, initdata)
			require.Equal(expectedDigest[:], digest)
		})
	}
}

// prepareInitdataImage is a variant of the eponymous function in Kata, simplified for tests.
//
// https://github.com/kata-containers/kata-containers/blob/077aaa6480953de3770b8c3e240dbb0dc44a186e/src/runtime/virtcontainers/hypervisor.go#L1222-L1281
func prepareInitdataImage(t *testing.T, initdata []byte) []byte {
	t.Helper()
	require := require.New(t)

	compressed, err := compress(initdata)
	require.NoError(err)

	padding := make([]byte, 512-((16+len(compressed))%512))
	buf := []byte("initdata")
	buf = binary.LittleEndian.AppendUint64(buf, uint64(len(compressed)))
	buf = append(buf, compressed...)
	buf = append(buf, padding...)
	require.Len(buf, 512)
	return buf
}
