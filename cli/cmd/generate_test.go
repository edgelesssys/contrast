// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"log/slog"
	"testing"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// TestStatefulSetInjections is a regression test for a nil dereference in the inject* functions.
func TestStatefulSetInjections(t *testing.T) {
	resources := []any{statefulSet()}

	t.Run("injectInitializer", func(t *testing.T) {
		require.NoError(t, injectInitializer(resources, "coordinator-namespace"))
	})

	t.Run("injectServiceMesh", func(t *testing.T) {
		require.NoError(t, injectServiceMesh(resources))
	})
}

func statefulSet() *applyappsv1.StatefulSetApplyConfiguration {
	return applyappsv1.StatefulSet("some-name", "some-namespace").
		WithSpec(applyappsv1.StatefulSetSpec().WithTemplate(applycorev1.PodTemplateSpec()))
}

func TestRuntimeClassesFromUnstructured(t *testing.T) {
	testCases := map[string]struct {
		yaml map[string]string
		want []platforms.Platform
	}{
		"empty": {
			yaml: map[string]string{},
			want: nil,
		},
		"single snp": {
			yaml: map[string]string{
				"file1.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: p1
spec:
  runtimeClassName: contrast-cc-metal-qemu-snp
`,
			},
			want: []platforms.Platform{platforms.MetalQEMUSNP},
		},
		"multiple files": {
			yaml: map[string]string{
				"file1.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: p1
spec:
  runtimeClassName: contrast-cc-metal-qemu-snp
`,
				"file2.yaml": `
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
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			fileMap := make(map[string][]*unstructured.Unstructured)
			for path, yaml := range tc.yaml {
				resources, err := kuberesource.UnmarshalApplyConfigurations([]byte(yaml))
				require.NoError(t, err)

				unstructured, err := kuberesource.ResourcesToUnstructured(resources)
				require.NoError(t, err)
				fileMap[path] = unstructured
			}

			got, err := runtimeClassesFromUnstructured(fileMap)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, got.Platforms())
		})
	}
}

func TestPatchRuntimeClassName(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	defaultHandler := "contrast-cc-metal-qemu-snp"

	testCases := map[string]struct {
		initial string
		want    string
	}{
		"no runtime class": {
			initial: "",
			want:    "",
		},
		"irrelevant class": {
			initial: "runc",
			want:    "runc",
		},
		"generic kata": {
			initial: "kata-cc-isolation",
			want:    defaultHandler,
		},
		"generic contrast": {
			initial: "contrast-cc",
			want:    defaultHandler,
		},
		"specific contrast-cc-metal-qemu-snp": {
			initial: "contrast-cc-metal-qemu-snp",
			want:    "contrast-cc-metal-qemu-snp",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			patch := patchRuntimeClassName(logger, defaultHandler)
			spec := applycorev1.PodSpec()
			if tc.initial != "" {
				spec.WithRuntimeClassName(tc.initial)
			}
			patch(spec)
			if tc.want == "" {
				assert.Nil(t, spec.RuntimeClassName)
			} else {
				require.NotNil(t, spec.RuntimeClassName)
				assert.Equal(t, tc.want, *spec.RuntimeClassName)
			}
		})
	}

	t.Run("nil spec returns nil", func(t *testing.T) {
		patch := patchRuntimeClassName(logger, defaultHandler)
		assert.Nil(t, patch(nil))
	})
}
