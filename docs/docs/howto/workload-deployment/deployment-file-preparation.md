# Prepare deployment files

To run your Kubernetes workloads as Confidential Containers, you'll need to make a few adjustments to your deployment files. This section walks you through the required changes and highlights important security considerations for your application.

## Applicability

Required for all Contrast deployments.

## Prerequisites

1. [Set up cluster](../cluster-setup/aks.md)
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

When calculating the VM resource requirements, init containers aren't taken into account.
If the sum of init container's memory limits surpass the sum of the main containers' memory limits, you need to increase the memory limit of one of the main containers in the pod.
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

On bare metal platforms, container images are pulled from within the guest CVM and stored in encrypted memory.
The images are pulled and uncompressed into encrypted memory, so the uncompressed image size needs to be taken into account when setting the container limits.
Registry interfaces often show the compressed size of an image, the decompressed image is usually a factor of 2-4x larger if the content is mostly binary.
For example, the `nginx:stable` image reports a compressed image size of 67MiB, but storing the uncompressed layers needs about 184MiB of memory.
Although only the extracted layers are stored, and those layers are reused across containers within the same pod, the memory limit should account for both the compressed and the decompressed layer simultaneously.
Altogether, setting the limit to 10x the compressed image size should be sufficient for small to medium images.

:::

Getting a predictable number of CPUs in your containers can be a bit more complicated than just setting a resource request since Contrast runs on Kata Containers.
Also note that, as mentioned above, resource allocations need to be known and static at the time of Pod creation and can not be adjusted at runtime. This prevents
us from implementing requests and limits as documented by Kubernetes.
In your YAML, set a resource limit instead of a request:

```yaml
spec: # v1.PodSpec
  containers:
    - name: my-container
      image: "my-image@sha256..."
      resources:
        limits:
          cpu: "1"
  runtimeClassName: contrast-cc
```

This limit is rounded up to the next whole integer. The final number of CPUs you can observe in a container is this ceiling + 1 (`numCPUs = ceil(limit) + 1`), so
in this case, the final number of CPUs would be 2. A limit of 1.25 would result in 3 CPUs and a limit of 0 would result in 1 CPU.

:::warning

CPU limits are currently only supported on AKS and will cause measurement errors on bare metal. Be aware
of things that might change container limits, like `LimitRange` or pod admission controllers.

:::

[Kubernetes resource management]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
