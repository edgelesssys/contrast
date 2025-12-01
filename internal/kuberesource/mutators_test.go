// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

var (
	//go:embed assets/cronjob.yaml
	cronjob []byte
	//go:embed assets/daemonset.yaml
	daemonset []byte
	//go:embed assets/nginx-deployment.yaml
	deployment []byte
	//go:embed assets/job.yaml
	job []byte
	//go:embed assets/pod-nginx.yaml
	pod []byte
)

func TestPatchNamespaces(t *testing.T) {
	require := require.New(t)

	coordinator := CoordinatorBundle()
	openssl := OpenSSL()
	emojivoto := Emojivoto(ServiceMeshIngressEgress)
	volumeStatefulSet := VolumeStatefulSet()
	mysql := MySQL()

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
		{
			name: "volume-stateful-set",
			set:  volumeStatefulSet,
		},
		{
			name: "mysql",
			set:  mysql,
		},
	} {
		t.Run(tc.name, func(_ *testing.T) {
			expectedNamespace := "right-namespace"
			set := PatchNamespaces(tc.set, expectedNamespace)
			u, err := ResourcesToUnstructured(set)
			require.NoError(err)
			require.NotEmpty(u)
			for _, obj := range u {
				require.Equal(expectedNamespace, obj.GetNamespace())
			}
		})
	}
}

func TestAddInitializer(t *testing.T) {
	initializer := Initializer("coordinator-ready.default")
	expectedInitializerContainerName := *initializer.Name
	expectedInitializerVolumeMountName := *initializer.VolumeMounts[0].Name
	for _, tc := range []struct {
		name      string
		d         *applyappsv1.DeploymentApplyConfiguration
		wantError bool
	}{
		{
			name: "default",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: false,
		},
		{
			name: "initializer replaced",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithInitContainers(initializer).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: false,
		},
		{
			name: "volume reused",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc").
							WithVolumes(Volume().
								WithName(*initializer.VolumeMounts[0].Name).
								WithEmptyDir(EmptyDirVolumeSource().Inner()),
							),
						))),
			wantError: false,
		},
		{
			name: "volume is not an EmptyDir",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc").
							WithVolumes(Volume().
								WithName(*initializer.VolumeMounts[0].Name).
								WithConfigMap(Volume().ConfigMap),
							),
						))),
			wantError: true,
		},
		{
			name: "volume mount reused",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithSpec(applycorev1.PodSpec().
							WithContainers(
								applycorev1.Container().
									WithVolumeMounts(
										VolumeMount().
											WithName(expectedInitializerVolumeMountName).
											WithMountPath("/some/other/path"),
									),
							).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: false,
		},
		{
			name: "unrelated initializers",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithInitContainers(applycorev1.Container().WithName("custom-init")).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: false,
		},
		{
			name: "volume mount reused in initializer",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithInitContainers(
								applycorev1.Container().
									WithVolumeMounts(
										VolumeMount().
											WithName(expectedInitializerVolumeMountName).
											WithMountPath("/some/other/path"),
									),
							).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: false,
		},
		{
			name: "cryptsetup default",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithAnnotations(map[string]string{securePVAnnotationKey: "device:mount"}).
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc").
							WithVolumes(
								Volume().
									WithName("device").
									WithPersistentVolumeClaim(applycorev1.PersistentVolumeClaimVolumeSource()),
							),
						))),
			wantError: false,
		},
		{
			name: "cryptsetup bad annotation",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithAnnotations(map[string]string{securePVAnnotationKey: "test"}).
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: true,
		},
		{
			name: "cryptsetup no device",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithAnnotations(map[string]string{securePVAnnotationKey: "device:mount"}).
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: true,
		},
		{
			name: "cryptsetup volume is not an block device",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithAnnotations(map[string]string{securePVAnnotationKey: "device:mount"}).
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc").
							WithVolumes(
								Volume().WithName("device"),
								Volume().
									WithName("mount").
									WithEmptyDir(EmptyDirVolumeSource().Inner()),
							),
						))),
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			_, err := AddInitializer(tc.d, initializer)
			if tc.wantError {
				require.Error(err)
				return
			}
			require.NoError(err)

			require.NotEmpty(tc.d.Spec.Template.Spec.InitContainers)
			require.Equal(expectedInitializerContainerName, *tc.d.Spec.Template.Spec.InitContainers[0].Name)
			require.NotEmpty(tc.d.Spec.Template.Spec.InitContainers[0].VolumeMounts)
			require.Equal(expectedInitializerVolumeMountName, *tc.d.Spec.Template.Spec.InitContainers[0].VolumeMounts[0].Name)

			if tc.d.Annotations[securePVAnnotationKey] != "" {
				securePVValue := strings.Split(tc.d.Annotations[securePVAnnotationKey], ":")
				_, mountName := securePVValue[0], securePVValue[1]
				sharedVolumeMountCount := 0
				for _, v := range tc.d.Spec.Template.Spec.InitContainers[0].VolumeMounts {
					if *v.Name == mountName {
						sharedVolumeMountCount++
					}
				}
				require.Equal(1, sharedVolumeMountCount)
				require.Equal(mountName, *tc.d.Spec.Template.Spec.InitContainers[0].VolumeMounts[1].Name)
			}

			initializerCount := 0
			for _, c := range tc.d.Spec.Template.Spec.InitContainers {
				if c.Name != nil && *c.Name == expectedInitializerContainerName {
					initializerCount++
				}
			}
			require.Equal(1, initializerCount)

			require.NotEmpty(tc.d.Spec.Template.Spec.Containers)
			for _, c := range tc.d.Spec.Template.Spec.Containers {
				initializerVolumeMountCount := 0
				for _, v := range c.VolumeMounts {
					if *v.Name == expectedInitializerVolumeMountName {
						initializerVolumeMountCount++
					}
				}
				require.Equal(1, initializerVolumeMountCount)
			}
			for _, c := range tc.d.Spec.Template.Spec.InitContainers {
				if c.Name == nil || *c.Name == expectedInitializerContainerName {
					continue
				}
				initializerVolumeMountCount := 0
				for _, v := range c.VolumeMounts {
					if *v.Name == expectedInitializerVolumeMountName {
						initializerVolumeMountCount++
					}
				}
				require.Equal(1, initializerVolumeMountCount)
			}

			require.NotEmpty(tc.d.Spec.Template.Spec.Volumes)
			initializerVolumeCount := 0
			for _, v := range tc.d.Spec.Template.Spec.Volumes {
				if *v.Name == expectedInitializerVolumeMountName {
					initializerVolumeCount++
				}
			}
			require.Equal(1, initializerVolumeCount)
		})
	}
}

func TestAddServiceMesh(t *testing.T) {
	for _, tc := range []struct {
		name            string
		d               *applyappsv1.DeploymentApplyConfiguration
		skipServiceMesh bool
		wantError       bool
	}{
		{
			name: "default",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithAnnotations(map[string]string{smIngressConfigAnnotationKey: ""}).
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: false,
		},
		{
			name: "no service mesh",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc"),
						))),
			skipServiceMesh: true,
			wantError:       false,
		},
		{
			name: "service mesh replaced",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithAnnotations(map[string]string{smIngressConfigAnnotationKey: ""}).
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithInitContainers(ServiceMeshProxy()).
							WithRuntimeClassName("contrast-cc"),
						))),
			wantError: false,
		},
		{
			name: "volume reused",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithAnnotations(map[string]string{smIngressConfigAnnotationKey: ""}).
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc").
							WithVolumes(Volume().
								WithName(*ServiceMeshProxy().VolumeMounts[0].Name).
								WithEmptyDir(EmptyDirVolumeSource().Inner()),
							),
						))),
			wantError: false,
		},
		{
			name: "volume is not an EmptyDir",
			d: applyappsv1.Deployment("test", "default").
				WithSpec(applyappsv1.DeploymentSpec().
					WithTemplate(applycorev1.PodTemplateSpec().
						WithAnnotations(map[string]string{smIngressConfigAnnotationKey: ""}).
						WithSpec(applycorev1.PodSpec().
							WithContainers(applycorev1.Container()).
							WithRuntimeClassName("contrast-cc").
							WithVolumes(Volume().
								WithName(*ServiceMeshProxy().VolumeMounts[0].Name).
								WithConfigMap(Volume().ConfigMap),
							),
						))),
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			_, err := AddServiceMesh(tc.d, ServiceMeshProxy())
			if tc.wantError {
				require.Error(err)
				return
			}
			require.NoError(err)

			if tc.skipServiceMesh {
				require.Empty(tc.d.Spec.Template.Spec.InitContainers)
				return
			}
			require.NotEmpty(tc.d.Spec.Template.Spec.InitContainers)
			require.Equal(*tc.d.Spec.Template.Spec.InitContainers[0].Name, *ServiceMeshProxy().Name)
			require.NotEmpty(tc.d.Spec.Template.Spec.InitContainers[0].VolumeMounts)
			require.Equal(*tc.d.Spec.Template.Spec.InitContainers[0].VolumeMounts[0].Name, *ServiceMeshProxy().VolumeMounts[0].Name)

			serviceMeshCount := 0
			for _, c := range tc.d.Spec.Template.Spec.InitContainers {
				if *c.Name == *ServiceMeshProxy().Name {
					serviceMeshCount++
				}
			}
			require.Equal(1, serviceMeshCount)

			require.NotEmpty(tc.d.Spec.Template.Spec.Volumes)
			serviceMeshVolumeCount := 0
			for _, v := range tc.d.Spec.Template.Spec.Volumes {
				if *v.Name == *ServiceMeshProxy().VolumeMounts[0].Name {
					serviceMeshVolumeCount++
				}
			}
			require.Equal(1, serviceMeshVolumeCount)
		})
	}
}

func TestAddImageStore_Regression(t *testing.T) {
	for name, tc := range map[string]struct {
		resource []byte
	}{
		"job":        {resource: job},
		"deployment": {resource: deployment},
		"cronjob":    {resource: cronjob},
		"daemonset":  {resource: daemonset},
		"pod":        {resource: pod},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			res, err := UnmarshalApplyConfigurations(tc.resource)
			require.NoError(err)
			res = PatchRuntimeHandlers(res, "contrast-cc")
			res = AddImageStore(res)
			encoded, err := EncodeResources(res...)
			require.NoError(err)
			require.Contains(string(encoded), "pvc-holder")
		})
	}
}
