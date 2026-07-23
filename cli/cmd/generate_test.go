// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"
	"testing"

	"github.com/edgelesssys/contrast/cli/genpolicy"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// TestStatefulSetInjections is a regression test for a nil dereference in the inject* functions.
func TestStatefulSetInjections(t *testing.T) {
	resources := []any{statefulSet()}

	t.Run("injectInitializer", func(t *testing.T) {
		require.NoError(t, injectInitializer(resources, "coordinator-namespace", "", kuberesource.MemoryProfileFull))
	})

	t.Run("injectServiceMesh", func(t *testing.T) {
		require.NoError(t, injectServiceMesh(resources, kuberesource.MemoryProfileFull))
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
		"single insecure": {
			yaml: map[string]string{
				"file1.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: p1
spec:
  runtimeClassName: contrast-insecure-metal-qemu
`,
			},
			want: []platforms.Platform{platforms.MetalQEMUInsecure},
		},
		"mixed cc and insecure": {
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
  runtimeClassName: contrast-insecure-metal-qemu
`,
			},
			want: []platforms.Platform{platforms.MetalQEMUSNP, platforms.MetalQEMUInsecure},
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
	ccHandler := "contrast-cc-metal-qemu-snp"
	insecureHandler := "contrast-insecure-metal-qemu"

	testCases := map[string]struct {
		defaultHandler string
		initial        string
		want           string
		updateHandler  bool
		wantErr        bool
	}{
		"no runtime class": {
			defaultHandler: ccHandler,
			initial:        "",
			want:           "",
		},
		"irrelevant class": {
			defaultHandler: ccHandler,
			initial:        "runc",
			want:           "runc",
		},
		"generic kata": {
			defaultHandler: ccHandler,
			initial:        "kata-cc-isolation",
			want:           ccHandler,
		},
		"generic contrast": {
			defaultHandler: ccHandler,
			initial:        "contrast-cc",
			want:           ccHandler,
		},
		"specific contrast-cc-metal-qemu-tdx": {
			defaultHandler: ccHandler,
			initial:        "contrast-cc-metal-qemu-tdx",
			want:           "contrast-cc-metal-qemu-tdx",
			updateHandler:  true,
		},
		"generic contrast-insecure with insecure handler": {
			defaultHandler: insecureHandler,
			initial:        "contrast-insecure",
			want:           insecureHandler,
		},
		"generic contrast-insecure with cc handler errors": {
			defaultHandler: ccHandler,
			initial:        "contrast-insecure",
			wantErr:        true,
		},
		"generic contrast-cc with insecure handler errors": {
			defaultHandler: insecureHandler,
			initial:        "contrast-cc",
			wantErr:        true,
		},
		"generic kata with insecure handler errors": {
			defaultHandler: insecureHandler,
			initial:        "kata-cc-isolation",
			wantErr:        true,
		},
		"specific contrast-insecure-metal-qemu": {
			defaultHandler: ccHandler,
			initial:        "contrast-insecure-metal-qemu",
			want:           "contrast-insecure-metal-qemu",
			updateHandler:  true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if _, err := manifest.GetEmbeddedReferenceValues(); err != nil && tc.updateHandler {
				// The embedded reference values are only available when running the test through nix
				// (e.g. nix build .#base.contrast.cli), not when using go test.
				// This only applies in cases where manifest.RuntimeHandler is called, in which case
				// we also need to update the handler to include the suffix (see below).
				t.Skip()
			} else if tc.updateHandler {
				tc.want = getHandler(t, tc.want)
			}

			patch := patchRuntimeClassName(tc.defaultHandler)
			spec := applycorev1.PodSpec()
			if tc.initial != "" {
				spec.WithRuntimeClassName(tc.initial)
			}
			_, err := patch(spec)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.want == "" {
				assert.Nil(t, spec.RuntimeClassName)
			} else {
				require.NotNil(t, spec.RuntimeClassName)
				assert.Equal(t, tc.want, *spec.RuntimeClassName)
			}
		})
	}

	t.Run("nil spec returns nil", func(t *testing.T) {
		patch := patchRuntimeClassName(ccHandler)
		result, err := patch(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestIsContrastWorkload(t *testing.T) {
	testCases := map[string]struct {
		runtimeClass string
		want         bool
	}{
		"no runtime class": {
			runtimeClass: "",
			want:         false,
		},
		"non-contrast runtime class": {
			runtimeClass: "foobar",
			want:         false,
		},
		"contrast-cc": {
			runtimeClass: "contrast-cc",
			want:         true,
		},
		"contrast-cc-metal-qemu-snp": {
			runtimeClass: "contrast-cc-metal-qemu-snp",
			want:         true,
		},
		"contrast-insecure": {
			runtimeClass: "contrast-insecure",
			want:         true,
		},
		"contrast-insecure-metal-qemu": {
			runtimeClass: "contrast-insecure-metal-qemu",
			want:         true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			spec := applycorev1.PodSpec()
			if tc.runtimeClass != "" {
				spec.WithRuntimeClassName(tc.runtimeClass)
			}
			pod := applycorev1.Pod("test", "default").WithSpec(spec)
			assert.Equal(t, tc.want, isContrastWorkload(pod))
		})
	}
}

func TestValidateInsecurePlatforms(t *testing.T) {
	testCases := map[string]struct {
		platforms      []platforms.Platform
		allowInsecure  bool
		setEnv         bool
		wantErr        bool
		wantErrContain string
	}{
		"no insecure platforms": {
			platforms: []platforms.Platform{platforms.MetalQEMUSNP},
			wantErr:   false,
		},
		"insecure without flag": {
			platforms:      []platforms.Platform{platforms.MetalQEMUInsecure},
			allowInsecure:  false,
			wantErr:        true,
			wantErrContain: "--INSECURE flag not set",
		},
		"insecure with flag but no env": {
			platforms:      []platforms.Platform{platforms.MetalQEMUInsecure},
			allowInsecure:  true,
			setEnv:         false,
			wantErr:        true,
			wantErrContain: "CONTRAST_ALLOW_INSECURE",
		},
		"insecure with flag and env": {
			platforms:     []platforms.Platform{platforms.MetalQEMUInsecure},
			allowInsecure: true,
			setEnv:        true,
			wantErr:       false,
		},
		"mixed with flag and env": {
			platforms:     []platforms.Platform{platforms.MetalQEMUSNP, platforms.MetalQEMUInsecure},
			allowInsecure: true,
			setEnv:        true,
			wantErr:       false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.setEnv {
				t.Setenv("CONTRAST_ALLOW_INSECURE", "true")
			} else {
				os.Unsetenv("CONTRAST_ALLOW_INSECURE")
			}

			collection := kuberesource.PlatformCollection{}
			for _, p := range tc.platforms {
				collection.Add(p)
			}

			err := validateInsecurePlatforms(collection, tc.allowInsecure)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrContain)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func getHandler(t *testing.T, name string) string {
	t.Helper()
	platform, err := platforms.FromRuntimeClassString(name)
	if platform == platforms.Unknown {
		// Testcase where we don't expect a supported platform.
		return name
	}
	require.NoError(t, err)
	handler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)
	return handler
}

func TestCalculatePodMemory(t *testing.T) {
	layersCache := &genpolicy.LayersCache{
		Index: map[string]genpolicy.ImageLayerIndex{
			"docker.io/library/some-image": {
				ImageRef: "some-image",
				Layers: []genpolicy.ImageLayerIndexEntry{
					{
						DiffID:         "layer1",
						CompressedSize: 10,
					},
				},
			},
			"ghcr.io/other/image": {
				ImageRef: "other-image",
				Layers: []genpolicy.ImageLayerIndexEntry{
					{
						DiffID:         "layer1",
						CompressedSize: 20,
					},
				},
			},
		},
		Layers: map[string]genpolicy.ImageLayer{
			"layer1": {
				DiffID:           "layer1",
				UncompressedSize: 20,
			},
		},
	}

	testCases := map[string]struct {
		pod  *applycorev1.PodApplyConfiguration
		want int64
	}{
		"main container without limits": {
			pod: kuberesource.Pod("test-pod", "default").
				WithSpec(
					kuberesource.PodSpec().
						WithContainers(
							kuberesource.Container().
								WithImage("some-image"),
						),
				),
			want: 30,
		},
		"main container with limits": {
			pod: kuberesource.Pod("test-pod", "default").
				WithSpec(
					kuberesource.PodSpec().
						WithContainers(
							kuberesource.Container().
								WithImage("some-image").
								WithResources(
									kuberesource.ResourceRequirements().
										WithMemoryLimitAndRequest(100),
								),
						),
				),
			want: 30 + 100*1024*1024,
		},
		"two containers with different images": {
			pod: kuberesource.Pod("test-pod", "default").
				WithSpec(
					kuberesource.PodSpec().
						WithContainers(
							kuberesource.Container().
								WithImage("some-image"),
							kuberesource.Container().
								WithImage("ghcr.io/other/image"),
						),
				),
			want: 70,
		},
		"init container with low limits": {
			pod: kuberesource.Pod("test-pod", "default").
				WithSpec(
					kuberesource.PodSpec().
						WithContainers(
							kuberesource.Container().
								WithImage("some-image").
								WithResources(
									kuberesource.ResourceRequirements().
										WithMemoryLimitAndRequest(100),
								),
						).
						WithInitContainers(
							kuberesource.Container().
								WithImage("some-image").
								WithResources(
									kuberesource.ResourceRequirements().
										WithMemoryLimitAndRequest(10),
								),
						),
				),
			want: 30 + 100*1024*1024,
		},
		"init container with high limits": {
			pod: kuberesource.Pod("test-pod", "default").
				WithSpec(
					kuberesource.PodSpec().
						WithContainers(
							kuberesource.Container().
								WithImage("some-image").
								WithResources(
									kuberesource.ResourceRequirements().
										WithMemoryLimitAndRequest(100),
								),
						).
						WithInitContainers(
							kuberesource.Container().
								WithImage("some-image").
								WithResources(
									kuberesource.ResourceRequirements().
										WithMemoryLimitAndRequest(200),
								),
						),
				),
			want: 30 + 200*1024*1024,
		},
		"side car container": {
			pod: kuberesource.Pod("test-pod", "default").
				WithSpec(
					kuberesource.PodSpec().
						WithContainers(
							kuberesource.Container().
								WithImage("some-image").
								WithResources(
									kuberesource.ResourceRequirements().
										WithMemoryLimitAndRequest(100),
								),
						).
						WithInitContainers(
							kuberesource.Container().
								WithImage("some-image").
								WithResources(
									kuberesource.ResourceRequirements().
										WithMemoryLimitAndRequest(200),
								).
								WithRestartPolicy(corev1.ContainerRestartPolicyAlways),
						),
				),
			want: 30 + 300*1024*1024,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			got, err := calculatePodMemory(tc.pod.Spec, layersCache)
			require.NoError(err)
			require.Equal(tc.want, got)
		})
	}
}
