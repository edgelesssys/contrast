// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	testCases := []struct {
		name    string
		fixture string
		wantErr bool
	}{
		{
			name: "valid",
			fixture: `apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
  - image: image
    name: bar
`,
			wantErr: false,
		},
		{
			name: "unknown field",
			fixture: `apiVersion: v1
kind: Pod
unknown: field
metadata:
  name: foo
spec:
  containers:
  - image: image
    name: bar
`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			resources, err := UnmarshalApplyConfigurations([]byte(tc.fixture))
			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			got, err := EncodeResources(resources...)
			require.NoError(err)

			require.Equal(tc.fixture, string(got))
		})
	}
}

const (
	podYAML = `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
`
	combinedYAML = `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
---
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
`
)

func TestYAMLBytesFromFiles(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "single valid file",
			files: map[string]string{
				"file1.yaml": podYAML,
			},
			want: podYAML,
		},
		{
			name: "multiple valid files",
			files: map[string]string{
				"file1.yaml": podYAML,
				"file2.yaml": podYAML,
			},
			want: combinedYAML,
		},
		{
			name: "invalid file",
			files: map[string]string{
				"file1.yaml": `invalid-yaml`,
			},
			wantErr: true,
		},
		{
			name: "format multiline string",
			files: map[string]string{
				"file1.yaml": `apiVersion: v1
data:
  foo: "bar\nbaz\n"
kind: ConfigMap
`,
			},
			want: `apiVersion: v1
data:
  foo: |
    bar
    baz
kind: ConfigMap
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			tempDir := t.TempDir()

			var paths []string
			for fileName, content := range tc.files {
				path := filepath.Join(tempDir, fileName)
				require.NoError(os.WriteFile(path, []byte(content), 0o644))
				paths = append(paths, path)
			}

			got, err := YAMLBytesFromFiles(paths...)
			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			require.Equal(tc.want, string(got))
		})
	}
}
