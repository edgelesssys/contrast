// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"fmt"
	"strings"

	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applybatchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

const exposeServiceAnnotation = "contrast.edgeless.systems/expose-service"

// AddInitializer adds an initializer and its shared volume to the resource.
//
// If the resource does not contain a PodSpec, this function does nothing.
// This function is not idempotent.
func AddInitializer(
	resource any,
	initializer *applycorev1.ContainerApplyConfiguration,
) any {
	return MapPodSpec(resource, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		// Add the initializer as an init container.
		spec.WithInitContainers(
			initializer,
		)
		if len(initializer.VolumeMounts) < 1 {
			return spec
		}
		// Create the volume written by the initializer.
		spec.WithVolumes(Volume().
			WithName(*initializer.VolumeMounts[0].Name).
			WithEmptyDir(EmptyDirVolumeSource().Inner()),
		)
		// Add the volume mount written by the initializer to the worker container.
		spec.Containers[0].WithVolumeMounts(VolumeMount().
			WithName(*initializer.VolumeMounts[0].Name).
			WithMountPath(*initializer.VolumeMounts[0].MountPath))
		return spec
	})
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

// MapPodSpec applies a function to a PodSpec in a Kubernetes resource.
func MapPodSpec(resource any, f func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration) any {
	if resource == nil {
		return nil
	}
	switch r := resource.(type) {
	case *applybatchv1.CronJobApplyConfiguration:
		r.Spec.JobTemplate.Spec.Template.Spec = f(r.Spec.JobTemplate.Spec.Template.Spec)
	case *applyappsv1.DaemonSetApplyConfiguration:
		r.Spec.Template.Spec = f(r.Spec.Template.Spec)
	case *applyappsv1.DeploymentApplyConfiguration:
		r.Spec.Template.Spec = f(r.Spec.Template.Spec)
	case *applybatchv1.JobApplyConfiguration:
		r.Spec.Template.Spec = f(r.Spec.Template.Spec)
	case *applycorev1.PodApplyConfiguration:
		r.Spec = f(r.Spec)
	case *applyappsv1.ReplicaSetApplyConfiguration:
		r.Spec.Template.Spec = f(r.Spec.Template.Spec)
	case *applycorev1.ReplicationControllerApplyConfiguration:
		r.Spec.Template.Spec = f(r.Spec.Template.Spec)
	case *applyappsv1.StatefulSetApplyConfiguration:
		r.Spec.Template.Spec = f(r.Spec.Template.Spec)
	}
	return resource
}
