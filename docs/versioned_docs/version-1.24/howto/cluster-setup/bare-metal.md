# Prepare a bare-metal instance

## Prerequisites

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">

- A supported CPU:
  - AMD Epyc 7003 series (Milan)
  - AMD Epyc 9004 series (Genoa)
- A Linux distribution for the Kubernetes nodes:
  - Kernel version must be at least 6.11.
  - The OS must be systemd-based.
  - The Kernel must be configured to just use cgroupsv2.

</TabItem>
<TabItem value="intel" label="Intel TDX">

- A supported CPU:
  - 5th Gen Intel Xeon Scalable Processor
  - Intel Xeon 6 Processors
- Platform must fulfill the [DIMM requirements](https://cc-enabling.trustedservices.intel.com/intel-tdx-enabling-guide/03/hardware_selection/#dimm-ie-main-memory-requirements).
- A Linux distribution for the Kubernetes nodes:
  - Kernel version must be at least 6.16.
  - The OS must be systemd-based.
  - The Kernel must be configured to just use cgroupsv2.

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

Follow Intel's [TDX Enabling Guide](https://cc-enabling.trustedservices.intel.com/intel-tdx-enabling-guide).
When deciding to update the Intel TDX module, be aware that the latest module might be incompatible with your CPU or host firmware.
Make sure to keep a backup of all files you're overwriting in this step until you're sure that the new module works correctly.

</TabItem>
</Tabs>

## Kernel setup

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">
Install Linux kernel 6.11 or greater.
</TabItem>
<TabItem value="intel" label="Intel TDX">
Install Ubuntu 26.04 and leave the kernel at the default version. Other distributions with a 6.16+ kernel might work as well, but we currently only provide support for Ubuntu 26.04.
Add the `nohibernate` and `kvm_intel.tdx=1` kernel command line parameters, for example by updating `GRUB_CMDLINE_LINUX` in `/etc/default/grub`.
</TabItem>
</Tabs>

Containerd uses a significant amount of `inotify` instances, so we recommend to allow at least 8192.
If necessary, the default can be increased by creating a config override file (for example in `/etc/sysctl.d/98-containerd.conf`) with the following content:

```ini
fs.inotify.max_user_instances = 8192
```

Apply this change by running `systemctl restart systemd-sysctl` and verify it using `sysctl fs.inotify.max_user_instances`.

## Kubernetes cluster setup

Contrast can be deployed with different Kubernetes distributions, provided that the following prerequisites are met:

1. The Container Runtime Interface (CRI) implementation must be containerd.
   Since older containerd versions [contain bugs that won't be fixed](../troubleshooting.md#contrast-attempts-to-pull-the-wrong-image-reference), we strongly recommend using v2.0.0 or higher.
2. The node directory `/opt` must be writable and mustn't be mounted with `noexec`.

The default configuration should work for a vanilla containerd installation.
Other Kubernetes variants may need subtle tweaks.
We'll show configurations for k3s and RKE2 below, see [the node installer configuration reference](../../reference/node-installer-configuration.md) for more details.

### K3s

[K3s](https://k3s.io/) is a lightweight Kubernetes distribution that's easy to set up.

1. Follow the [K3s setup instructions](https://docs.k3s.io/) to create a cluster.
   Contrast is currently tested with K3s version `v1.34.1+k3s1`.
2. Install a block storage provider such as [Longhorn](https://longhorn.io/docs/1.9.1/deploy/install/install-with-helm/) and mark it as the default storage class.
3. Ensure that a load balancer controller is installed.
   For development and testing purposes, the built-in [ServiceLB](https://docs.k3s.io/networking/networking-services#service-load-balancer) should suffice.

Then, install the ConfigMap to configure the Contrast node-installer for use with K3s:

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/node-installer-target-config-k3s.yml
```

If you need to pull large images, configure K3s to use a longer `runtime-request-timeout` duration than the [default value of 2 minutes](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) used by the kubelet,
for example by setting

```yaml
kubelet-arg:
  - "runtime-request-timeout=5m"
```

in `/etc/rancher/k3s/config.yaml`.

### RKE2

[RKE2](https://docs.rke2.io/) is Rancher's enterprise-ready next-generation Kubernetes distribution.

1. Follow the [RKE2 setup instructions](https://docs.rke2.io/) to create a cluster.
   Only the Contrast runtime installation is currently tested on RKE2.
2. Install a block storage provider such as [Longhorn](https://longhorn.io/docs/1.9.1/deploy/install/install-with-helm/) and mark it as the default storage class.
3. Ensure that a load balancer controller is installed.
   RKE2 doesn't come with a bundled load balancer controller.

Then, install the ConfigMap to configure the Contrast node-installer for use with RKE2:

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/node-installer-target-config-rke2.yml
```

If you need to pull large images, configure RKE2 to use a longer `runtime-request-timeout` duration than the [default value of 2 minutes](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) used by the kubelet,
for example by setting

```yaml
kubelet-arg:
  - "runtime-request-timeout=5m"
```

in `/etc/rancher/rke2/config.yaml`.

<!-- TODO(burgerdev): add instructions for kubeadm and make the k8s distros tabs instead of sections. -->

## Preparing a cluster for GPU usage

### Supported GPU hardware

Contrast can only be used with the following Confidential Computing enabled GPUs:

<!-- generated with `nix run .#base.scripts.get-nvidia-cc-gpus` -->
<!-- vale off -->

- NVIDIA HGX B200, 8-GPU, SXM6 180GB HBM3e, AC
- NVIDIA HGX B200-850, 8-GPU, SXM6 180GB HBM3e, AC
- NVIDIA HGX H100 4-GPU 64GB HBM2e (Partner Cooled)
- NVIDIA HGX H100 4-GPU 80GB HBM3 (Partner Cooled)
- NVIDIA HGX H100 4-GPU 94GB HBM2e (Partner Cooled)
- NVIDIA HGX H100 8-GPU 80GB (Air Cooled)
- NVIDIA HGX H100 8-GPU 96GB (Air Cooled)
- NVIDIA HGX H20 141GB HBM3e 8-GPU (Air Cooled)
- NVIDIA HGX H200 8-GPU 141GB (Air Cooled)
- NVIDIA HGX H20A HBM3 96GB 8-GPU (Air Cooled)
- NVIDIA HGX H800 8-GPU 80GB (Air Cooled)
- NVIDIA HGX H800 8-GPU 80GB (Partner Cooled)
- NVIDIA H100 NVL
- NVIDIA H100 PCIe
- NVIDIA H200 NVL
- NVIDIA H800 NVL
- NVIDIA H800 PCIe
- NVIDIA RTX PRO 6000 Blackwell Server Edition

<!-- vale on -->

:::warning

Currently, only the `NVIDIA H100 PCIe` and `NVIDIA HGX B200` models are covered by tests. Other GPUs aren't guaranteed to work.

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

[Ubuntu Server 26.04](https://releases.ubuntu.com/resolute/), for example, fulfills these requirements out-of-the-box, and no further changes are necessary here.

If the per-node requirements are fulfilled, deploy the [NVIDIA GPU Operator](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest) to the cluster. It provisions pod-VMs with GPUs via VFIO.

For a GPU-enabled Contrast cluster, you can then deploy the operator with the following commands:

<!-- ! $ echo "GPU_OPERATOR_VERSION=$(jq -r '."gpu-operator".currentValue' ../../../../tools/bm-maintenance/versions.json)" -->

<!-- > sh $
sed -n '/^# Add the NVIDIA Helm repository$/,/^$/p' ../../../../packages/by-name/scripts/upgrade-gpu-operator/upgrade-gpu-operator.sh
sed -n '/^# Install the GPU Operator$/,/^$/p' ../../../../packages/by-name/scripts/upgrade-gpu-operator/upgrade-gpu-operator.sh |
  sed '$d' | sed "s/\"\$GPU_OPERATOR_VERSION\"/$GPU_OPERATOR_VERSION/g"
-->

<!-- BEGIN mdsh -->
```sh
# Add the NVIDIA Helm repository
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia && helm repo update

# Install the GPU Operator
# The Kata sandbox plugin defaults to the pgpu alias when P_GPU_ALIAS is unset.
# An explicitly empty value disables the alias and exposes model-specific resources.
helm install --wait --generate-name \
  -n gpu-operator --create-namespace \
  nvidia/gpu-operator \
  --version=v26.3.3 \
  --set sandboxWorkloads.enabled=true \
  --set sandboxWorkloads.defaultWorkload=vm-passthrough \
  --set sandboxWorkloads.mode=kata \
  --set 'kataSandboxDevicePlugin.env[0].name=P_GPU_ALIAS' \
  --set 'kataSandboxDevicePlugin.env[0].value=' \
  --set nfd.enabled=true \
  --set nfd.nodefeaturerules=true
```
<!-- END mdsh -->

Refer to the [official installation instructions](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/getting-started.html) for details and further options.

Version 26.3 of the GPU operator supports Blackwell GPU detection and confidential-computing configuration without additional node feature or CC manager patches.
For B300 GPUs, however, a small patch is necessary as the sandbox device plugin bundled with this release carries an outdated PCI database lacking the B300 device ID (`10de:3182`).

<details>
<summary>Enabling Support for B300 GPUs</summary>

A complete current `pci.ids` exceeds the ConfigMap size limit, so retain only the NVIDIA vendor records and PCI class definitions before mounting the database into the sandbox device plugin:

```sh
pci_ids_path=/path/to/current/pci.ids
awk '
  $1 == "10de" && $0 ~ /^[0-9a-f][0-9a-f][0-9a-f][0-9a-f]  / { nvidia = 1 }
  nvidia && $0 ~ /^[0-9a-f][0-9a-f][0-9a-f][0-9a-f]  / && $1 != "10de" { nvidia = 0 }
  nvidia { print }
  $1 == "C" { classes = 1 }
  classes { print }
' "$pci_ids_path" |
  kubectl create configmap nvidia-pci-ids \
    -n gpu-operator \
    --from-file=pci.ids=/dev/stdin \
    --dry-run=client \
    -o yaml |
  kubectl apply -f -

kubectl patch daemonset nvidia-kata-sandbox-device-plugin-daemonset \
  -n gpu-operator \
  --type=strategic \
  --patch='spec:
    template:
      spec:
        containers:
          - name: nvidia-kata-sandbox-device-plugin-ctr
            volumeMounts:
              - name: nvidia-pci-ids
                mountPath: /usr/share/misc
                readOnly: true
        volumes:
          - name: nvidia-pci-ids
            emptyDir: null
            configMap:
              name: nvidia-pci-ids
              items:
                - key: pci.ids
                  path: pci.ids'
kubectl rollout status daemonset/nvidia-kata-sandbox-device-plugin-daemonset \
  -n gpu-operator \
  --timeout=5m
```

</details>

Once the operator is fully deployed, which can take a few minutes, check the available GPUs in the cluster:

```sh
kubectl get nodes -l nvidia.com/gpu.present -o json | \
  jq '.items[] | {name: .metadata.name, gpus: (.status.allocatable |
    to_entries | map(select(.key | startswith("nvidia.com/")) |
    select(.value != "0")) | from_entries)}'
```

The above command should yield an output similar to the following, depending on what GPUs are available:

```json
{
  "name": "node-name",
  "gpus": {
    "nvidia.com/GB100_B200": "8"
  }
}
```

These identifiers are then used to [run GPU workloads on the cluster](../../howto/workload-deployment/GPU-configuration.md).
