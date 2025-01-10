// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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

	r := RuntimeClass(runtimeHandler).
		WithHandler(runtimeHandler).
		WithLabels(map[string]string{"addonmanager.kubernetes.io/mode": "Reconcile"}).
		WithOverhead(Overhead(corev1.ResourceList{"memory": resource.MustParse("1152Mi")}))

	if platform == platforms.AKSCloudHypervisorSNP {
		r.WithScheduling(Scheduling(map[string]string{"kubernetes.azure.com/kata-cc-isolation": "true"}))
	}

	return &RuntimeClassConfig{r}, nil
}

// NodeInstallerConfig wraps a DaemonSetApplyConfiguration for a node installer.
type NodeInstallerConfig struct {
	*applyappsv1.DaemonSetApplyConfiguration
	*applycorev1.ServiceAccountApplyConfiguration
	*applyrbacv1.ClusterRoleApplyConfiguration
	*applyrbacv1.ClusterRoleBindingApplyConfiguration
}

// NodeInstaller constructs a node installer daemon set.
func NodeInstaller(namespace string, platform platforms.Platform) (*NodeInstallerConfig, error) {
	runtimeHandler, err := manifest.RuntimeHandler(platform)
	if err != nil {
		return nil, fmt.Errorf("getting default runtime handler: %w", err)
	}

	name := fmt.Sprintf("%s-nodeinstaller", runtimeHandler)

	tardevSnapshotter := Container().
		WithName("tardev-snapshotter").
		WithImage("ghcr.io/edgelesssys/contrast/tardev-snapshotter:latest").
		WithResources(ResourceRequirements().
			WithMemoryRequest(800),
		).
		WithVolumeMounts(
			VolumeMount().
				WithName("host-mount").
				WithMountPath("/host"),
			VolumeMount().
				WithName("var-lib-containerd").
				WithMountPath("/var/lib/containerd"),
		).
		WithArgs(
			"tardev-snapshotter",
			fmt.Sprintf("/var/lib/containerd/io.containerd.snapshotter.v1.tardev-%s", runtimeHandler),
			fmt.Sprintf("/host/run/containerd/tardev-snapshotter-%s.sock", runtimeHandler),
			"/host/var/run/containerd/containerd.sock",
		).
		WithEnv(
			NewEnvVar("RUST_LOG", "tardev_snapshotter=trace"),
		)
	tardevSnapshotterVolumes := []*applycorev1.VolumeApplyConfiguration{
		Volume().
			WithName("var-lib-containerd").
			WithHostPath(HostPathVolumeSource().
				WithPath("/var/lib/containerd").
				WithType(corev1.HostPathDirectory),
			),
	}

	nydusSnapshotter := Container().
		WithName("nydus-snapshotter").
		WithImage("ghcr.io/edgelesssys/contrast/nydus-snapshotter:latest").
		WithResources(ResourceRequirements().
			WithMemoryRequest(800),
		).
		WithVolumeMounts(
			VolumeMount().
				WithName("host-mount").
				WithMountPath("/host"),
			VolumeMount().
				WithName("var-lib-containerd").
				WithMountPath("/var/lib/containerd"),
			VolumeMount().
				WithName("var-lib-nydus-snapshotter").
				WithMountPath(fmt.Sprintf("/var/lib/nydus-snapshotter/%s", runtimeHandler)),
		).
		WithArgs(
			"containerd-nydus-grpc",
			// Snapshotter will write to this path and tell containerd to read from it, so
			// path must be shared and the same on the host. See 'var-lib-nydus-snapshotter' volume.
			fmt.Sprintf("--root=/var/lib/nydus-snapshotter/%s", runtimeHandler),
			"--config=/share/nydus-snapshotter/config-coco-guest-pulling.toml",
			fmt.Sprintf("--address=/host/run/containerd/containerd-nydus-grpc-%s.sock", runtimeHandler),
			"--log-to-stdout",
			fmt.Sprintf("--nydus-overlayfs-path=/opt/edgeless/%s/bin/nydus-overlayfs", runtimeHandler),
		)
	nydusSnapshotterVolumes := []*applycorev1.VolumeApplyConfiguration{
		Volume().
			WithName("var-lib-nydus-snapshotter").
			WithHostPath(HostPathVolumeSource().
				WithPath(fmt.Sprintf("/var/lib/nydus-snapshotter/%s", runtimeHandler)).
				WithType(corev1.HostPathDirectoryOrCreate),
			),
	}

	nydusPull := Container().
		WithName("nydus-pull").
		WithImage("ghcr.io/edgelesssys/contrast/nydus-pull:latest").
		WithArgs(runtimeHandler).
		WithEnv(
			EnvVar().
				WithName("NODE_NAME").
				WithValueFrom(
					applycorev1.EnvVarSource().
						WithFieldRef(
							applycorev1.ObjectFieldSelector().
								WithFieldPath("spec.nodeName"),
						),
				),
		).
		WithVolumeMounts(
			VolumeMount().
				WithName("containerd-socket").
				WithMountPath("/run/containerd/containerd.sock"),
		)

	var nodeInstallerImageURL string
	var containers []*applycorev1.ContainerApplyConfiguration
	var snapshotterVolumes []*applycorev1.VolumeApplyConfiguration
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		nodeInstallerImageURL = "ghcr.io/edgelesssys/contrast/node-installer-microsoft:latest"
		containers = append(containers, tardevSnapshotter)
		snapshotterVolumes = tardevSnapshotterVolumes
	case platforms.MetalQEMUSNP, platforms.MetalQEMUTDX, platforms.MetalQEMUSNPGPU:
		nodeInstallerImageURL = "ghcr.io/edgelesssys/contrast/node-installer-kata:latest"
		containers = append(containers, nydusSnapshotter, nydusPull)
		nydusSnapshotterVolumes = append(nydusSnapshotterVolumes,
			Volume().
				WithName("var-lib-containerd").
				WithHostPath(HostPathVolumeSource().
					WithPath("/var/lib/containerd").
					WithType(corev1.HostPathDirectory),
				),
			Volume().
				WithName("containerd-socket").
				WithHostPath(HostPathVolumeSource().
					WithPath("/run/containerd/containerd.sock").
					WithType(corev1.HostPathSocket),
				),
		)
		snapshotterVolumes = nydusSnapshotterVolumes
	case platforms.K3sQEMUTDX, platforms.K3sQEMUSNP, platforms.K3sQEMUSNPGPU, platforms.RKE2QEMUTDX:
		nodeInstallerImageURL = "ghcr.io/edgelesssys/contrast/node-installer-kata:latest"
		containers = append(containers, nydusSnapshotter, nydusPull)
		nydusSnapshotterVolumes = append(nydusSnapshotterVolumes,
			Volume().
				WithName("var-lib-containerd").
				WithHostPath(HostPathVolumeSource().
					WithPath("/var/lib/rancher/k3s/agent/containerd").
					WithType(corev1.HostPathDirectory),
				),
			Volume().
				WithName("containerd-socket").
				WithHostPath(HostPathVolumeSource().
					WithPath("/run/k3s/containerd/containerd.sock").
					WithType(corev1.HostPathSocket),
				),
		)
		snapshotterVolumes = nydusSnapshotterVolumes
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
					WithServiceAccountName(name).
					WithHostPID(true).
					WithInitContainers(Container().
						WithName("installer").
						WithImage(nodeInstallerImageURL).
						WithResources(ResourceRequirements().
							WithMemoryRequest(700),
						).
						WithSecurityContext(SecurityContext().WithPrivileged(true).SecurityContextApplyConfiguration).
						WithVolumeMounts(VolumeMount().
							WithName("host-mount").
							WithMountPath("/host")).
						WithCommand("/bin/node-installer", platform.String()),
					).
					WithContainers(
						containers...,
					).
					WithVolumes(append(
						snapshotterVolumes,
						Volume().
							WithName("host-mount").
							WithHostPath(HostPathVolumeSource().
								WithPath("/").
								WithType(corev1.HostPathDirectory),
							),
					)...,
					),
				),
			),
		)

	serviceAccount := applycorev1.ServiceAccount(name, namespace)

	clusterRole := applyrbacv1.ClusterRole(name).
		WithRules(
			applyrbacv1.PolicyRule().
				WithAPIGroups("").
				WithResources("pods").
				WithVerbs("watch"),
		)

	clusterRoleBinding := applyrbacv1.ClusterRoleBinding(name).
		WithSubjects(
			applyrbacv1.Subject().
				WithKind("ServiceAccount").
				WithName(name).
				WithNamespace(namespace),
		).
		WithRoleRef(
			applyrbacv1.RoleRef().
				WithKind("ClusterRole").
				WithName(name).
				WithAPIGroup("rbac.authorization.k8s.io"),
		)

	return &NodeInstallerConfig{
		DaemonSetApplyConfiguration:          d,
		ServiceAccountApplyConfiguration:     serviceAccount,
		ClusterRoleApplyConfiguration:        clusterRole,
		ClusterRoleBindingApplyConfiguration: clusterRoleBinding,
	}, nil
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
		).
		WithStartupProbe(Probe().
			WithInitialDelaySeconds(1).
			WithPeriodSeconds(1).
			WithTCPSocket(TCPSocketAction().
				WithPort(intstr.FromInt32(port))),
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

// PortForwarderMultiplePorts constructs a port forwarder pod for multiple ports.
func PortForwarderMultiplePorts(name, namespace string) *PortForwarderConfig {
	name = "port-forwarder-" + name

	p := Pod(name, namespace).
		WithLabels(map[string]string{"app.kubernetes.io/name": name}).
		WithSpec(PodSpec().
			WithContainers(
				Container().
					WithName("port-forwarder").
					WithImage("ghcr.io/edgelesssys/contrast/port-forwarder:latest").
					WithCommand("/bin/bash", "-c", "echo Starting port-forward with socat; for port in ${LISTEN_PORTS}; do socat -d -d TCP-LISTEN:$port,fork TCP:${FORWARD_HOST}:$port & done; wait").
					WithResources(ResourceRequirements().
						WithMemoryLimitAndRequest(50),
					),
			),
		)

	return &PortForwarderConfig{p}
}

// WithListenPorts sets multiple ports to listen on. Should only be used if PortForwarderMultiplePorts was used initially.
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
					WithContainers(
						Container().
							WithName("coordinator").
							WithImage("ghcr.io/edgelesssys/contrast/coordinator:latest").
							WithVolumeDevices(applycorev1.VolumeDevice().
								WithName("state-device").
								WithDevicePath("/dev/csi0"),
							).
							WithSecurityContext(SecurityContext().
								WithCapabilities(applycorev1.Capabilities().
									WithAdd("SYS_ADMIN"),
								),
							).
							WithPorts(
								ContainerPort().
									WithName("userapi").
									WithContainerPort(1313),
								ContainerPort().
									WithName("meshapi").
									WithContainerPort(7777),
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
			).
			WithVolumeClaimTemplates(applycorev1.PersistentVolumeClaim("state-device", namespace).
				WithSpec(applycorev1.PersistentVolumeClaimSpec().
					WithVolumeMode(corev1.PersistentVolumeBlock).
					WithAccessModes(corev1.ReadWriteOnce).
					WithResources(applycorev1.VolumeResourceRequirements().
						WithRequests(map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: resource.MustParse("1Gi")}),
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

// ServiceForStatefulSet creates a service for a StatefulSet by exposing the configured ports
// of the first container.
func ServiceForStatefulSet(s *applyappsv1.StatefulSetApplyConfiguration) *applycorev1.ServiceApplyConfiguration {
	selector := s.Spec.Selector.MatchLabels
	ports := s.Spec.Template.Spec.Containers[0].Ports

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
// Port forwarders are named "port-forwarder-SVCNAME" and forward the first port in the ServiceSpec.
func PortForwarderForService(svc *applycorev1.ServiceApplyConfiguration) *applycorev1.PodApplyConfiguration {
	namespace := ""
	if svc.Namespace != nil {
		namespace = *svc.Namespace
	}

	var ports []int32
	for _, port := range svc.Spec.Ports {
		ports = append(ports, *port.Port)
	}

	forwarder := PortForwarderMultiplePorts(*svc.Name, namespace).
		WithListenPorts(ports).
		WithForwardTarget(*svc.Name, -1) // port can be -1 since MultiplePortsForwarder ignores FORWARD_PORT env

	return forwarder.PodApplyConfiguration
}

// Initializer creates a new InitializerConfig.
func Initializer() *applycorev1.ContainerApplyConfiguration {
	return applycorev1.Container().
		WithName("contrast-initializer").
		WithImage("ghcr.io/edgelesssys/contrast/initializer:latest").
		WithResources(ResourceRequirements().
			WithMemoryRequest(50),
		).
		WithEnv(NewEnvVar("COORDINATOR_HOST", "coordinator")).
		WithVolumeMounts(VolumeMount().
			WithName("contrast-secrets").
			WithMountPath("/contrast"),
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

// CryptsetupInitCommand returns the init command for the cryptsetup
// container to setup an encrypted LUKS mount.
func CryptsetupInitCommand() string {
	return `#!/bin/bash
set -e
# device is the path to the block device to be encrypted.
device="/dev/csi0"
# workload_secret_path is the path to the Contrast workload secret.
workload_secret_path="/contrast/secrets/workload-secret-seed"

if ! cryptsetup isLuks "${device}"; then
	# cryptsetup derives the encryption key of the master key using PBKDF2 based on
	# the workloadSecret as passphrase and a random 32 byte salt (stored in LUKS header)
	# which ensures uniqueness of encryption key per disk.
	cryptsetup luksFormat --pbkdf-memory=10240 $device "${workload_secret_path}" </dev/null
	cryptsetup open "${device}" state -d "${workload_secret_path}"

	# Create the ext4 filesystem on the mapper device.
	mkfs.ext4 /dev/mapper/state
else
	cryptsetup open "${device}" state -d "${workload_secret_path}"
fi

mount /dev/mapper/state /state
touch /done
sleep inf
`
}
