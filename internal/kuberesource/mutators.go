// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applybatchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	applyrbacv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
)

const (
	exposeServiceAnnotation       = "contrast.edgeless.systems/expose-service"
	contrastRoleAnnotationKey     = "contrast.edgeless.systems/pod-role"
	skipInitializerAnnotationKey  = "contrast.edgeless.systems/skip-initializer"
	smIngressConfigAnnotationKey  = "contrast.edgeless.systems/servicemesh-ingress"
	smEgressConfigAnnotationKey   = "contrast.edgeless.systems/servicemesh-egress"
	smAdminInterfaceAnnotationKey = "contrast.edgeless.systems/servicemesh-admin-interface-port"
)

// AddInitializer adds an initializer and its shared volume to the resource.
//
// If the resource does not contain a PodSpec, this function does nothing.
// This function is idempotent.
func AddInitializer(
	resource any,
	initializer *applycorev1.ContainerApplyConfiguration,
) (res any, retErr error) {
	res = MapPodSpecWithMeta(resource, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
		if meta.Annotations[skipInitializerAnnotationKey] == "true" {
			return meta, spec
		}
		if spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			return meta, spec
		}

		// Remove already existing init containers with unique initializer name.
		spec.InitContainers = slices.DeleteFunc(spec.InitContainers, func(c applycorev1.ContainerApplyConfiguration) bool {
			return c.Name != nil && *c.Name == *initializer.Name
		})
		// Add the initializer as first init container.
		spec.InitContainers = append([]applycorev1.ContainerApplyConfiguration{*initializer}, spec.InitContainers...)
		// Initializer has to have a volume mount.
		// This should never error because the Initializer is configured to have a volume mount.
		if len(initializer.VolumeMounts) < 1 {
			retErr = fmt.Errorf("initializer volume mount list is empty")
			return nil, nil
		}

		retErr = ensureVolumeExists(spec, *initializer.VolumeMounts[0].Name)
		if retErr != nil {
			return nil, nil
		}

		for i := range spec.Containers {
			addOrReplaceVolumeMount(&spec.Containers[i], initializer.VolumeMounts[0])
		}

		for i := range spec.InitContainers {
			addOrReplaceVolumeMount(&spec.InitContainers[i], initializer.VolumeMounts[0])
		}
		return meta, spec
	})
	return res, retErr
}

func addOrReplaceVolumeMount(container *applycorev1.ContainerApplyConfiguration, volumeMount applycorev1.VolumeMountApplyConfiguration) {
	// Remove already existing volume mounts on the worker containers with unique volume mount name.
	container.VolumeMounts = slices.DeleteFunc(container.VolumeMounts, func(v applycorev1.VolumeMountApplyConfiguration) bool {
		return v.Name != nil && *v.Name == *volumeMount.Name
	})

	// Add the volume mount written by the initializer to the worker container.
	container.WithVolumeMounts(&volumeMount)
}

type serviceMeshMode string

const (
	// ServiceMeshIngressEgress sets the service mesh mode to ingress and egress.
	ServiceMeshIngressEgress serviceMeshMode = "service-mesh-ingress-egress"
	// ServiceMeshDisabled disables the service mesh.
	ServiceMeshDisabled serviceMeshMode = "service-mesh-disabled"
)

// AddServiceMesh adds a service mesh proxy to the resource with the proxy
// configuration given in the object annotations.
//
// If the resource does not contain a PodSpec, this function does nothing.
// This function is idempotent.
func AddServiceMesh(
	resource any,
	serviceMeshProxy *applycorev1.ContainerApplyConfiguration,
) (res any, retErr error) {
	res = MapPodSpecWithMeta(resource, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
		if spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			return meta, spec
		}

		ingressConfig, ingressOk := meta.Annotations[smIngressConfigAnnotationKey]
		egressConfig, egressOk := meta.Annotations[smEgressConfigAnnotationKey]
		portAnnotation, portOk := meta.Annotations[smAdminInterfaceAnnotationKey]

		// Don't change anything if automatic service mesh injection isn't enabled.
		if !ingressOk && !egressOk && !portOk {
			return meta, spec
		}

		// Remove already existing init containers with unique service mesh name.
		spec.InitContainers = slices.DeleteFunc(spec.InitContainers, func(c applycorev1.ContainerApplyConfiguration) bool {
			return c.Name != nil && *c.Name == *serviceMeshProxy.Name
		})

		retErr = ensureVolumeExists(spec, *serviceMeshProxy.VolumeMounts[0].Name)
		if retErr != nil {
			return nil, nil
		}

		if portAnnotation != "" {
			port, err := strconv.Atoi(portAnnotation)
			if err != nil {
				retErr = fmt.Errorf("parsing service mesh admin interface port: %w", err)
				return nil, nil
			}

			serviceMeshProxy.
				WithEnv(NewEnvVar("CONTRAST_ADMIN_PORT", portAnnotation)).
				WithPorts(
					ContainerPort().
						WithName("contrast-admin").
						WithContainerPort(int32(port)),
				)
		}

		if ingressConfig != "" {
			serviceMeshProxy.WithEnv(NewEnvVar("CONTRAST_INGRESS_PROXY_CONFIG", ingressConfig))
		}
		if egressConfig != "" {
			serviceMeshProxy.WithEnv(NewEnvVar("CONTRAST_EGRESS_PROXY_CONFIG", egressConfig))
		}

		return meta, spec.WithInitContainers(serviceMeshProxy)
	})
	return res, retErr
}

func ensureVolumeExists(spec *applycorev1.PodSpecApplyConfiguration, volumeName string) error {
	// Existing volume with unique name has to be of type EmptyDir.
	for _, volume := range spec.Volumes {
		if volume.Name != nil && *volume.Name == volumeName {
			if volume.EmptyDir == nil {
				return fmt.Errorf("volume %s has to be of type EmptyDir", *volume.Name)
			}
			return nil
		}
	}
	// Create the volume written if it not already exists.
	spec.WithVolumes(Volume().
		WithName(volumeName).
		WithEmptyDir(EmptyDirVolumeSource().Inner()),
	)
	return nil
}

// AddPortForwarders adds a port-forwarder for each Service.
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

// AddDmesg adds a dmesg logging container.
func AddDmesg(resources []any) []any {
	dmesgContainer := Container().
		WithName("dmesg").
		WithImage("ghcr.io/edgelesssys/contrast/dmesg:v0.0.1@sha256:6ad6bbb5735b84b10af42d2441e8d686b1d9a6cbf096b53842711ef5ddabd28d").
		WithSecurityContext(SecurityContext().
			WithPrivileged(true).SecurityContextApplyConfiguration)

	addDmesg := func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration,
	) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
		if spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			return meta, spec
		}
		spec.Containers = append(spec.Containers, *dmesgContainer)
		return meta, spec
	}

	var out []any
	for _, resource := range resources {
		out = append(out, MapPodSpecWithMeta(resource, addDmesg))
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
				if obj.Spec != nil {
					obj.Spec.WithType("LoadBalancer")
				}
			}
		}
		out = append(out, resource)
	}
	return out
}

// AddLogging modifies Contrast Coordinators among the resources to enable debug logging.
func AddLogging(resources []any, level, subsystem string) []any {
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applyappsv1.StatefulSetApplyConfiguration:
			if r.Spec != nil && r.Spec.Template != nil &&
				r.Spec.Template.Annotations["contrast.edgeless.systems/pod-role"] == "coordinator" {
				r.Spec.Template.Spec.Containers[0].WithEnv(
					NewEnvVar("CONTRAST_LOG_LEVEL", level),
					NewEnvVar("CONTRAST_LOG_SUBSYSTEMS", subsystem),
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
			for i := range len(spec.InitContainers) {
				if spec.InitContainers[i].Image != nil {
					if replacement, ok := replacements[*spec.InitContainers[i].Image]; ok {
						spec.InitContainers[i].Image = &replacement
					}
				}
			}
			for i := range len(spec.Containers) {
				if spec.Containers[i].Image != nil {
					if replacement, ok := replacements[*spec.Containers[i].Image]; ok {
						spec.Containers[i].Image = &replacement
					}
				}
			}
			return spec
		}))
	}
	return out
}

// PatchRuntimeHandlers replaces runtime handlers in a set of resources.
func PatchRuntimeHandlers(resources []any, runtimeHandler string) []any {
	var out []any
	for _, resource := range resources {
		out = append(out, MapPodSpec(resource, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
			spec.RuntimeClassName = &runtimeHandler
			return spec
		}))
	}
	return out
}

// PatchNamespaces replaces namespaces in a set of resources.
func PatchNamespaces(resources []any, namespace string) []any {
	nsPtr := &namespace
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
		case *applyrbacv1.ClusterRoleBindingApplyConfiguration:
			if namespace != "" {
				*r.Name = fmt.Sprintf("%s-%s", *r.Name, *nsPtr)
			}
			for i := range len(r.Subjects) {
				r.Subjects[i].Namespace = nsPtr
			}
		}
	}
	return resources
}

// PatchServiceMeshAdminInterface activates the admin interface on the
// specified port for all Service Mesh components in a set of resources.
func PatchServiceMeshAdminInterface(resources []any, port int32) []any {
	var out []any
	for _, resource := range resources {
		out = append(out, MapPodSpecWithMeta(resource, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
			_, ingressOk := meta.Annotations[smIngressConfigAnnotationKey]
			_, egressOk := meta.Annotations[smEgressConfigAnnotationKey]
			if ingressOk || egressOk {
				meta.WithAnnotations(map[string]string{smAdminInterfaceAnnotationKey: fmt.Sprint(port)})
				meta.Annotations[smIngressConfigAnnotationKey] += fmt.Sprintf("##admin#%d#true", port)
			}
			return meta, spec
		}))
	}
	return out
}

// PatchCoordinatorMetrics enables Coordinator metrics on the specified port.
func PatchCoordinatorMetrics(resources []any, port int32) []any {
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applyappsv1.StatefulSetApplyConfiguration:
			if r.Spec != nil && r.Spec.Template != nil && r.Spec.Template.Spec != nil &&
				len(r.Spec.Template.Spec.Containers) > 0 &&
				r.Spec.Template.Annotations[contrastRoleAnnotationKey] == "coordinator" {
				r.Spec.Template.Spec.Containers[0].WithEnv(NewEnvVar("CONTRAST_METRICS_PORT", fmt.Sprint(port)))
				r.Spec.Template.Spec.Containers[0].WithPorts(
					ContainerPort().
						WithName("prometheus").
						WithContainerPort(port),
				)
			}
		}
	}
	return resources
}

// MapPodSpecWithMeta applies a function to a PodSpec in a Kubernetes resource,
// and its corresponding object metadata.
func MapPodSpecWithMeta(
	resource any,
	f func(
		meta *applymetav1.ObjectMetaApplyConfiguration,
		spec *applycorev1.PodSpecApplyConfiguration,
	) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration),
) any {
	if resource == nil {
		return nil
	}
	switch r := resource.(type) {
	case *applybatchv1.CronJobApplyConfiguration:
		if r.ObjectMetaApplyConfiguration != nil &&
			r.Spec != nil &&
			r.Spec.JobTemplate != nil &&
			r.Spec.JobTemplate.Spec != nil &&
			r.Spec.JobTemplate.Spec.Template != nil &&
			r.Spec.JobTemplate.Spec.Template.Spec != nil {
			r.ObjectMetaApplyConfiguration, r.Spec.JobTemplate.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.JobTemplate.Spec.Template.Spec)
		}
	case *applyappsv1.DaemonSetApplyConfiguration:
		if r.ObjectMetaApplyConfiguration != nil &&
			r.Spec != nil &&
			r.Spec.Template != nil &&
			r.Spec.Template.Spec != nil {
			r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
		}
	case *applyappsv1.DeploymentApplyConfiguration:
		if r.ObjectMetaApplyConfiguration != nil &&
			r.Spec != nil &&
			r.Spec.Template != nil &&
			r.Spec.Template.Spec != nil {
			r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
		}
	case *applybatchv1.JobApplyConfiguration:
		if r.ObjectMetaApplyConfiguration != nil &&
			r.Spec != nil &&
			r.Spec.Template != nil &&
			r.Spec.Template.Spec != nil {
			r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
		}
	case *applycorev1.PodApplyConfiguration:
		if r.ObjectMetaApplyConfiguration != nil &&
			r.Spec != nil {
			r.ObjectMetaApplyConfiguration, r.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec)
		}
	case *applyappsv1.ReplicaSetApplyConfiguration:
		if r.ObjectMetaApplyConfiguration != nil &&
			r.Spec != nil &&
			r.Spec.Template != nil &&
			r.Spec.Template.Spec != nil {
			r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
		}
	case *applycorev1.ReplicationControllerApplyConfiguration:
		if r.ObjectMetaApplyConfiguration != nil &&
			r.Spec != nil &&
			r.Spec.Template != nil &&
			r.Spec.Template.Spec != nil {
			r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
		}
	case *applyappsv1.StatefulSetApplyConfiguration:
		if r.ObjectMetaApplyConfiguration != nil &&
			r.Spec != nil &&
			r.Spec.Template != nil &&
			r.Spec.Template.Spec != nil {
			r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec = f(r.ObjectMetaApplyConfiguration, r.Spec.Template.Spec)
		}
	}
	return resource
}

// MapPodSpec applies a function to a PodSpec in a Kubernetes resource.
func MapPodSpec(resource any, f func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration) any {
	return MapPodSpecWithMeta(
		resource,
		func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (
			*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration,
		) {
			return meta, f(spec)
		})
}
