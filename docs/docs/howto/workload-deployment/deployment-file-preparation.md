# Prepare deployment files

To run your Kubernetes workloads as confidential containers with Contrast, you'll need to make a few adjustments to your deployment files.
This section walks you through the required changes and highlights important security considerations for your application.

## Applicability

Required for all Contrast deployments.

## Prerequisites

1. [Set up cluster](../cluster-setup/bare-metal.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Add Coordinator to resources](./add-coordinator.md)

## How-to

This page explains how to update your deployment files for Contrast and outlines key security consideration for your application.

### Security review

Contrast ensures integrity and confidentiality of the applications, but interactions with untrusted systems require the developers' attention.
Review the [security considerations](../hardening.md) and the [certificates section](../../architecture/components/service-mesh.md#public-key-infrastructure) for deploying applications securely with Contrast.

### Resource backup

Contrast will add annotations to your Kubernetes YAML files. If you want to keep the original files
unchanged, you can copy the files into a separate local directory.
You can also generate files from a Helm chart or from a Kustomization.

<Tabs groupId="yaml-source">
<TabItem value="kustomize" label="kustomize">

```sh
mkdir resources
kustomize build $MY_RESOURCE_DIR > resources/all.yml
```

</TabItem>
<TabItem value="helm" label="helm">

```sh
mkdir resources
helm template $RELEASE_NAME $CHART_NAME > resources/all.yml
```

</TabItem>
<TabItem value="copy" label="copy">

```sh
cp -R $MY_RESOURCE_DIR resources/
```

</TabItem>
</Tabs>

### RuntimeClass

To specify that a workload (pod, deployment, etc.) should be executed with Contrast,
add `runtimeClassName: contrast-cc` to the pod spec (pod definition or template).

```yaml
spec: # v1.PodSpec
  runtimeClassName: contrast-cc
```

This is a placeholder name that will be replaced by a versioned `runtimeClassName` when generating policies.

<!-- TODO(katexochen): Describe how runtimeClass is handled after first generate -->

#### Multiple RuntimeClasses

Depending on the configuration of your cluster and the workloads to deploy, it can be desirable to use different runtime classes for pods in the same deployment.
For example, in a cluster consisting of both SEV-SNP and TDX machines, you might want to distribute the workload over all nodes.

Contrast supports these multi-runtime class configurations. For details, please see [the multi-runtime class How To](../multi-runtime-class.md).

### Namespaces

Contrast workloads can run in any namespace and a Contrast deployment can be spread across multiple namespaces.
The Coordinator doesn't have to be in the same namespace as the workloads and is configured to use the `default` namespace when first downloading the resource.
To better organize your deployment, you can create a dedicated namespace and change the Coordinator namespace in the deployment file.

### Volumes

Contrast doesn't support sharing filesystems with the host.
This means that most [Kubernetes volume](https://kubernetes.io/docs/concepts/storage/volumes/) mounts aren't supported.

The easiest and safest way to mount a persistent filesystem into a container is Contrast's [secure persistent volume](../encrypted-storage.md) support.
Alternatively, you can add a persistent volume with [`volumeMode: Block`](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#volume-mode) to your pod and mount it directly in your container.

The only supported filesystem volume types are:

- `configMap`
- `downwardAPI`
- `emptyDir`
- `projected`
- `secret`

<!-- TODO(burgerdev): ensure all of these are tested! -->

However, the content of these volumes isn't integrity protected and must be assumed untrustworthy, except for `emptyDir`.

All unsupported volume types will result in a temporary mount that's only available during the lifetime of the container.
If the volume mount on the host contains data (for example, mounts of type `hostPath` or `image`), this data will be copied into the VM, but not written back.
Mounted UNIX domain sockets won't work in the guest container.

### Pod resources

Contrast workloads are deployed as one confidential virtual machine (CVM) per pod.
The resources provided to the CVM are static and can't be adjusted at runtime.
In order to configure the CVM resources correctly, Contrast workloads require a stricter specification of pod resources compared to standard [Kubernetes resource management].

The total memory available to the CVM is calculated from the sum of the individual containers' memory `limits` and a static `RuntimeClass` overhead that accounts for services running inside the CVM.
Consider the following abbreviated example resource definitions:

```yaml
kind: RuntimeClass
handler: contrast-cc
overhead:
  podFixed:
    memory: 256Mi
---
spec: # v1.PodSpec
  containers:
    - name: my-container
      image: "my-image@sha256:..."
      resources:
        limits:
          memory: 128Mi
    - name: my-sidecar
      image: "my-other-image@sha256:..."
      resources:
        limits:
          memory: 64Mi
```

Contrast launches this pod as a VM with 448MiB of memory: 192MiB for the containers and 256MiB for the guest operating system.
In general, you don't need to care about the memory requirements of the guest system, as they're covered by the static `RuntimeClass` overhead.

Since memory can't be shared dynamically with the host, each container should have a memory limit that covers its worst-case requirements.

Kubernetes packs a node until the sum of pod _requests_ reaches the node's total memory.
Since a Contrast pod is always going to consume node memory according to the _limits_, the accounting is only correct if the request is equal to the limit.
Thus, once you determined the memory requirements of your application, you should add a resource section to the pod specification with request and limit:

```yaml
spec: # v1.PodSpec
  containers:
    - name: my-container
      image: "my-image@sha256:..."
      resources:
        requests:
          memory: 50Mi
        limits:
          memory: 50Mi
```

:::note

Container images are pulled from within the pod-VMs using the [Contrast image puller](../../architecture/components/runtime#pod-vm-image).
By default, images will be pulled into encrypted memory, and the memory limits must be set to a high enough value to take the uncompressed image sizes into account.
Alternatively, the images can instead be pulled and uncompressed into an encrypted storage device using [Contrast secure image store](../secure-image-store.md).
In this case, image sizes don't need to be taken into account when setting the container memory limits.

Registry interfaces often show the compressed size of an image, the decompressed image is usually a factor of 2-4x larger if the content is mostly binary.
For example, the `nginx:stable` image reports a compressed image size of 67MiB, but storing the uncompressed layers needs about 184MiB of memory.
Although only the extracted layers are stored, and those layers are reused across containers within the same pod, the memory limit should account for both the compressed and the decompressed layer simultaneously.
Altogether, setting the limit to 10x the compressed image size should be sufficient for small to medium images when not using the Contrast secure image store feature.

:::

:::warning

CPU limits are currently not supported and will cause measurement errors on bare metal. Be aware
of things that might change container limits, like `LimitRange` or pod admission controllers.

:::

[Kubernetes resource management]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
