// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func TestSplitYAML(t *testing.T) {
	testCases := map[string]struct {
		resources string
		wantSplit []string
		wantErr   bool
	}{
		"empty": {
			resources: "",
			wantSplit: []string{},
		},
		"single resource": {
			resources: `apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
			wantSplit: []string{
				`apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
			},
		},
		"single resource with doc separator": {
			resources: `
---
apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
			wantSplit: []string{
				`apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
			},
		},
		"2 documents": {
			resources: `---
apiVersion: v1
kind: Namespace
metadata:
    name: test1
---
apiVersion: v1
kind: Namespace
metadata:
    name: test2
`,
			wantSplit: []string{
				`apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
				`apiVersion: v1
kind: Namespace
metadata:
    name: test2
`,
			},
		},
		"3 documents": {
			resources: `---
apiVersion: v1
kind: Namespace
metadata:
    name: test1
---
apiVersion: v1
kind: Namespace
metadata:
    name: test2
---
apiVersion: v1
kind: Namespace
metadata:
    name: test3
`,
			wantSplit: []string{
				`apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
				`apiVersion: v1
kind: Namespace
metadata:
    name: test2
`,
				`apiVersion: v1
kind: Namespace
metadata:
    name: test3
`,
			},
		},
		"2 document invalid": {
			resources: `---
apiVersion: v1
kind: Namespace
metadata:
	name: test1
---
apiVersion
`,
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotSplit, err := splitYAML([]byte(tc.resources))

			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tc.wantSplit, bytesToStrings(gotSplit))
		})
	}
}

func TestEncodeUnstructured(t *testing.T) {
	for name, tc := range map[string]struct {
		input  []*unstructured.Unstructured
		output string
	}{
		"quotation marks are added where needed": {
			input:  []*unstructured.Unstructured{{Object: map[string]any{"foo": "1e2"}}},
			output: "foo: \"1e2\"\n",
		},
		"multi-line strings use literal-style ": {
			input:  []*unstructured.Unstructured{{Object: map[string]any{"foo": "1\n2\n3\n"}}},
			output: "foo: |\n  1\n  2\n  3\n",
		},
		"multiple objects are --- separated": {
			input: []*unstructured.Unstructured{
				{Object: map[string]any{"foo": "1"}},
				{Object: map[string]any{"bar": 2}},
			},
			output: "foo: \"1\"\n---\nbar: 2\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			output, err := EncodeUnstructured(tc.input)
			require.NoError(err)
			require.Equal(tc.output, string(output))
		})
	}
}

func bytesToStrings(b [][]byte) []string {
	s := make([]string, len(b))
	for i, bb := range b {
		s[i] = string(bb)
	}
	return s
}
