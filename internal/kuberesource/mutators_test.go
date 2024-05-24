// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"testing"

	"github.com/stretchr/testify/require"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func TestPatchNamespaces(t *testing.T) {
	coordinator := CoordinatorBundle()
	openssl := OpenSSL()
	emojivoto := Emojivoto(ServiceMeshIngressEgress)

	for _, tc := range []struct {
		name string
		set  []any
	}{
		{
			name: "coordinator",
			set:  coordinator,
		},
		{
			name: "openssl",
			set:  openssl,
		},
		{
			name: "emojivoto",
			set:  emojivoto,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			expectedNamespace := "right-namespace"
			set := PatchNamespaces(tc.set, expectedNamespace)
			u, err := ResourcesToUnstructured(set)
			require.NoError(err)
			require.NotEmpty(u)
			for _, obj := range u {
				require.Equal(expectedNamespace, obj.GetNamespace())
			}
		})
		t.Run(tc.name+"-empty-namespace", func(t *testing.T) {
			require := require.New(t)
			set := PatchNamespaces(tc.set, "some-namespace")
			set = PatchNamespaces(set, "")
			u, err := ResourcesToUnstructured(set)
			require.NoError(err)
			require.NotEmpty(u)
			for _, obj := range u {
				meta := obj.Object["metadata"].(map[string]any)
				_, ok := meta["namespace"]
				require.False(ok, "namespace should have been deleted")
			}
		})
	}
}

func TestAddInitializer(t *testing.T) {
	require := require.New(t)
	d := applyappsv1.Deployment("test", "default").
		WithSpec(applyappsv1.DeploymentSpec().
			WithTemplate(applycorev1.PodTemplateSpec().
				WithSpec(applycorev1.PodSpec().
					WithContainers(applycorev1.Container()))))

	AddInitializer(d, Initializer())

	require.NotEmpty(d.Spec.Template.Spec.InitContainers)
	require.NotEmpty(d.Spec.Template.Spec.InitContainers[0].VolumeMounts)
	require.NotEmpty(d.Spec.Template.Spec.Containers)
	require.NotEmpty(d.Spec.Template.Spec.Containers[0].VolumeMounts)
	require.NotEmpty(d.Spec.Template.Spec.Volumes)
	require.Equal(*d.Spec.Template.Spec.Volumes[0].Name, *d.Spec.Template.Spec.InitContainers[0].VolumeMounts[0].Name)
	require.Equal(*d.Spec.Template.Spec.Volumes[0].Name, *d.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
}

func TestAddServiceMesh(t *testing.T) {
	require := require.New(t)
	d := applyappsv1.Deployment("test", "default").
		WithSpec(applyappsv1.DeploymentSpec().
			WithTemplate(applycorev1.PodTemplateSpec().
				WithSpec(applycorev1.PodSpec().
					WithContainers(applycorev1.Container()))))

	smProxy := ServiceMeshProxy()
	AddServiceMesh(d, smProxy)

	require.NotEmpty(d.Spec.Template.Spec.InitContainers)
	require.Equal(d.Spec.Template.Spec.InitContainers[0], *smProxy)
}
