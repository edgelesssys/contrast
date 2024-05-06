# Known Limitations

As Contrast is currently in an early development stage, it's built on several projects that are also under active development.
This section outlines the most significant known limitations, providing stakeholders with clear expectations and understanding of the current state.

## Availability

- **Platform Support**: At present, Contrast is exclusively available on Azure AKS, supported by the [Confidential Container preview for AKS](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview). Expansion to other cloud platforms is planned, pending the availability of necessary infrastructure enhancements.

## Kubernetes Features

- **Persistent Volumes**: Not currently supported within Confidential Containers.
- **Port-Forwarding**: This feature [isn't yet supported by Kata Containers](https://github.com/kata-containers/kata-containers/issues/1693). You can [deploy a port-forwarder](https://docs.edgeless.systems/contrast/deployment#connect-to-the-contrast-coordinator) as a workaround.
- **Resource Limits**: There is an existing bug on AKS where container memory limits are incorrectly applied. The current workaround involves using only memory requests instead of limits.

## Runtime Policies

- **Coverage**: While the enforcement of workload policies generally functions well, [there are scenarios not yet fully covered](https://github.com/microsoft/kata-containers/releases/tag/3.2.0.azl0.genpolicy). It's crucial to review deployments specifically for these edge cases.
- **Order of events**: The current policy evaluation mechanism on API requests isn't stateful, so it can't ensure a prescribed order of events. Consequently, there's no guaranteed enforcement that the [service mesh sidecar](components/service-mesh.md) container runs *before* the workload container. This order ensures that all traffic between pods is securely encapsulated within TLS connections.
- **Absence of events**: Policies can't ensure certain events have happened. A container, such as the [service mesh sidecar](components/service-mesh.md), can be omitted entirely. Environment variables may be missing.
- **Volume integrity checks**: While persistent volumes aren't supported yet, integrity checks don't currently cover other objects such as `ConfigMaps` and `Secrets`.

:::warning
The policy limitations, in particular the missing guarantee that our service mesh sidecar has been started before the workload container affects the service mesh implementation of Contrast. Currently, this requires inspecting the iptables rules on startup or terminating TLS connections in the workload directly.
:::

## Tooling Integration

- **CLI Availability**: The CLI tool is currently only available for Linux. This limitation arises because certain upstream dependencies haven't yet been ported to other platforms.
