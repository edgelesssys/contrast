# Configure GPU support

Contrast supports running GPU workloads inside the confidential computing environment.

## Applicability

This step is optional and only necessary if your application includes a GPU workload (for example an AI model) that should run confidentially.

:::warning

Currently, confidential GPU workloads are supported **only** on bare-metal systems with AMD SEV-SNP.
They're **not** supported on bare-metal systems using Intel TDX.

GPUs without Confidential Computing support can't be used with Contrast.

See the section on [Supported GPU hardware](../cluster-setup/bare-metal.md#supported-gpu-hardware) for more information.

:::

## Prerequisites

1. [Set up cluster](../cluster-setup/bare-metal.md)
2. [Configure for GPU usage](../../howto/cluster-setup/bare-metal.md#preparing-a-cluster-for-gpu-usage)
3. [Install CLI](../install-cli.md)
4. [Deploy the Contrast runtime](./runtime-deployment.md)
5. [Add Coordinator to resources](add-coordinator.md)
6. [Prepare deployment files](./deployment-file-preparation.md)

## How-to

If the cluster is [configured for GPU usage](../../howto/cluster-setup/bare-metal.md#preparing-a-cluster-for-gpu-usage), pods can use GPUs by adding an extended resource limit to the pod definition.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-demo
spec:
  runtimeClassName: contrast-cc
  containers:
  - name: gpu-demo
    image: ghcr.io/edgelesssys/tensorflow@sha256:73fe35b67dad5fa5ab0824ed7efeb586820317566a705dff76142f8949ffcaff
    resources:
      limits:
        nvidia.com/pgpu: "1"
```

If you deployed the NVIDIA GPU operator as described above, the resource name is `nvidia.com/pgpu`.
Otherwise, you need to consult your CDI definitions (in `/var/run/cdi` or `/etc/cdi`) to find the appropriate device class.

Finally, the environment variable `NVIDIA_VISIBLE_DEVICES` can be set to `all` to grant other containers access to the GPUs directly assigned to other containers in the pod.
This can be useful for configuring a GPU in an init container, for example.

:::note
A pod configured to use GPU support may take a few minutes to come up, as the VM creation and boot procedure needs to do more work compared to a non-GPU pod.
:::
