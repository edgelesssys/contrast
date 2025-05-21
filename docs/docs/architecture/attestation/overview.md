# Attestation in Contrast

Contrast uses remote attestation to ensure that each workload pod runs in a verifiable and trusted environment. Every pod is executed inside a Confidential Virtual Machine (CVM), and its integrity and configuration are verified at launch using hardware-based attestation.

The attestation process builds on Remote Attestation Procedures (RATS) as described in [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html), but is tailored specifically for Kubernetes and the Contrast architecture.

## Attestation roles in Contrast

Contrast separates responsibilities across three attestation roles:

### Attester: Confidential pods

Each pod is launched inside a CVM using a secure runtime based on Kata Containers. During launch:

- The CPU (AMD SEV-SNP or Intel TDX) measures the initial guest memory—including the kernel, initramfs and kernel command line.
- The measurement is embedded in a hardware-signed attestation report.
- A hash of the pod’s runtime policy is embedded in a dedicated field of the report(`HOSTDATA` on SEV-SNP or `MRCONFIGID` on TDX).
- The report is signed by the CPU firmware and verifiable via vendor-provided public keys.

The runtime policy is enforced inside the CVM by the `kata-agent`. On startup, the agent:

- Reads the Base64-encoded policy passed in as a pod annotation.
- Computes a SHA-256 hash of the policy document.
- Compares this hash against the value embedded in the attestation report.
- Aborts execution if the hash doesn't match.

The runtime policy specifies:

- Which container images (by cryptographic hash) may be executed
- Which environment variables are permitted
- Which mount points may be used
- Which host-to-guest calls are allowed

This guarantees that the CVM launches in a well-defined state and enforces only the explicitly declared configuration.

### Verifier: Contrast Coordinator & CLI

The Coordinator runs inside a CVM and verifies attestation reports from other pods. It:

- Checks launch measurements and policy hashes against a trusted **manifest**
- Issues service mesh certificates to verified pods
- Functions as the verifier for the full deployment

The **manifest** is a JSON configuration that defines the trusted state of the deployment. It includes:

- **ReferenceValues**: Expected CVM launch measurements
- **Policies**: Hashes of accepted runtime policies
- **WorkloadOwnerKeyDigests**: Public key digests used to authorize future manifest updates
- **SeedshareOwnerPubKeys**: Used for securely recovering workload secrets and restoring trust

Only pods whose attestation evidence matches the manifest are accepted into the trusted service mesh.

The Contrast Coordinator itself also runs as a confidential pod and is attested using the Contrast CLI.
The CLI includes embedded reference values for the Coordinator, allowing it to verify the Coordinator's identity and integrity during attestation.
Because these reference values are part of the CLI build, the CLI effectively serves as the root of trust for the deployment.
Verifying the CLI’s integrity and authenticity is therefore essential.

### Relying party: Operator and data owner

The Contrast CLI is used by operators or data owners to verify the deployment and establish trust. It:

- Verifies the attestation of the Coordinator using embedded reference values
- Uploads the Manifest to define the trusted deployment state
- Retrieves the Root CA and Mesh CA certificates for the service mesh

## Evidence generation and verification

### What's included in the attestation report?

Each attestation report contains:

- **Launch Measurement**: A cryptographic digest of the guest memory at CVM startup
- **Runtime Policy Hash**: The policy is embedded by the host and verified by the `kata-agent`
- **Platform Info**: CPU type, TCB version, microcode versions
- **REPORTDATA**: A hash of the CVM’s public key and a nonce, ensuring freshness and binding the attestation to a specific TLS session

### How verification works

- The Coordinator receives the attestation report from each pod.
- It compares the launch measurement and policy hash to the manifest.
- If the evidence matches, the pod is approved and issued a Mesh CA certificate.
- If the evidence doesn't match, the pod is rejected and can't join the mesh.

The CLI verifies the Coordinator in the same way, using reference values embedded during the Contrast build process.

## Summary

Contrast enforces trust across a Kubernetes deployment using hardware-based attestation. Each pod’s launch state and configuration are verified before it can access secrets or participate in the mesh. The Coordinator acts as a centralized verifier, using a declarative manifest to define the trusted state. The CLI provides an interface for verification and certificate retrieval, completing a robust and transparent attestation workflow.

## FAQ

### What's the purpose of remote attestation in Contrast?

Remote attestation in Contrast ensures that software runs within a secure, isolated confidential computing environment.
This process certifies that the memory is encrypted and confirms the integrity and authenticity of the software running within the deployment.
By validating the runtime environment and the policies enforced on it, Contrast ensures that the system operates in a trustworthy state and hasn't been tampered with.

### How does Contrast ensure the security of the attestation process?

Contrast leverages hardware-rooted security features such as AMD SEV-SNP or Intel TDX to generate cryptographic evidence of a pod’s current state and configuration.
This evidence is checked against pre-defined appraisal policies to guarantee that only verified and authorized pods are part of a Contrast deployment.

### What security benefits does attestation provide?

Attestation confirms the integrity of the runtime environment and the identity of the workloads.
It plays a critical role in preventing unauthorized changes and detecting potential modifications at runtime.
The attestation provides integrity and authenticity guarantees, enabling relying parties—such as workload operators or data owners—to confirm the effective protection against potential threats, including malicious cloud insiders, co-tenants, or compromised workload operators.
More details on the specific security benefits can be found [here](../../security.md).

### How can you verify the authenticity of attestation results?

Attestation results in Contrast are tied to cryptographic proofs generated and signed by the hardware itself.
These proofs are then verified using public keys from trusted hardware vendors, ensuring that the results aren't only accurate but also resistant to tampering.
For further authenticity verification, all of Contrast's code is reproducibly built, and the attestation evidence can be verified locally from the source code.

### How are attestation results used by relying parties?

Relying parties use attestation results to make informed security decisions, such as allowing access to sensitive data or resources only if the attestation verifies the system's integrity.
Thereafter, the use of Contrast's [CA certificates in TLS connections](../components/service-mesh.md) provides a practical approach to communicate securely with the application.
