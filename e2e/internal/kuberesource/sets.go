package kuberesource

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

	opensslBackend := Deployment("openssl-backend", ns).
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
							WithCommand("/bin/sh", "-c", "echo Workload started  \n openssl s_server -port 443 -Verify 2 -CAfile /tls-config/MeshCACert.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem").
							WithEnv(
								NewEnvVar("COORDINATOR_HOST", "coordinator"),
							).
							WithPorts(
								ContainerPort().
									WithName("openssl").
									WithContainerPort(443),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)

	opensslBackend, err := AddInitializer(opensslBackend, Initializer())
	if err != nil {
		return nil, err
	}

	opensslBackendService := ServiceForDeployment(opensslBackend)

	opensslClient := Deployment("openssl-client", ns).
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
							WithCommand("/bin/sh", "-c", "echo Workload started \nwhile true; do \n  echo \"THIS IS A TEST MESSAGE\" | openssl s_client -connect openssl-frontend:443 -verify_return_error -CAfile /tls-config/RootCACert.pem\n  sleep 30\ndone\n").
							WithEnv(
								NewEnvVar("COORDINATOR_HOST", "coordinator"),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)
	opensslClient, err = AddInitializer(opensslClient, Initializer())
	if err != nil {
		return nil, err
	}

	opensslFrontend := Deployment("openssl-frontend", ns).
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
							WithCommand("/bin/sh", "-c", "echo Workload started\nopenssl s_server -www -port 443 -cert /tls-config/certChain.pem -key /tls-config/key.pem -cert_chain /tls-config/certChain.pem &\nwhile true; do \n  echo \"THIS IS A TEST MESSAGE\" | openssl s_client -connect openssl-backend:443 -verify_return_error -CAfile /tls-config/MeshCACert.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem\n  sleep 10\ndone\n").
							WithEnv(
								NewEnvVar("COORDINATOR_HOST", "coordinator"),
							).
							WithResources(ResourceRequirements().
								WithMemoryLimitAndRequest(50),
							),
					),
				),
			),
		)
	opensslFrontend, err = AddInitializer(opensslFrontend, Initializer())
	if err != nil {
		return nil, err
	}

	opensslFrontendService := ServiceForDeployment(opensslFrontend)

	portforwarderCoordinator := PortForwarder("coordinator", ns).
		WithListenPort(1313).
		WithForwardTarget("coordinator", 1313).
		PodApplyConfiguration
	portforwarderopensslFrontend := PortForwarder("openssl-frontend", ns).
		WithListenPort(443).
		WithForwardTarget("openssl-frontend", 443).
		PodApplyConfiguration

	resources := []any{
		namespace,
		coordinator,
		coordinatorService,
		coordinatorForwarder,
		opensslBackend,
		opensslBackendService,
		opensslClient,
		opensslFrontend,
		opensslFrontendService,
		portforwarderCoordinator,
		portforwarderopensslFrontend,
	}

	return resources, nil
}
