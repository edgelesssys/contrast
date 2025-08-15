# Planned features and limitations

This section lists planned features and current limitations of Contrast.

## Availability

- **Bare-metal support**: Support for running [Contrast on bare-metal Kubernetes](../howto/cluster-setup/bare-metal.md) is available for AMD SEV-SNP and Intel TDX.
- **Public cloud** is limited to cloud vendors providing the required hardware as bare-metal instances, ideally as part of their managed Kubernetes offerings.

## Kubernetes features

- **Persistent volumes**: Contrast only supports volumes with [`volumeMode: Block`](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#volume-mode). These block devices are provided by the untrusted environment and should be treated accordingly. The [transparent encryption feature](secrets.md#secure-persistence) is recommended for secure persistence.
- **Volume sub-paths**: Mounting only a `subPath` of a volume [isn't yet supported by Kata Containers](https://github.com/kata-containers/kata-containers/issues/10487).
- **Port forwarding**: This feature [isn't yet supported by Kata Containers](https://github.com/kata-containers/kata-containers/issues/1693). You can [deploy a port-forwarder](../howto/workload-deployment/set-manifest.md#connect-to-the-contrast-coordinator) as a workaround.
- **Resource limits**: Contrast doesn't support setting CPU limits on bare metal. Adding a resource request for CPUs will lead to attestation failures.
- **Image pull secrets**: registry authentication isn't yet supported.

## Runtime policies

- **Order of events**: The current policy evaluation mechanism on API requests isn't stateful, so it can't ensure a prescribed order of events.
- **Absence of events**: Policies can't ensure certain events have happened. A container, such as the [service mesh sidecar](components/service-mesh.md), can be omitted entirely. Environment variables may be missing.
- **Volume integrity checks**: Integrity checks don't cover any volume mounts, such as `ConfigMaps` and `Secrets`.
- **Supported resource kinds**: There are some resources not yet covered.
  It's crucial to ensure that only [supported resource kinds](components/policies.md#supported-resource-kinds) are passed to `contrast generate`.

:::note
The missing guarantee for startup order doesn't affect the security of Contrast's service mesh, see [Service mesh startup enforcement](components/service-mesh.md#service-mesh-startup-enforcement).
:::

## Tooling integration

- **CLI availability**: The CLI tool is currently only available for Linux. This limitation arises because certain upstream dependencies haven't yet been ported to other platforms.

## GPU attestation

While Contrast supports integration with confidential computing-enabled GPUs, such as NVIDIA's H100 series, attesting the integrity of the GPU device currently must be handled at the workload layer.
This means the workload needs to verify that the GPU is indeed an NVIDIA H100 running in confidential computing mode.

To simplify this process, the NVIDIA CC-Manager, which is
[deployed alongside the NVIDIA GPU operator](../howto/cluster-setup/bare-metal.md#preparing-a-cluster-for-gpu-usage), enables the use of confidential computing GPUs (CC GPUs) within the workload. With the CC-Manager in place, the workload is responsible only for attesting the GPU's integrity.

One way to perform this attestation is by using
[nvTrust](https://github.com/NVIDIA/nvtrust), NVIDIA's reference implementation for GPU attestation.
nvTrust provides tools and utilities to perform attestation within the workload.
