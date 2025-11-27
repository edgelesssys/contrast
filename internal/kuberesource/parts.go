// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"fmt"
	"strconv"

	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyrbacv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
)

// ContrastRuntimeClass creates a new RuntimeClassConfig.
func ContrastRuntimeClass(platform platforms.Platform) (*RuntimeClassConfig, error) {
	runtimeHandler, err := manifest.RuntimeHandler(platform)
	if err != nil {
		return nil, fmt.Errorf("getting default runtime handler: %w", err)
	}

	// Consists of the default VM memory, 70MiB for the Kata shim and 100MiB for qemu overhead.
	memoryOverhead := platforms.DefaultMemoryInMebiBytes(platform) + 170

	r := RuntimeClass(runtimeHandler).
		WithHandler(runtimeHandler).
		WithLabels(map[string]string{"addonmanager.kubernetes.io/mode": "Reconcile"}).
		WithOverhead(Overhead(corev1.ResourceList{"memory": *resource.NewQuantity(int64(memoryOverhead)*1024*1024, resource.BinarySI)}))

	return &RuntimeClassConfig{r}, nil
}

// NodeInstaller constructs a node installer daemon set.
func NodeInstaller(namespace string, platform platforms.Platform) (*applyappsv1.DaemonSetApplyConfiguration, error) {
	runtimeHandler, err := manifest.RuntimeHandler(platform)
	if err != nil {
		return nil, fmt.Errorf("getting default runtime handler: %w", err)
	}

	name := fmt.Sprintf("%s-nodeinstaller", runtimeHandler)

	var nodeInstallerImageURL string
	switch {
	case platforms.IsQEMU(platform) && platforms.IsGPU(platform):
		nodeInstallerImageURL = "ghcr.io/edgelesssys/contrast/node-installer-kata-gpu:latest"
	case platforms.IsQEMU(platform):
		nodeInstallerImageURL = "ghcr.io/edgelesssys/contrast/node-installer-kata:latest"
	default:
		return nil, fmt.Errorf("unsupported platform %q", platform)
	}

	d := DaemonSet(name, namespace).
		WithLabels(map[string]string{"app.kubernetes.io/name": name}).
		WithSpec(DaemonSetSpec().
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": name}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": name}).
				WithAnnotations(map[string]string{
					"contrast.edgeless.systems/pod-role": "contrast-node-installer",
					"contrast.edgeless.systems/platform": platform.String(),
				}).
				WithSpec(PodSpec().
					WithHostPID(true).
					WithInitContainers(Container().
						WithName("installer").
						WithImage(nodeInstallerImageURL).
						WithResources(ResourceRequirements().
							// node-installer doesn't run in as VM, no need to set a limit here.
							WithMemoryRequest(700),
						).
						WithSecurityContext(SecurityContext().WithPrivileged(true).SecurityContextApplyConfiguration).
						WithVolumeMounts(
							VolumeMount().
								WithName("host-mount").
								WithMountPath("/host"),
							VolumeMount().
								WithName("var-run-dbus-socket").
								WithMountPath("/var/run/dbus/system_bus_socket"),
							VolumeMount().
								WithName("target-config").
								WithMountPath("/target-config"),
							VolumeMount().
								WithName("imagepuller-config").
								WithMountPath("/imagepuller-config"),
						).
						WithCommand("/bin/node-installer", platform.String()),
					).
					WithContainers(
						Container().
							WithName("pause").
							WithImage("registry.k8s.io/pause:3.6@sha256:3d380ca8864549e74af4b29c10f9cb0956236dfb01c40ca076fb6c37253234db"),
					).
					WithVolumes(
						Volume().
							WithName("host-mount").
							WithHostPath(HostPathVolumeSource().
								WithPath("/").
								WithType(corev1.HostPathDirectory),
							),
						Volume().
							WithName("var-run-dbus-socket").
							WithHostPath(HostPathVolumeSource().
								WithPath("/var/run/dbus/system_bus_socket").
								WithType(corev1.HostPathSocket),
							),
						Volume().
							WithName("target-config").
							WithConfigMap(ConfigMapVolumeSource().
								WithName("contrast-node-installer-target-config").
								WithOptional(true),
							),
						Volume().
							WithName("imagepuller-config").
							WithSecret(SecretVolumeSource().
								WithSecretName("contrast-node-installer-imagepuller-config").
								WithOptional(true),
							),
					),
				),
			),
		)

	return d, nil
}

// NodeInstallerTargetConfig returns a ConfigMap for the passed target.
func NodeInstallerTargetConfig(target string) (*applycorev1.ConfigMapApplyConfiguration, error) {
	ns := "kube-system"
	switch target {
	case "k3s":
		return applycorev1.ConfigMap("contrast-node-installer-target-config", ns).
			WithData(map[string]string{
				"containerd-config-path": "var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl",
				"systemd-unit-name":      "k3s.service,k3s-agent.service",
			}), nil
	default:
		return nil, fmt.Errorf("unsupported target %q", target)
	}
}

// NodeInstallerImagePullerSecret returns an imagepuller Secret for the passed namespace.
func NodeInstallerImagePullerSecret(namespace string, content []byte) *applycorev1.SecretApplyConfiguration {
	return applycorev1.Secret("contrast-node-installer-imagepuller-config", namespace).
		WithType(corev1.SecretTypeOpaque).
		WithData(map[string][]byte{
			"contrast-imagepuller.toml": content,
		})
}

// PortForwarderConfig wraps a PodApplyConfiguration for a port forwarder.
type PortForwarderConfig struct {
	*applycorev1.PodApplyConfiguration
}

// WithForwardTarget sets the target host to forward to.
func (p *PortForwarderConfig) WithForwardTarget(host string) *PortForwarderConfig {
	p.Spec.Containers[0].WithEnv(NewEnvVar("FORWARD_HOST", host))
	return p
}

const portForwarderScript = `echo Starting port-forward with socat >&2
handler() {
  echo "Received SIGTERM, forwarding to children" >&2
  kill -TERM -1
}
trap handler TERM
set -x
for port in ${LISTEN_PORTS}; do
  socat -d -d TCP-LISTEN:$port,fork TCP:${FORWARD_HOST}:$port,connect-timeout=2,retry=1 &
done
wait
`

// PortForwarder constructs a port forwarder pod for multiple ports.
func PortForwarder(name, namespace string) *PortForwarderConfig {
	name = "port-forwarder-" + name

	p := Pod(name, namespace).
		WithLabels(map[string]string{"app.kubernetes.io/name": name}).
		WithSpec(PodSpec().
			WithContainers(
				Container().
					WithName("port-forwarder").
					WithImage("ghcr.io/edgelesssys/contrast/port-forwarder:latest").
					WithCommand("/bin/bash", "-c", portForwarderScript).
					WithResources(ResourceRequirements().
						WithMemoryLimitAndRequest(50),
					),
			),
		)

	return &PortForwarderConfig{p}
}

// WithListenPorts sets multiple ports to listen on.
func (p *PortForwarderConfig) WithListenPorts(ports []int32) *PortForwarderConfig {
	var containerPorts []*applycorev1.ContainerPortApplyConfiguration
	var envVar string
	for _, port := range ports {
		containerPorts = append(containerPorts, ContainerPort().WithContainerPort(port))
		envVar += " " + strconv.Itoa(int(port))
	}
	p.Spec.Containers[0].
		WithPorts(containerPorts...).
		WithEnv(NewEnvVar("LISTEN_PORTS", envVar))
	return p
}

// CoordinatorConfig wraps applyappsv1.DeploymentApplyConfiguration for a coordinator.
type CoordinatorConfig struct {
	*applyappsv1.StatefulSetApplyConfiguration
	*applycorev1.ServiceAccountApplyConfiguration
	*applyrbacv1.RoleApplyConfiguration
	*applyrbacv1.RoleBindingApplyConfiguration
}

// Coordinator constructs a new CoordinatorConfig.
func Coordinator(namespace string) *CoordinatorConfig {
	c := StatefulSet("coordinator", namespace).
		WithSpec(StatefulSetSpec().
			WithReplicas(1).
			WithServiceName("coordinator").
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "coordinator"}),
			).
			WithPersistentVolumeClaimRetentionPolicy(applyappsv1.StatefulSetPersistentVolumeClaimRetentionPolicy().
				WithWhenDeleted(appsv1.DeletePersistentVolumeClaimRetentionPolicyType).
				WithWhenScaled(appsv1.DeletePersistentVolumeClaimRetentionPolicyType)). // TODO(burgerdev): this should be RETAIN for released coordinators.
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "coordinator"}).
				WithAnnotations(map[string]string{"contrast.edgeless.systems/pod-role": "coordinator"}).
				WithSpec(PodSpec().
					WithServiceAccountName("coordinator").
					WithContainers(
						Container().
							WithName("coordinator").
							WithImage("ghcr.io/edgelesssys/contrast/coordinator:latest").
							WithSecurityContext(SecurityContext().
								WithCapabilities(applycorev1.Capabilities().
									WithAdd(
										"NET_ADMIN", // Needed for removing the default deny iptables rule.
									),
								),
							).
							WithPorts(
								ContainerPort().
									WithName("userapi").
									WithContainerPort(1313),
								ContainerPort().
									WithName("httpapi").
									WithContainerPort(1314),
								ContainerPort().
									WithName("meshapi").
									WithContainerPort(7777),
								ContainerPort().
									WithName("transitapi").
									WithContainerPort(8200),
							).
							WithStartupProbe(Probe().
								WithInitialDelaySeconds(1).
								WithPeriodSeconds(1).
								WithFailureThreshold(60).
								WithHTTPGet(applycorev1.HTTPGetAction().
									WithPort(intstr.FromInt(9102)).
									WithPath("/probe/startup")),
							).
							WithLivenessProbe(Probe().
								WithPeriodSeconds(10).
								WithFailureThreshold(3).
								WithHTTPGet(applycorev1.HTTPGetAction().
									WithPort(intstr.FromInt(9102)).
									WithPath("/probe/liveness")),
							).
							WithReadinessProbe(Probe().
								WithPeriodSeconds(5).
								WithHTTPGet(applycorev1.HTTPGetAction().
									WithPort(intstr.FromInt(9102)).
									WithPath("/probe/readiness")),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					).
					WithAffinity(
						applycorev1.Affinity().
							WithPodAntiAffinity(
								applycorev1.PodAntiAffinity().
									WithPreferredDuringSchedulingIgnoredDuringExecution(
										applycorev1.WeightedPodAffinityTerm().
											WithWeight(100).
											WithPodAffinityTerm(
												applycorev1.PodAffinityTerm().
													WithLabelSelector(
														LabelSelector().
															WithMatchLabels(map[string]string{"contrast.edgeless.systems/pod-role": "coordinator"}),
													).
													WithTopologyKey("kubernetes.io/hostname"),
											),
									),
							),
					),
				),
			),
		)

	sa := ServiceAccount("coordinator", namespace).ServiceAccountApplyConfiguration

	role := Role("coordinator", namespace).
		WithRules(
			applyrbacv1.PolicyRule().
				WithAPIGroups("").
				WithResources("configmaps").
				WithVerbs("get", "create", "update", "watch"),
			applyrbacv1.PolicyRule().
				WithAPIGroups("").
				WithResources("pods").
				WithVerbs("get", "list"),
		)

	roleBinding := RoleBinding("coordinator", namespace).
		WithSubjects(
			applyrbacv1.Subject().
				WithKind("ServiceAccount").
				WithName("coordinator").
				WithNamespace(namespace),
		).
		WithRoleRef(
			applyrbacv1.RoleRef().
				WithKind("Role").
				WithName("coordinator").
				WithAPIGroup("rbac.authorization.k8s.io"),
		)

	return &CoordinatorConfig{
		StatefulSetApplyConfiguration:    c,
		ServiceAccountApplyConfiguration: sa,
		RoleApplyConfiguration:           role,
		RoleBindingApplyConfiguration:    roleBinding,
	}
}

// WithImage sets the image of the coordinator.
func (c *CoordinatorConfig) WithImage(image string) *CoordinatorConfig {
	c.Spec.Template.Spec.Containers[0].WithImage(image)
	return c
}

// ServiceForDeployment creates a service for a deployment by exposing the configured ports
// of the deployment's first container.
func ServiceForDeployment(d *applyappsv1.DeploymentApplyConfiguration) *applycorev1.ServiceApplyConfiguration {
	selector := d.Spec.Selector.MatchLabels
	ports := getPorts(d.Spec.Template.Spec)

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

// ServiceForStatefulSet creates a service for a StatefulSet by exposing the configured ports
// of the first container.
func ServiceForStatefulSet(s *applyappsv1.StatefulSetApplyConfiguration) *applycorev1.ServiceApplyConfiguration {
	selector := s.Spec.Selector.MatchLabels
	ports := getPorts(s.Spec.Template.Spec)

	var ns string
	if s.Namespace != nil {
		ns = *s.Namespace
	}
	svc := Service(*s.Name, ns).
		WithSpec(ServiceSpec().
			WithSelector(selector),
		)

	for _, p := range ports {
		svc.Spec.WithPorts(
			ServicePort().
				WithName(*p.Name).
				WithPort(*p.ContainerPort).
				WithTargetPort(intstr.FromInt32(*p.ContainerPort)),
		)
	}

	return svc
}

// PortForwarderForService creates a Pod that forwards network traffic to the given service.
//
// Port forwarders are named "port-forwarder-SVCNAME" and forward all TCP ports in the ServiceSpec.
func PortForwarderForService(svc *applycorev1.ServiceApplyConfiguration) (*applycorev1.PodApplyConfiguration, error) {
	namespace := ""
	if svc.Namespace != nil {
		namespace = *svc.Namespace
	}

	var ports []int32
	for _, port := range svc.Spec.Ports {
		if port.Protocol == nil || *port.Protocol == corev1.ProtocolTCP {
			ports = append(ports, *port.Port)
		}
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("no TCP ports in service spec")
	}

	forwarder := PortForwarder(*svc.Name, namespace).
		WithListenPorts(ports).
		WithForwardTarget(*svc.Name)

	return forwarder.PodApplyConfiguration, nil
}

// Initializer creates a new InitializerConfig.
func Initializer(coordinatorHost string) *applycorev1.ContainerApplyConfiguration {
	return applycorev1.Container().
		WithName("contrast-initializer").
		WithImage("ghcr.io/edgelesssys/contrast/initializer:latest").
		WithResources(ResourceRequirements().
			WithMemoryLimitAndRequest(50),
		).
		WithEnv(NewEnvVar("COORDINATOR_HOST", coordinatorHost)).
		WithVolumeMounts(VolumeMount().
			WithName("contrast-secrets").
			WithMountPath("/contrast"),
		).
		WithSecurityContext(
			SecurityContext().
				WithCapabilities(
					applycorev1.Capabilities().
						WithAdd(
							"NET_ADMIN", // Needed for removing the default deny iptables rule.
						),
				),
		)
}

// ServiceMeshProxy creates a new service mesh proxy sidecar container.
func ServiceMeshProxy() *applycorev1.ContainerApplyConfiguration {
	return applycorev1.Container().
		WithName("contrast-service-mesh").
		WithImage("ghcr.io/edgelesssys/contrast/service-mesh-proxy:latest").
		WithRestartPolicy(corev1.ContainerRestartPolicyAlways).
		WithVolumeMounts(VolumeMount().
			WithName("contrast-secrets").
			WithMountPath("/contrast"),
		).
		WithSecurityContext(SecurityContext().
			WithPrivileged(true).
			AddCapabilities("NET_ADMIN").
			SecurityContextApplyConfiguration,
		).
		WithStartupProbe(Probe().
			WithInitialDelaySeconds(1).
			WithPeriodSeconds(5).
			WithFailureThreshold(5).
			WithExec(ExecAction().
				WithCommand("test", "-f", "/ready")),
		).
		WithArgs(
			"-l", "debug",
		)
}

// DebugShell creates a new debug shell container.
func DebugShell() *applycorev1.ContainerApplyConfiguration {
	return applycorev1.Container().
		WithName("contrast-debug-shell").
		WithImage("ghcr.io/edgelesssys/contrast/debugshell:latest").
		WithResources(ResourceRequirements().
			WithMemoryLimitAndRequest(400),
		)
}

func getPorts(podSpec *applycorev1.PodSpecApplyConfiguration) (ports []applycorev1.ContainerPortApplyConfiguration) {
	for _, c := range podSpec.Containers {
		ports = append(ports, c.Ports...)
	}
	return ports
}
