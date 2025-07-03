---
slug: /
id: intro
---

# Contrast

Welcome to the documentation of Contrast! Contrast runs confidential container deployments on Kubernetes at scale.

![Contrast concept](/img/concept.svg)

Contrast is based on the [Kata Containers](https://github.com/kata-containers/kata-containers) and
[Confidential Containers](https://github.com/confidential-containers) projects.
Confidential Containers are Kubernetes pods that are executed inside a confidential micro-VM and provide strong hardware-based isolation from the surrounding environment.
This works with unmodified containers in a lift-and-shift approach.
Contrast currently targets the [CoCo preview on AKS](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview).

:::tip
See the 📄[whitepaper](https://content.edgeless.systems/hubfs/Confidential%20Computing%20Whitepaper.pdf) for more information on confidential computing.
:::

## Goal

Contrast is designed to keep all data always encrypted and to prevent access from the infrastructure layer. It removes the infrastructure provider from the trusted computing base (TCB). This includes access from datacenter employees, privileged cloud admins, own cluster administrators, and attackers coming through the infrastructure, for example, malicious co-tenants escalating their privileges.

Contrast integrates fluently with the existing Kubernetes workflows. It's compatible with managed Kubernetes, can be installed as a day-2 operation and imposes only minimal changes to your deployment flow.

## Use Cases

Contrast provides unique security [features](basics/features.md) and [benefits](basics/security-benefits.md). The core use cases are:

* Increasing the security of your containers
* Moving sensitive workloads from on-prem to the cloud with Confidential Computing
* Shielding the code and data even from the own cluster administrators
* Increasing the trustworthiness of your SaaS offerings
* Simplifying regulatory compliance
* Multi-party computation for data collaboration

## Next steps

You can learn more about the concept of [Confidential Containers](basics/confidential-containers.md), [features](basics/features.md), and [security benefits](basics/security-benefits.md) of Contrast in this section. To jump right into the action head to [*Getting started*](getting-started/install.md).
