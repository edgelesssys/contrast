# Prepare deployment files

To run your Kubernetes workloads as Confidential Containers, you'll need to make a few adjustments to your deployment files. This section walks you through the required changes and highlights important security considerations for your application.

## Applicability

Required for all Contrast deployments.

## Prerequisites

1. [Set up cluster](../cluster-setup/aks.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)

## How-to

This page explains how to update your deployment files for Contrast and outlines key security consideration for your application.

### Security review

Contrast ensures integrity and confidentiality of the applications, but interactions with untrusted systems require the developers' attention.
Review the [security considerations](../hardening.md) and the [certificates](../../architecture/components/service-mesh.md#certificate-authority) section for writing secure Contrast application.

### RuntimeClass

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

To specify that a workload (pod, deployment, etc.) should be deployed as confidential containers,
add `runtimeClassName: contrast-cc` to the pod spec (pod definition or template).
This is a placeholder name that will be replaced by a versioned `runtimeClassName` when generating policies.

```yaml
spec: # v1.PodSpec
  runtimeClassName: contrast-cc
```

### Pod resources

Contrast workloads are deployed as one confidential virtual machine (CVM) per pod.
In order to configure the CVM resources correctly, Contrast workloads require a stricter specification of pod resources compared to standard [Kubernetes resource management].

The total memory available to the CVM is calculated from the sum of the individual containers' memory limits and a static `RuntimeClass` overhead that accounts for services running inside the CVM.
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

Contrast launches this pod as a VM with 448MiB of memory: 192MiB for the containers and 256MiB for the Linux kernel, the Kata agent and other base processes.

When calculating the VM resource requirements, init containers aren't taken into account.
If you have an init container that requires large amounts of memory, you need to adjust the memory limit of one of the main containers in the pod.
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
The CVM mounts a `tmpfs` for the image layers that's capped at 50% of the total VM memory.
This tmpfs holds the extracted image layers, so the uncompressed image size needs to be taken into account when setting the container limits.
Registry interfaces often show the compressed size of an image, the decompressed image is usually a factor of 2-4x larger if the content is mostly binary.
For example, the `nginx:stable` image reports a compressed image size of 67MiB, but storing the uncompressed layers needs about 184MiB of memory.
Although only the extracted layers are stored, and those layers are reused across containers within the same pod, the memory limit should account for both the compressed and the decompressed layer simultaneously.
Altogether, setting the limit to 10x the compressed image size should be sufficient for small to medium images.

:::

[Kubernetes resource management]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
