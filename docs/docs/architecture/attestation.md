# Attestation in Contrast

This document describes the attestation architecture of Contrast, adhering to the definitions of Remote ATtestation procedureS (RATS) in [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html).
The following gives a detailed description of Contrast's attestation architecture.
At the end of this document, we included an [FAQ](#frequently-asked-questions-about-attestation-in-contrast) that answers the most common questions regarding attestation in hindsight of the [security benefits](../basics/security-benefits.md).

## Attestation Architecture
Contrast integrates with the RATS architecture, leveraging their definition of roles and processes including *Attesters*, *Verifiers*, and *Relying Parties*.

![Conceptual attestation architecture](../_media/attestation-rats-architecture.svg)

Figure 1: Conceptual attestation architecture. Taken from [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html#figure-1).


- **Attester**: Assigned to entities that are responsible for creating *Evidence* which is then sent to a *Verifier*.
- **Verifier**: These entities utilize the *Evidence*, *Reference Values*, and *Endorsements*. They assess the trustworthiness of the *Attester* by applying an *Appraisal Policy* for *Evidence*. Following this assessment, *Verifiers* generate *Attestation Results* for use by *Relying Parties*. The *Appraisal Policy* for *Evidence* may be provided by the *Verifier Owner*, programmed into the *Verifier*, or acquired through other means.
- **Relying Party**: Assigned to entities that utilize *Attestation Results*, applying their own appraisal policies to make specific decisions, such as authorization decisions. This process is referred to as the "appraisal of Attestation Results." The *Appraisal Policy* for *Attestation Results* might be sourced from the *Relying Party Owner*, configured by the owner, embedded in the *Relying Party*, or obtained through other protocols or mechanisms.

## Components of Contrast's Attestation
The key components involved in the attestation process of Contrast are detailed below:

### Attester: Application Pods
This includes all Pods of the Contrast deployment that run inside Confidential Containers and generate cryptographic evidence reflecting their current configuration and state.
Their evidence is rooted in the [hardware measurements](../basics/confidential-containers.md) from the CPU and their [confidential VM environment](../components/runtime.md).
The details of this evidence are given below in the section on [Evidence Generation and Appraisal](#evidence-generation-and-appraisal).

![Attestation flow of a confidential pod](../_media/attestation-pod.svg)

Figure 2: ttestation flow of a confidential pod. Based on the layered attester graphic in [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html#figure-3).

Pods run in Contrast's [runtime environment](../components/runtime.md) (B), effectively within a confidential VM.
During launch, the CPU (A) measures the initial memory content of the confidential VM that contains Contrast's pod-VM image and generates the corresponding attestation evidence.
The image is in [IGVM format](https://github.com/microsoft/igvm), encapsulating all information required to launch a virtual machine, including the kernel, the initramfs, and kernel cmdline.
The kernel cmdline contains the root hash for [dm-verity](https://www.kernel.org/doc/html/latest/admin-guide/device-mapper/verity.html) that ensures the integrity of the root filesystem.
The root filesystem contains all  components of the container's runtime environment including the [guest agent](../basics/confidential-containers.md#kata-containers) (C).

In the userland, the guest agent takes care of enforcing the containers [runtime policy](../components/index.md#runtime-policies).
While the policy is passed in during the initialization procedure via the Kata host, the evidence for the runtime policy is part of the CPU measurements.
During the [deployment](../deployment.md#generate-policy-annotations-and-manifest) the expected policy hash is annotated to the Kubernetes Pod resources.
On AMD SEV-SNP the hash of the policy is then added to the attestation report via the `HOSTDATA` field by the hypervisor.
When provided with the policy from the Kata host, the guest agent verifies that the policy's hash matches the one in the `HOSTDATA` field.

In summary a Pod's evidence is the attestation report of the CPU that provides evidence for runtime environment and the runtime policy.

### Verifier: Coordinator and CLI
The [Coordinator](../components/index.md#the-coordinator) acts as a verifier within the Contrast deployment, configured with a [Manifest](../components/index.md#the-manifest) that defines the reference values and serves as an appraisal policy for all pods in the deployment.
It also pulls endorsements from hardware vendors to verify the hardware claims.
The Coordinator operates within the cluster as a confidential container and provides similar evidence as any other Pod when it acts as an attester.
In RATS terminology, the Coordinator's dual role is defined as a lead attester in a composite device which spans the entire deployment: Coordinator and the workload pods.
It collects evidence from other attesters and conveys it to a verifier, generating evidence about the layout of the whole composite device based on the Manifest as the appraisal policy.


![Deployment attestation as a composite device](../_media/attestation-composite-device.svg)

Figure 3: Contrast deployment as a composite device. Based on the composite device in [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html#figure-4).

The [CLI](../components/index.md#the-cli-command-line-interface) serves as the verifier for the Coordinator and the entire Contrast deployment, containing the reference values for the Coordinator and the endorsements from hardware vendors.
These reference values are built into the CLI during our release process and can be reproduced offline via reproducible builds.

### Relying Party: Data Owner
A relying party in the Contrast scenario could be, for example, the [data owner](../basics/security-benefits.md) that interacts with the application.
The relying party can use the CLI to obtain the attestation results and Contrast's [CA certificates](certificates.md) bound to these results.
The CA certificates can then be used by the relying party to authenticate the application, for example through TLS connections.


## Evidence Generation and Appraisal

### Evidence Types and Formats
Several types of attestation evidence exist in Contrast:
- **The hardware attestation report**: For AMD SEV-SNP, this includes information such as chip identifier, platform info, microcode versions, and guest measurements. The report also contains the runtime policy hash and is signed by the CPU's private key.
- **The guest measurements**: A launch digest generated by the CPU, which is the hash of all initial guest memory pages, containing the kernel, initramfs, and kernel command line including the root filesystem's dm-verity root hash.
- **The runtime policy hash**: The hash of the Rego policy that defines all expected API commands and their values from the host to the Kata guest agent, including the dm-verity hashes for the container image layers, environment variables, and mount points.

### Appraisal Policies for Evidence
The appraisal policies in Contrast consist of two parts:
- **The Manifest**: A JSON file configuring the Coordinator with reference values for the runtime policy hashes for each pod in the deployment and the expected hardware attestation report evidence.
- **The CLI's appraisal policy**: Contains the Coordinator's guest measurements and its runtime policy. The policy is baked into the CLI during the build process and can be compared against the source code reference via reproducible builds.



## Frequently Asked Questions about Attestation in Contrast

### What is the purpose of remote attestation in Contrast?

Remote attestation in Contrast provides a mechanism to verify the integrity and authenticity of the software running within the deployment.
By validating the runtime environment and the policies enforced on it, Contrast ensures that the system operates in a trustworthy state and has not been tampered with.

### How does Contrast ensure the security of the attestation process?

Contrast leverages hardware-rooted security features such as AMD SEV-SNP to generate cryptographic evidence of a pod’s current state and configuration.
This evidence is checked against pre-defined appraisal policies to guarantee that only verified and authorized configurations are operational, significantly reducing the risk of malicious modifications.

### What security benefits does attestation provide?

Attestation confirms the integrity of the operating environment and the compliance of the system with the security policies set by the organization.
It plays a critical role in preventing unauthorized changes and detecting potential attacks at runtime.
More details on the specific security benefits can be found [here](../basics/security-benefits.md).

### How can I verify the authenticity of Attestation Results?

Attestation Results in Contrast are tied to cryptographic proofs generated and signed by the hardware itself.
These proofs are then verified using public keys from trusted hardware vendors, ensuring that the results are not only accurate but also resistant to tampering.
For further authenticity verification, all of Contrast's code is reproducibly built, and the attestation evidence can be verified locally from the source code.

### How are Attestation Results used by Relying Parties?

Relying Parties use *Attestation Results* to make informed security decisions, such as allowing access to sensitive data or resources only if the attestation verifies the system's integrity.
Thereafter, the use of Contrast's [CA certificates in TLS connections](certificates.md) provides a practical approach to communicate securely with the application.

## Summary

In summary, Contrast's attestation strategy adheres to the RATS guidelines and consists of robust verification mechanisms that ensure each component of the deployment is secure and trustworthy.
This comprehensive approach allows Contrast to provide a high level of security assurance to its users.
