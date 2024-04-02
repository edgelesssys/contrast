package kuberesource

import "k8s.io/apimachinery/pkg/util/intstr"

// Simple returns a simple set of resources for testing.
func Simple() ([]any, error) {
	ns := "edg-default"

	namespace := Namespace(ns)
	coordinator := Coordinator(ns).DeploymentApplyConfiguration
	coordinatorService := ServiceForDeployment(coordinator)
	coordinatorForwarder := PortForwarder("coordinator", ns).
		WithListenPort(1313).
		WithForwardTarget("coordinator", 1313).
		PodApplyConfiguration

	workload := Deployment("workload", ns).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "workload"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "workload"}).
				WithSpec(PodSpec().
					WithRuntimeClassName("kata-cc-isolation").
					WithContainers(
						Container().
							WithName("workload").
							WithImage("docker.io/library/busybox:1.36.1-musl@sha256:d4707523ce6e12afdbe9a3be5ad69027150a834870ca0933baf7516dd1fe0f56").
							WithCommand("/bin/sh", "-c", "echo Workload started ; while true; do sleep 60; done").
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)
	workload, err := AddInitializer(workload, Initializer())
	if err != nil {
		return nil, err
	}

	resources := []any{
		namespace,
		coordinator,
		coordinatorService,
		coordinatorForwarder,
		workload,
	}

	return resources, nil
}

// OpenSSL returns a set of resources for testing with OpenSSL.
func OpenSSL() ([]any, error) {
	ns := "edg-default"
	namespace := Namespace(ns)
	coordinator := Coordinator(ns).DeploymentApplyConfiguration
	coordinatorService := ServiceForDeployment(coordinator)
	coordinatorForwarder := PortForwarder("coordinator", ns).
		WithListenPort(1313).
		WithForwardTarget("coordinator", 1313).
		PodApplyConfiguration

	backend := Deployment("openssl-backend", ns).
		WithSpec(DeploymentSpec().
			WithReplicas(1).
			WithSelector(LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "openssl-backend"}),
			).
			WithTemplate(PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": "openssl-backend"}).
				WithSpec(PodSpec().
					WithRuntimeClassName("kata-cc-isolation").
					WithContainers(
						Container().
							WithName("openssl-backend").
							WithImage("ghcr.io/edgelesssys/contrast/openssl:latest").
							WithCommand("/bin/bash", "-c", "echo Workload started \nopenssl s_server -port 443 -Verify 2 -CAfile /tls-config/MeshCACert.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem").
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
					WithRuntimeClassName("kata-cc-isolation").
					WithContainers(
						Container().
							WithName("openssl-client").
							WithImage("ghcr.io/edgelesssys/contrast/openssl:latest").
							WithCommand("/bin/bash", "-c", "echo Workload started \nwhile true; do \n  echo \"THIS IS A TEST MESSAGE\" |\n    openssl s_client -connect openssl-frontend:443 -verify_return_error -CAfile /tls-config/RootCACert.pem\n  sleep 30\ndone\n").
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
					WithRuntimeClassName("kata-cc-isolation").
					WithContainers(
						Container().
							WithName("openssl-frontend").
							WithImage("ghcr.io/edgelesssys/contrast/openssl:latest").
							WithCommand("/bin/bash", "-c", "echo Workload started\nopenssl s_server -www -port 443 -cert /tls-config/certChain.pem -key /tls-config/key.pem -cert_chain /tls-config/certChain.pem &\nwhile true; do \n  echo \"THIS IS A TEST MESSAGE\" |\n    openssl s_client -connect openssl-backend:443 -verify_return_error -CAfile /tls-config/MeshCACert.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem\n  sleep 10\ndone\n").
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

	portforwarderCoordinator := PortForwarder("coordinator", ns).
		WithListenPort(1313).
		WithForwardTarget("coordinator", 1313).
		PodApplyConfiguration

	portforwarderOpenSSLFrontend := PortForwarder("openssl-frontend", ns).
		WithListenPort(443).
		WithForwardTarget("openssl-frontend", 443).
		PodApplyConfiguration

	resources := []any{
		namespace,
		coordinator,
		coordinatorService,
		coordinatorForwarder,
		backend,
		backendService,
		client,
		frontend,
		frontendService,
		portforwarderCoordinator,
		portforwarderOpenSSLFrontend,
	}

	return resources, nil
}

// Emojivoto returns resources for deploying EmojiVoto application.
func Emojivoto() ([]any, error) {
	ns := "edg-default"
	namespace := Namespace(ns)
	coordinator := Coordinator(ns).DeploymentApplyConfiguration
	coordinatorService := ServiceForDeployment(coordinator)
	coordinatorForwarder := PortForwarder("coordinator", ns).
		WithListenPort(1313).
		WithForwardTarget("coordinator", 1313).
		PodApplyConfiguration

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
					WithRuntimeClassName("kata-cc-isolation").
					WithServiceAccountName("emoji").
					WithContainers(
						Container().
							WithName("emoji-svc").
							WithImage("ghcr.io/3u13r/emojivoto-emoji-svc:coco-1").
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
							WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/MeshCACert.pem")).
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
							WithImage("ghcr.io/3u13r/emojivoto-web:coco-1").
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
					WithRuntimeClassName("kata-cc-isolation").
					WithServiceAccountName("voting").
					WithContainers(
						Container().
							WithName("voting-svc").
							WithImage("ghcr.io/3u13r/emojivoto-voting-svc:coco-1").
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
							WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/MeshCACert.pem")).
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
					WithRuntimeClassName("kata-cc-isolation").
					WithServiceAccountName("web").
					WithContainers(
						Container().
							WithName("web-svc").
							WithImage("ghcr.io/3u13r/emojivoto-web:coco-1").
							WithPorts(
								ContainerPort().
									WithName("https").
									WithContainerPort(8080),
							).
							WithEnv(EnvVar().WithName("WEB_PORT").WithValue("8080")).
							WithEnv(EnvVar().WithName("EMOJISVC_HOST").WithValue("emoji-svc:8080")).
							WithEnv(EnvVar().WithName("VOTINGSVC_HOST").WithValue("voting-svc:8080")).
							WithEnv(EnvVar().WithName("INDEX_BUNDLE").WithValue("dist/index_bundle.js")).
							WithEnv(EnvVar().WithName("EDG_CERT_PATH").WithValue("/tls-config/certChain.pem")).
							WithEnv(EnvVar().WithName("EDG_CA_PATH").WithValue("/tls-config/MeshCACert.pem")).
							WithEnv(EnvVar().WithName("EDG_KEY_PATH").WithValue("/tls-config/key.pem")).
							WithEnv(EnvVar().WithName("EDG_DISABLE_CLIENT_AUTH").WithValue("true")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)
	web, err = AddInitializer(web, Initializer())
	if err != nil {
		return nil, err
	}

	webService := ServiceForDeployment(web).
		WithName("web-svc").
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

	portforwarderCoordinator := PortForwarder("coordinator", ns).
		WithListenPort(1313).
		WithForwardTarget("coordinator", 1313).
		PodApplyConfiguration

	portforwarderemojivotoWeb := PortForwarder("emojivoto-web", ns).
		WithListenPort(8080).
		WithForwardTarget("web-svc", 443).
		PodApplyConfiguration

	resources := []any{
		namespace,
		coordinator,
		coordinatorService,
		coordinatorForwarder,
		emoji,
		emojiService,
		emojiserviceAccount,
		portforwarderCoordinator,
		portforwarderemojivotoWeb,
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
