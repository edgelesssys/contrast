# Overview

This page introduces the core components of Contrast and explains its architectural principles.

## Components

Contrast consists of four main components:

- **Contrast Kubernetes runtime**:
  Contrast provides a custom Kubernetes `RuntimeClass` that defines a runtime handler for `containerd`.
  This handler runs containers inside Confidential Virtual Machines (CVMs).
  The runtime is based on Kata Containers and the Confidential Containers (CoCo) project.

- **Contrast Coordinator**:
  This is an additional service deployed to the cluster.
  It also runs inside a CVM using the Contrast runtime.
  It acts as the central attestation service, making sure that only verified workloads can join the trusted service mesh.
  The Coordinator uses a _manifest_—a `.json` file that defines the trusted state of your cluster using cryptographic hashes for all workloads.
  The Coordinator verifies each CVM's attestation against this manifest before trusting the workload.

- **Service mesh**:
  The Contrast Coordinator also acts as a Certificate Authority (CA), issuing certificates only to workloads that successfully pass attestation.
  These certificates can be used to establish a trusted service mesh for secure pod-to-pod communication.
  It can also be presented to external clients, allowing them to verify the service’s identity.

- **Contrast CLI**:
  This command-line tool verifies the integrity and authenticity of both the Coordinator and the full deployment using remote attestation.
  Data owners can use it to verify that a deployment is trustworthy.

  The CLI also pre-processes deployment files, adjusting them automatically for a secure Contrast integration.

## Architectural goals

Contrast is designed to meet four key goals:

- **Isolation**:
  All workloads run in CVMs, isolating them from the infrastructure provider.
  Memory is encrypted at runtime, and all cluster communication is confidential and authenticated end-to-end.

- **Attestation**:
  Workload integrity is verified using hardware-based, cloud-agnostic attestation.
  Only trusted workloads are allowed to run.

- **Integration**:
  Contrast adds a custom runtime and attestation components to Kubernetes.
  These can be added to existing clusters with minimal changes to your workflow.

- **Transparency**:
  All reference values for the trusted state are fully open and auditable.
  Contrast uses open-source code and reproducible builds.
