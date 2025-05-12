# Initializer

<!-- TODO: Maybe not enough content for seperate page -->

Contrast provides an Initializer that handles the remote attestation on the workload side transparently and
fetches the workload certificate.
The Initializer runs as an init container before your workload is started.
It provides the workload container and the [service mesh sidecar](service-mesh.md) with the workload certificates.
