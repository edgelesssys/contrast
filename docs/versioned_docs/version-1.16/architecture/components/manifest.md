# The manifest

The manifest is the configuration file for the Coordinator, defining your confidential deployment.
It's automatically generated from your deployment by the Contrast CLI and uses JSON format (`manifest.json`).

The manifest has the following higher level structure:

```json
{
  "Policies": {
    "<policy-hash>": {
      "SANs": [ "<san1>", "<san2>", ... ],
      "WorkloadSecretID": "<workload-secret-id>",
      "Role": "<role>"
    },
    ...
  },
  "ReferenceValues": {
    "snp": [
      {
        "ProductName": "<product-name>",
        "TrustedMeasurement": "<trusted-measurement>",
        "MinimumTCB": { },
        "GuestPolicy": { },
        "PlatformInfo": { }
      },
      ...
    ],
    "tdx": [
      {
        "MrTd": "<mr-td>",
        "MrSeam": "<mr-seam>",
        "Rtmrs": [ "<rtmr0>", "<rtmr1>", "<rtmr2>", "<rtmr3>" ],
        "Xfam": "<xfam>"
      },
      ...
    ]
  },
  "WorkloadOwnerKeyDigests": [ "<workload-owner-key-digest1>", "<workload-owner-key-digest2>", ... ],
  "SeedshareOwnerKeys": [ "<seedshare-owner-key1>", "<seedshare-owner-key2>", ... ]
}
```

## `Policies` {#policies}

The identities of your Pods, represented by the hashes of their respective initdata documents.

### `Policies.*.SANs` {#policies-sans}

The Coordinator will add the Subject Alternative Names (SANs) to the workload certificate for the workload with the respective policy hash after verification.
Allowed values are:

- IP addresses
- URIs
- DNS names

By default, the Contrast CLI will add two SANs for each workload on generate: The pod name as DNS name under which the pod can be reached inside the cluster,
and a wildcard DNS name `*` to allow the certificate to be used with any other hostname (for example an external load balancer).
As DNS is untrusted in the context of Contrast, issuing a wildcard certificate won't weaken the security of your workload.
In addition, the Coordinator will add the pod IP address to the certificate.

If the pod is exposed via a different IP than the pod IP, for example a load balancer, and you want to include that IP in the certificate for verification,
insert the desired IP address into the list of SANs.
The change could look like this:

```diff
   "Policies": {
     ...
     "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": {
       "SANs": [
         "web",
-        "*"
+        "*",
+        "203.0.113.34"
       ],
     },
```

Some authentication and authorization schemes rely on URI SANs in X.509 certificates.
For example, [SPIFFE IDs] are URIs and are often found as certificate SANs.
The change to add such an URI SAN to the manifest could look like this:

```diff
   "Policies": {
     ...
     "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": {
       "SANs": [
         "web",
-        "*"
+        "*",
+        "spiffe://acme.com/billing/payments"
       ],
     },
```

[SPIFFE IDs]: https://spiffe.io/docs/latest/spiffe-about/spiffe-concepts/#spiffe-id

### `Policies.*.WorkloadSecretID` {#policies-workload-secret-id}

The `WorkloadSecretID` is used in addition to the secret seed to derive the workload secret.
It's a non-interpreted string and defaults to the qualified Kubernetes resource name of the pod.
As long as the `WorkloadSecretID` remains unchanged, the derived workload secret will remain stable across manifest updates and Coordinator recovery.

See [Workload secrets](../secrets.md#workload-secrets) for more information.

### `Policies.*.Role` {#policies-role}

Contrast role assigned to the workload.
The only supported value is `coordinator`, which identifies the Coordinator within the manifest.
Workloads don't set this field.

## `ReferenceValues` {#reference-values}

The remote attestation reference values for the confidential micro-VM that's the runtime environment of your Pods.
The reference values cover both the platform configuration as well as the guest TCB.
They're independent from the workload executed inside the Contrast pod VM and only differ between platforms or Contrast versions.

The reference values are grouped by confidential computing technology, there is a `snp` and a `tdx` section.
Each of those sections contains a list of reference value sets.
The Coordinator will accept a workload if its attestation report matches _any_ of the listed reverence value sets exactly.

### `ReferenceValues.snp.*.ProductName` {#snp-product-name}

The product name of your platform.
`Milan` and `Genoa` are supported by Contrast.

### `ReferenceValues.snp.*.TrustedMeasurement` {#snp-trusted-measurement}

The `TrustedMeasurement` is a hash over the initial memory contents and state of the confidential VM.
It covers the guest firmware, the initrd and kernel as well as the kernel command line.
The kernel command line contains the dm-verity hash of the root filesystem, which contains all Contrast components that run inside the guest.

It's the (launch) `MEASUREMENT` from the SNP `ATTESTATION_REPORT`, according to Table 23 in the [SEV ABI Spec].

### `ReferenceValues.snp.*.MinimumTCB` {#snp-minimum-tcb}

The `MinimumTCB` defines the minimum secure version numbers (SVNs) for the platform components.
The Contrast Coordinator compares these value against both the `COMMITTED_TCB` and `LAUNCH_TCB`,
where the `LAUNCH_TCB` is the TCB that was committed when the VM was first launched,
and the `COMMITTED_TCB` is the TCB that's currently committed on the platform.
Provisional firmware isn't considered, as it can be rolled back by a malicious platform operator at any time.

AMD doesn't provide an accessible way to acquire the latest TCB values for your platform.
Visit the [AMD SEV developer portal](https://www.amd.com/en/developer/sev.html) and download the latest firmware package for your processor family.
Unpack and inspect the contained release notes, which state the SNP firmware SVN (called `SPL` (security patch level) in that document).
Contact your hardware vendor or BIOS firmware provider for information about the other TCB components

To check the current TCB level of your platform, use [`snphost`]:

```sh
snphost show tcb
```
```console
Reported TCB: TCB Version:
  Microcode:   72
  SNP:         23
  TEE:         0
  Boot Loader: 9
  FMC:         None
Platform TCB: TCB Version:
  Microcode:   72
  SNP:         23
  TEE:         0
  Boot Loader: 9
  FMC:         None
```

The values listed as `Reported TCB` to should be greater or equal to the `MinimumTCB` values in `manifest.json`.
The `Platform TCB` can be higher than the `Reported TCB`, in this case, the platform has provisional firmware enrolled.
Contrast relies on the committed TCB values, as provisional firmware can be rolled back anytime by the platform operator.

:::warning

The TCB values observed on the target platform using `snphost` might not be trustworthy.
Your channel to the system or the system itself might be compromised.
The deployed firmware could be outdated and vulnerable.

:::

#### `ReferenceValues.snp.*.MinimumTCB.BootloaderVersion` {#snp-tcb-bootloader-version}

SVN of the bootloader of the secure processor.

#### `ReferenceValues.snp.*.MinimumTCB.TEEVersion` {#snp-tcb-tee-version}

SVN of the OS of the secure processor.
See [SEV ABI Spec], Section 2.2.

#### `ReferenceValues.snp.*.MinimumTCB.SNPVersion` {#snp-tcb-snp-version}

SVN of the SNP firmware.
See [SEV ABI Spec], Section 2.2.

#### `ReferenceValues.snp.*.MinimumTCB.MicrocodeVersion` {#snp-tcb-microcode-version}

SVN of the CPU microcode.
See [SEV ABI Spec], Section 2.2.

#### `ReferenceValues.snp.*.MinimumTCB.FMCVersion` {#snp-tcb-fmc-version}

Always `None` for Milan and Genoa platforms.

### `ReferenceValues.snp.*.GuestPolicy` {#snp-guest-policy}

This is the guest policy according to Section 4.3 of the [SEV ABI Spec].
It's enforced during the launch of the confidential VM.

The guest policy is currently static in Contrast, values can't be changed.

<!-- TODO(katexochen): Add more detailed description and recommendation for these fields.
#### `ReferenceValues.snp.*.GuestPolicy.ABIMinor`
#### `ReferenceValues.snp.*.GuestPolicy.ABIMajor` {#snp-guest-policy-abi-major}
#### `ReferenceValues.snp.*.GuestPolicy.SMT` {#snp-guest-policy-smt}
#### `ReferenceValues.snp.*.GuestPolicy.MigrateMA` {#snp-guest-policy-migrate-ma}
#### `ReferenceValues.snp.*.GuestPolicy.Debug` {#snp-guest-policy-debug}
#### `ReferenceValues.snp.*.GuestPolicy.SingleSocket` {#snp-guest-policy-single-socket}
#### `ReferenceValues.snp.*.GuestPolicy.CXLAllowed` {#snp-guest-policy-cxl-allowed}
#### `ReferenceValues.snp.*.GuestPolicy.MemAES256XTS` {#snp-guest-policy-mem-aes256xts}
#### `ReferenceValues.snp.*.GuestPolicy.RAPLDis` {#snp-guest-policy-rapl-dis}
#### `ReferenceValues.snp.*.GuestPolicy.CipherTextHidingDRAM` {snp-guest-policy-ciphertext-hiding-dram}
#### `ReferenceValues.snp.*.GuestPolicy.PageSwapDisable` {#snp-guest-policy-page-swap-disable}
-->

### `ReferenceValues.snp.*.PlatformInfo` {#snp-platform-info}

The `PLATFORM_INFO` structure according to Table 24 in the [SEV ABI Spec].

This has some overlap with the [`GuestPolicy`](#snp-guest-policy), but is checked as part of the attestation report.

<!-- TODO(katexochen): Add more detailed description and recommendation for these fields.
#### `ReferenceValues.snp.*.PlatformInfo.SMTEnabled` {#snp-platform-info-smt-enabled}
#### `ReferenceValues.snp.*.PlatformInfo.TSMEEnabled` {#snp-platform-info-tsme-enabled}
#### `ReferenceValues.snp.*.PlatformInfo.ECCEnabled` {#snp-platform-info-ecc-enabled}
#### `ReferenceValues.snp.*.PlatformInfo.RAPLDisabled` {#snp-platform-info-rapl-disabled}
#### `ReferenceValues.snp.*.PlatformInfo.CiphertextHidingDRAMEnabled` {#snp-platform-info-ciphertext-hiding-dram-enabled}
#### `ReferenceValues.snp.*.PlatformInfo.AliasCheckComplete` {#snp-platform-info-alias-check-complete}
#### `ReferenceValues.snp.*.PlatformInfo.TIOEnabled` {#snp-platform-info-tio-enabled}
-->

### `ReferenceValues.snp.AllowedChipIDs`

These are matched against the `CHIP_ID` field from the SNP attestation report, as documented in Table 23 in the [SEV ABI Spec].
If the list is empty or null, all chip IDs are accepted.

In case hardware is operated by you instead of a third party, or you are able to gain physical access to the hardware to audit it,
you can list the known-good chip IDs of the hardware to ensure that the hardware is genuine and operated in a trusted physical environment.

To show the chip ID of a system, use [`snphost`]:

```sh
snphost show identifier
```

The returned value can be inserted into the `AllowedChipIDs` list.

:::warning

The chip ID must be retrieved from a machine by physically accessing it.
If you retrieve the chip ID via a remote channel, your traffic could already be redirected to a hostile environment that allows an attacker physical access.

:::

### `ReferenceValues.tdx.*.MrTd` {#tdx-mr-td}

The TD measurement register (`MRTD`) is a hash over the initial memory content of the confidential VM.
The initial memory content for Contrast is the OVMF guest firmware.

See Table 3.50 (`TDINFO_BASE`) and Section 5.4.53.3.2 in the [TDX ABI Spec] for more information.

### `ReferenceValues.tdx.*.MrSeam` {#tdx-mr-seam}

`MrSeam` is the SHA384 hash of the TDX module that created the confidential VM, according to Table 3.46 (`TEE_TCB_INFO`) in the [TDX ABI Spec].

You should retrieve the TDX module via a trustworthy channel from Intel, for example by downloading the TDX module from [Intel's GitHub repository] and hashing the module on a trusted machine.
You can also reproduce the release artifact by following the build instructions linked in the release notes.

You can check the hash of the in-use TDX module by executing

```sh
sha384sum /boot/efi/EFI/TDX/TDX-SEAM.so | cut -d' ' -f1
```

:::warning

The TDX module hash (`MrSeam`) observed on the target platform might not be trustworthy.
Your channel to the system or the system itself might be compromised.
Make sure to retrieve or reproduce the value on a trusted machine.

:::

[Intel's GitHub repository]: https://github.com/intel/confidential-computing.tdx.tdx-module/releases

### `ReferenceValues.tdx.*.Rtmrs[4]` {#tdx-rtmrs}

RTMRs are the runtime extendable measurement registers of TDX, as specified in Table 3.50 (`TDINFO_BASE`) in the [TDX ABI Spec].
They cover the guest firmware, the initrd and kernel as well as the kernel command line.
The kernel command line contains the dm-verity hash of the root filesystem, which contains all Contrast components that run inside the guest.

### `ReferenceValues.tdx.*.Xfam` {#tdx-xfam}

The extended features available mask (`XFAM`) determines the set of extended features available for use by the guest and is documented in Section 3.4.2 (`XFAM`) in the [TDX ABI Spec].

### `ReferenceValues.tdx.AllowedPIIDs`

These are matched against the `PIID` field from the PCK certificate, as documented in section 1.3.5 of the [SGX PCK Spec].
If the list is empty or null, all PIIDs are accepted.

In case hardware is operated by you instead of a third party, or you are able to gain physical access to the hardware to audit it,
you can obtain the PIID with the following steps:

1. Install and run Intel's [`PCKIDRetrievalTool`].
   This should place a CSV file in your working directory.
2. Retrieve the following fields from the CSV file:
   - `EncryptedPPID`
   - `PCE_ID`
   - `CPUSVN`
   - `PCE ISVSVN`
3. Use these values to [request a PCK certificate from Intel PCS](https://api.portal.trustedservices.intel.com/content/documentation.html#pcs-certificate-v4).
   Note that the response contains intermediate certificates in the `SGX-PCK-Certificate-Issuer-Chain` header that are required to verify the PCK certificate's signature.
4. Verify that the PCK certificate chains back to Intel's root, for example with `openssl verify`.
5. Parse the PCK certificate to find the SGX extension address:

   ```sh
   openssl asn1parse -in pck.pem
   ```

   Example output, showing the extension address `624` right after its ASN.1 OID:

   ```txt
     613:d=5  hl=2 l=   9 prim: OBJECT            :1.2.840.113741.1.13.1
     624:d=5  hl=4 l= 554 prim: OCTET STRING      [HEX DUMP]: [...]
   ```

6. Parse the SGX extension to find the PIID:

   ```sh
   openssl asn1parse -in pck.pem --strparse $ADDRESS
   ```

   Example output with `ADDRESS=624`, showing the PIID right after its ASN.1 OID:

   ```txt
     454:d=2  hl=2 l=  10 prim: OBJECT            :1.2.840.113741.1.13.1.6
     466:d=2  hl=2 l=  16 prim: OCTET STRING      [HEX DUMP]:E90210702A2CC5AD9764F29DDC8FDE8C
   ```

   Copy the value shown after `[HEX DUMP]` into the `AllowedPIIDs` field.

:::warning

The EncryptedPPID must be retrieved from a machine by physically accessing it.
If you retrieve this value via a remote channel, your traffic could already be redirected to a hostile environment that allows an attacker physical access.

:::

## `WorkloadOwnerKeyDigests` {#workload-owner-key-digests}

A list of workload owner public key digests.
Used for authenticating subsequent manifest updates.

By default, the list contains the digest of the key that was passed to the Contrast CLI on `contrast generate` via the `--add-workload-owner-key` flag.
If the flag wasn't used, the workload owner key was generated and stored in the workspace as `workload-owner.pem`.

The Coordinator uses this list to authenticate manifest updates submitted via `contrast set`.
If multiple workload owner keys are specified, any of the corresponding private keys can be used to set a new manifest.

If the manifest is generated with the `--disable-updates` flag, the `WorkloadOwnerKeyDigests` list is empty.
In this case, updates to the manifest are disabled and the [deployment is immutable](../../howto/immutable-deployments.md).

## `SeedshareOwnerKeys` {#seedshare-owner-keys}

Public keys of seed share owners.
Used to authenticate user recovery and permission to handle the secret seed.

Setting a manifest where the `WorkloadOwnerKeyDigests` has been removed will render the deployment [immutable](../../howto/immutable-deployments.md).
Doing the same for the `SeedshareOwnerKeys` field makes Coordinator recovery and workload secret recovery impossible.

[`snphost`]: https://github.com/virtee/snphost
[SEV ABI Spec]: https://www.amd.com/content/dam/amd/en/documents/developer/56860.pdf
[TDX ABI Spec]: https://cdrdv2.intel.com/v1/dl/getContent/733579
[SGX PCK Spec]: https://api.trustedservices.intel.com/documents/Intel_SGX_PCK_Certificate_CRL_Spec-1.5.pdf
[`PCKIDRetrievalTool`]: https://github.com/intel/confidential-computing.tee.dcap/blob/717f2a91ca732c3309b0c59d21757463133eb440/tools/PCKRetrievalTool/README.txt
