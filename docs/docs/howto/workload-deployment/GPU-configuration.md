# Configure GPU support

Contrast supports running GPU workloads inside the confidential computing environment.

## Applicability

This step is optional and only necessary if your application includes a GPU workload (for example an AI model) that should run confidentially.

Currently, confidential GPU workloads are supported **only** on bare-metal systems with AMD SEV-SNP.
They're **not** supported on AKS or on bare-metal systems using Intel TDX.

## Prerequisites

1. [Set up cluster](../cluster-setup/aks.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Prepare deployment files](./deployment-file-preparation.md)
5. [Configure TLS (optional)](./TLS-configuration.md)

## How-to

If the cluster is [configured for GPU usage](../../howto/cluster-setup/bare-metal.md#preparing-a-cluster-for-gpu-usage), Pods can use GPU devices if needed.

To do so, a CDI annotation needs to be added, specifying to use the `pgpu` (passthrough GPU) mode. The `0` corresponds to the PCI device index.

- For nodes with a single GPU, this value is always `0`.
- For nodes with multiple GPUs, the value needs to correspond to the device's order as enumerated on the PCI bus. You can identify this order by inspecting the `/var/run/cdi/nvidia.com-pgpu.yaml` file on the specific node.

This process ensures the correct GPU is allocated to the workload.

As the footprint of a GPU-enabled pod-VM is larger than one of a non-GPU one, the memory of the pod-VM can be adjusted by using the `io.katacontainers.config.hypervisor.default_memory` annotation, which receives the memory the
VM should receive in MiB. The example below sets it to 16 GB. A reasonable minimum for a GPU pod with a light workload is 8 GB.

```yaml
metadata:
  # ...
  annotations:
    # ...
    cdi.k8s.io/gpu: "nvidia.com/pgpu=0"
    io.katacontainers.config.hypervisor.default_memory: "16384"
```

In addition, the container within the pod that requires GPU access must include a device request.
This request specifies the number of GPUs the container should use.
The identifiers for the GPUs, obtained during the [deployment of the NVIDIA GPU Operator](../cluster-setup/bare-metal.md#preparing-a-cluster-for-gpu-usage), must be included in the request.
In the provided example, the container is allocated a single NVIDIA H100 GPU.

Finally, the environment variable `NVIDIA_VISIBLE_DEVICES` must be set to `all` to grant the container access to GPU utilities provided by the pod-VM. This includes essential tools like CUDA libraries, which are required for running GPU workloads.

```yaml
spec:
  # ...
  containers:
    - # ...
      resources:
        limits:
          "nvidia.com/GH100_H100_PCIE": 1
      env:
        # ...
        - name: NVIDIA_VISIBLE_DEVICES
          value: all
```

:::note
A pod configured to use GPU support may take a few minutes to come up, as the VM creation and boot procedure needs to do more work compared to a non-GPU pod.
:::
