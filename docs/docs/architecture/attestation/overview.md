# Attestation in Contrast

Contrast uses remote attestation to ensure that each workload pod runs in a
verifiable and trusted environment. Every pod is executed inside a Confidential
Virtual Machine (CVM), and its integrity and configuration are verified at
launch using hardware-based attestation.

The attestation process builds on Remote Attestation Procedures (RATS) as
described in [RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html), but is
tailored specifically for Kubernetes and the Contrast architecture.

## Attestation roles in Contrast

Contrast separates responsibilities across three attestation roles:

### Attester: Confidential pods

Each pod is launched inside a CVM using a secure runtime based on Kata
Containers. During launch:

- The CPU (AMD SEV-SNP or Intel TDX) measures the initial guest memory—including
  the kernel, initramfs and kernel command line.
- The measurement is embedded in a hardware-signed attestation report.
- A hash of the pod’s runtime policy is embedded in a dedicated field of the
  report(`HOSTDATA` on SEV-SNP or `MRCONFIGID` on TDX).
- The report is signed by the CPU firmware and verifiable via vendor-provided
  public keys.

The runtime policy is enforced inside the CVM by the `kata-agent`. On startup,
the agent:

- Reads the Base64-encoded policy passed in as a pod annotation.
- Computes a SHA-256 hash of the policy document.
- Compares this hash against the value embedded in the attestation report.
- Aborts execution if the hash doesn't match.

The runtime policy specifies:

- Which container images (by cryptographic hash) may be executed
- Which environment variables are permitted
- Which mount points may be used
- Which host-to-guest calls are allowed

This guarantees that the CVM launches in a well-defined state and enforces only
the explicitly declared configuration.

### Verifier: Contrast Coordinator & CLI

The Coordinator runs inside a CVM and verifies attestation reports from other
pods. It:

- Checks launch measurements and policy hashes against a trusted **manifest**
- Issues service mesh certificates to verified pods
- Functions as the verifier for the full deployment

The **manifest** is a JSON configuration that defines the trusted state of the
deployment. It includes:

- **ReferenceValues**: Expected CVM launch measurements
- **Policies**: Hashes of accepted runtime policies
- **WorkloadOwnerKeyDigests**: Public key digests used to authorize future
  manifest updates
- **SeedshareOwnerPubKeys**: Used for securely recovering workload secrets and
  restoring trust

Only pods whose attestation evidence matches the manifest are accepted into the
trusted service mesh.

The Contrast Coordinator itself also runs as a confidential pod and is attested
using the Contrast CLI. The CLI includes embedded reference values for the
Coordinator, allowing it to verify the Coordinator's identity and integrity
during attestation. Because these reference values are part of the CLI build,
the CLI effectively serves as the root of trust for the deployment. Verifying
the CLI’s integrity and authenticity is therefore essential.

### Relying party: Operator and data owner

The Contrast CLI is used by operators or data owners to verify the deployment
and establish trust. It:

- Verifies the attestation of the Coordinator using embedded reference values
- Uploads the Manifest to define the trusted deployment state
- Retrieves the Root CA and Mesh CA certificates for the service mesh

## Evidence generation and verification

### What's included in the attestation report?

Each attestation report contains:

- **Launch Measurement**: A cryptographic digest of the guest memory at CVM
  startup
- **Runtime Policy Hash**: The policy is embedded by the host and verified by
  the `kata-agent`
- **Platform Info**: CPU type, TCB version, microcode versions
- **REPORTDATA**: A hash of the CVM’s public key and a nonce, ensuring freshness
  and binding the attestation to a specific TLS session

### How verification works

- The Coordinator receives the attestation report from each pod.
- It compares the launch measurement and policy hash to the manifest.
- If the evidence matches, the pod is approved and issued a Mesh CA certificate.
- If the evidence doesn't match, the pod is rejected and can't join the mesh.

The CLI verifies the Coordinator in the same way, using reference values
embedded during the Contrast build process.

## Summary

Contrast enforces trust across a Kubernetes deployment using hardware-based
attestation. Each pod’s launch state and configuration are verified before it
can access secrets or participate in the mesh. The Coordinator acts as a
centralized verifier, using a declarative manifest to define the trusted state.
The CLI provides an interface for verification and certificate retrieval,
completing a robust and transparent attestation workflow.
