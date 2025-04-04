# SNP Attestation

The key component for attesting AMD SEV-SNP is the Security Processor (SP),
which measures the CVM and metadata and returns an attestation report reflecting those
measurements.


## Startup

The SP extends the launch digest every time the hypervisor donates a page to the CVM during startup via `SNP_LAUNCH_UPDATE`. On an abstract level, the launch digest is extended as follows:
```
LD := Hash(LD || Page || Page Metadata)
```

When the Hypervisor calls `SNP_LAUNCH_FINISH`, it provides the SP with the `HOST_DATA`,
the `ID_BLOCK`, and `ID_AUTH` block.

The `HOSTDATA` is opaque for the SP. Kata writes the hash of the kata policy in
this field to bind the policy to the CVM. `HOSTDATA` is later reflected in the
attestation report.

### ID Block Structure
The complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 75.

| Field    | Description    | Contrast usage |
| -------- | -------        | ------- |
| LD |   The expected launch digest of the guest.      | Expected launch digest over kernel, initrd, and cmdline of the CVM. |
| POLICY |   The policy of the guest.      |  |

The SP checks during startup if the measurement it calculated matches the `LD` in the ID Block.
If they don't match, the SP aborts the boot process.
Similarly, if the policy doesn't match the configuration of the CVM, the SP aborts also.

### ID Auth Structure
The ID auth structure exists to be able to verify the ID block structure.
A CVM image can be started with for example various different policies.
Moreover, the ID block itself can't be verified at a later date, since
it's not part of the attestation report.
The intended use is that a trusted party creates an ECDSA-384 ID key pair
and signs the ID block structure. Both the signature and the public part of the
ID key are then passed via the hypervisor to the SP which verifies the signature and
keeps a SHA-384 digest of the public key. The SP adds this digest in every attestation
report requested by the CVM.

The complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 76.

| Field    | Description    | Contrast usage |
| -------- | -------        | ------- |
| ID_BLOCK_SIG |   The signature of all bytes of the ID block.      | Constant value of (r,s) = (2,1) |
| ID_KEY |   The public component of the ID key.      | Deterministically derived from the ID Block and ID_BLOCK_SIG (see [Anonymous ID Block Signing](#anonymous-id-block-signing)) |

### Guest Policy Structure
The gest policy structure is embedded in the ID block in the `POLICY` field.
The complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 10.

| Field    | Description    | Value on cloud-hypervisor  | Value on QEMU |
| -------- | -------        | -------  | ------- |
| CIPHERTEXT_HIDING_DRAM |   0: Ciphertext hiding for the DRAM may be enabled or disabled. <br />1: Ciphertext hiding for the DRAM must be enabled.      | 0 | 0 |
| RAPL_DIS |   0: Allow Running Average Power Limit (RAPL).  <br />1: RAPL must be disabled     | 0 | 0 |
| MEM_AES_256_XTS |   0: Allow either AES 128 XEX or AES 256 XTS for memory encryption. <br /> 1: Require AES 256 XTS for memory encryption.     | 0 | 0 |
| CXL_ALLOW |   0: CXL can't be populated with devices or memory. <br /> 1: CXL can be populated with devices or memory.     | 0 | 0 |
| SINGLE_SOCKET |   0: Guest can be activated on multiple sockets. <br /> 1: Guest can be activated only on one socket.   | 0 | 0 |
| DEBUG |  0: Debugging is disallowed. <br />  1: Debugging is allowed  | 0 | 0 |
| MIGRATE_MA |  0: Association with a migration agent is disallowed. <br /> 1: Association with a migration agent is allowed.  | 0 | 0 |
| SMT |  0: SMT is disallowed. <br /> 1: SMT is allowed.  | 1 | 1 |
| ABI_MAJOR |  The minimum ABI major version required for this guest to run.  | 0 | 0 |
| ABI_MINOR |  The minimum ABI minor version required for this guest to run.  | 0 | 31 |



## Attestation Report

The attestation report is signed by the Versioned Chip Endorsement Key (VCEK).
The SP derives this key from the chip unique secret and the following REPORTED_TCB
information:
1. SP Bootloader SVN
1. SP OS SVN
1. SNP firmware SVN
1. Microcode patch level

With those parameters one can also retrieve a certificate signing the VCEK from the
AMD Key Distribution Service (KDS) by querying `https://kdsintf.amd.com/vcek/v1/{product_name}/{hwid}?{params}`

This VCEK certificate is signed by the AMD SEV CA certificate, which is signed by the AMD Root CA.
```
AMD Root CA --> AMD SEV CA --> VCEK -- signs --> Report
```

The Contrast CLI embeds the AMD Root CA and AMD SEV CA certificate

### Attestation Report Structure
The complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 23.

| Field    | Description    | Contrast usage |
| -------- | -------        | ------- |
| VERSION  | Version number of this attestation report. Set to `3h` for this specification.           |
| VMPL    | The firmware sets this value depending on whether a guest (MSG_REPORT_REQ) or host (SNP_HV_REPORT_REQ) requested the guest attestation report. For a Guest requested attestation report this field will contain the value (0-3). A Host requested attestation report will have a value of 0xffffffff.           |
| PLATFORM_INFO    | Information about the platform. See Table below         |
| REPORT_DATA    | If REQUEST_SOURCE is guest provided, then contains Guest-provided data, else host request and zero (0) filled by firmware.           | Digest of nonce provided by the relying party and TLS public key of the CVM. |
| MEASUREMENT    | The measurement calculated at launch.           | Digest over kernel, initrd, and cmdline. |
| HOST_DATA    | Data provided by the hypervisor at launch.           | Digest of the kata policy.
| ID_KEY_DIGEST    | SHA-384 digest of the ID public key that signed the ID block provided in SNP_LAUNCH_FINISH.           | Deterministic function of the SNP policy and launch digest in the ID_BLOCK. (see [Anonymous ID Block Signing](#anonymous-id-block-signing))
| CPUID_FAM_ID    | Family ID (Combined Extended Family ID and Family ID)           |
| CPUID_MOD_ID    | Model (combined Extended Model and Model fields)           |
| CPUID_STEP    | Stepping.           |
| LAUNCH_TCB    | The CurrentTcb at the time the guest was launched or imported.      | Lowest TCB the guest ever executed with. |
| SIGNATURE    | Signature of bytes `0h` to `29Fh` inclusive of this report.            | Used to verify the integrity and authenticity of the report.


### Platform Info Structure
The platform info structure is embedded in the attestation report in the `PLATFORM_INFO` field.
The complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 24.

| Field    | Description  |
| -------- | -------    |
| ALIAS_CHECK_COMPLETE | Indicates that alias detection has completed since the last system reset and there are no aliasing addresses. Resets to 0. Contains mitigation for CVE-2024-21944.        |  |
| CIPHERTEXT_HIDING_DRAM_EN | Indicates ciphertext hiding is enabled for the DRAM.        |  |
| RAPL_DIS | Indicates that the RAPL feature is disabled.        |  |
| ECC_EN | Indicates that the platform is using error correcting codes for memory. Present when EccMemReporting feature bit is set.        |  |
| TSME_EN | Indicates that TSME is enabled in the system.        |  |
| SMT_EN | Indicates that SMT is enabled in the system.        |  |


## Anonymous ID Block Signing
As described in [startup](#startup), the SP checks the signature of the ID block
with the public key provided in the ID auth block. The common usage of such signatures
is to know that a trusted party holding the private key has signed the ID block.
Since the ID block is part of for example the [IGVM](https://github.com/microsoft/igvm) headers of
the VM image, they're bound to the `runtimeClass` Contrast sets-up
during [installation](../getting-started/install.md).
Therefore, the ID auth block and the signature and public key has to be provided by
Contrast, but the authors of contrast shouldn't be part of the TCB.

To both have the ability to sign ID Blocks and not be part of the TCB, we must ensure
that there exists no private key for the `ID_KEY` in the ID Auth structure.
For this, we implement ECDSA public key recovery. The algorithm is defined in [SEC 1: Elliptic Curve Cryptography](https://www.secg.org/sec1-v2.pdf).
The algorithm calculates an ECDSA public key given a message and its signature.
We keep the signature constant as `(r,s) = (2,1)` for all versions and
use the given ID Block containing the policy and launch digest as an input.
The recovery algorithm returns two valid public keys from which we choose the smaller one, meaning
the one with the smaller x value and, if equal, the one with the smaller y value.

Since we don't generate any private key material during recovery and calculating the private
key from only the message, signature, and public key is cryptographically hard, no one
can forge (ID Block, signature) combinations under the same public key.
