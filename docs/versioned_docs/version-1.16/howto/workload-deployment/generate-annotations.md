# Generate initdata annotations and manifest

This step updates your deployment files with initdata annotations and automatically generates the deployment manifest.

## Applicability

This step is required for all Contrast deployments.

## Prerequisites

1. [Set up cluster](../cluster-setup/bare-metal.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Add Coordinator to resources](./add-coordinator.md)
5. [Prepare deployment files](./deployment-file-preparation.md)
6. [Configure TLS (optional)](./TLS-configuration.md)
7. [Enable GPU support (optional)](./GPU-configuration.md)

## How-to

Run the `generate` command to add the necessary components to your deployment files.
This will add the Contrast Initializer to every workload with the specified `contrast-cc` runtime class
and the Contrast Service Mesh to all workloads that have a specified configuration.
After that, it will generate the [execution policies](../../architecture/components/policies.md), wrap them in initdata documents and add them as annotations to your deployment files.
A `manifest.json` with the reference values of your deployment will be created.

<Tabs queryString="platform">
<TabItem value="metal-qemu-snp" label="Bare metal (SEV-SNP)">

```sh
contrast generate --reference-values metal-qemu-snp resources/
```

On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values as they can vary between platforms and CPU models.
They will have to be filled in manually.
Find a detailed description of the meaning of these values and how to acquire them in the [`MinimumTCB` section of the manifest documentation](../../architecture/components/manifest.md#snp-minimum-tcb).

</TabItem>
<TabItem value="metal-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">

```sh
contrast generate --reference-values metal-qemu-snp-gpu resources/
```

On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values as they can vary between platforms and CPU models.
They will have to be filled in manually.
Find a detailed description of the meaning of these values and how to acquire them in the [`MinimumTCB` section of the manifest documentation](../../architecture/components/manifest.md#snp-minimum-tcb).

</TabItem>
<TabItem value="metal-qemu-tdx" label="Bare metal (TDX)">

```sh
contrast generate --reference-values metal-qemu-tdx resources/
```

On bare-metal TDX, `contrast generate` is unable to fill in the `MrSeam` value as it depends on your platform configuration.
It will have to be filled in manually.
Find a detailed description of the meaning of this value and how to acquire it in the [`TDXReferenceValues` section of the manifest documentation](../../architecture/components/manifest.md#tdx-mr-seam).

</TabItem>
<TabItem value="metal-qemu-tdx-gpu" label="Bare metal (TDX, with GPU support)">

```sh
contrast generate --reference-values metal-qemu-tdx-gpu resources/
```

On bare-metal TDX, `contrast generate` is unable to fill in the `MrSeam` value as it depends on your platform configuration.
It will have to be filled in manually.
Find a detailed description of the meaning of this value and how to acquire it in the [`TDXReferenceValues` section of the manifest documentation](../../architecture/components/manifest.md#tdx-mr-seam).

</TabItem>
</Tabs>

The `generate` command needs to pull the container images to derive policies.
Running `generate` for the first time can take a while, especially if the images are large.
If your container registry requires authentication, you can create the necessary credentials with `docker login` or `podman login`.
Be aware of the [registry authentication limitation](../../architecture/features-limitations.md#kubernetes-features) on bare metal.

:::warning
Please be aware that runtime policies currently have some blind spots.
For example, they can't guarantee the starting order of containers.
See the [current limitations](../../architecture/features-limitations.md#runtime-policies) for more details.
:::

Running `contrast generate` for the first time creates some additional files in the working directory:

- `seedshare-owner.pem` is required for handling the secret seed and recovering the Coordinator (see [Secrets & recovery](../../architecture/secrets.md)).
- `workload-owner.pem` is required for manifest updates after the initial `contrast set`.
- `rules.rego` and `settings.json` are the basis for [runtime policies](../../architecture/components/policies.md).
- `layers-cache.json` caches container image layer information for your deployments to speed up subsequent runs of `contrast generate`.

### Fine-tuning initializer injection

If you don't want the Contrast Initializer to automatically be added to your
workloads, there are two ways you can skip the Initializer injection step,
depending on how you want to customize your deployment.

#### `--skip-initializer` flag

You can disable the Initializer injection completely by specifying the
`--skip-initializer` flag in the `generate` command.

<Tabs queryString="platform">
<TabItem value="metal-qemu-snp" label="Bare metal (SEV-SNP)">

```sh
contrast generate --reference-values metal-qemu-snp --skip-initializer resources/
```

</TabItem>
<TabItem value="metal-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">

```sh
contrast generate --reference-values metal-qemu-snp-gpu --skip-initializer resources/
```

</TabItem>
<TabItem value="metal-qemu-tdx" label="Bare metal (TDX)">

```sh
contrast generate --reference-values metal-qemu-tdx --skip-initializer resources/
```

</TabItem>
<TabItem value="metal-qemu-tdx-gpu" label="Bare metal (TDX, with GPU support)">

```sh
contrast generate --reference-values metal-qemu-tdx-gpu --skip-initializer resources/
```

</TabItem>
</Tabs>

#### `skip-initializer` annotation

If you want to disable the Initializer injection for a specific workload with
the `contrast-cc` runtime class, you can do so by adding an annotation to the workload.

```yaml
metadata: # v1.Pod, v1.PodTemplateSpec
  annotations:
    contrast.edgeless.systems/skip-initializer: "true"
```

#### Manual Initializer injection

When disabling the automatic Initializer injection, you can manually add the
Initializer as a sidecar container to your workload before generating the
policies. Configure the workload to use the certificates written to the
`contrast-secrets` `volumeMount`.

```yaml
# v1.PodSpec
spec:
  initContainers:
    - env:
        - name: COORDINATOR_HOST
          value: coordinator-ready
      image: "ghcr.io/edgelesssys/contrast/initializer:v1.16.1@sha256:c3058cd5afb8f433ddf35bde7bbc14e187634f26cb1f000a76b4d3248943e5ae"
      name: contrast-initializer
      volumeMounts:
        - mountPath: /contrast
          name: contrast-secrets
  volumes:
    - emptyDir: {}
      name: contrast-secrets
```

### Fine-tuning service mesh injection

The service mesh is only injected for workload that have a [service mesh annotation](../../architecture/components/service-mesh.md#configuring-the-proxy).

#### `--skip-service-mesh` flag

You can disable the service mesh injection completely by specifying the
`--skip-service-mesh` flag in the `generate` command.

<Tabs queryString="platform">
<TabItem value="metal-qemu-snp" label="Bare metal (SEV-SNP)">

```sh
contrast generate --reference-values metal-qemu-snp --skip-service-mesh resources/
```

</TabItem>
<TabItem value="metal-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">

```sh
contrast generate --reference-values metal-qemu-snp-gpu --skip-service-mesh resources/
```

</TabItem>
<TabItem value="metal-qemu-tdx" label="Bare metal (TDX)">

```sh
contrast generate --reference-values metal-qemu-tdx --skip-service-mesh resources/
```
</TabItem>
<TabItem value="metal-qemu-tdx-gpu" label="Bare metal (TDX, with GPU support)">

```sh
contrast generate --reference-values metal-qemu-tdx-gpu --skip-service-mesh resources/
```
</TabItem>
</Tabs>

In this case, you can manually add the service mesh sidecar container to your workload before generating the policies, or [authenticate peers on the application level](./TLS-configuration.md#go-integration).
