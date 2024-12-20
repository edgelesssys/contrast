# Confidential Containers

Contrast uses some building blocks from [Confidential Containers](https://confidentialcontainers.org) (CoCo), a [CNCF Sandbox project](https://www.cncf.io/projects/confidential-containers/) that aims to standardize confidential computing at the pod level.
The project is under active development and many of the high-level features are still in flux.
Contrast uses the more stable core primitive provided by CoCo: its Kubernetes runtime.

## Kubernetes RuntimeClass

Kubernetes can be extended to use more than one container runtime with [`RuntimeClass`](https://kubernetes.io/docs/concepts/containers/runtime-class/) objects.
The [Container Runtime Interface](https://kubernetes.io/docs/concepts/architecture/cri/) (CRI) implementation, for example containerd, dispatches pod management API calls to the appropriate `RuntimeClass`.
`RuntimeClass` implementations are usually based on an [OCI runtime](https://github.com/opencontainers/runtime-spec), such as `runc`, `runsc` or `crun`.
In CoCo's case, the runtime is Kata Containers with added confidential computing capabilities.

## Kata Containers

[Kata Containers](https://katacontainers.io/) is an OCI runtime that runs pods in VMs.
The pod VM spawns an agent process that accepts management commands from the Kata runtime running on the host.
There are two options for creating pod VMs: local to the Kubernetes node, or remote VMs created with cloud provider APIs.
Using local VMs requires either bare-metal servers or VMs with support for nested virtualization.
Local VMs communicate with the host over a virtual socket.
For remote VMs, host-to-agent communication is tunnelled through the cloud provider's network.

Kata Containers was originally designed to isolate the guest from the host, but it can also run pods in confidential VMs (CVMs) to shield pods from their underlying infrastructure.
In confidential mode, the guest agent is configured with an [Open Policy Agent](https://www.openpolicyagent.org/) (OPA) policy to authorize API calls from the host.
This policy also contains checksums for the expected container images.
It's derived from Kubernetes resource definitions and its checksum is included in the attestation report.

## AKS CoCo preview

[Azure Kubernetes Service](https://learn.microsoft.com/en-us/azure/aks/) (AKS) provides CoCo-enabled node pools as a [preview offering](https://learn.microsoft.com/en-us/azure/aks/confidential-containers-overview).
These node pools leverage Azure VM types capable of nested virtualization (CVM-in-VM) and the CoCo stack is pre-installed.
Contrast can be deployed directly into a CoCo-enabled AKS cluster.
