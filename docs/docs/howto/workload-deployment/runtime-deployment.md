# Deploy the Contrast runtime

This step configures the host environment on your Kubernetes worker nodes.

## Applicability

Required for all Contrast deployments.

## Prerequisites

1. [Set up cluster](../cluster-setup/aks.md)
2. [Install CLI](../install-cli.md)

## How-to

Contrast depends on a [custom Kubernetes `RuntimeClass` (`contrast-cc`)](../../architecture/components/runtime.md), which needs to be installed in the cluster prior to the Coordinator or any confidential workloads.
This consists of a `RuntimeClass` resource and a `DaemonSet` that performs installation on worker nodes.
This step is only required once for each version of the runtime.
It can be shared between Contrast deployments.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-aks-clh-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-snp-gpu.yml
```
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-tdx.yml
```
</TabItem>
</Tabs>
