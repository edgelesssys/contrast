# Overview

This page introduces the core components of Contrast and explains its architectural principles.

## Components

Contrast consists of the following main components:

- **Contrast Kubernetes runtime**:
  Contrast provides a custom Kubernetes `RuntimeClass` that defines a runtime handler for `containerd`.
  This handler runs containers inside Confidential Virtual Machines (CVMs).
  The runtime is based on Kata Containers and the Confidential Containers (CoCo) project.

- **Runtime policies**:
  strictly control Host-to-CVM communication on worker nodes and ensure that only approved workloads are allowed to start inside CVMs.

- **Contrast Coordinator**:
  The Contrast Coordinator is an additional service deployed to the cluster.
  Like other components, it runs inside a CVM using the Contrast runtime.
  It serves as the central attestation authority, ensuring that only verified workloads can join the trusted service mesh.
  The Coordinator uses a _manifest_, a `.json` file that defines the trusted state of your cluster by listing cryptographic hashes for all approved workloads.
  Each CVM’s attestation is verified against this manifest before the workload is trusted.

- **Initializer**:
  The initializer runs as an init container within confidential pods.
  It implements the attestation logic and constitutes the attestation endpoint on the workload side.

- **Service mesh**:
  The Contrast Coordinator also acts as a Certificate Authority (CA), issuing certificates only to workloads that successfully pass attestation.
  These certificates can be used to establish a trusted service mesh for secure pod-to-pod communication.
  It can also be presented to external clients, allowing them to verify the service’s identity.

- **Contrast CLI**:
  This command-line tool verifies the integrity and authenticity of both the Coordinator and the full deployment using remote attestation.
  Data owners can use it to verify that a deployment is trustworthy.
  The CLI also pre-processes deployment files, adjusting them automatically for a secure Contrast integration.

## Architectural goals

Contrast is designed to achieve the following architectural goals:

- **Isolation**
  All workloads run in CVMs, isolating them from the underlying infrastructure and cloud provider.

- **End-to-end encryption**
  Memory is encrypted at runtime, and all cluster communication is confidential and authenticated from end to end.

- **Integrity & authenticity**
  Workload integrity and the security of the environment are verified using hardware-based attestation.
  Only trusted workloads are permitted to run.

- **Seamless integration**
  Contrast integrates with existing Kubernetes clusters via a custom runtime and attestation components, requiring minimal changes to existing workflows.

- **Transparency**
  All reference values for the trusted state are fully open and auditable.
  Contrast relies on open-source code and reproducible builds to ensure full transparency.
