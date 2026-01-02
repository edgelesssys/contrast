# Deploy the Contrast runtime

This step configures the host environment on your Kubernetes worker nodes.

## Applicability

Required for all Contrast deployments.

## Prerequisites

1. [Set up cluster](../cluster-setup/bare-metal.md)
2. [Install CLI](../install-cli.md)

## How-to

Contrast depends on a [custom Kubernetes `RuntimeClass` (`contrast-cc`)](../../architecture/components/runtime.md), which needs to be installed in the cluster prior to the Coordinator or any confidential workloads.
This consists of a `RuntimeClass` resource and the node installer `DaemonSet` that performs installation on worker nodes.
This step is only required once for each version of the runtime.
It can be shared between Contrast deployments.
Also, different Contrast runtime versions can be installed in the same cluster.

<Tabs queryString="platform">
<TabItem value="metal-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-metal-qemu-snp.yml
```
</TabItem>
<TabItem value="metal-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-metal-qemu-snp-gpu.yml
```
</TabItem>
<TabItem value="metal-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-metal-qemu-tdx.yml
```
</TabItem>
<TabItem value="metal-qemu-tdx-gpu" label="Bare metal (TDX, with GPU support)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-metal-qemu-tdx-gpu.yml
```
</TabItem>
</Tabs>

:::warning[Modifications to containerd configuration]

The Contrast node installer will modify the containerd configuration on the worker nodes to add the runtime class.
A backup will be created for the original configuration.

Some Kubernetes platforms, for example K3s, use a template for the containerd configuration.
Notice that Contrast can't modify these templates, but will write the templated version instead.
Any modifications made to the template afterward won't take effect.

:::
