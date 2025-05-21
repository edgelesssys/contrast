// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"fmt"

	"github.com/edgelesssys/contrast/internal/platforms"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// CoordinatorBundle returns the Coordinator and a matching Service.
func CoordinatorBundle() []any {
	coordinator := Coordinator("")

	coordinatorService := ServiceForStatefulSet(coordinator.StatefulSetApplyConfiguration).
		WithAnnotations(map[string]string{exposeServiceAnnotation: "true"})
	coordinatorService.Spec.WithPublishNotReadyAddresses(true)

	coordinatorReadyService := ServiceForStatefulSet(coordinator.StatefulSetApplyConfiguration).
		WithName(*coordinatorService.GetName() + "-ready")

	return []any{
		coordinator.StatefulSetApplyConfiguration,
		coordinator.ServiceAccountApplyConfiguration,
		coordinator.RoleApplyConfiguration,
		coordinator.RoleBindingApplyConfiguration,
		coordinatorService,
		coordinatorReadyService,
	}
}

// Runtime returns a set of resources for registering and installing the runtime.
func Runtime(platform platforms.Platform) ([]any, error) {
	ns := "kube-system"

	runtimeClass, err := ContrastRuntimeClass(platform)
	if err != nil {
		return nil, fmt.Errorf("creating runtime class: %w", err)
	}

	runtimeClassApplyConfig := runtimeClass.RuntimeClassApplyConfiguration
	nodeInstaller, err := NodeInstaller(ns, platform)
	if err != nil {
		return nil, fmt.Errorf("creating node installer: %w", err)
	}

	return []any{
		runtimeClassApplyConfig,
		nodeInstaller.DaemonSetApplyConfiguration,
		nodeInstaller.ServiceAccountApplyConfiguration,
		nodeInstaller.ClusterRoleApplyConfiguration,
		nodeInstaller.ClusterRoleBindingApplyConfiguration,
	}, nil
}

// OpenSSL returns a set of resources for testing with OpenSSL.
func OpenSSL() []any {
	ns := ""
	backend := Deployment("openssl-backend", ns).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "openssl-backend"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "openssl-backend"}).
				WithSpec(PodSpec().
					WithContainers(
						Container().
							WithName("openssl-backend").
							WithImage("ghcr.io/edgelesssys/contrast/openssl:latest").
							WithCommand("/bin/sh", "-c", "openssl s_server -port 443 -Verify 2 -CAfile /contrast/tls-config/mesh-ca.pem -cert /contrast/tls-config/certChain.pem -key /contrast/tls-config/key.pem").
							WithPorts(
								ContainerPort().
									WithName("https").
									WithContainerPort(443),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(250),
							).
							WithReadinessProbe(Probe().
								WithInitialDelaySeconds(1).
								WithPeriodSeconds(5).
								WithTCPSocket(TCPSocketAction().
									WithPort(intstr.FromInt(443))),
							),
					),
				),
			),
		)

	backendService := ServiceForDeployment(backend)

	frontend := Deployment("openssl-frontend", ns).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "openssl-frontend"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "openssl-frontend"}).
				WithSpec(PodSpec().
					WithContainers(
						Container().
							WithName("openssl-frontend").
							WithImage("ghcr.io/edgelesssys/contrast/openssl:latest").
							WithCommand("/bin/sh", "-c", "openssl s_server -www -port 443 -cert /contrast/tls-config/certChain.pem -key /contrast/tls-config/key.pem -cert_chain /contrast/tls-config/certChain.pem").
							WithPorts(
								ContainerPort().
									WithName("https").
									WithContainerPort(443),
							).
							WithReadinessProbe(Probe().
								WithInitialDelaySeconds(1).
								WithPeriodSeconds(5).
								WithTCPSocket(TCPSocketAction().
									WithPort(intstr.FromInt(443))),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(250),
							),
					),
				),
			),
		)

	frontendService := ServiceForDeployment(frontend)

	resources := []any{
		backend,
		backendService,
		frontend,
		frontendService,
	}

	return resources
}

// GetDEnts returns a set of resources for testing getdents entry limits.
func GetDEnts() []any {
	tester := Deployment("getdents-tester", "").
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "getdents-tester"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "getdents-tester"}).
				WithSpec(PodSpec().
					WithContainers(
						Container().
							WithName("getdents-tester").
							WithImage("ghcr.io/edgelesssys/contrast/getdents-e2e-test:1").
							WithCommand("/bin/sh", "-c", "sleep inf").
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)

	return []any{tester}
}

// GenpolicyRegressionTests returns deployments for regression testing genpolicy.
func GenpolicyRegressionTests() map[string]*applyappsv1.DeploymentApplyConfiguration {
	out := make(map[string]*applyappsv1.DeploymentApplyConfiguration)

	// Reproduces https://github.com/edgelesssys/contrast/issues/624.
	badLayer := "bad-layer"
	out[badLayer] = Deployment(badLayer, "").
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": badLayer}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": badLayer}).
				WithSpec(PodSpec().
					WithContainers(
						Container().
							WithName(badLayer).
							WithImage("docker.io/library/httpd:2.4.59-bookworm@sha256:10182d88d7fbc5161ae0f6f758cba7adc56d4aae2dc950e51d72c0cf68967cea").
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)

	return out
}

// Emojivoto returns resources for deploying Emojivoto application.
func Emojivoto(smMode serviceMeshMode) []any {
	ns := ""
	var emojiSvcImage, emojiVotingSvcImage, emojiWebImage, emojiWebVoteBotImage, emojiSvcHost, votingSvcHost string
	var memoryLimitMiB int64
	switch smMode {
	case ServiceMeshDisabled:
		// Source: https://github.com/3u13r/emojivoto/tree/8ba877681c297721cde63eb7be95c98c4c1186ee
		emojiSvcImage = "ghcr.io/edgelesssys/contrast/emojivoto-emoji-svc:coco-1@sha256:fa80600859cda06079a542632713b2cc67ed836e429753a799a6c313322d1426"
		emojiVotingSvcImage = "ghcr.io/edgelesssys/contrast/emojivoto-voting-svc:coco-1@sha256:bb7fbea32bf28c6201602b473bf7e0f40290642e3f783dcfa4b8e3c693531cba"
		emojiWebImage = "ghcr.io/edgelesssys/contrast/emojivoto-web:coco-1@sha256:0fd9bf6f7dcb99bdb076144546b663ba6c3eb457cbb48c1d3fceb591d207289c"
		emojiWebVoteBotImage = emojiWebImage
		emojiSvcHost = "emoji-svc:8080"
		votingSvcHost = "voting-svc:8080"
		// Our modified images are around 100MiB compressed.
		memoryLimitMiB = 600
	case ServiceMeshIngressEgress:
		emojiSvcImage = "docker.io/buoyantio/emojivoto-emoji-svc:v11@sha256:957949355653776b65fafc2ee22f737cd21e090d4ace63f3b99f6e16976f0458"
		emojiVotingSvcImage = "docker.io/buoyantio/emojivoto-voting-svc:v11@sha256:a57ac67af7a5b05988a38b49568eca6a078ef27a71c148c44c9db4efb1dac58b"
		emojiWebImage = "docker.io/buoyantio/emojivoto-web:v11@sha256:d21f9fdb794f754b076344ce01c4858c499617c952cc6a941ac3cefbf5ccedfd"
		// Source: https://github.com/3u13r/emojivoto/tree/8ba877681c297721cde63eb7be95c98c4c1186ee
		emojiWebVoteBotImage = "ghcr.io/edgelesssys/contrast/emojivoto-web:coco-1@sha256:0fd9bf6f7dcb99bdb076144546b663ba6c3eb457cbb48c1d3fceb591d207289c"
		emojiSvcHost = "127.137.0.1:8081"
		votingSvcHost = "127.137.0.2:8081"
		// Upstream images are at most 75MiB compressed, but we're adding the service mesh image with 50MiB.
		memoryLimitMiB = 800
	default:
		panic(fmt.Sprintf("unknown service mesh mode: %s", smMode))
	}

	emoji := Deployment("emoji", ns).
		WithLabels(map[string]string{
			"app.kubernetes.io/name":    "emoji",
			"app.kubernetes.io/part-of": "emojivoto",
			"app.kubernetes.io/version": "v11",
		}).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{
					"app.kubernetes.io/name": "emoji-svc",
					"version":                "v11",
				}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{
					"app.kubernetes.io/name": "emoji-svc",
					"version":                "v11",
				}).
				WithSpec(PodSpec().
					WithServiceAccountName("emoji").
					WithContainers(
						Container().
							WithName("emoji-svc").
							WithImage(emojiSvcImage).
							WithPorts(
								ContainerPort().
									WithName("grpc").
									WithContainerPort(8080),
								ContainerPort().
									WithName("prom").
									WithContainerPort(8801),
							).
							WithEnv(EnvVar().WithName("GRPC_PORT").WithValue("8080")).
							WithEnv(EnvVar().WithName("PROM_PORT").WithValue("8801")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(memoryLimitMiB),
							).
							WithReadinessProbe(Probe().
								WithInitialDelaySeconds(1).
								WithPeriodSeconds(5).
								WithTCPSocket(TCPSocketAction().
									WithPort(intstr.FromInt(8080))),
							),
					),
				),
			),
		)

	emojiService := ServiceForDeployment(emoji).
		WithName("emoji-svc").
		WithSpec(ServiceSpec().
			WithSelector(
				map[string]string{"app.kubernetes.io/name": "emoji-svc"},
			).
			WithPorts(
				ServicePort().
					WithName("grpc").
					WithPort(8080).
					WithTargetPort(intstr.FromInt(8080)),
				ServicePort().
					WithName("prom").
					WithPort(8801).
					WithTargetPort(intstr.FromInt(8801)),
			),
		)

	emojiserviceAccount := ServiceAccount("emoji", ns).
		WithAPIVersion("v1").
		WithKind("ServiceAccount")

	voteBot := Deployment("vote-bot", ns).
		WithLabels(map[string]string{
			"app.kubernetes.io/name":    "vote-bot",
			"app.kubernetes.io/part-of": "emojivoto",
			"app.kubernetes.io/version": "v11",
		}).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{
					"app.kubernetes.io/name": "vote-bot",
					"version":                "v11",
				}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{
					"app.kubernetes.io/name": "vote-bot",
					"version":                "v11",
				}).
				WithSpec(PodSpec().
					WithContainers(
						Container().
							WithName("vote-bot").
							WithImage(emojiWebVoteBotImage).
							WithCommand("emojivoto-vote-bot").
							WithEnv(EnvVar().WithName("WEB_HOST").WithValue("web-svc:443")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(memoryLimitMiB),
							),
					),
				),
			),
		)

	voting := Deployment("voting", ns).
		WithLabels(map[string]string{
			"app.kubernetes.io/name":    "voting",
			"app.kubernetes.io/part-of": "emojivoto",
			"app.kubernetes.io/version": "v11",
		}).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{
					"app.kubernetes.io/name": "voting-svc",
					"version":                "v11",
				}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{
					"app.kubernetes.io/name": "voting-svc",
					"version":                "v11",
				}).
				WithSpec(PodSpec().
					WithServiceAccountName("voting").
					WithContainers(
						Container().
							WithName("voting-svc").
							WithImage(emojiVotingSvcImage).
							WithPorts(
								ContainerPort().
									WithName("grpc").
									WithContainerPort(8080),
								ContainerPort().
									WithName("prom").
									WithContainerPort(8801),
							).
							WithEnv(EnvVar().WithName("GRPC_PORT").WithValue("8080")).
							WithEnv(EnvVar().WithName("PROM_PORT").WithValue("8801")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(memoryLimitMiB),
							).
							WithReadinessProbe(Probe().
								WithInitialDelaySeconds(1).
								WithPeriodSeconds(5).
								WithTCPSocket(TCPSocketAction().
									WithPort(intstr.FromInt(8080))),
							),
					),
				),
			),
		)

	votingService := ServiceForDeployment(voting).
		WithName("voting-svc").
		WithSpec(ServiceSpec().
			WithSelector(
				map[string]string{"app.kubernetes.io/name": "voting-svc"},
			).
			WithPorts(
				ServicePort().
					WithName("grpc").
					WithPort(8080).
					WithTargetPort(intstr.FromInt(8080)),
				ServicePort().
					WithName("prom").
					WithPort(8801).
					WithTargetPort(intstr.FromInt(8801)),
			),
		)

	votingserviceAccount := ServiceAccount("voting", ns).
		WithAPIVersion("v1").
		WithKind("ServiceAccount")

	web := Deployment("web", ns).
		WithLabels(map[string]string{
			"app.kubernetes.io/name":    "web",
			"app.kubernetes.io/part-of": "emojivoto",
			"app.kubernetes.io/version": "v11",
		}).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{
					"app.kubernetes.io/name": "web-svc",
					"version":                "v11",
				}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{
					"app.kubernetes.io/name": "web-svc",
					"version":                "v11",
				}).
				WithSpec(PodSpec().
					WithServiceAccountName("web").
					WithContainers(
						Container().
							WithName("web-svc").
							WithImage(emojiWebImage).
							WithPorts(
								ContainerPort().
									WithName("https").
									WithContainerPort(8080),
							).
							WithEnv(EnvVar().WithName("WEB_PORT").WithValue("8080")).
							WithEnv(EnvVar().WithName("EMOJISVC_HOST").WithValue(emojiSvcHost)).
							WithEnv(EnvVar().WithName("VOTINGSVC_HOST").WithValue(votingSvcHost)).
							WithEnv(EnvVar().WithName("INDEX_BUNDLE").WithValue("dist/index_bundle.js")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(memoryLimitMiB),
							).
							WithReadinessProbe(applycorev1.Probe().
								WithHTTPGet(applycorev1.HTTPGetAction().
									WithPort(intstr.FromInt(8080)).
									WithScheme(corev1.URISchemeHTTPS),
								).
								WithInitialDelaySeconds(1).
								WithPeriodSeconds(5),
							),
					),
				),
			),
		)

	webService := ServiceForDeployment(web).
		WithName("web-svc").
		WithAnnotations(map[string]string{exposeServiceAnnotation: "true"}).
		WithSpec(ServiceSpec().
			WithSelector(
				map[string]string{"app.kubernetes.io/name": "web-svc"},
			).
			WithType("ClusterIP").
			WithPorts(
				ServicePort().
					WithName("https").
					WithPort(443).
					WithTargetPort(intstr.FromInt(8080)),
			),
		)

	webserviceAccount := ServiceAccount("web", ns).
		WithAPIVersion("v1").
		WithKind("ServiceAccount")

	resources := []any{
		emoji,
		emojiService,
		emojiserviceAccount,
		voteBot,
		voting,
		votingService,
		votingserviceAccount,
		web,
		webService,
		webserviceAccount,
	}

	if smMode == ServiceMeshDisabled {
		emoji.Spec.Template.Spec.Containers[0].
			WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/contrast/tls-config/certChain.pem")).
			WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/contrast/tls-config/mesh-ca.pem")).
			WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/contrast/tls-config/key.pem"))
		voting.Spec.Template.Spec.Containers[0].
			WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/contrast/tls-config/certChain.pem")).
			WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/contrast/tls-config/mesh-ca.pem")).
			WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/contrast/tls-config/key.pem"))
		web.Spec.Template.Spec.Containers[0].
			WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/contrast/tls-config/certChain.pem")).
			WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/contrast/tls-config/mesh-ca.pem")).
			WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/contrast/tls-config/key.pem")).
			WithEnv(EnvVar().WithName("EDG_DISABLE_CLIENT_AUTH").WithValue("true"))
		return resources
	}

	emoji.WithAnnotations(map[string]string{smIngressConfigAnnotationKey: ""})
	voting.WithAnnotations(map[string]string{smIngressConfigAnnotationKey: ""})
	web.WithAnnotations(map[string]string{
		smIngressConfigAnnotationKey: "web#8080#false",
		smEgressConfigAnnotationKey:  "emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080",
	})

	return resources
}

// VolumeStatefulSet returns a stateful set for testing volume mounts and the
// mounting of encrypted luks volumes using the workload-secret.
func VolumeStatefulSet() []any {
	vss := StatefulSet("volume-tester", "").
		WithAnnotations(map[string]string{securePVAnnotationKey: "state:share"}).
		WithSpec(StatefulSetSpec().
			WithPersistentVolumeClaimRetentionPolicy(applyappsv1.StatefulSetPersistentVolumeClaimRetentionPolicy().
				WithWhenDeleted(appsv1.DeletePersistentVolumeClaimRetentionPolicyType).
				WithWhenScaled(appsv1.DeletePersistentVolumeClaimRetentionPolicyType)).
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "volume-tester"}),
			).
			WithServiceName("volume-tester").
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "volume-tester"}).
				WithSpec(
					PodSpec().
						WithContainers(
							Container().
								WithName("volume-tester").
								WithImage("ghcr.io/edgelesssys/contrast/initializer:latest").
								WithCommand("/bin/sh", "-c", "sleep inf").
								WithVolumeMounts(
									VolumeMount().
										WithName("share").
										WithMountPath("/state").
										WithMountPropagation(corev1.MountPropagationHostToContainer),
								),
						),
				),
			).
			WithVolumeClaimTemplates(PersistentVolumeClaim("state", "").
				WithSpec(applycorev1.PersistentVolumeClaimSpec().
					WithVolumeMode(corev1.PersistentVolumeBlock).
					WithAccessModes(corev1.ReadWriteOnce).
					WithResources(applycorev1.VolumeResourceRequirements().
						WithRequests(map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: resource.MustParse("1Gi")}),
					),
				),
			),
		)

	return []any{vss}
}

// MySQL returns the resources for deploying a MySQL database
// with an encrypted luks volume using the workload-secret.
func MySQL() []any {
	backend := StatefulSet("mysql-backend", "").
		WithAnnotations(map[string]string{
			smIngressConfigAnnotationKey: "",
			securePVAnnotationKey:        "state:share",
		}).
		WithSpec(StatefulSetSpec().
			WithPersistentVolumeClaimRetentionPolicy(applyappsv1.StatefulSetPersistentVolumeClaimRetentionPolicy().
				WithWhenDeleted(appsv1.DeletePersistentVolumeClaimRetentionPolicyType).
				WithWhenScaled(appsv1.DeletePersistentVolumeClaimRetentionPolicyType)).
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "mysql-backend"}),
			).
			WithServiceName("mysql-backend").
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "mysql-backend"}).
				WithSpec(
					PodSpec().
						WithContainers(
							Container().
								WithName("mysql-backend").
								WithImage("docker.io/library/mysql:9.1.0@sha256:0255b469f0135a0236d672d60e3154ae2f4538b146744966d96440318cc822c6").
								WithEnv(NewEnvVar("MYSQL_ALLOW_EMPTY_PASSWORD", "1")).
								WithPorts(
									ContainerPort().
										WithName("mysql").
										WithContainerPort(3306),
								).
								WithVolumeMounts(
									VolumeMount().
										WithName("share").
										WithMountPath("/var/lib/mysql").
										WithMountPropagation(corev1.MountPropagationHostToContainer),
								).
								WithResources(ResourceRequirements().
									WithMemoryLimitAndRequest(2000),
								),
						).
						WithVolumes(
							applycorev1.Volume().
								WithName("share").
								WithEmptyDir(applycorev1.EmptyDirVolumeSource()),
						),
				),
			).
			WithVolumeClaimTemplates(PersistentVolumeClaim("state", "").
				WithSpec(applycorev1.PersistentVolumeClaimSpec().
					WithVolumeMode(corev1.PersistentVolumeBlock).
					WithAccessModes(corev1.ReadWriteOnce).
					WithResources(applycorev1.VolumeResourceRequirements().
						WithRequests(map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: resource.MustParse("1Gi")}),
					),
				),
			),
		)

	backendService := ServiceForStatefulSet(backend)

	clientCmd := `#!/bin/bash
while ! mysqladmin ping -h 127.137.0.1 -u root --silent; do
	echo "Waiting for MySQL server ...";
	sleep 5;
done
mysql -h 127.137.0.1 -u root -e "CREATE DATABASE my_db;"
mysql -h 127.137.0.1 -u root -D my_db -e "CREATE TABLE my_table (id INT NOT NULL AUTO_INCREMENT, uuid CHAR(36), PRIMARY KEY (id));"
while true; do
	mysql -h 127.137.0.1 -u root -D my_db -e "INSERT INTO my_table (uuid) VALUES (UUID());"
	mysql -h 127.137.0.1 -u root -D my_db -e "SELECT * FROM my_table;"
	sleep 5;
done
`

	client := Deployment("mysql-client", "").
		WithAnnotations(map[string]string{smEgressConfigAnnotationKey: "mysql-backend#127.137.0.1:3306#mysql-backend:3306"}).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "mysql-client"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "mysql-client"}).
				WithSpec(
					PodSpec().
						WithContainers(
							Container().
								WithName("mysql-client").
								WithImage("docker.io/library/mysql:9.1.0@sha256:0255b469f0135a0236d672d60e3154ae2f4538b146744966d96440318cc822c6").
								WithEnv(NewEnvVar("MYSQL_ALLOW_EMPTY_PASSWORD", "1")).
								WithCommand("/bin/sh", "-c", clientCmd).
								WithResources(ResourceRequirements().
									WithMemoryLimitAndRequest(2000),
								),
						),
				),
			),
		)

	return []any{
		backend,
		backendService,
		client,
	}
}

// GPU returns the resources for deploying a GPU test pod.
func GPU() []any {
	tester := Deployment("gpu-tester", "").
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "gpu-tester"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "gpu-tester"}).
				WithAnnotations(map[string]string{
					"cdi.k8s.io/gpu": "nvidia.com/pgpu=0",
				}).
				WithSpec(PodSpec().
					WithContainers(
						Container().
							WithName("gpu-tester").
							WithImage("ghcr.io/edgelesssys/contrast/ubuntu:24.04").
							WithCommand("/bin/sh", "-c", "sleep inf").
							WithEnv(EnvVar().
								WithName("NVIDIA_VISIBLE_DEVICES").WithValue("all"),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(500). // This accounts for nvidia-smi and the guest pull overhead.
								WithLimits(corev1.ResourceList{
									corev1.ResourceName("nvidia.com/GH100_H100_PCIE"): resource.MustParse("1"),
								}),
							),
					),
				),
			),
		)

	return []any{tester}
}
