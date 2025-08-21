# Secure Image Store

This section provides guidance on how to configure Contrast's secure image store feature to match your needs.

## Applicability

Storing container images on an encrypted ephemeral disk instead of in-memory reduces the amount of memory required.
This is especially beneficial in deployments with tight memory constraints, or if your container images are very large in size.

## Prerequisites

A running Contrast deployment.

## How-To

The secure image store is enabled by default, providing each pod with `10Gi` of storage.
This amount can be adjusted on a per-pod basis, the feature can be disabled for individual pods, or it can be disabled globally.

### Adjusting the size of the image store

Add the following annotation to one of your pod definitions:

```yaml
metadata: # v1.Pod, v1.PodTemplateSpec
  annotations:
    contrast.edgeless.systems/image-store-size: 250Gi
```

Rerun `contrast generate` and reapply your deployment for the change to take effect:

```bash
contrast generate resources/
kubectl apply -f resources/
```

### Disabling the image store for a single pod

To disable the secure image store for a specific pod, set the value of the shown annotation to `0`, without a unit:

```yaml
metadata: # v1.Pod, v1.PodTemplateSpec
  annotations:
    contrast.edgeless.systems/image-store-size: 0
```

Rerun `contrast generate` and reapply your deployment for the change to take effect:

```bash
contrast generate resources/
kubectl apply -f resources/
```

### Disabling the image store globally

To disable the image store globally, specify the `--skip-image-store` flag in the `contrast generate` command, then reapply your deployment:

```bash
contrast generate --skip-image-store resources/
kubectl apply -f resources/
```

## Memory considerations when disabling the secure image store

If the Contrast secure image store feature is disabled, container images are pulled and uncompressed into encrypted memory.
Memory limits must be adjusted accordingly.
See [Pod resources](./workload-deployment/deployment-file-preparation#pod-resources) for details.
