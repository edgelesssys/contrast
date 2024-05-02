# Known Limitations

As Contrast is currently in an early development stage, it's built on several projects that are also under active development.
This section outlines the most significant known limitations, providing stakeholders with clear expectations and understanding of the current state.

## Availability

- **Platform Support**: At present, Contrast is exclusively available on Azure AKS, supported by the [Confidential Container preview for AKS](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview). Expansion to other cloud platforms is planned, pending the availability of necessary infrastructure enhancements.

## Kubernetes Features

- **Persistent Volumes**: Not currently supported within Confidential Containers.
- **Port-Forwarding**: This feature isn't yet supported by Kata Containers.
- **Resource Limits**: There is an existing bug on AKS where container memory limits are incorrectly applied. The current workaround involves using only memory requests instead of limits.

## Runtime Policies

- **Coverage**: While the enforcement of workload policies generally functions well, [there are scenarios not yet fully covered](https://github.com/microsoft/kata-containers/releases/tag/genpolicy-0.6.2-5). It's crucial to review deployments specifically for these edge cases.
- **Policy Evaluation**: The current policy evaluation mechanism on API requests isn't stateful, which means it can't ensure a prescribed order of events. Consequently, there's no guaranteed enforcement that the [initializer](components/index.md#the-initializer) container runs *before* the workload container. This order is vital for ensuring that all traffic between pods is securely encapsulated within TLS connections. TODO: Consequences

## Tooling Integration

- **CLI Availability**: The CLI tool is currently only available for Linux. This limitation arises because certain upstream dependencies haven't yet been ported to other platforms.
