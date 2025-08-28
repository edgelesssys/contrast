# Add the Contrast Coordinator to your resources

This step adds an additional service to your resources. The Coordinator takes care of verifying your deployment.

## Applicability

This step is mandatory for all Contrast deployments.

## Prerequisite

1. [Set up cluster](../cluster-setup/aks.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)

## How-to
Download the Kubernetes resource of the Contrast Coordinator, comprising a single replica deployment and a LoadBalancer service. Put it next to your resources:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/download/v1.12.2/coordinator.yml --output-dir resources
```
