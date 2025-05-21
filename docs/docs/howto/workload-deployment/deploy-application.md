# Deploy application

Now its time to deploy your application resources.

## Applicability

This step is mandatory for all Contrast deployments.

## Prerequisites

1. [Set up cluster](../cluster-setup/aks.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Prepare deployment files](./deployment-file-preparation.md)
5. [Configure TLS (optional)](./TLS-configuration.md)
6. [Enable GPU support (optional)](./GPU-configuration.md)
7. [Generate annotations and manifest](./generate-annotations.md)

## How-to

Apply the resources to the cluster. Your workloads will block in the initialization phase until a
manifest is set at the Coordinator.

```sh
kubectl apply -f resources/
```
