# Deploy application

Now its time to deploy your application resources.

## Applicability

This step is mandatory for all Contrast deployments.

## Prerequisites

1. [Set up cluster](../cluster-setup/aks.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Add Coordinator to resources](./add-coordinator.md)
5. [Prepare deployment files](./deployment-file-preparation.md)
6. [Configure TLS (optional)](./TLS-configuration.md)
7. [Enable GPU support (optional)](./GPU-configuration.md)
8. [Generate annotations and manifest](./generate-annotations.md)

## How-to

Apply the resources to the cluster.

```sh
kubectl apply -f resources/
```

Until [a manifest is set](set-manifest.md), the Coordinator will report unready
and the workload pods will stay in the initialization phase.
