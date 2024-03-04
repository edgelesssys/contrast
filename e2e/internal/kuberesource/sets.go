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
