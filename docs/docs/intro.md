---
slug: /
id: intro
---

# Introduction

Welcome to the Contrast documentation! Contrast makes your Kubernetes deployments confidential by running workloads securely within confidential computing environments.

![Contrast concept](/img/concept.svg)

Contrast is built upon the open-source [Kata Containers](https://github.com/kata-containers/kata-containers) project.

Contrast provides confidential containers: Kubernetes pods that are executed within confidential micro-VMs, providing strong, hardware-based isolation from the surrounding environment.
You can use your existing containers without modification—enabling easy adoption through a lift-and-shift approach.

:::tip
Contrast leverages a technology called confidential computing. If you're new to confidential computing, check out our 📄[whitepaper](https://content.edgeless.systems/hubfs/Confidential%20Computing%20Whitepaper.pdf) for an overview.
:::

## Why use Contrast?

Contrast keeps your data encrypted at all times, ensuring it remains inaccessible from the underlying infrastructure.
It effectively removes the infrastructure provider—including datacenter employees, privileged cloud administrators, cluster operators, and potential attackers—from your trusted computing base (TCB).
This protects your workloads even from sophisticated threats like malicious co-tenants attempting privilege escalation.

Contrast integrates seamlessly into your existing Kubernetes workflows. It can be deployed into your existing Kubernetes cluster, and requires minimal adjustments to your existing processes.

## Key use cases

Contrast provides powerful [security features and benefits](./security.md). Common scenarios include:

- Strengthening container security with hardware-backed isolation
- Securely migrating sensitive workloads from on-premises to cloud environments
- Protecting workloads and data from internal threats, including cluster administrators
- Enhancing trust and security for SaaS offerings
- Streamlining regulatory compliance efforts
- Facilitating secure multi-party data collaboration

## Supported Kubernetes environments

Contrast supports bare-metal setups based on AMD SEV-SNP and Intel TDX hardware. It also supports managed Kubernetes with hybrid bare-metal nodes.

## Getting started

Use these entry points to quickly explore Contrast:

- **Hands-on example**: The [Getting Started](./getting-started/overview.md) section walks you step-by-step through securing a deployment using Contrast—a practical and beginner-friendly way to get started.

- **Guides**: The [How-to](./howto/cluster-setup/bare-metal.md) section provides concise instructions for common workflows. After grasping the basics, these guides will help you accomplish specific tasks quickly.

- **Troubleshooting**: Facing issues? Check our [Troubleshooting](./howto/troubleshooting.md) section for solutions to common problems and pitfalls.

- **Security**: Explore the [Security](./security.md) section to understand Contrast’s security properties and threat model.

- **Architecture**: For a deeper technical dive, the [Architecture](./architecture/overview.md) section explains how Contrast achieves its strong security features.
