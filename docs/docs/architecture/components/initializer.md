# Initializer

Contrast provides an Initializer that handles the remote attestation on the workload side transparently and
fetches the workload certificate.
The Initializer runs as an init container before your workload is started.
It provides the workload container and the [service mesh sidecar](service-mesh.md) with the workload certificates.

The initializer periodically connects to the [`coordinator-ready`](coordinator.md#services) service in its own namespace until it receives a successful response.
This means that workload pods with an initializer will stay in the `Init` phase until a Coordinator manifest is set that allows their specific configuration.

If your workload is configured with [persistent encrypted storage](../../howto/encrypted-storage.md), the initializer will prepare and mount the device and continue running as a sidecar container alongside your application.
