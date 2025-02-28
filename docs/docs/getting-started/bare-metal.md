# Prepare a bare-metal instance

## Hardware and firmware setup

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">
1. Update your BIOS to a version that supports AMD SEV-SNP. Updating to the latest available version is recommended as newer versions will likely contain security patches for AMD SEV-SNP.
2. Enter BIOS setup to enable SMEE, IOMMU, RMP coverage, and SEV-SNP. Set the SEV-ES ASID Space Limit to a non-zero number (higher is better).
3. Download the latest firmware version for your processor from [AMD](https://www.amd.com/de/developer/sev.html), unpack it, and place it in `/lib/firmware/amd`.

Consult AMD's [Using SEV with AMD EPYC Processors user guide](https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/tuning-guides/58207-using-sev-with-amd-epyc-processors.pdf) for more information.
</TabItem>
<TabItem value="intel" label="Intel TDX">
Follow Canonical's instructions on [setting up Intel TDX in the host's BIOS](https://github.com/canonical/tdx?tab=readme-ov-file#43-enable-intel-tdx-in-the-hosts-bios).
</TabItem>
</Tabs>

## Kernel setup

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">
Install a kernel with version 6.11 or greater. If you're following this guide before 6.11 has been released, use 6.11-rc3. Don't use 6.11-rc4 - 6.11-rc6 as they contain a regression. 6.11-rc7+ might work.
</TabItem>
<TabItem value="intel" label="Intel TDX">
Follow Canonical's instructions on [setting up Intel TDX on Ubuntu 24.04](https://github.com/canonical/tdx?tab=readme-ov-file#41-install-ubuntu-2404-server-image). Note that Contrast currently only supports Intel TDX with Ubuntu 24.04.
</TabItem>
</Tabs>

Increase the `user.max_inotify_instances` sysctl limit by adding `user.max_inotify_instances=8192` to `/etc/sysctl.d/99-sysctl.conf` and running `sysctl --system`.

## K3s setup

1. Follow the [K3s setup instructions](https://docs.k3s.io/) to create a cluster.
2. Install a block storage provider such as [Longhorn](https://longhorn.io/docs/latest/deploy/install/install-with-kubectl/) and mark it as the default storage class.

## Preparing a cluster for GPU usage

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">
To enable GPU usage on a Contrast cluster, some conditions need to be fulfilled for *each cluster node* that should host GPU workloads:

1. Ensure that GPUs supporting confidential computing (CC) are available on the machine.

   ```sh
   lspci -nnk | grep '3D controller' -A3
   ```

   This should show a [CC-capable](https://www.nvidia.com/en-us/data-center/solutions/confidential-computing/) GPU like the NVIDIA H100:

   ```shell-session
   41:00.0 3D controller [0302]: NVIDIA Corporation GH100 [H100 PCIe] [10de:2331] (rev a1)
      Subsystem: NVIDIA Corporation GH100 [H100 PCIe] [10de:1626]
      Kernel driver in use: vfio-pci
      Kernel modules: nvidiafb, nouveau
   ```

   :::info
   Contrast doesn't support non-CC GPUs.
   :::

2. You must activate the IOMMU. You can check by running:

   ```sh
   ls /sys/kernel/iommu_groups
   ```

   If the output contains the group indices (`0`, `1`, ...), the IOMMU is supported on the host.
   Otherwise, add `intel_iommu=on` to the kernel command line.
3. Additionally, the host kernel needs to have the following kernel configuration options enabled:
    - `CONFIG_VFIO`
    - `CONFIG_VFIO_IOMMU_TYPE1`
    - `CONFIG_VFIO_MDEV`
    - `CONFIG_VFIO_MDEV_DEVICE`
    - `CONFIG_VFIO_PCI`
4. A CDI configuration needs to be present on the node. To generate it, you can use the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html).
   Refer to the official instructions on [how to generate a CDI configuration with it](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/cdi-support.html).


If the per-node requirements are fulfilled, deploy the [NVIDIA GPU Operator](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest) to the cluster. It provisions pod-VMs with GPUs via VFIO.

Initially, label all nodes that *should run GPU workloads*:

```sh
kubectl label node <node-name> nvidia.com/gpu.workload.config=vm-passthrough
```

For a GPU-enabled Contrast cluster, you can then deploy the operator with the following command:

```sh
helm install --wait --generate-name \
   -n gpu-operator --create-namespace \
   nvidia/gpu-operator \
   --version=v24.9.1 \
   --set sandboxWorkloads.enabled=true \
   --set sandboxWorkloads.defaultWorkload='vm-passthrough' \
   --set nfd.nodefeaturerules=true \
   --set vfioManager.enabled=true \
   --set ccManager.enabled=true
```

Refer to the [official installation instructions](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/getting-started.html) for details and further options.

Once the operator is deployed, check the available GPUs in the cluster:

```sh
kubectl get nodes -l nvidia.com/gpu.present -o json | \
  jq '.items[0].status.allocatable |
    with_entries(select(.key | startswith("nvidia.com/"))) |
    with_entries(select(.value != "0"))'
```

The above command should yield an output similar to the following, depending on what GPUs are available:

```json
{
   "nvidia.com/GH100_H100_PCIE": "1"
}
```

These identifiers are then used to [run GPU workloads on the cluster](../deployment.md).

</TabItem>
<TabItem value="intel" label="Intel TDX">
:::warning
Currently, Contrast only supports GPU workloads on SEV-SNP-based clusters.
:::
</TabItem>
</Tabs>
