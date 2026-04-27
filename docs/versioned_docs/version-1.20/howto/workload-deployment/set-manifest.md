# Set manifest

Setting the manifest enables the Contrast Coordinator to verify the deployment.

## Applicability

This step is mandatory for all Contrast deployments. Workloads won't start until a valid manifest has been configured.

## Prerequisites

1. [Set up cluster](../cluster-setup/bare-metal.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Add Coordinator to resources](./add-coordinator.md)
5. [Prepare deployment files](./deployment-file-preparation.md)
6. [Configure TLS (optional)](./TLS-configuration.md)
7. [Enable GPU support (optional)](./GPU-configuration.md)
8. [Generate annotations and manifest](./generate-annotations.md)
9. [Deploy application](./deploy-application.md)
10. [Connect to Coordinator](./connect-to-coordinator.md)

## How-to

Attest the Coordinator and set the manifest:

```sh
contrast set -c "${coordinator}:1313" resources/
```

This will use the reference values from the manifest file to attest the Coordinator. After this step, the Coordinator will start issuing TLS certificates to the workloads. The init container will fetch a certificate for the workload and the workload is started.
