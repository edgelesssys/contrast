# Overview

This page introduces the core components of Contrast and explains its architectural principles.

## Components

Contrast consists of the following main components:

- **Contrast Kubernetes runtime**:
  Contrast provides a custom Kubernetes `RuntimeClass` that defines a runtime handler for `containerd`.
  This handler runs containers inside Confidential Virtual Machines (CVMs).
  The runtime is based on Kata Containers.

- **Runtime policies**:
  Strictly control host-to-CVM communication on worker nodes and ensure that only approved workloads are allowed to start inside CVMs.

- **Contrast Coordinator**:
  The Contrast Coordinator is an additional service deployed to the cluster.
  Like other components, it runs inside a CVM using the Contrast runtime.
  It serves as the central attestation authority, ensuring that only verified workloads can join the trusted service mesh.
  The Coordinator uses a _manifest_, a `.json` file that defines the trusted state of your cluster by listing cryptographic hashes for all approved workloads.
  Each CVM's attestation is verified against this manifest before the workload is trusted.

- **Initializer**:
  The initializer runs as an init container within confidential pods.
  It implements the attestation logic and constitutes the attestation endpoint on the workload side.

- **Service mesh**:
  The Contrast Coordinator also acts as a Certificate Authority (CA), issuing certificates only to workloads that successfully pass attestation.
  These certificates can be used to establish a trusted service mesh for secure pod-to-pod communication.
  It can also be presented to external clients, allowing them to verify the service's identity.

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

## Why choose Contrast?

Contrast isn't the only software that runs containers in confidential VMs.
However, we believe that Contrast has some unique advantages that you should know about, which are listed in the following sections.

### The team

Contrast was created by Edgeless Systems, the confidential computing pioneers from Germany.
We've been scaling confidential computing since 2020, with a team of cybersecurity and system engineering experts.
Contrast is our third product at the intersection of confidential computing and Kubernetes, and our designs draw from this vast experience.

### Ready for production

Contrast is stable and secure software you can rely on in production.
The threat model is documented and we ensure the confidentiality and integrity of your containers.
Workloads automatically receive a restrictive policy, without leaving anything for you to figure out.
Contrast doesn't have a single-point-of-failure in the verification path and scales with your cluster.

### Lift and shift

Contrast is designed to support existing Kubernetes workloads without modifications.
You don't need to interact with remote attestation unless you want to, and you can even leave storage and network encryption to Contrast components.

### Reproducible from available source

Our software is developed in the open, without hidden components or gated enterprise features.
With access to the source code you can convince yourself of its security, and security researchers are invited to do so, too.
Our build system ensures outputs are reproducible, so you can verify binary artifacts yourself and don't need to trust our processes.

### Measurements included

Managing reference values for confidential computing can be quite cumbersome, in particular with diverse workloads.
Contrast manages all that complexity for you.
You don't need to use any third party tools or empirically derive appraisal policies.

### SaaS-ready

A Contrast deployment can be verified end-to-end, without requiring administrative access.
This allows offering software-as-a-service to users and excluding yourself from the trusted computing base.
Changes in software or configuration are measured and observable by your users.

### Plugs into standards

Contrast aims to comply with standards and established practice, in order to ease lifting a service into a confidential environment.
Contrast verification is X.509 based, making it compatible with most software stacks in existence.
Standard OCI images and registry authentication are supported out-of-the-box.
Adding a Vault to manage your secrets is straightforward and fully integrated into attestation.
