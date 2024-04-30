// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// ContrastRuntimeClass creates a new RuntimeClassConfig.
func ContrastRuntimeClass() *RuntimeClassConfig {
	r := RuntimeClass(runtimeHandler).
		WithHandler(runtimeHandler).
		WithLabels(map[string]string{"addonmanager.kubernetes.io/mode": "Reconcile"}).
		WithOverhead(Overhead(corev1.ResourceList{"memory": resource.MustParse("1152Mi")})).
		WithScheduling(Scheduling(map[string]string{"kubernetes.azure.com/kata-cc-isolation": "true"}))

	return &RuntimeClassConfig{r}
}

// NodeInstallerConfig wraps a DaemonSetApplyConfiguration for a node installer.
type NodeInstallerConfig struct {
	*applyappsv1.DaemonSetApplyConfiguration
}

// NodeInstaller constructs a node installer daemon set.
func NodeInstaller(namespace string) *NodeInstallerConfig {
	name := "contrast-node-installer"

	d := DaemonSet(name, namespace).
		WithLabels(map[string]string{"app.kubernetes.io/name": name}).
		WithSpec(DaemonSetSpec().
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": name}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": name}).
				WithAnnotations(map[string]string{"contrast.edgeless.systems/pod-role": "contrast-node-installer"}).
				WithSpec(PodSpec().
					WithHostPID(true).
					WithInitContainers(Container().
						WithName("installer").
						WithImage("ghcr.io/edgelesssys/contrast/node-installer:latest").
						WithResources(ResourceRequirements().
							WithMemoryLimitAndRequest(100),
						).
						WithSecurityContext(SecurityContext().WithPrivileged(true).SecurityContextApplyConfiguration).
						WithVolumeMounts(VolumeMount().
							WithName("host-mount").
							WithMountPath("/host")),
					).
					WithContainers(
						Container().
							WithName("pause").
							WithImage("k8s.gcr.io/pause").
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(10),
							),
					).
					WithVolumes(
						Volume().
							WithName("host-mount").
							WithHostPath(HostPathVolumeSource().
								WithPath("/").
								WithType(corev1.HostPathDirectory),
							),
					),
				),
			),
		)

	return &NodeInstallerConfig{d}
}

// PortForwarderConfig wraps a PodApplyConfiguration for a port forwarder.
type PortForwarderConfig struct {
	*applycorev1.PodApplyConfiguration
}

// PortForwarder constructs a port forwarder pod.
func PortForwarder(name, namespace string) *PortForwarderConfig {
	name = "port-forwarder-" + name

	p := Pod(name, namespace).
		WithLabels(map[string]string{"app.kubernetes.io/name": name}).
		WithSpec(PodSpec().
			WithContainers(
				Container().
					WithName("port-forwarder").
					WithImage("ghcr.io/edgelesssys/contrast/port-forwarder:latest").
					WithCommand("/bin/bash", "-c", "echo Starting port-forward with socat; exec socat -d -d TCP-LISTEN:${LISTEN_PORT},fork TCP:${FORWARD_HOST}:${FORWARD_PORT}").
					WithResources(ResourceRequirements().
						WithMemoryLimitAndRequest(50),
					),
			),
		)

	return &PortForwarderConfig{p}
}

// WithListenPort sets the port to listen on.
func (p *PortForwarderConfig) WithListenPort(port int32) *PortForwarderConfig {
	p.Spec.Containers[0].
		WithPorts(
			ContainerPort().
				WithContainerPort(port),
		).
		WithEnv(
			NewEnvVar("LISTEN_PORT", strconv.Itoa(int(port))),
		)
	return p
}

// WithForwardTarget sets the target host and port to forward to.
func (p *PortForwarderConfig) WithForwardTarget(host string, port int32) *PortForwarderConfig {
	p.Spec.Containers[0].
		WithEnv(
			NewEnvVar("FORWARD_HOST", host),
			NewEnvVar("FORWARD_PORT", strconv.Itoa(int(port))),
		)
	return p
}

// CoordinatorConfig wraps applyappsv1.DeploymentApplyConfiguration for a coordinator.
type CoordinatorConfig struct {
	*applyappsv1.DeploymentApplyConfiguration
}

// Coordinator constructs a new CoordinatorConfig.
func Coordinator(namespace string) *CoordinatorConfig {
	c := Deployment("coordinator", namespace).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "coordinator"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "coordinator"}).
				WithAnnotations(map[string]string{"contrast.edgeless.systems/pod-role": "coordinator", "prometheus.io/scrape": "true"}).
				WithSpec(PodSpec().
					WithRuntimeClassName(runtimeHandler).
					WithContainers(
						Container().
							WithName("coordinator").
							WithImage("ghcr.io/edgelesssys/contrast/coordinator:latest").
							WithPorts(
								ContainerPort().
									WithName("userapi").
									WithContainerPort(1313),
								ContainerPort().
									WithName("meshapi").
									WithContainerPort(7777),
								ContainerPort().
									WithName("prometheus").
									WithContainerPort(9102),
							).
							WithReadinessProbe(Probe().
								WithInitialDelaySeconds(1).
								WithPeriodSeconds(5).
								WithTCPSocket(TCPSocketAction().
									WithPort(intstr.FromInt(1313))),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(100),
							),
					),
				),
			),
		)

	return &CoordinatorConfig{c}
}

// WithImage sets the image of the coordinator.
func (c *CoordinatorConfig) WithImage(image string) *CoordinatorConfig {
	c.Spec.Template.Spec.Containers[0].WithImage(image)
	return c
}

// GetDeploymentConfig returns the DeploymentConfig of the coordinator.
func (c *CoordinatorConfig) GetDeploymentConfig() *DeploymentConfig {
	return &DeploymentConfig{c.DeploymentApplyConfiguration}
}

// ServiceForDeployment creates a service for a deployment by exposing the configured ports
// of the deployment's first container.
func ServiceForDeployment(d *applyappsv1.DeploymentApplyConfiguration) *applycorev1.ServiceApplyConfiguration {
	selector := d.Spec.Selector.MatchLabels
	ports := d.Spec.Template.Spec.Containers[0].Ports

	var ns string
	if d.Namespace != nil {
		ns = *d.Namespace
	}
	s := Service(*d.Name, ns).
		WithSpec(ServiceSpec().
			WithSelector(selector),
		)

	for _, p := range ports {
		s.Spec.WithPorts(
			ServicePort().
				WithName(*p.Name).
				WithPort(*p.ContainerPort).
				WithTargetPort(intstr.FromInt32(*p.ContainerPort)),
		)
	}

	return s
}

// PortForwarderForService creates a Pod that forwards network traffic to the given service.
//
// Port forwarders are named "port-forwarder-SVCNAME" and forward the first port in the ServiceSpec.
func PortForwarderForService(svc *applycorev1.ServiceApplyConfiguration) *applycorev1.PodApplyConfiguration {
	port := *svc.Spec.Ports[0].Port
	namespace := ""
	if svc.Namespace != nil {
		namespace = *svc.Namespace
	}
	return PortForwarder(*svc.Name, namespace).
		WithListenPort(port).
		WithForwardTarget(*svc.Name, port).
		PodApplyConfiguration
}

// Initializer creates a new InitializerConfig.
func Initializer() *applycorev1.ContainerApplyConfiguration {
	return applycorev1.Container().
		WithName("initializer").
		WithImage("ghcr.io/edgelesssys/contrast/initializer:latest").
		WithResources(ResourceRequirements().
			WithMemoryRequest(50),
		).
		WithEnv(NewEnvVar("COORDINATOR_HOST", "coordinator")).
		WithVolumeMounts(VolumeMount().
			WithName("tls-certs").
			WithMountPath("/tls-config"),
		)
}

// ServiceMeshProxy creates a new service mesh proxy sidecar container.
func ServiceMeshProxy() *applycorev1.ContainerApplyConfiguration {
	return applycorev1.Container().
		WithName("sidecar").
		WithImage("ghcr.io/edgelesssys/contrast/service-mesh-proxy:latest").
		WithRestartPolicy(corev1.ContainerRestartPolicyAlways).
		WithVolumeMounts(VolumeMount().
			WithName("tls-certs").
			WithMountPath("/tls-config"),
		).
		WithSecurityContext(SecurityContext().
			WithPrivileged(true).
			AddCapabilities("NET_ADMIN").
			SecurityContextApplyConfiguration,
		)
}
