package kuberesource

import (
	"k8s.io/apimachinery/pkg/util/intstr"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

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

// GenerateEmojivoto returns resources for deploying EmojiVoto application with custom images.
func GenerateEmojivoto(ns string, emojiImage, initializerImage, portforwarderImage, votingImage, webImage string, generateCoordinatorService bool) ([]any, error) {
	var resources []any

	if ns == "" {
		ns = "edg-default"
	}

	namespace := Namespace(ns)

	if generateCoordinatorService {
		coordinator := Coordinator(ns).DeploymentApplyConfiguration
		coordinatorService := ServiceForDeployment(coordinator)
		coordinatorForwarder := PortForwarder("coordinator", ns).
			WithListenPort(1313).
			WithForwardTarget("coordinator", 1313).
			PodApplyConfiguration
		resources = append(resources, namespace, coordinator, coordinatorService, coordinatorForwarder)
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
					WithRuntimeClassName("kata-cc-isolation").
					WithServiceAccountName("emoji").
					WithContainers(
						Container().
							WithName("emoji-svc").
							WithImage(emojiImage).
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
	emoji, err := AddInitializer(emoji, Initializer().WithImage(initializerImage))
	if err != nil {
		return nil, err
	}
	resources = append(resources, emoji)

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
	resources = append(resources, emojiService)

	emojiserviceAccount := ServiceAccount("emoji", ns).
		WithAPIVersion("v1").
		WithKind("ServiceAccount")
	resources = append(resources, emojiserviceAccount)

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
							WithImage(webImage).
							WithCommand("emojivoto-vote-bot").
							WithEnv(EnvVar().WithName("WEB_HOST").WithValue("web-svc:443")).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(25),
							),
					),
				),
			),
		)
	resources = append(resources, voteBot)

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
							WithImage(votingImage).
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
	voting, err = AddInitializer(voting, Initializer().WithImage(initializerImage))
	if err != nil {
		return nil, err
	}
	resources = append(resources, voting)

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
	resources = append(resources, votingService)

	votingserviceAccount := ServiceAccount("voting", ns).
		WithAPIVersion("v1").
		WithKind("ServiceAccount")
	resources = append(resources, votingserviceAccount)

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
							WithImage(webImage).
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
	web, err = AddInitializer(web, Initializer().WithImage(initializerImage))
	if err != nil {
		return nil, err
	}
	resources = append(resources, web)

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
	resources = append(resources, webService)

	webserviceAccount := ServiceAccount("web", ns).
		WithAPIVersion("v1").
		WithKind("ServiceAccount")
	resources = append(resources, webserviceAccount)

	portforwarderCoordinator := PortForwarder("coordinator", ns).
		WithListenPort(1313).
		WithForwardTarget("coordinator", 1313).
		PodApplyConfiguration
	resources = append(resources, portforwarderCoordinator)

	portforwarderWeb := PortForwarder("emojivoto-web", ns).
		WithSpec(PodSpec().
			WithContainers(
				Container().
					WithName("port-forwarder").
					WithImage(portforwarderImage).
					WithEnv(EnvVar().WithName("LISTEN_PORT").WithValue("8080")).
					WithEnv(EnvVar().WithName("FORWARD_HOST").WithValue("web-svc")).
					WithEnv(EnvVar().WithName("FORWARD_PORT").WithValue("443")).
					WithCommand("/bin/bash", "-c", "echo Starting port-forward with socat; exec socat -d -d TCP-LISTEN:${LISTEN_PORT},fork TCP:${FORWARD_HOST}:${FORWARD_PORT}").
					WithPorts(
						ContainerPort().
							WithContainerPort(8080),
					).
					WithResources(ResourceRequirements().
						WithMemoryLimitAndRequest(50),
					),
			),
		)
	resources = append(resources, portforwarderWeb)

	return resources, nil
}

// Emojivoto returns resources for deploying Emojivoto application.
func Emojivoto() ([]any, error) {
	return GenerateEmojivoto(
		"edg-default",
		"ghcr.io/3u13r/emojivoto-emoji-svc:coco-1",
		"ghcr.io/edgelesssys/contrast/initializer:latest",
		"ghcr.io/edgelesssys/contrast/port-forwarder:latest",
		"ghcr.io/3u13r/emojivoto-voting-svc:coco-1",
		"ghcr.io/3u13r/emojivoto-web:coco-1",
		false,
	)
}

// EmojivotoDemo returns resources for deploying a simple Emojivoto demo.
func EmojivotoDemo() ([]any, error) {
	vanilla, _ := Emojivoto()
	replacements := map[string]string{
		"ghcr.io/edgelesssys/contrast/initializer:latest":    "ghcr.io/3u13r/contrast/initializer@sha256:3f0e76ffd1c62af460d2a7407ca0ab13cd49b3f07a00d5ef5bd636bcb6d8381f",
		"ghcr.io/edgelesssys/contrast/port-forwarder:latest": "ghcr.io/3u13r/contrast/port-forwarder@sha256:00b02378ceb33df7db46a0b6b56fd7fe1e7b2e7dade0404957f16235c01e80e0",
	}
	patched := PatchImages(vanilla, replacements)
	patched = PatchNamespaces(patched, "default")
	return patched, nil
}

// PatchImages replaces images in a set of resources.
func PatchImages(resources []any, replacements map[string]string) []any {
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applyappsv1.DeploymentApplyConfiguration:
			for i := 0; i < len(r.Spec.Template.Spec.InitContainers); i++ {
				if replacement, ok := replacements[*r.Spec.Template.Spec.InitContainers[i].Image]; ok {
					r.Spec.Template.Spec.InitContainers[i].Image = &replacement
				}
			}
			for i := 0; i < len(r.Spec.Template.Spec.Containers); i++ {
				if replacement, ok := replacements[*r.Spec.Template.Spec.Containers[i].Image]; ok {
					r.Spec.Template.Spec.Containers[i].Image = &replacement
				}
			}
		case *applycorev1.PodApplyConfiguration:
			for i := 0; i < len(r.Spec.Containers); i++ {
				if replacement, ok := replacements[*r.Spec.Containers[i].Image]; ok {
					r.Spec.Containers[i].Image = &replacement
				}
			}
		case *applycorev1.ServiceApplyConfiguration:
			// Do nothing
		case *applycorev1.ServiceAccountApplyConfiguration:
			// Do nothing
		}
	}
	return resources
}

// PatchNamespaces replaces namespaces in a set of resources.
func PatchNamespaces(resources []any, namespace string) []any {
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applycorev1.PodApplyConfiguration:
			r.Namespace = &namespace
		case *applyappsv1.DeploymentApplyConfiguration:
			r.Namespace = &namespace
		case *applycorev1.ServiceApplyConfiguration:
			r.Namespace = &namespace
		case *applycorev1.ServiceAccountApplyConfiguration:
			r.Namespace = &namespace
		}
	}
	return resources
}
