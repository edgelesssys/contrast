// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/edgelesssys/contrast/cli/genpolicy"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func TestSelectTDXReferenceValues(t *testing.T) {
	const handler = "contrast-cc-metal-qemu-tdx-test"
	base := manifest.TDXReferenceValues{
		Platform:          "Metal-QEMU-TDX",
		Rtmrs:             [4]manifest.HexString{"01", "11", "22", "33"},
		Rtmr0Alternatives: []manifest.HexString{"02", "03"},
	}
	embedded := manifest.EmbeddedReferenceValues{handler: {
		ReferenceValues: manifest.ReferenceValues{TDX: []manifest.TDXReferenceValues{base}},
		RTMR0ByVCPU:     map[int]manifest.HexString{1: "01", 2: "02", 3: "03"},
	}}
	testCases := map[string]struct {
		cpus           []string
		want           []manifest.HexString
		custom         bool
		wantErr        bool
		kind           string
		missingMapping bool
	}{
		"coordinator and workload": {cpus: []string{"0", "1"}, want: []manifest.HexString{"01", "02"}},
		"only two CPUs":            {cpus: []string{"1"}, want: []manifest.HexString{"02"}},
		"duplicate counts":         {cpus: []string{"1", "1"}, want: []manifest.HexString{"02"}},
		"order independent":        {cpus: []string{"2", "0"}, want: []manifest.HexString{"01", "03"}},
		"unsupported count":        {cpus: []string{"220"}, wantErr: true},
		"replace custom RTMR0":     {cpus: []string{"1"}, custom: true, want: []manifest.HexString{"02"}},
		"missing mapping":          {cpus: []string{"1"}, missingMapping: true, wantErr: true},
		"deployment":               {cpus: []string{"1"}, kind: "Deployment", want: []manifest.HexString{"02"}},
		"statefulset":              {cpus: []string{"1"}, kind: "StatefulSet", want: []manifest.HexString{"02"}},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mnf := &manifest.Manifest{ReferenceValues: manifest.ReferenceValues{TDX: []manifest.TDXReferenceValues{base}}}
			if tc.custom {
				mnf.ReferenceValues.TDX[0].Rtmrs[0] = "ff"
			}
			var pods []any
			for _, cpus := range tc.cpus {
				spec := applycorev1.PodSpec().
					WithRuntimeClassName(handler).WithContainers(applycorev1.Container().WithName("workload").
					WithResources(applycorev1.ResourceRequirements().WithLimits(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpus)})))
				switch tc.kind {
				case "Deployment":
					pods = append(pods, applyappsv1.Deployment("workload", "test").WithSpec(applyappsv1.DeploymentSpec().WithTemplate(applycorev1.PodTemplateSpec().WithSpec(spec))))
				case "StatefulSet":
					pods = append(pods, applyappsv1.StatefulSet("workload", "test").WithSpec(applyappsv1.StatefulSetSpec().WithTemplate(applycorev1.PodTemplateSpec().WithSpec(spec))))
				default:
					pods = append(pods, applycorev1.Pod("pod", "test").WithSpec(spec))
				}
			}
			resources, err := kuberesource.ResourcesToUnstructured(pods)
			require.NoError(t, err)
			values := embedded
			if tc.missingMapping {
				values = nil
			}
			err = selectTDXReferenceValues(map[string][]*unstructured.Unstructured{"pods.yml": resources}, mnf, values)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got := mnf.ReferenceValues.TDX[0]
			assert.Equal(t, tc.want, append([]manifest.HexString{got.Rtmrs[0]}, got.Rtmr0Alternatives...))
			assert.Equal(t, base.Rtmrs[1:], got.Rtmrs[1:])
		})
	}
}

func TestSelectTDXReferenceValuesUpdates(t *testing.T) {
	const handler = "contrast-cc-metal-qemu-tdx-test"
	base := manifest.TDXReferenceValues{
		Platform: "Metal-QEMU-TDX", MrTd: "aa",
		Rtmrs:             [4]manifest.HexString{"01", "11", "22", "33"},
		Rtmr0Alternatives: []manifest.HexString{"02", "04"},
	}
	embedded := manifest.EmbeddedReferenceValues{handler: {
		ReferenceValues: manifest.ReferenceValues{TDX: []manifest.TDXReferenceValues{base}},
		RTMR0ByVCPU:     map[int]manifest.HexString{1: "01", 2: "02", 4: "04"},
	}}
	mnf := &manifest.Manifest{ReferenceValues: manifest.ReferenceValues{TDX: []manifest.TDXReferenceValues{base}}}
	mnf.ReferenceValues.TDX[0].MrSeam = "bb"
	for _, tc := range []struct {
		name string
		cpu  string
		want manifest.HexString
	}{
		{"initial generation", "1", "02"},
		{"increase CPUs", "3", "04"},
		{"unchanged CPUs", "3", "04"},
		{"decrease CPUs", "1", "02"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resources, err := kuberesource.ResourcesToUnstructured([]any{
				applycorev1.Pod("workload", "test").WithSpec(applycorev1.PodSpec().WithRuntimeClassName(handler).
					WithContainers(applycorev1.Container().WithResources(applycorev1.ResourceRequirements().
						WithLimits(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(tc.cpu)})))),
			})
			require.NoError(t, err)
			require.NoError(t, selectTDXReferenceValues(map[string][]*unstructured.Unstructured{"pods.yml": resources}, mnf, embedded))
			got := mnf.ReferenceValues.TDX[0]
			assert.Equal(t, tc.want, got.Rtmrs[0])
			assert.Empty(t, got.Rtmr0Alternatives)
			assert.Equal(t, base.Rtmrs[1:], got.Rtmrs[1:])
			assert.Equal(t, base.MrTd, got.MrTd)
			assert.Equal(t, manifest.HexString("bb"), got.MrSeam)
			encoded, err := json.Marshal(mnf)
			require.NoError(t, err)
			var loaded manifest.Manifest
			require.NoError(t, json.Unmarshal(encoded, &loaded))
			mnf = &loaded
		})
	}
}

func TestSelectTDXReferenceValuesIncompatibleRuntime(t *testing.T) {
	const handler = "contrast-cc-metal-qemu-tdx-test"
	base := manifest.TDXReferenceValues{Platform: "Metal-QEMU-TDX", MrTd: "aa", Rtmrs: [4]manifest.HexString{"01", "11", "22", "33"}}
	embedded := manifest.EmbeddedReferenceValues{handler: {
		ReferenceValues: manifest.ReferenceValues{TDX: []manifest.TDXReferenceValues{base}},
		RTMR0ByVCPU:     map[int]manifest.HexString{1: "01"},
	}}
	resources, err := kuberesource.ResourcesToUnstructured([]any{
		applycorev1.Pod("workload", "test").WithSpec(applycorev1.PodSpec().WithRuntimeClassName(handler)),
	})
	require.NoError(t, err)
	testCases := map[string]func(*manifest.TDXReferenceValues){
		"MRTD":  func(ref *manifest.TDXReferenceValues) { ref.MrTd = "ff" },
		"RTMR1": func(ref *manifest.TDXReferenceValues) { ref.Rtmrs[1] = "ff" },
		"RTMR2": func(ref *manifest.TDXReferenceValues) { ref.Rtmrs[2] = "ff" },
		"RTMR3": func(ref *manifest.TDXReferenceValues) { ref.Rtmrs[3] = "ff" },
	}
	for name, change := range testCases {
		t.Run(name, func(t *testing.T) {
			ref := base
			change(&ref)
			mnf := &manifest.Manifest{ReferenceValues: manifest.ReferenceValues{TDX: []manifest.TDXReferenceValues{ref}}}
			err := selectTDXReferenceValues(map[string][]*unstructured.Unstructured{"pods.yml": resources}, mnf, embedded)
			require.ErrorContains(t, err, "do not match embedded reference values")
			assert.Equal(t, ref, mnf.ReferenceValues.TDX[0])
		})
	}
	t.Run("missing reference values", func(t *testing.T) {
		err := selectTDXReferenceValues(map[string][]*unstructured.Unstructured{"pods.yml": resources}, &manifest.Manifest{}, embedded)
		require.ErrorContains(t, err, "manifest has no TDX reference values")
	})
}

func TestPodVCPUCount(t *testing.T) {
	container := func(cpu string) *applycorev1.ContainerApplyConfiguration {
		return applycorev1.Container().WithResources(applycorev1.ResourceRequirements().WithLimits(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu)}))
	}
	testCases := map[string]struct {
		spec *applycorev1.PodSpecApplyConfiguration
		want int64
	}{
		"empty":         {applycorev1.PodSpec(), 1},
		"fractional":    {applycorev1.PodSpec().WithContainers(container("100m")), 2},
		"regular sum":   {applycorev1.PodSpec().WithContainers(container("600m"), container("600m")), 3},
		"init sum":      {applycorev1.PodSpec().WithInitContainers(container("600m"), container("600m")), 3},
		"sidecar":       {applycorev1.PodSpec().WithContainers(container("1")).WithInitContainers(container("500m").WithRestartPolicy(corev1.ContainerRestartPolicyAlways)), 3},
		"pod resources": {applycorev1.PodSpec().WithResources(applycorev1.ResourceRequirements().WithLimits(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")})), 5},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, podVCPUCount(tc.spec))
		})
	}
}

func TestSelectTDXReferenceValuesOtherPlatforms(t *testing.T) {
	for _, handler := range []string{"contrast-cc-metal-qemu-snp-test", "contrast-cc-metal-qemu-tdx-gpu-test"} {
		t.Run(handler, func(t *testing.T) {
			resources, err := kuberesource.ResourcesToUnstructured([]any{
				applycorev1.Pod("pod", "test").WithSpec(applycorev1.PodSpec().WithRuntimeClassName(handler)),
			})
			require.NoError(t, err)
			newManifest := func() *manifest.Manifest {
				return &manifest.Manifest{ReferenceValues: manifest.ReferenceValues{
					SNP: []manifest.SNPReferenceValues{{TrustedMeasurement: "01"}},
					TDX: []manifest.TDXReferenceValues{{Platform: "Metal-QEMU-TDX-GPU", Rtmrs: [4]manifest.HexString{"01", "11", "22", "33"}, Rtmr0Alternatives: []manifest.HexString{"02"}}},
				}}
			}
			mnf := newManifest()
			require.NoError(t, selectTDXReferenceValues(map[string][]*unstructured.Unstructured{"pods.yml": resources}, mnf, nil))
			assert.Equal(t, newManifest(), mnf)
		})
	}
}

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

func TestPatchCoordinatorAllowInsecure(t *testing.T) {
	insecurePlatforms := kuberesource.PlatformCollection{
		platforms.MetalQEMUInsecure: {},
	}
	securePlatforms := kuberesource.PlatformCollection{
		platforms.MetalQEMUSNP: {},
	}
	mixedPlatforms := kuberesource.PlatformCollection{
		platforms.MetalQEMUSNP:      {},
		platforms.MetalQEMUInsecure: {},
	}

	newCoordinator := func() *applyappsv1.StatefulSetApplyConfiguration {
		podSpec := applycorev1.PodSpec().WithContainers(
			applycorev1.Container().WithName("sidecar"),
			applycorev1.Container().WithName("coordinator").WithEnv(
				kuberesource.NewEnvVar("KEEP_ME", "value"),
				kuberesource.NewEnvVar(allowInsecureEnvVar, "false"),
			),
		)
		template := applycorev1.PodTemplateSpec().
			WithLabels(map[string]string{kuberesource.ContrastRoleLabelKey: string(manifest.RoleCoordinator)}).
			WithSpec(podSpec)
		return applyappsv1.StatefulSet("coordinator", "default").
			WithSpec(applyappsv1.StatefulSetSpec().WithTemplate(template))
	}

	t.Run("patches named coordinator container idempotently", func(t *testing.T) {
		coordinator := newCoordinator()
		patchCoordinatorAllowInsecure(coordinator, insecurePlatforms, true)
		patchCoordinatorAllowInsecure(coordinator, insecurePlatforms, true)

		containers := coordinator.Spec.Template.Spec.Containers
		require.Len(t, containers, 2)
		assert.Empty(t, containers[0].Env)
		require.Len(t, containers[1].Env, 2)

		env := map[string]string{}
		for _, envVar := range containers[1].Env {
			env[*envVar.Name] = *envVar.Value
		}
		assert.Equal(t, "value", env["KEEP_ME"])
		assert.Equal(t, "1", env[allowInsecureEnvVar])
	})

	t.Run("does not patch secure platforms", func(t *testing.T) {
		coordinator := newCoordinator()
		patchCoordinatorAllowInsecure(coordinator, securePlatforms, true)
		assert.Equal(t, "false", *coordinator.Spec.Template.Spec.Containers[1].Env[1].Value)
	})

	t.Run("does not patch mixed platforms", func(t *testing.T) {
		coordinator := newCoordinator()
		patchCoordinatorAllowInsecure(coordinator, mixedPlatforms, true)
		assert.Equal(t, "false", *coordinator.Spec.Template.Spec.Containers[1].Env[1].Value)
	})

	t.Run("does not patch without opt-in", func(t *testing.T) {
		coordinator := newCoordinator()
		patchCoordinatorAllowInsecure(coordinator, insecurePlatforms, false)
		assert.Equal(t, "false", *coordinator.Spec.Template.Spec.Containers[1].Env[1].Value)
	})

	t.Run("handles missing pod spec", func(t *testing.T) {
		template := applycorev1.PodTemplateSpec().
			WithLabels(map[string]string{kuberesource.ContrastRoleLabelKey: string(manifest.RoleCoordinator)})
		coordinator := applyappsv1.StatefulSet("coordinator", "default").
			WithSpec(applyappsv1.StatefulSetSpec().WithTemplate(template))

		assert.False(t, isCoordinator(coordinator))
		assert.NotPanics(t, func() {
			patchCoordinatorAllowInsecure(coordinator, insecurePlatforms, true)
		})
	})
}

func TestValidateInsecurePlatforms(t *testing.T) {
	testCases := map[string]struct {
		platforms      []platforms.Platform
		allowInsecure  bool
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
		"insecure with flag": {
			platforms:     []platforms.Platform{platforms.MetalQEMUInsecure},
			allowInsecure: true,
			wantErr:       false,
		},
		"mixed with flag": {
			platforms:     []platforms.Platform{platforms.MetalQEMUSNP, platforms.MetalQEMUInsecure},
			allowInsecure: true,
			wantErr:       false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
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

func TestValidateInsecureManifest(t *testing.T) {
	testCases := map[string]struct {
		platform      platforms.Platform
		allowInsecure bool
		wantErr       bool
	}{
		"secure without flag": {
			platform: platforms.MetalQEMUSNP,
		},
		"secure with flag": {
			platform:      platforms.MetalQEMUSNP,
			allowInsecure: true,
		},
		"insecure without flag": {
			platform: platforms.MetalQEMUInsecure,
			wantErr:  true,
		},
		"insecure with flag": {
			platform:      platforms.MetalQEMUInsecure,
			allowInsecure: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mnf := &manifest.Manifest{
				ReferenceValues: manifest.ReferenceValues{
					SNP: []manifest.SNPReferenceValues{{Platform: tc.platform.String()}},
				},
			}
			err := validateInsecureManifest(mnf, tc.allowInsecure)
			if tc.wantErr {
				require.ErrorContains(t, err, "--INSECURE flag not set")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInsecureRuntimesAllowed(t *testing.T) {
	testCases := map[string]struct {
		set   bool
		value string
		want  bool
	}{
		"unset":   {set: false, want: false},
		"empty":   {set: true, value: "", want: false},
		"true":    {set: true, value: "true", want: true},
		"one":     {set: true, value: "1", want: true},
		"false":   {set: true, value: "false", want: false},
		"garbage": {set: true, value: "yes please", want: false},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(allowInsecureEnvVar, tc.value)
			if !tc.set {
				require.NoError(t, os.Unsetenv(allowInsecureEnvVar))
			}

			assert.Equal(t, tc.want, insecureRuntimesAllowed())
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
