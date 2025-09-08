// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package initdata

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTemplate(algorithm, version string, data bool) string {
	var builder strings.Builder

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
	return builder.String()
}

func tomlToAnnotation(t *testing.T, toml string) string {
	t.Helper()
	compressed, err := compress([]byte(toml))
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(compressed)
	return encoded
}

func annotationToToml(t *testing.T, annotation string) string {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(annotation)
	require.NoError(t, err)
	decompressed, err := decompress(decoded)
	require.NoError(t, err)
	return string(decompressed)
}

func TestDecodeKataAnnotation(t *testing.T) {
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

			annotation := tomlToAnnotation(t, buildTemplate(tc.algorithm, tc.version, tc.data))
			i, err := DecodeKataAnnotation(annotation)
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

func TestEncodeKataAnnotation(t *testing.T) {
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

			annotation, _, err := i.EncodeKataAnnotation()
			if tc.wantErr != nil {
				require.ErrorIs(err, tc.wantErr)
				return
			}

			require.NoError(err)
			tomlString := annotationToToml(t, annotation)
			fmt.Println(tomlString)

			assert.Contains(tomlString, fmt.Sprintf("version = '%s'", tc.version))
			assert.Contains(tomlString, fmt.Sprintf("algorithm = '%s'", tc.algorithm))
			for key, value := range tc.data {
				if strings.Contains(key, ".") {
					key = fmt.Sprintf("'%s'", key)
				}
				assert.Contains(tomlString, fmt.Sprintf("%s = '%s'", key, value))
			}

			decoded, err := DecodeKataAnnotation(annotation)
			require.NoError(err)
			assert.Equal(&i, decoded)
		})
	}
}
