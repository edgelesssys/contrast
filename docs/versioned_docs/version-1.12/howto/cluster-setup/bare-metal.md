# Prepare a bare-metal instance

## Prerequisites

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">

- A supported CPU:
  - AMD Epyc 7003 series (Milan)
  - AMD Epyc 9004 series (Genoa)

</TabItem>
<TabItem value="intel" label="Intel TDX">

- A supported CPU:
  - 5th Gen Intel Xeon Scalable Processor
  - Intel Xeon 6 Processors
- Platform must fulfill the [DIMM requirements](https://cc-enabling.trustedservices.intel.com/intel-tdx-enabling-guide/03/hardware_selection/#dimm-ie-main-memory-requirements).

</TabItem>
</Tabs>

## Hardware and firmware setup

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">

1. Update your BIOS to a version that supports AMD SEV-SNP. Updating to the latest available version is recommended as newer versions will likely contain security patches for AMD SEV-SNP.
2. Enter BIOS setup to enable SMEE, IOMMU, RMP coverage, and SEV-SNP. Set the SEV-ES ASID Space Limit to a non-zero number (higher is better).
3. Download the latest firmware version for your processor from [AMD](https://www.amd.com/de/developer/sev.html), unpack it, and place it in `/lib/firmware/amd`.

Consult AMD's [Using SEV with AMD EPYC Processors user guide](https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/tuning-guides/58207-using-sev-with-amd-epyc-processors.pdf) for more information.

</TabItem>
<TabItem value="intel" label="Intel TDX">

Follow Canonical's instructions in [4.2 Enable Intel TDX in Host OS](https://github.com/canonical/tdx?tab=readme-ov-file#42-enable-intel-tdx-in-host-os) (set `TDX_SETUP_ATTESTATION=1` in `setup-tdx-config`), [4.3 Enable Intel TDX in the Host's BIOS](https://github.com/canonical/tdx?tab=readme-ov-file#43-enable-intel-tdx-in-the-hosts-bios) and [9.2 Setup Intel® SGX Data Center Attestation Primitives (Intel® SGX DCAP) on the Host OS](https://github.com/canonical/tdx?tab=readme-ov-file#92-setup-intel-sgx-data-center-attestation-primitives-intel-sgx-dcap-on-the-host-os) (skipping step 9.2.1).
You can ignore the other sections of the document.

Follow Intel's guide to [Update Intel TDX Module via Binary Deployment](https://cc-enabling.trustedservices.intel.com/intel-tdx-enabling-guide/04/hardware_setup/#update-intel-tdx-module-via-binary-deployment). Intel recommends to install the latest TDX module version available.

</TabItem>
</Tabs>

## Kernel setup

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">
Install Linux kernel 6.11 or greater.
</TabItem>
<TabItem value="intel" label="Intel TDX">
Follow Canonical's instructions on [setting up Intel TDX on Ubuntu 24.04](https://github.com/canonical/tdx?tab=readme-ov-file#41-install-ubuntu-server-image). Note that Contrast currently only supports Intel TDX with Ubuntu 24.04.
</TabItem>
</Tabs>

Containerd uses a significant amount of `inotify` instances, so we recommend to allow at least 8192.
If necessary, the default can be increased by creating a config override file (for example in `/etc/sysctl.d/98-containerd.conf`) with the following content:

```ini
fs.inotify.max_user_instances = 8192
```

Apply this change by running `systemctl restart systemd-sysctl` and verify it using `sysctl fs.inotify.max_user_instances`.

## K3s setup

1. Follow the [K3s setup instructions](https://docs.k3s.io/) to create a cluster.
   Contrast is currently tested with K3s version `v1.31.5+k3s1`.
2. Install a block storage provider such as [Longhorn](https://longhorn.io/docs/latest/deploy/install/install-with-kubectl/) and mark it as the default storage class.
3. Ensure that a load balancer controller is installed. For development and testing purposes, the built-in [ServiceLB](https://docs.k3s.io/networking/networking-services#service-load-balancer) should suffice.

## Preparing a cluster for GPU usage

### Supported GPU hardware

Contrast can only be used with the following Confidential Computing enabled GPUs:

<!-- generated with `nix run .#scripts.get-nvidia-cc-gpus` -->
<!-- vale off -->

- NVIDIA HGX H100 4-GPU 64GB HBM2e (Partner Cooled)
- NVIDIA HGX H100 4-GPU 80GB HBM3 (Partner Cooled)
- NVIDIA HGX H100 4-GPU 94GB HBM2e (Partner Cooled)
- NVIDIA HGX H100 8-GPU 80GB (Air Cooled)
- NVIDIA HGX H100 8-GPU 96GB (Air Cooled)
- NVIDIA HGX H20 141GB HBM3e 8-GPU (Air Cooled)
- NVIDIA HGX H200 8-GPU 141GB (Air Cooled)
- NVIDIA HGX H20A HBM3 96gb 8-GPU (Air Cooled)
- NVIDIA HGX H800 8-GPU 80GB (Air Cooled)
- NVIDIA H100 NVL
- NVIDIA H100 PCIe
- NVIDIA H200 NVL
- NVIDIA H800 PCIe

<!-- vale on -->

:::warning

Currently, only use of `NVIDIA H100 PCIe` is covered by tests. Use of other GPUs isn't guaranteed to work.

:::

To check what GPUs are available on your system, run:

```sh
lspci -nnk | grep '3D controller' -A3
```

```shell-session
41:00.0 3D controller [0302]: NVIDIA Corporation GH100 [H100 PCIe] [10de:2331] (rev a1)
   Subsystem: NVIDIA Corporation GH100 [H100 PCIe] [10de:1626]
   Kernel driver in use: vfio-pci
   Kernel modules: nvidiafb, nouveau
```

Further information is provided in [NVIDIA's Secure AI Compatibility Matrix](https://www.nvidia.com/en-us/data-center/solutions/confidential-computing/secure-ai-compatibility-matrix/).

### Setup

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">

To enable GPU usage on a Contrast cluster, some conditions need to be fulfilled for *each cluster node* that should host GPU workloads:

1. You must activate the IOMMU. You can check by running:

   ```sh
   ls /sys/kernel/iommu_groups
   ```

   If the output contains the group indices (`0`, `1`, ...), the IOMMU is supported on the host.
   Otherwise, add `intel_iommu=on` to the kernel command line.

2. Additionally, the host kernel needs to have the following kernel configuration options enabled:
   - `CONFIG_VFIO`
   - `CONFIG_VFIO_IOMMU_TYPE1`
   - `CONFIG_VFIO_MDEV`
   - `CONFIG_VFIO_MDEV_DEVICE`
   - `CONFIG_VFIO_PCI`

3. A CDI configuration needs to be present on the node. To generate it, you can use the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html).
   Refer to the official instructions on [how to generate a CDI configuration with it](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/cdi-support.html).

If the per-node requirements are fulfilled, deploy the [NVIDIA GPU Operator](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest) to the cluster. It provisions pod-VMs with GPUs via VFIO.

Initially, label all nodes that _should run GPU workloads_:

```sh
kubectl label node <node-name> nvidia.com/gpu.workload.config=vm-passthrough
```

For a GPU-enabled Contrast cluster, you can then deploy the operator with the following commands:

```sh
# Add the NVIDIA Helm repository
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia && helm repo update

# Install the GPU Operator
helm install --wait --generate-name \
   -n gpu-operator --create-namespace \
   nvidia/gpu-operator \
   --version=v25.3.0 \
   --set sandboxWorkloads.enabled=true \
   --set sandboxWorkloads.defaultWorkload='vm-passthrough' \
   --set nfd.nodefeaturerules=true \
   --set vfioManager.enabled=true \
   --set ccManager.enabled=true \
   --set ccManager.defaultMode=on
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

These identifiers are then used to [run GPU workloads on the cluster](../../howto/workload-deployment/GPU-configuration.md).

</TabItem>
<TabItem value="intel" label="Intel TDX">
:::warning
Currently, Contrast only supports GPU workloads on SEV-SNP-based clusters.
:::
</TabItem>
</Tabs>
