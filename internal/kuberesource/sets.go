// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/intstr"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// CoordinatorBundle returns the Coordinator and a matching Service.
func CoordinatorBundle() []any {
	coordinator := Coordinator("").DeploymentApplyConfiguration
	coordinatorService := ServiceForDeployment(coordinator).
		WithAnnotations(map[string]string{exposeServiceAnnotation: "true"})

	resources := []any{
		coordinator,
		coordinatorService,
	}

	return resources
}

// Runtime returns a set of resources for registering and installing the runtime.
func Runtime() ([]any, error) {
	ns := ""

	runtimeClass := ContrastRuntimeClass().RuntimeClassApplyConfiguration
	nodeInstaller := NodeInstaller(ns).DaemonSetApplyConfiguration

	resources := []any{
		runtimeClass,
		nodeInstaller,
	}

	return resources, nil
}

// OpenSSL returns a set of resources for testing with OpenSSL.
func OpenSSL() ([]any, error) {
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
					WithRuntimeClassName(runtimeHandler).
					WithContainers(
						Container().
							WithName("openssl-backend").
							WithImage("ghcr.io/edgelesssys/contrast/openssl:latest").
							WithCommand("/bin/bash", "-c", "echo Workload started \nopenssl s_server -port 443 -Verify 2 -CAfile /tls-config/mesh-ca.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem").
							WithPorts(
								ContainerPort().
									WithName("openssl").
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

	backend, err := AddInitializer(backend, Initializer())
	if err != nil {
		return nil, err
	}

	backendService := ServiceForDeployment(backend)

	client := Deployment("openssl-client", ns).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "openssl-client"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "openssl-client"}).
				WithSpec(PodSpec().
					WithRuntimeClassName(runtimeHandler).
					WithContainers(
						Container().
							WithName("openssl-client").
							WithImage("ghcr.io/edgelesssys/contrast/openssl:latest").
							WithCommand("/bin/bash", "-c", "echo Workload started \nwhile true; do \n  echo \"THIS IS A TEST MESSAGE\" |\n    openssl s_client -connect openssl-frontend:443 -verify_return_error -CAfile /tls-config/coordinator-root-ca.pem\n  sleep 30\ndone\n").
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)
	client, err = AddInitializer(client, Initializer())
	if err != nil {
		return nil, err
	}

	frontend := Deployment("openssl-frontend", ns).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "openssl-frontend"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "openssl-frontend"}).
				WithSpec(PodSpec().
					WithRuntimeClassName(runtimeHandler).
					WithContainers(
						Container().
							WithName("openssl-frontend").
							WithImage("ghcr.io/edgelesssys/contrast/openssl:latest").
							WithCommand("/bin/bash", "-c", "echo Workload started\nopenssl s_server -www -port 443 -cert /tls-config/certChain.pem -key /tls-config/key.pem -cert_chain /tls-config/certChain.pem &\nwhile true; do \n  echo \"THIS IS A TEST MESSAGE\" |\n    openssl s_client -connect openssl-backend:443 -verify_return_error -CAfile /tls-config/mesh-ca.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem\n  sleep 10\ndone\n").
							WithPorts(
								ContainerPort().
									WithName("openssl").
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
	frontend, err = AddInitializer(frontend, Initializer())
	if err != nil {
		return nil, err
	}

	frontendService := ServiceForDeployment(frontend)

	resources := []any{
		backend,
		backendService,
		client,
		frontend,
		frontendService,
	}

	return resources, nil
}

// GetDEnts returns a set of resources for testing getdents entry limits.
func GetDEnts() ([]any, error) {
	tester := Deployment("getdents-tester", "").
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "getdents-tester"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "getdents-tester"}).
				WithSpec(PodSpec().
					WithRuntimeClassName(runtimeHandler).
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

	return []any{tester}, nil
}

// Emojivoto returns resources for deploying Emojivoto application.
func Emojivoto(smMode serviceMeshMode) ([]any, error) {
	ns := ""
	var emojiSvcImage, emojiVotingSvcImage, emojiWebImage, emojiWebVoteBotImage, emojiSvcHost, votingSvcHost string
	smProxyEmoji := ServiceMeshProxy()
	smProxyWeb := ServiceMeshProxy()
	smProxyVoting := ServiceMeshProxy()
	switch smMode {
	case ServiceMeshDisabled:
		emojiSvcImage = "ghcr.io/3u13r/emojivoto-emoji-svc:coco-1"
		emojiVotingSvcImage = "ghcr.io/3u13r/emojivoto-voting-svc:coco-1"
		emojiWebImage = "ghcr.io/3u13r/emojivoto-web:coco-1"
		emojiWebVoteBotImage = emojiWebImage
		emojiSvcHost = "emoji-svc:8080"
		votingSvcHost = "voting-svc:8080"
		smProxyEmoji = nil
		smProxyWeb = nil
		smProxyVoting = nil
	case ServiceMeshIngressEgress:
		emojiSvcImage = "docker.l5d.io/buoyantio/emojivoto-emoji-svc:v11"
		emojiVotingSvcImage = "docker.l5d.io/buoyantio/emojivoto-voting-svc:v11"
		emojiWebImage = "docker.l5d.io/buoyantio/emojivoto-web:v11"
		emojiWebVoteBotImage = "ghcr.io/3u13r/emojivoto-web:coco-1"
		emojiSvcHost = "127.137.0.1:8081"
		votingSvcHost = "127.137.0.2:8081"
		smProxyWeb = smProxyWeb.
			WithEnv(EnvVar().
				WithName("EDG_INGRESS_PROXY_CONFIG").
				WithValue("web#8080#false"),
			).
			WithEnv(EnvVar().
				WithName("EDG_EGRESS_PROXY_CONFIG").
				WithValue("emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080"),
			)
	case ServiceMeshEgress:
		emojiSvcImage = "ghcr.io/3u13r/emojivoto-emoji-svc:coco-1"
		emojiVotingSvcImage = "ghcr.io/3u13r/emojivoto-voting-svc:coco-1"
		emojiWebImage = "docker.l5d.io/buoyantio/emojivoto-web:v11"
		emojiWebVoteBotImage = emojiWebImage
		emojiSvcHost = "127.137.0.1:8081"
		votingSvcHost = "127.137.0.2:8081"
		smProxyWeb = smProxyWeb.
			WithSecurityContext(SecurityContext().
				WithPrivileged(true).
				AddCapabilities("NET_ADMIN").
				AddCapabilities("NET_RAW").
				SecurityContextApplyConfiguration,
			).
			WithEnv(EnvVar().
				WithName("EDG_EGRESS_PROXY_CONFIG").
				WithValue("emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080"),
			)
		smProxyEmoji = nil
		smProxyVoting = nil
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
					WithRuntimeClassName(runtimeHandler).
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
							WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/tls-config/certChain.pem")).
							WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/mesh-ca.pem")).
							WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/tls-config/key.pem")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)
	emoji, err := AddInitializer(emoji, Initializer())
	if err != nil {
		return nil, err
	}

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
					WithRuntimeClassName(runtimeHandler).
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
							WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/tls-config/certChain.pem")).
							WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/mesh-ca.pem")).
							WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/tls-config/key.pem")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)
	voting, err = AddInitializer(voting, Initializer())
	if err != nil {
		return nil, err
	}

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
					WithRuntimeClassName(runtimeHandler).
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
							WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/tls-config/certChain.pem")).
							WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/mesh-ca.pem")).
							WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/tls-config/key.pem")).
							WithEnv(EnvVar().WithName("EDG_DISABLE_CLIENT_AUTH").WithValue("true")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							).
							WithReadinessProbe(applycorev1.Probe().
								WithTCPSocket(TCPSocketAction().WithPort(intstr.FromInt(8080))).
								WithInitialDelaySeconds(1).
								WithPeriodSeconds(5),
							),
					),
				),
			),
		)
	web, err = AddInitializer(web, Initializer())
	if err != nil {
		return nil, err
	}

	if smProxyEmoji != nil {
		emoji, err = AddServiceMesh(emoji, smProxyEmoji, smMode)
		if err != nil {
			return nil, err
		}
	}
	if smProxyWeb != nil {
		web, err = AddServiceMesh(web, smProxyWeb, smMode)
		if err != nil {
			return nil, err
		}
	}
	if smProxyVoting != nil {
		voting, err = AddServiceMesh(voting, smProxyVoting, smMode)
		if err != nil {
			return nil, err
		}
	}

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

	return resources, nil
}
