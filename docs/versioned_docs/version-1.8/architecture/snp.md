# AMD SEV-SNP attestation

This page explains details of the AMD SEV-SNP attestation that are especially
relevant for the implementation in Contrast. You should be familiar with
[attestation basics](attestation.md). For more details on SEV-SNP, see the
[whitepapers and specifications from AMD](https://www.amd.com/en/developer/sev.html).

The key component for attesting AMD SEV-SNP is the Secure Processor (SP). It
measures the confidential VM (CVM) and metadata and returns an attestation
report reflecting those measurements.

## Startup

The SP extends the launch digest each time the hypervisor donates a page to the
CVM during startup via `SNP_LAUNCH_UPDATE`. On an abstract level, the launch
digest is extended as follows:

```
LD := Hash(LD || Page || Page Metadata)
```

When the Hypervisor calls `SNP_LAUNCH_FINISH`, it provides the SP with the
`HOST_DATA`, the `ID_BLOCK`, and `ID_AUTH` block.

The `HOST_DATA` is opaque to the SP. Kata writes the hash of the kata policy in
this field to bind the policy to the CVM. `HOST_DATA` is later reflected in the
attestation report.

### ID block structure

The ID block contains fields that identify the VM. The following table shows the
fields that are relevant for Contrast. The complete structure can be found in
the SEV Secure Nested Paging Firmware ABI Specification in table 75.

| Field  | Description                              | Contrast usage                                                      |
| ------ | ---------------------------------------- | ------------------------------------------------------------------- |
| LD     | The expected launch digest of the guest. | Expected launch digest over kernel, initrd, and cmdline of the CVM. |
| POLICY | The policy of the guest.                 |                                                                     |

During startup, the SP compares the measurement it calculated to the `LD` in the
ID block. If they don't match, the SP aborts the boot process. Similarly, if the
policy doesn't match the configuration of the CVM, the SP aborts also.

### ID auth structure

The ID auth structure exists to be able to verify the ID block structure. A CVM
image can be started with for example various different policies. Moreover, the
ID block itself can't be verified later, since it's not part of the attestation
report. The intended use is that a trusted party creates an ECDSA-384 ID key
pair and signs the ID block structure. Both the signature and the public part of
the ID key are then passed via the hypervisor to the SP. The SP verifies the
signature and keeps a SHA-384 digest of the public key. The SP adds this digest
in every attestation report requested by the CVM.

The complete structure can be found in the SEV Secure Nested Paging Firmware ABI
Specification in table 76.

| Field        | Description                                 | Contrast usage                                                                                                               |
| ------------ | ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| ID_BLOCK_SIG | The signature of all bytes of the ID block. | Constant value of (r,s) = (2,1)                                                                                              |
| ID_KEY       | The public component of the ID key.         | Deterministically derived from the ID block and ID_BLOCK_SIG (see [Anonymous ID block signing](#anonymous-id-block-signing)) |

### Guest policy structure

The guest policy structure is embedded in the ID block in the `POLICY` field.
The complete structure can be found in the SEV Secure Nested Paging Firmware ABI
Specification in table 10.

| Field                  | Description                                                                                                            | Value on cloud-hypervisor | Value on QEMU |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------- | ------------------------- | ------------- |
| CIPHERTEXT_HIDING_DRAM | 0: Ciphertext hiding for the DRAM may be enabled or disabled. <br />1: Ciphertext hiding for the DRAM must be enabled. | 0                         | 0             |
| RAPL_DIS               | 0: Allow Running Average Power Limit (RAPL). <br />1: RAPL must be disabled                                            | 0                         | 0             |
| MEM_AES_256_XTS        | 0: Allow either AES 128 XEX or AES 256 XTS for memory encryption. <br /> 1: Require AES 256 XTS for memory encryption. | 0                         | 0             |
| CXL_ALLOW              | 0: CXL can't be populated with devices or memory. <br /> 1: CXL can be populated with devices or memory.               | 0                         | 0             |
| SINGLE_SOCKET          | 0: Guest can be activated on multiple sockets. <br /> 1: Guest can be activated only on one socket.                    | 0                         | 0             |
| DEBUG                  | 0: Debugging is disallowed. <br /> 1: Debugging is allowed                                                             | 0                         | 0             |
| MIGRATE_MA             | 0: Association with a migration agent is disallowed. <br /> 1: Association with a migration agent is allowed.          | 0                         | 0             |
| SMT                    | 0: SMT is disallowed. <br /> 1: SMT is allowed.                                                                        | 1                         | 1             |
| ABI_MAJOR              | The minimum ABI major version required for this guest to run.                                                          | 0                         | 0             |
| ABI_MINOR              | The minimum ABI minor version required for this guest to run.                                                          | 31                        | 0             |

## Attestation report

The attestation report is signed by the Versioned Chip Endorsement Key (VCEK).
The SP derives this key from the chip unique secret and the following
REPORTED_TCB information:

1. SP bootloader SVN
1. SP OS SVN
1. SNP firmware SVN
1. Microcode patch level

With those parameters, one can retrieve a certificate signing the VCEK from the
AMD Key Distribution Service (KDS) by querying
`https://kdsintf.amd.com/vcek/v1/{product_name}/{hwid}?{params}`

This VCEK certificate is signed by the AMD SEV CA certificate, which is signed
by the AMD Root CA.

```
AMD Root CA --> AMD SEV CA --> VCEK -- signs --> Report
```

The Contrast CLI embeds the AMD Root CA and AMD SEV CA certificate.

### Attestation report structure

The following table shows the most important fields of the attestation report
and how they're used by Contrast. The complete structure can be found in the SEV
Secure Nested Paging Firmware ABI Specification in table 23.

| Field         | Description                                                                                                                | Contrast usage                                                                                                                              |
| ------------- | -------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| VERSION       | Version number of this attestation report. Set to `3h` for this specification.                                             |                                                                                                                                             |
| PLATFORM_INFO | Information about the platform. See Table below                                                                            |                                                                                                                                             |
| REPORT_DATA   | If REQUEST_SOURCE is guest provided, then contains Guest-provided data, else host request and zero (0) filled by firmware. | Digest of nonce provided by the relying party and TLS public key of the CVM.                                                                |
| MEASUREMENT   | The measurement calculated at launch.                                                                                      | Digest over kernel, initrd, and cmdline.                                                                                                    |
| HOST_DATA     | Data provided by the hypervisor at launch.                                                                                 | Digest of the kata policy.                                                                                                                  |
| ID_KEY_DIGEST | SHA-384 digest of the ID public key that signed the ID block provided in SNP_LAUNCH_FINISH.                                | Deterministic function of the SNP policy and launch digest in the ID_BLOCK. (see [Anonymous ID block signing](#anonymous-id-block-signing)) |
| LAUNCH_TCB    | The CurrentTcb at the time the guest was launched or imported.                                                             | Lowest TCB the guest ever executed with.                                                                                                    |
| SIGNATURE     | Signature of bytes `0h` to `29Fh` inclusive of this report.                                                                | Used to verify the integrity and authenticity of the report.                                                                                |

### Platform info structure

The platform info structure is embedded in the attestation report in the
`PLATFORM_INFO` field. The complete structure can be found in the SEV Secure
Nested Paging Firmware ABI Specification in table 24.

| Field                     | Description                                                                                                                                                        |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| ALIAS_CHECK_COMPLETE      | Indicates that alias detection has completed since the last system reset and there are no aliasing addresses. Resets to 0. Contains mitigation for CVE-2024-21944. |
| CIPHERTEXT_HIDING_DRAM_EN | Indicates ciphertext hiding is enabled for the DRAM.                                                                                                               |
| RAPL_DIS                  | Indicates that the RAPL feature is disabled.                                                                                                                       |
| ECC_EN                    | Indicates that the platform is using error correcting codes for memory. Present when EccMemReporting feature bit is set.                                           |
| TSME_EN                   | Indicates that TSME is enabled in the system.                                                                                                                      |
| SMT_EN                    | Indicates that SMT is enabled in the system.                                                                                                                       |

## Anonymous ID block signing

As described in [startup](#startup), the SP checks the signature of the ID block
with the public key provided in the ID auth block. The common usage of such
signatures is to know that a trusted party holding the private key has signed
the ID block. Since the ID block is part of for example the
[IGVM](https://github.com/microsoft/igvm) headers of the VM image, they're bound
to the `runtimeClass` Contrast sets up during
[installation](../getting-started/install.md). Therefore, the ID auth block and
the signature and public key has to be provided by Contrast, but the authors of
Contrast shouldn't be part of the TCB.

To have both the ability to sign ID blocks and not be part of the TCB, we must
ensure that there exists no private key for the `ID_KEY` in the ID auth
structure. For this, we implement ECDSA public key recovery. The algorithm is
defined in
[SEC 1: Elliptic Curve Cryptography](https://www.secg.org/sec1-v2.pdf). The
algorithm calculates an ECDSA public key given a message and its signature. We
keep the signature constant as `(r,s) = (2,1)` for all versions and use the
given ID block containing the guest policy and launch digest as an input. The
recovery algorithm returns two valid public keys. We choose the smaller one,
meaning the one with the smaller x value and, if equal, the one with the smaller
y value.

Since we don't generate any private key material during recovery and calculating
the private key from only the message, signature, and public key is
cryptographically hard, no one can forge (ID block, signature) combinations
under the same public key.
