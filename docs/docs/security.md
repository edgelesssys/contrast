# Security overview

Contrast is designed to thoroughly protect your deployment and data from the
underlying infrastructure. This section outlines the security goals and the
associated threat model.

## Security goals

- **Confidentiality**: All data processed remains encrypted at all times: During
  transit, at rest and even while processing through runtime encryption.

- **Isolation**: By design, Contrast strictly isolates workloads from the
  underlying infrastructure. It prevents access by infrastructure providers,
  data center personnel, privileged cloud administrators, and external malicious
  actors.

- **Integrity and authenticity**: Contrast ensures that all workloads are
  running in a trusted and intended state. The integrity and authenticity of
  workloads is ensured through remote attestation.

## Threat model and mitigations

This section outlines the types of threats Contrast is designed to mitigate.

> **Out of scope:**
>
> - Vulnerabilities in application logic (for example broken access controls)
> - Hardware-level attacks on Confidential Computing (for example side-channel
>   exploits)
> - Denial-of-service (DoS) and other availability-focused attacks

### Threat actors

Contrast protects against five main types of attackers:

- **Malicious cloud insider:** Cloud provider employees or contractors with
  privileged access across physical infrastructure, hypervisors, or Kubernetes
  control planes. They may tamper with VM resources, intercept data, or restrict
  runtime behavior (for example limit memory, alter disk volumes, or change
  firewall rules).

- **Malicious cloud co-tenant:** A cloud user who breaks out of isolation to
  target neighboring tenants. Though lacking physical access, they may achieve
  similar effects to insiders through persistent exploitation.

- **Malicious workload operator:** Kubernetes administrators or DevOps engineers
  with access to workload deployment and orchestration tools. Their influence
  spans everything above the hypervisor.

- **Malicious attestation client:** Attempts to disrupt or bypass the
  attestation service by sending malformed or intentionally deceptive requests.

- **Malicious container image provider:** Publishes container images that
  include malicious functionality (for example backdoors or unauthorized data
  access logic).

### Attack surfaces

| Attacker                           | Target                           | Surface                  | Risk                                                                           |
| ---------------------------------- | -------------------------------- | ------------------------ | ------------------------------------------------------------------------------ |
| Cloud insider                      | Confidential Container, Workload | Physical memory          | May extract secrets by dumping VM memory                                       |
| Cloud insider, co-tenant, operator | Confidential Container, Workload | Disk (read/write)        | May inspect or modify data stored on disk                                      |
| Cloud insider, co-tenant, operator | Confidential Container, Workload | Kubernetes control plane | Can alter environment variables, mounts, and workload metadata                 |
| Cloud insider, co-tenant, operator | Confidential Container, Workload | Container runtime        | May use APIs (for example `kubectl exec`) to access workloads                  |
| Cloud insider, co-tenant, operator | Confidential Container, Workload | Network                  | Can intercept traffic to registries, attestation endpoints, or other workloads |
| Malicious attestation client       | Attestation service              | Attestation interface    | May disrupt the attestation flow with invalid or malformed input               |
| Malicious image provider           | Workload                         | Container image          | May introduce compromised logic or hidden behavior into the workload           |

#### Mitigations

Contrast mitigates these threats using three core components:

1. [**Runtime environment**](./architecture/components/runtime.md): protects
   memory, disk, and VM integrity
2. [**Runtime policies**](./architecture/components/policies.md): define and
   enforce workload integrity and configuration
3. [**Service mesh**](./architecture/components/service-mesh.md): secures all
   internal and external communication

### Attacks on the confidential container environment

| Threat                                                            | Mitigation                                                                                 | Where it's enforced                                                                                      |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| Intercepting network traffic during workload launch or image pull | Reflected in attestation report; assumes images are public and contain no embedded secrets | [Policies](./architecture/components/policies.md), [Attestation](./architecture/attestation/overview.md) |
| Modifying the workload image post-download                        | Prevented by dm-verity-protected read-only partitions                                      | [Runtime environment](./architecture/components/runtime.md)                                              |
| Changing runtime settings via the Kubernetes control plane        | Detected by runtime policies and validated through attestation                             | [Policies](./architecture/components/policies.md), [Attestation](./architecture/attestation/overview.md) |

### Attacks on the attestation service

| Threat                                                           | Mitigation                                                                                            | Where it's enforced                                     |
| ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| Modifying or hijacking the Coordinator deployment                | Coordinator is itself attested; images are reproducible and protected via a secured supply chain      | [Attestation](./architecture/attestation/overview.md)   |
| Intercepting secrets in transit between workload and Coordinator | TLS with attested identities ensures encryption and prevents impersonation                            | Service mesh and attestation protocol                   |
| Exploiting attestation parsing edge cases                        | Handled by a memory-safe Go parser tested against vendor specifications; policies are fully auditable | [Coordinator](./architecture/components/coordinator.md) |
| Overloading the attestation service (DoS)                        | Will be mitigated by making the Coordinator scalable and fault-tolerant                               | [Coordinator](./architecture/components/coordinator.md) |

### Attacks on workloads

| Threat                                         | Mitigation                                                                                                | Where it's enforced                                         |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| Eavesdropping on inter-container communication | Prevented by automatic TLS encryption across all intra-cluster communication                              | [Service mesh](./architecture/components/service-mesh.md)   |
| Reading or altering data on persistent volumes | Persistent volumes aren't yet supported; future support will include encryption and integrity protections | [Runtime environment](./architecture/components/runtime.md) |
| Publishing compromised workload images         | Updates require explicit policy approval and must match attested, verified workload configurations        | [Attestation](./architecture/attestation/overview.md)       |

## Real-world scenarios

| Use Case                   | Example                                                                                                                                                                                                                  |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Secure cloud migration** | _TechSolve Inc._ moves sensitive workloads to the cloud. As both image provider and data owner, they're exposed to threats from insiders and co-tenants. Contrast ensures data isolation and workload integrity.         |
| **Trusted SaaS delivery**  | _SaaSProviderX_ wants to prove to customers that even internal admins can't access their data. With Contrast, customers retain control, while the SaaS provider is excluded from the trusted base.                       |
| **Regulatory compliance**  | _HealthSecure Inc._ migrates analytics to the cloud while handling patient data. Regulators require verifiable isolation. Contrast provides attestable guarantees that only authorized workloads process sensitive data. |

In all scenarios, Contrast ensures that only authorized workloads can access
sensitive data. It offers verifiable isolation from infrastructure and
control-plane actors, while giving data owners full visibility and control over
the runtime environment.
