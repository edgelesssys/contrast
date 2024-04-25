// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"errors"

	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// AddInitializer adds an initializer to a deployment.
func AddInitializer(
	deployment *applyappsv1.DeploymentApplyConfiguration,
	initializer *applycorev1.ContainerApplyConfiguration,
) (*applyappsv1.DeploymentApplyConfiguration, error) {
	if initializer == nil {
		return nil, errors.New("initializer is nil")
	}
	if deployment == nil {
		return nil, errors.New("deployment is nil")
	}
	if deployment.Spec == nil {
		return nil, errors.New("deployment.Spec is nil")
	}
	if deployment.Spec.Template == nil {
		return nil, errors.New("deployment.Spec.Template is nil")
	}
	if deployment.Spec.Template.Spec == nil {
		return nil, errors.New("deployment.Spec.Template.Spec is nil")
	}
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return nil, errors.New("deployment.Spec.Template.Spec.Containers is empty")
	}

	// Add the initializer as an init container.
	deployment.Spec.Template.Spec.WithInitContainers(
		initializer,
	)
	// Create the volume written by the initializer.
	deployment.Spec.Template.Spec.WithVolumes(Volume().
		WithName("tls-certs").
		WithEmptyDir(EmptyDirVolumeSource().Inner()),
	)
	// Add the volume mount written by the initializer to the worker container.
	deployment.Spec.Template.Spec.Containers[0].WithVolumeMounts(VolumeMount().
		WithName("tls-certs").
		WithMountPath("/tls-config"),
	)
	return deployment, nil
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

// AddServiceMesh adds a service mesh proxy to a deployment.
func AddServiceMesh(
	deployment *applyappsv1.DeploymentApplyConfiguration,
	serviceMeshProxy *applycorev1.ContainerApplyConfiguration,
	mode serviceMeshMode,
) (*applyappsv1.DeploymentApplyConfiguration, error) {
	if mode == ServiceMeshDisabled {
		return deployment, nil
	}
	if serviceMeshProxy == nil {
		return nil, errors.New("serviceMeshProxy is nil")
	}
	if deployment == nil {
		return nil, errors.New("deployment is nil")
	}
	if deployment.Spec == nil {
		return nil, errors.New("deployment.Spec is nil")
	}
	if deployment.Spec.Template == nil {
		return nil, errors.New("deployment.Spec.Template is nil")
	}
	if deployment.Spec.Template.Spec == nil {
		return nil, errors.New("deployment.Spec.Template.Spec is nil")
	}

	// Add the proxy as an init container.
	deployment.Spec.Template.Spec.WithInitContainers(
		serviceMeshProxy,
	)
	return deployment, nil
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
