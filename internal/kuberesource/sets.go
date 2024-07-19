// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"fmt"

	"github.com/edgelesssys/contrast/node-installer/platforms"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// CoordinatorBundle returns the Coordinator and a matching Service.
func CoordinatorBundle() []any {
	coordinatorSfSets := Coordinator("").StatefulSetApplyConfiguration
	coordinatorService := ServiceForStatefulSet(coordinatorSfSets).
		WithAnnotations(map[string]string{exposeServiceAnnotation: "true"})

	resources := []any{
		coordinatorSfSets,
		coordinatorService,
	}

	return resources
}

// Runtime returns a set of resources for registering and installing the runtime.
func Runtime(platform platforms.Platform) ([]any, error) {
	ns := ""

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
							WithCommand("/bin/bash", "-c", "openssl s_server -port 443 -Verify 2 -CAfile /tls-config/mesh-ca.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem").
							WithPorts(
								ContainerPort().
									WithName("https").
									WithContainerPort(443),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
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
							WithCommand("/bin/bash", "-c", "openssl s_server -www -port 443 -cert /tls-config/certChain.pem -key /tls-config/key.pem -cert_chain /tls-config/certChain.pem").
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
								WithMemoryLimitAndRequest(50),
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
	var httpProbeScheme corev1.URIScheme
	switch smMode {
	case ServiceMeshDisabled:
		emojiSvcImage = "ghcr.io/3u13r/emojivoto-emoji-svc:coco-1"
		emojiVotingSvcImage = "ghcr.io/3u13r/emojivoto-voting-svc:coco-1"
		emojiWebImage = "ghcr.io/3u13r/emojivoto-web:coco-1"
		emojiWebVoteBotImage = emojiWebImage
		httpProbeScheme = corev1.URISchemeHTTP
		emojiSvcHost = "emoji-svc:8080"
		votingSvcHost = "voting-svc:8080"
	case ServiceMeshIngressEgress:
		emojiSvcImage = "docker.io/buoyantio/emojivoto-emoji-svc:v11"
		emojiVotingSvcImage = "docker.io/buoyantio/emojivoto-voting-svc:v11"
		emojiWebImage = "docker.io/buoyantio/emojivoto-web:v11"
		emojiWebVoteBotImage = "ghcr.io/3u13r/emojivoto-web:coco-1"
		httpProbeScheme = corev1.URISchemeHTTPS
		emojiSvcHost = "127.137.0.1:8081"
		votingSvcHost = "127.137.0.2:8081"
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
								WithMemoryLimitAndRequest(50),
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
								WithMemoryLimitAndRequest(25),
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
								WithMemoryLimitAndRequest(50),
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
								WithMemoryLimitAndRequest(50),
							).
							WithReadinessProbe(applycorev1.Probe().
								WithHTTPGet(applycorev1.HTTPGetAction().
									WithPort(intstr.FromInt(8080)).
									WithScheme(httpProbeScheme),
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
			WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/tls-config/certChain.pem")).
			WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/mesh-ca.pem")).
			WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/tls-config/key.pem"))
		voting.Spec.Template.Spec.Containers[0].
			WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/tls-config/certChain.pem")).
			WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/mesh-ca.pem")).
			WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/tls-config/key.pem"))
		web.Spec.Template.Spec.Containers[0].
			WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/tls-config/certChain.pem")).
			WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/mesh-ca.pem")).
			WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/tls-config/key.pem")).
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
