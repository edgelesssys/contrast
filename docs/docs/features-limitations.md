# Planned features and limitations

This section lists planned features and current limitations of Contrast.

## Availability

- **Platform support**: At present, Contrast is exclusively available on Azure AKS, supported by the [Confidential Container preview for AKS](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview). Expansion to other cloud platforms is planned, pending the availability of necessary infrastructure enhancements.
- **Bare-metal support**: Support for running [Contrast on bare-metal Kubernetes](getting-started/bare-metal.md) is available for AMD SEV-SNP and Intel TDX.

## Kubernetes features

- **Persistent volumes**: Contrast only supports volumes with [`volumeMode: Block`](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#volume-mode). These block devices are provided by the untrusted environment and should be treated accordingly. We plan to provide transparent encryption on top of block devices in a future release.
- **Port forwarding**: This feature [isn't yet supported by Kata Containers](https://github.com/kata-containers/kata-containers/issues/1693). You can [deploy a port-forwarder](https://docs.edgeless.systems/contrast/deployment#connect-to-the-contrast-coordinator) as a workaround.
- **Resource limits**: Contrast doesn't support setting CPU limits on bare metal. Adding a resource request for CPUs will lead to attestation failures.
- **Image pull secrets**: registry authentication is only supported on AKS, see [Registry Authentication](howto/registry-authentication.md#bare-metal) for details.

## Runtime policies

- **Coverage**: While the enforcement of workload policies generally functions well, [there are scenarios not yet fully covered](https://github.com/microsoft/kata-containers/releases/tag/3.2.0.azl0.genpolicy). It's crucial to review deployments specifically for these edge cases.
- **Order of events**: The current policy evaluation mechanism on API requests isn't stateful, so it can't ensure a prescribed order of events.
- **Absence of events**: Policies can't ensure certain events have happened. A container, such as the [service mesh sidecar](components/service-mesh.md), can be omitted entirely. Environment variables may be missing.
- **Volume integrity checks**: Integrity checks don't cover any volume mounts, such as `ConfigMaps` and `Secrets`.

:::note
The missing guarantee for startup order doesn't affect the security of Contrast's service mesh, see [Service mesh startup enforcement](components/service-mesh.md#service-mesh-startup-enforcement).
:::

## Tooling integration

- **CLI availability**: The CLI tool is currently only available for Linux. This limitation arises because certain upstream dependencies haven't yet been ported to other platforms.

## Automatic recovery and high availability

The Contrast Coordinator is a singleton and can't be scaled to more than one instance.
When this instance's pod is restarted, for example for node maintenance, it needs to be recovered manually.
In a future release, we plan to support distributed Coordinator instances that can recover automatically.

## Overriding Kata configuration

Kata Containers supports [overriding certain configuration values via Kubernetes annotations](https://github.com/kata-containers/kata-containers/blob/b4da4b5e3b9b21048af9333b071235a57a3e9493/docs/how-to/how-to-set-sandbox-config-kata.md).

It needs to be noted that setting these values is unsupported, and doing so may lead to unexpected
behaviour, as Contrast isn't tested against all possible configuration combinations.

## GPU attestation

While Contrast supports integration with confidential computing-enabled GPUs, such as NVIDIA's H100 series, attesting the integrity of the GPU device currently must be handled at the workload layer.
This means the workload needs to verify that the GPU is indeed an NVIDIA H100 running in confidential computing mode.

To simplify this process, the NVIDIA CC-Manager, which is
[deployed alongside the NVIDIA GPU operator](./getting-started/bare-metal.md#preparing-a-cluster-for-gpu-usage), enables the use of confidential computing GPUs (CC GPUs) within the workload. With the CC-Manager in place, the workload is responsible only for attesting the GPU's integrity.

One way to perform this attestation is by using
[nvTrust](https://github.com/NVIDIA/nvtrust), NVIDIA's reference implementation for GPU attestation.
nvTrust provides tools and utilities to perform attestation within the workload.
