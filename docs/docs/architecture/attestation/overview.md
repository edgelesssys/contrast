# Attestation in Contrast

Contrast uses remote attestation to ensure that each workload pod runs in a verifiable and trusted environment. Every pod is executed inside a Confidential Virtual Machine (CVM), and its integrity and configuration are verified at launch using hardware-based attestation.

The attestation process builds on Remote Attestation Procedures (RATS) as described in [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html), but is tailored specifically for Kubernetes and the Contrast architecture.

## Attestation roles in Contrast

Contrast separates responsibilities across three attestation roles:

### Attester: Confidential pods

Each pod is launched inside a CVM using a secure runtime based on Kata Containers. During launch:

- The CPU (AMD SEV-SNP or Intel TDX) measures the initial guest memory including the kernel, initramfs and kernel command line.
- The measurement is embedded in a hardware-signed attestation report.
- A hash of the pod's initdata document is embedded in a dedicated field of the report(`HOSTDATA` on SEV-SNP or `MRCONFIGID` on TDX).
- The report is signed by the CPU firmware and verifiable via vendor-provided public keys.

The initdata document contains an agent policy, which restricts requests from the untrusted Kata runtime to the agent.
On VM startup and before the Kata agent starts, a special process called `initdata-processor`:

- Finds the initdata document in a special block device.
- Computes a SHA-256 hash of the initdata document.
- Compares this hash against the value embedded in the attestation report.
- Aborts execution if the hash doesn't match.
- Reads the policy from the initdata document and writes it to a secure path where the Kata agent can read it.

The runtime policy specifies:

- Which container images (by cryptographic hash) may be executed
- Which environment variables are permitted
- Which mount points may be used
- Which host-to-guest calls are allowed

This guarantees that the CVM launches in a well-defined state and enforces only the explicitly declared configuration.

### Verifier: Contrast Coordinator & CLI

The Coordinator runs inside a CVM and verifies attestation reports from other pods. It:

- Checks attestation reports, including the initdata hash, against a trusted [manifest](../components/manifest.md)
- Issues service mesh certificates to verified pods

Only pods whose attestation evidence matches the manifest are accepted into the trusted service mesh.
The Coordinator is verifier for all workloads of a Contrast deployment and issues certificates as attestation result, is therefore the certificate authority for a deployment.

The Contrast Coordinator itself also runs as a confidential pod and is attested using the Contrast CLI.
The CLI includes embedded reference values for the Coordinator, allowing it to verify the Coordinator's identity and integrity during attestation.
Because these reference values are part of the CLI build, the CLI effectively serves as the root of trust for the deployment.
Verifying the CLI's integrity and authenticity is therefore essential.

The verification of the Coordinator by the CLI enables trust in the Coordinator to verify other workloads based on the manifest.
The workloads are then _attested transitively_ through the Coordinator.

### Relying party: Operator and data owner

The Contrast CLI is used by operators or data owners to verify the deployment and establish trust. It:

- Verifies the attestation of the Coordinator using embedded reference values
- Uploads the Manifest to define the trusted deployment state
- Retrieves the Root CA and Mesh CA certificates for the service mesh

## Evidence generation and verification

### What's included in the attestation report?

Each attestation report contains:

- **Launch Measurement**: A cryptographic digest of the guest memory at CVM startup
- **Initdata Hash**: The initdata document is embedded by the host and verified by the `initdata-processor`
- **Platform Info**: CPU type, TCB version, microcode versions
- **REPORTDATA**: A hash of the CVM's public key and a nonce, ensuring freshness and binding the attestation to a specific TLS session

### How verification works

- The Coordinator receives the attestation report from each pod.
- It compares the launch measurement and initdata hash to the manifest.
- If the evidence matches, the pod is approved and issued a Mesh CA certificate.
- If the evidence doesn't match, the pod is rejected and can't join the mesh.

The CLI verifies the Coordinator in the same way, using reference values embedded during the Contrast build process.

## Summary

Contrast enforces trust across a Kubernetes deployment using hardware-based attestation. Each pod's launch state and configuration are verified before it can access secrets or participate in the mesh. The Coordinator acts as a centralized verifier, using a declarative manifest to define the trusted state. The CLI provides an interface for verification and certificate retrieval, completing a robust and transparent attestation workflow.
