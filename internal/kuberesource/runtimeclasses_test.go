// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNames(t *testing.T) {
	testCases := map[string]struct {
		p    PlatformCollection
		want []string
	}{
		"empty": {
			p:    PlatformCollection{},
			want: nil,
		},
		"single snp": {
			p:    PlatformCollection{platforms.MetalQEMUSNP: {}},
			want: []string{"Metal-QEMU-SNP"},
		},
		"single tdx": {
			p:    PlatformCollection{platforms.MetalQEMUTDX: {}},
			want: []string{"Metal-QEMU-TDX"},
		},
		"single snp-gpu": {
			p:    PlatformCollection{platforms.MetalQEMUSNPGPU: {}},
			want: []string{"Metal-QEMU-SNP-GPU"},
		},
		"single tdx-gpu": {
			p:    PlatformCollection{platforms.MetalQEMUTDXGPU: {}},
			want: []string{"Metal-QEMU-TDX-GPU"},
		},
		"multiple": {
			p: PlatformCollection{
				platforms.MetalQEMUSNP:    {},
				platforms.MetalQEMUTDX:    {},
				platforms.MetalQEMUSNPGPU: {},
				platforms.MetalQEMUTDXGPU: {},
			},
			want: []string{"Metal-QEMU-SNP", "Metal-QEMU-TDX", "Metal-QEMU-SNP-GPU", "Metal-QEMU-TDX-GPU"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.ElementsMatch(t, tc.want, tc.p.Names())
		})
	}
}

func TestAddFromString(t *testing.T) {
	testCases := map[string]struct {
		input   string
		want    []platforms.Platform
		wantErr bool
	}{
		"valid snp": {
			input: "contrast-cc-metal-qemu-snp",
			want:  []platforms.Platform{platforms.MetalQEMUSNP},
		},
		"valid tdx": {
			input: "contrast-cc-metal-qemu-tdx",
			want:  []platforms.Platform{platforms.MetalQEMUTDX},
		},
		"valid snp-gpu": {
			input: "contrast-cc-metal-qemu-snp-gpu",
			want:  []platforms.Platform{platforms.MetalQEMUSNPGPU},
		},
		"valid tdx-gpu": {
			input: "contrast-cc-metal-qemu-tdx-gpu",
			want:  []platforms.Platform{platforms.MetalQEMUTDXGPU},
		},
		"invalid": {
			input:   "invalid",
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			p := PlatformCollection{}
			err := p.AddFromString(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, p.Platforms())
		})
	}
}

func TestAddFromCommaSeparated(t *testing.T) {
	testCases := map[string]struct {
		input   string
		want    []platforms.Platform
		wantErr bool
	}{
		"single snp": {
			input: "metal-qemu-snp",
			want:  []platforms.Platform{platforms.MetalQEMUSNP},
		},
		"single tdx": {
			input: "metal-qemu-tdx",
			want:  []platforms.Platform{platforms.MetalQEMUTDX},
		},
		"single snp-gpu": {
			input: "metal-qemu-snp-gpu",
			want:  []platforms.Platform{platforms.MetalQEMUSNPGPU},
		},
		"single tdx-gpu": {
			input: "metal-qemu-tdx-gpu",
			want:  []platforms.Platform{platforms.MetalQEMUTDXGPU},
		},
		"multiple": {
			input: "metal-qemu-snp,metal-qemu-tdx,metal-qemu-snp-gpu,metal-qemu-tdx-gpu",
			want: []platforms.Platform{
				platforms.MetalQEMUSNP,
				platforms.MetalQEMUTDX,
				platforms.MetalQEMUSNPGPU,
				platforms.MetalQEMUTDXGPU,
			},
		},
		"invalid": {
			input:   "metal-qemu-snp,invalid",
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			p := PlatformCollection{}
			err := p.AddFromCommaSeparated(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, p.Platforms())
		})
	}
}

func TestAddFromResources(t *testing.T) {
	testCases := map[string]struct {
		yaml    string
		want    []platforms.Platform
		wantErr bool
	}{
		"pod with snp": {
			yaml: `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc-metal-qemu-snp
`,
			want: []platforms.Platform{platforms.MetalQEMUSNP},
		},
		"pod with tdx": {
			yaml: `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc-metal-qemu-tdx
`,
			want: []platforms.Platform{platforms.MetalQEMUTDX},
		},
		"pod with snp-gpu": {
			yaml: `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc-metal-qemu-snp-gpu
`,
			want: []platforms.Platform{platforms.MetalQEMUSNPGPU},
		},
		"pod with tdx-gpu": {
			yaml: `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc-metal-qemu-tdx-gpu
`,
			want: []platforms.Platform{platforms.MetalQEMUTDXGPU},
		},
		"pod without runtime class": {
			yaml: `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: pause
    image: pause
`,
			want: nil,
		},
		"pod with different prefix": {
			yaml: `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: other-cc-metal-qemu-snp
`,
			want: nil,
		},
		"multiple resources in one yaml": {
			yaml: `
apiVersion: v1
kind: Pod
metadata:
  name: p1
spec:
  runtimeClassName: contrast-cc-metal-qemu-snp
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
spec:
  template:
    spec:
      runtimeClassName: contrast-cc-metal-qemu-tdx
`,
			want: []platforms.Platform{platforms.MetalQEMUSNP, platforms.MetalQEMUTDX},
		},
		"invalid runtime class": {
			yaml: `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc-invalid
`,
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			resources, err := UnmarshalApplyConfigurations([]byte(tc.yaml))
			require.NoError(t, err)

			p := PlatformCollection{}
			err = p.AddFromResources(resources)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, p.Platforms())
		})
	}
}

func TestAddFromYamlFiles(t *testing.T) {
	testCases := map[string]struct {
		files   map[string]string
		want    []platforms.Platform
		wantErr bool
	}{
		"single file": {
			files: map[string]string{
				"pod.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc-metal-qemu-snp
`,
			},
			want: []platforms.Platform{platforms.MetalQEMUSNP},
		},
		"multiple files": {
			files: map[string]string{
				"pod1.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: p1
spec:
  runtimeClassName: contrast-cc-metal-qemu-snp
`,
				"pod2.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: p2
spec:
  runtimeClassName: contrast-cc-metal-qemu-tdx
`,
			},
			want: []platforms.Platform{platforms.MetalQEMUSNP, platforms.MetalQEMUTDX},
		},
		"nested directory": {
			files: map[string]string{
				"subdir/pod.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc-metal-qemu-snp-gpu
`,
			},
			want: []platforms.Platform{platforms.MetalQEMUSNPGPU},
		},
		"non-yaml file ignored": {
			files: map[string]string{
				"pod.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc-metal-qemu-tdx-gpu
`,
				"README.md": "This is not a YAML file",
			},
			want: []platforms.Platform{platforms.MetalQEMUTDXGPU},
		},
		"empty directory": {
			files: map[string]string{},
			want:  nil,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			defer os.RemoveAll(tmpDir)

			for path, content := range tc.files {
				fullPath := filepath.Join(tmpDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0o644)
				require.NoError(t, err)
			}

			p := PlatformCollection{}
			err := p.AddFromYamlFiles(tmpDir)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, p.Platforms())
		})
	}

	t.Run("invalid yaml in file", func(t *testing.T) {
		tmpDir := t.TempDir()
		defer os.RemoveAll(tmpDir)

		err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(":::invalid:::"), 0o644)
		require.NoError(t, err)

		p := PlatformCollection{}
		err = p.AddFromYamlFiles(tmpDir)
		require.Error(t, err)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		p := PlatformCollection{}
		err := p.AddFromYamlFiles("/non/existent/path")
		require.Error(t, err)
	})
}
