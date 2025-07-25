# Attestation in Contrast

This document describes the attestation architecture of Contrast, adhering to
the definitions of Remote ATtestation procedureS (RATS) in
[RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html). The following gives a
detailed description of Contrast's attestation architecture. At the end of this
document, we included an
[FAQ](#frequently-asked-questions-about-attestation-in-contrast) that answers
the most common questions regarding attestation in hindsight of the
[security benefits](../basics/security-benefits.md).

## Attestation architecture

Contrast integrates with the RATS architecture, leveraging their definition of
roles and processes including _Attesters_, _Verifiers_, and _Relying Parties_.

![Conceptual attestation architecture](../_media/attestation-rats-architecture.svg)

Figure 1: Conceptual attestation architecture. Taken from
[RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html#figure-1).

- **Attester**: Assigned to entities that are responsible for creating
  _Evidence_ which is then sent to a _Verifier_.
- **Verifier**: These entities utilize the _Evidence_, _Reference Values_, and
  _Endorsements_. They assess the trustworthiness of the _Attester_ by applying
  an _Appraisal Policy_ for _Evidence_. Following this assessment, _Verifiers_
  generate _Attestation Results_ for use by _Relying Parties_. The _Appraisal
  Policy_ for _Evidence_ may be provided by the _Verifier Owner_, programmed
  into the _Verifier_, or acquired through other means.
- **Relying Party**: Assigned to entities that utilize _Attestation Results_,
  applying their own appraisal policies to make specific decisions, such as
  authorization decisions. This process is referred to as the "appraisal of
  Attestation Results." The _Appraisal Policy_ for _Attestation Results_ might
  be sourced from the _Relying Party Owner_, configured by the owner, embedded
  in the _Relying Party_, or obtained through other protocols or mechanisms.

## Components of Contrast's attestation

The key components involved in the attestation process of Contrast are detailed
below:

### Attester: Application Pods

This includes all Pods of the Contrast deployment that run inside Confidential
Containers and generate cryptographic evidence reflecting their current
configuration and state. Their evidence is rooted in the
[hardware measurements](../basics/confidential-containers.md) from the CPU and
their [confidential VM environment](../components/runtime.md). The details of
this evidence are given below in the section on
[evidence generation and appraisal](#evidence-generation-and-appraisal).

![Attestation flow of a confidential pod](../_media/attestation-pod.svg)

Figure 2: Attestation flow of a confidential pod. Based on the layered attester
graphic in [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html#figure-3).

Pods run in Contrast's [runtime environment](../components/runtime.md) (B),
effectively within a confidential VM. During launch, the CPU (A) measures the
initial memory content of the confidential VM that contains Contrast's pod-VM
image and generates the corresponding attestation evidence. On AKS the image is
in [IGVM format](https://github.com/microsoft/igvm), encapsulating all
information required to launch a virtual machine, including the kernel, the
initramfs, and kernel cmdline. On bare-metal QEMU boots the image via
[direct Linux boot](https://qemu-project.gitlab.io/qemu/system/linuxboot.html)
and `kernel-hashes=on`, which signals QEMU to measure the kernel, initrd, and
cmdline passed via the flags. The kernel cmdline contains the root hash for
[dm-verity](https://www.kernel.org/doc/html/latest/admin-guide/device-mapper/verity.html)
that ensures the integrity of the root filesystem. The root filesystem contains
all components of the container's runtime environment including the
[guest agent](../basics/confidential-containers.md#kata-containers) (C).

In the userland, the guest agent takes care of enforcing the
[runtime policy](../components/overview.md#runtime-policies) of the pod. While
the policy is passed in during the initialization procedure via the host, the
evidence for the runtime policy is part of the CPU measurements. During the
[deployment](../deployment.md#generate-policy-annotations-and-manifest) the
policy is annotated to the Kubernetes Pod resources. The hypervisor adds the
hash of the policy to the attestation report via the HOSTDATA (on AMD SEV-SNP)
or MRCONFIGID (Intel TDX) fields. When provided with the policy from the Kata
host, the guest agent verifies that the policy's hash matches the one in the
`HOSTDATA`/`MRCONFIGID` field.

In summary a Pod's evidence is the attestation report of the CPU that provides
evidence for runtime environment and the runtime policy.

### Verifier: Coordinator and CLI

The [Coordinator](../components/overview.md#the-coordinator) acts as a verifier
within the Contrast deployment, configured with a
[Manifest](../components/overview.md#the-manifest) that defines the reference
values and serves as an appraisal policy for all pods in the deployment. It also
pulls endorsements from hardware vendors to verify the hardware claims. The
Coordinator operates within the cluster as a confidential container and provides
similar evidence as any other Pod when it acts as an attester. In RATS
terminology, the Coordinator's dual role is defined as a lead attester in a
composite device which spans the entire deployment: Coordinator and the workload
pods. It collects evidence from other attesters and conveys it to a verifier,
generating evidence about the layout of the whole composite device based on the
Manifest as the appraisal policy.

![Deployment attestation as a composite device](../_media/attestation-composite-device.svg)

Figure 3: Contrast deployment as a composite device. Based on the composite
device in [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html#figure-4).

The [CLI](../components/overview.md#the-cli-command-line-interface) serves as
the verifier for the Coordinator and the entire Contrast deployment, containing
the reference values for the Coordinator and the endorsements from hardware
vendors. These reference values are built into the CLI during our release
process and can be reproduced offline via reproducible builds.

### Relying Party: Data owner

A relying party in the Contrast scenario could be, for example, the
[data owner](../basics/security-benefits.md) that interacts with the
application. The relying party can use the CLI to obtain the attestation results
and Contrast's [CA certificates](certificates.md) bound to these results. The CA
certificates can then be used by the relying party to authenticate the
application, for example through TLS connections.

## Evidence generation and appraisal

### Evidence types and formats

In Contrast, attestation evidence revolves around a hardware-generated
attestation report, which contains several critical pieces of information:

- **The hardware attestation report**: This report includes details such as the
  chip identifier, platform information, microcode versions, and comprehensive
  guest measurements. The entire report is signed by the CPU's private key,
  ensuring the authenticity and integrity of the data provided.
- **The launch measurements**: Included within the hardware attestation report,
  this is a digest generated by the CPU that represents a hash of all initial
  guest memory pages. This includes essential components like the kernel,
  initramfs, and the kernel command line. Notably, it incorporates the root
  filesystem's dm-verity root hash, verifying the integrity of the root
  filesystem.
- **The runtime policy hash**: Also part of the hardware attestation report,
  this field contains the hash of the Rego policy which dictates all expected
  API commands and their values from the host to the Kata guest agent. It
  encompasses crucial settings such as dm-verity hashes for the container image
  layers, environment variables, and mount points.

### Appraisal policies for evidence

The appraisal of this evidence in Contrast is governed by two main components:

- **The Manifest**: A JSON file used by the Coordinator to align with reference
  values. It sets the expectations for runtime policy hashes for each pod and
  includes what should be reported in the hardware attestation report for each
  component of the deployment.
- **The CLI's appraisal policy**: This policy encompasses expected values of the
  Coordinator’s guest measurements and its runtime policy. It's embedded into
  the CLI during the build process and ensures that any discrepancy between the
  built-in values and those reported by the hardware attestation can be
  identified and addressed. The integrity of this policy is safeguardable
  through reproducible builds, allowing verification against the source code
  reference.

## Frequently asked questions about attestation in Contrast

### What's the purpose of remote attestation in Contrast?

Remote attestation in Contrast ensures that software runs within a secure,
isolated confidential computing environment. This process certifies that the
memory is encrypted and confirms the integrity and authenticity of the software
running within the deployment. By validating the runtime environment and the
policies enforced on it, Contrast ensures that the system operates in a
trustworthy state and hasn't been tampered with.

### How does Contrast ensure the security of the attestation process?

Contrast leverages hardware-rooted security features such as AMD SEV-SNP or
Intel TDX to generate cryptographic evidence of a pod’s current state and
configuration. This evidence is checked against pre-defined appraisal policies
to guarantee that only verified and authorized pods are part of a Contrast
deployment.

### What security benefits does attestation provide?

Attestation confirms the integrity of the runtime environment and the identity
of the workloads. It plays a critical role in preventing unauthorized changes
and detecting potential modifications at runtime. The attestation provides
integrity and authenticity guarantees, enabling relying parties—such as workload
operators or data owners—to confirm the effective protection against potential
threats, including malicious cloud insiders, co-tenants, or compromised workload
operators. More details on the specific security benefits can be found
[here](../basics/security-benefits.md).

### How can you verify the authenticity of attestation results?

Attestation results in Contrast are tied to cryptographic proofs generated and
signed by the hardware itself. These proofs are then verified using public keys
from trusted hardware vendors, ensuring that the results aren't only accurate
but also resistant to tampering. For further authenticity verification, all of
Contrast's code is reproducibly built, and the attestation evidence can be
verified locally from the source code.

### How are attestation results used by relying parties?

Relying parties use attestation results to make informed security decisions,
such as allowing access to sensitive data or resources only if the attestation
verifies the system's integrity. Thereafter, the use of Contrast's
[CA certificates in TLS connections](certificates.md) provides a practical
approach to communicate securely with the application.

## Summary

In summary, Contrast's attestation strategy adheres to the RATS guidelines and
consists of robust verification mechanisms that ensure each component of the
deployment is secure and trustworthy. This comprehensive approach allows
Contrast to provide a high level of security assurance to its users.
