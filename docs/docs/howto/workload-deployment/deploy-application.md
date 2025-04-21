# Deploy application

Now its time to deploy your application resources.

## Applicability

This step is mandatory for all Contrast deployments.

## Prerequisites

1. [Set up cluster](.)
2. [Deploy runtime](.)
3. [Prepare deployment files](.)
4. [Configure TLS (optional)](.)
5. [Enable GPU support (optional)](.)
6. [Generate annotations and manifest](.)

## How-to

Apply the resources to the cluster. Your workloads will block in the initialization phase until a
manifest is set at the Coordinator.

```sh
kubectl apply -f resources/
```
