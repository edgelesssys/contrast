# Configuring the secure image store

This section provides guidance on how to configure Contrast's secure image store feature, allowing you to adapt it to fit with your memory constraints and security requirements.

## Applicability

Storing container images on an encrypted ephemeral disk instead of in-memory reduces the amount of memory required.
This is especially beneficial in deployments with tight memory constraints, or if your container images are very large in size.

## Prerequisites

A running Contrast deployment.

## How-To

The secure image store is disabled by default, and images are therefore pulled into memory.
To enable the image store, specify the `--inject-image-store` flag in the `contrast generate` command, then apply your deployment:

```bash
contrast generate --inject-image-store resources/
kubectl apply -f resources/
```

By default, this provides each pod with `10Gi` of storage.
This amount can be adjusted on a per-pod basis, and the feature can be disabled for individual pods.

Possible use-cases include the accommodation of very large images, or working in environments where memory restrictions are tight, and the [slight weakening of the security posture](./hardening#limitations-inherent-to-policy-checking) is acceptable.

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

To disable the secure image store for a specific pod, set the value of the shown annotation to `"0"`, without a unit:

```yaml
metadata: # v1.Pod, v1.PodTemplateSpec
  annotations:
    contrast.edgeless.systems/image-store-size: "0"
```

*(The quotation marks around `"0"` are required for the value to be treated as a string.)*

Rerun `contrast generate` and reapply your deployment for the change to take effect:

```bash
contrast generate resources/
kubectl apply -f resources/
```

### Manual image store setup

If you need further customization, for example because you require the use of a custom [CSI driver](https://kubernetes-csi.github.io/docs/drivers.html),
you can disable the image store injection (either [for a single pod](#disabling-the-image-store-for-a-single-pod) or globally by omitting the `--inject-image-store` flag, and instead provide your own image store.

Internally, whether or not the image store is used depends exclusively on the presence of the magic device `/dev/image_store`.
If a device with this name is present in the pod-VM, it will be prepared, mounted and used as the secure image store.
This requires that the device is bound to a [sidecar container](https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/).
To the definition of the pod for which you wish to handle the secure image store manually, add a sidecar `initContainer` and an ephemeral volume:

```yaml
spec: # v1.Pod, v1.PodTemplateSpec
  initContainers:
    - image: "my-image@sha256:..."
      command:
        - /usr/local/bin/bash
        - "-c"
        - sleep infinity
      name: my-image-store
      resources:
        limits:
          memory: 50Mi
        requests:
          memory: 50Mi
      restartPolicy: Always
      securityContext:
        privileged: true
      volumeDevices:
        - devicePath: /dev/image_store
          name: image-store
  volumes:
    - ephemeral:
        volumeClaimTemplate:
          spec:
            accessModes:
              - ReadWriteOnce
            resources:
              requests:
                storage: 10Gi
            volumeMode: Block
      name: image-store
```

The configuration shown above creates a volume and a sidecar container to which it's bound.
All subsequent image pulls will target this volume.
Adjust the configuration to your liking, within the bounds of the following requirements:

- The volume **must** be bound to an `initContainer`.
- This `initContainer` **must** run for the entire lifetime of the deployment.
- Its `restartPolicy` **must** be `Always`, as this is what turns the `initContainer` into a sidecar container.
- The `devicePath` under which the volume is mounted **must** be `/dev/image_store`.

## Memory considerations when disabling the secure image store

If the Contrast secure image store feature is disabled, container images are pulled and uncompressed into encrypted memory.
Memory limits must be adjusted accordingly.
See [Pod resources](./workload-deployment/deployment-file-preparation#pod-resources) for details.

## CSI driver considerations

Since the image store is ephemeral by nature, most features and safeties provided by typical CSI implementations (for example, replication or snapshotting features) aren't required in this context.
Indeed, due to the complexity and overhead associated with these drivers, they may be detrimental.
For best results, choose a CSI provider focused on efficient handling of short-lived volumes.

To use a specific CSI driver only for the image stores, manually configure the appropriate `storageClassName` in the `volumeClaimTemplate` as shown above.
