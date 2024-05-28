// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"fmt"
	"slices"
	"strings"

	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applybatchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

const (
	exposeServiceAnnotation      = "contrast.edgeless.systems/expose-service"
	contrastRoleAnnotationKey    = "contrast.edgeless.systems/pod-role"
	skipInitializerAnnotationKey = "contrast.edgeless.systems/skip-initializer"
)

// AddInitializer adds an initializer and its shared volume to the resource.
//
// If the resource does not contain a PodSpec, this function does nothing.
// This function is idempotent.
func AddInitializer(
	resource any,
	initializer *applycorev1.ContainerApplyConfiguration,
) (res any, err error) {
	res = MapPodSpecWithMeta(resource, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if meta.Annotations[skipInitializerAnnotationKey] == "true" {
			return spec
		}
		if spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			return spec
		}

		// Remove already existing init containers with unique initializer name.
		spec.InitContainers = slices.DeleteFunc(spec.InitContainers, func(c applycorev1.ContainerApplyConfiguration) bool {
			return *c.Name == *initializer.Name
		})
		// Add the initializer as first init container.
		spec.InitContainers = append([]applycorev1.ContainerApplyConfiguration{*initializer}, spec.InitContainers...)
		if len(initializer.VolumeMounts) < 1 {
			return spec
		}

		// Existing volume with unique name has to be of type EmptyDir.
		var volumeExists bool
		for _, volume := range spec.Volumes {
			if *volume.Name == *initializer.VolumeMounts[0].Name {
				volumeExists = true
				if volume.EmptyDir == nil {
					err = fmt.Errorf("volume %s has to be of type EmptyDir", *volume.Name)
					return nil
				}
			}
		}
		// Create the volume written by the initializer if it not already exists.
		if !volumeExists {
			spec.WithVolumes(Volume().
				WithName(*initializer.VolumeMounts[0].Name).
				WithEmptyDir(EmptyDirVolumeSource().Inner()),
			)
		}

		// Remove already existing volume mounts on the worker container with unique volume mount name.
		for i := range spec.Containers {
			spec.Containers[i].VolumeMounts = slices.DeleteFunc(spec.Containers[i].VolumeMounts, func(v applycorev1.VolumeMountApplyConfiguration) bool {
				return *v.Name == *initializer.VolumeMounts[0].Name
			})

			// Add the volume mount written by the initializer to the worker container.
			spec.Containers[i].WithVolumeMounts(VolumeMount().
				WithName(*initializer.VolumeMounts[0].Name).
				WithMountPath(*initializer.VolumeMounts[0].MountPath))
		}
		return spec
	})
	return res, err
}

type serviceMeshMode string

const (
	// ServiceMeshIngressEgress sets the service mesh mode to ingress and egress.
	ServiceMeshIngressEgress serviceMeshMode = "service-mesh-ingress-egress"
	// ServiceMeshEgress sets the service mesh mode to egress.
	ServiceMeshEgress serviceMeshMode = "service-mesh-egress"
	// ServiceMeshDisabled disables the service mesh.
	ServiceMeshDisabled serviceMeshMode = "service-mesh-disabled"
)

// AddServiceMesh adds a service mesh proxy to the resource.
//
// If the resource does not contain a PodSpec, this function does nothing.
// This function is not idempotent.
func AddServiceMesh(
	resource any,
	serviceMeshProxy *applycorev1.ContainerApplyConfiguration,
) any {
	return MapPodSpec(resource, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		return spec.WithInitContainers(serviceMeshProxy)
	})
}

// AddPortForwarders adds a port-forwarder for each Service resource.
func AddPortForwarders(resources []any) []any {
	var out []any
	for _, resource := range resources {
		switch obj := resource.(type) {
		case *applycorev1.ServiceApplyConfiguration:
			out = append(out, PortForwarderForService(obj))
		}
		out = append(out, resource)
	}
	return out
}

// AddLoadBalancers adds a load balancer to each Service resource.
func AddLoadBalancers(resources []any) []any {
	var out []any
	for _, resource := range resources {
		switch obj := resource.(type) {
		case *applycorev1.ServiceApplyConfiguration:
			if obj.Annotations[exposeServiceAnnotation] == "true" {
				obj.Spec.WithType("LoadBalancer")
			}
		}
		out = append(out, resource)
	}
	return out
}

// AddLogging modifies Contrast Coordinators among the resources to enable debug logging.
func AddLogging(resources []any, level string) []any {
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applyappsv1.DeploymentApplyConfiguration:
			if r.Spec.Template.Annotations["contrast.edgeless.systems/pod-role"] == "coordinator" {
				r.Spec.Template.Spec.Containers[0].WithEnv(
					NewEnvVar("CONTRAST_LOG_LEVEL", level),
					NewEnvVar("CONTRAST_LOG_SUBSYSTEMS", "*"),
				)
			}
		}
	}
	return resources
}

// PatchImages replaces images in a set of resources.
func PatchImages(resources []any, replacements map[string]string) []any {
	var out []any
	for _, resource := range resources {
		out = append(out, MapPodSpec(resource, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
			for i := 0; i < len(spec.InitContainers); i++ {
				if replacement, ok := replacements[*spec.InitContainers[i].Image]; ok {
					spec.InitContainers[i].Image = &replacement
				}
			}
			for i := 0; i < len(spec.Containers); i++ {
				if replacement, ok := replacements[*spec.Containers[i].Image]; ok {
					spec.Containers[i].Image = &replacement
				}
			}
			return spec
		}))
	}
	return out
}

// PatchNamespaces replaces namespaces in a set of resources.
func PatchNamespaces(resources []any, namespace string) []any {
	var nsPtr *string
	if namespace != "" {
		nsPtr = &namespace
	}
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applycorev1.PodApplyConfiguration:
			r.Namespace = nsPtr
		case *applyappsv1.DeploymentApplyConfiguration:
			r.Namespace = nsPtr
		case *applyappsv1.DaemonSetApplyConfiguration:
			r.Namespace = nsPtr
		case *applyappsv1.StatefulSetApplyConfiguration:
			r.Namespace = nsPtr
		case *applycorev1.ServiceApplyConfiguration:
			r.Namespace = nsPtr
		case *applycorev1.ServiceAccountApplyConfiguration:
			r.Namespace = nsPtr
		}
	}
	return resources
}

// PatchServiceMeshAdminInterface activates the admin interface on the
// specified port for all Service Mesh components in a set of resources.
func PatchServiceMeshAdminInterface(resources []any, port int32) []any {
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applyappsv1.DeploymentApplyConfiguration:
			for i := 0; i < len(r.Spec.Template.Spec.InitContainers); i++ {
				// TODO(davidweisse): find service mesh containers by unique name as specified in RFC 005.
				if strings.Contains(*r.Spec.Template.Spec.InitContainers[i].Image, "service-mesh-proxy") {
					r.Spec.Template.Spec.InitContainers[i] = *r.Spec.Template.Spec.InitContainers[i].
						WithEnv(NewEnvVar("EDG_ADMIN_PORT", fmt.Sprint(port))).
						WithPorts(
							ContainerPort().
								WithName("admin-interface").
								WithContainerPort(port),
						)
					ingressProxyConfig := false
					for j, env := range r.Spec.Template.Spec.InitContainers[i].Env {
						if *env.Name == "EDG_INGRESS_PROXY_CONFIG" {
							ingressProxyConfig = true
							env.WithValue(fmt.Sprintf("%s##admin#%d#true", *env.Value, port))
							r.Spec.Template.Spec.InitContainers[i].Env[j] = env
							break
						}
					}
					if !ingressProxyConfig {
						r.Spec.Template.Spec.InitContainers[i].WithEnv(
							NewEnvVar("EDG_INGRESS_PROXY_CONFIG", fmt.Sprintf("admin#%d#true", port)),
						)
					}
				}
			}
		}
	}
	return resources
}

// MapPodSpecWithMeta applies a function to a PodSpec in a Kubernetes resource.
func MapPodSpecWithMeta(
	resource any,
	f func(
		meta *applymetav1.ObjectMetaApplyConfiguration,
		spec *applycorev1.PodSpecApplyConfiguration,
	) *applycorev1.PodSpecApplyConfiguration,
) any {
	if resource == nil {
		return nil
	}
	switch r := resource.(type) {
	case *applybatchv1.CronJobApplyConfiguration:
		r.Spec.JobTemplate.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.JobTemplate.Spec.Template.Spec)
	case *applyappsv1.DaemonSetApplyConfiguration:
		r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
	case *applyappsv1.DeploymentApplyConfiguration:
		r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
	case *applybatchv1.JobApplyConfiguration:
		r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
	case *applycorev1.PodApplyConfiguration:
		r.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec)
	case *applyappsv1.ReplicaSetApplyConfiguration:
		r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
	case *applycorev1.ReplicationControllerApplyConfiguration:
		r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
	case *applyappsv1.StatefulSetApplyConfiguration:
		r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
	}
	return resource
}

// MapPodSpec applies a function to a PodSpec in a Kubernetes resource.
func MapPodSpec(resource any, f func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration) any {
	return MapPodSpecWithMeta(resource, func(_ *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		return f(spec)
	})
}
