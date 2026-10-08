# Manifest update

This section guides you through the process of updating a manifest at the Contrast Coordinator.

## Applicability

If the manifest changes, for example, due to modifications in the service mesh configuration, it must be updated at the Coordinator.

## Prerequisites

1. A running Contrast deployment
2. [Connect to the Coordinator](./workload-deployment/connect-to-coordinator.md)

## How-to

Set the changed manifest at the Coordinator with:

```sh
contrast set -c "${coordinator}:1313" resources/
```

The Contrast Coordinator will rotate the mesh ca certificate on the manifest update. Workload certificates issued
after the manifest update are thus issued by another certificate authority and services receiving the new CA certificate chain
won't trust parts of the deployment that got their certificate issued before the update. This way, Contrast ensures
that parts of the deployment that received a security update won't be infected by parts of the deployment at an older
patch level that may have been compromised. The `mesh-ca.pem` is updated with the new CA certificate chain.

### Rolling out the update

The Coordinator has the new manifest set, but the different containers of the app are still
using the older certificate authority. The Contrast Initializer terminates after the initial attestation
flow and won't pull new certificates on manifest updates.

To roll out the update, use:

```sh
kubectl rollout restart <resource>
```

for all your application resources.

### Updates for certificate rotation

As described above, a manifest update triggers rotation of the mesh CA certificate, the intermediate CA certificate and the workload certificates.
You can use this to force a certificate rotation or to constrain the certificate validity period.
Setting the current manifest once more causes a certificate rotation, without changing the reference values enforced by the Coordinator.

### Atomic manifest updates

Setting the manifest won't consider the previous state of the Coordinator.
This means that after a manifest update, you may have accidentally overwritten a previous Coordinator state set by another party.
To prevent this, use the `--atomic` flag:

```sh
contrast set -c "${coordinator}:1313" --atomic resources/
```

This will only update the manifest if the manifest history at the Coordinator matches the expected history.
When setting the manifest on an already initialized Coordinator, the latest transition hash has to be obtained by running `contrast verify`.
An atomic manifest update will then automatically read the hash from `verify/latest-transition`.
When setting the manifest for the first time, the expected transition hash is `00...00` (32 zero bytes, hex-encoded) and will be set automatically if the `verify/latest-transition` file doesn't exist.
Optionally, you can specify a transition hash using the `--latest-transition` flag:

```sh
contrast set -c "${coordinator}:1313" --atomic --latest-transition ab...cd resources/
```

### Signed manifest updates

Only authorized users with access to a trusted workload owner key can set a manifest at the Coordinator.
During a normal manifest update, the workload owner key is passed to the CLI and used directly in the TLS handshake with the Coordinator.
It's also possible to use externally managed keys, for example, in a hardware security module (HSM) or a cloud key management service (KMS).
In this case, the manifest update can be signed with the workload owner key and only the signature is needed to set the manifest.

The signature is generated over the manifest content and the latest transition hash, which is obtained from a previous `contrast verify`.
Use the `contrast sign` subcommand with the `--prepare` flag to get the blob that needs to be signed:

```sh
contrast sign --prepare --out next-transition
```

Then, sign the blob with the workload owner key and pass the signature to the CLI:

```sh
openssl dgst -sha256 -sign <workload-owner-key> -out transition.sig next-transition
contrast set -c "${coordinator}:1313" -s transition.sig resources/
```

If you have direct access to the workload owner key, you can also sign the manifest update using the CLI:

```sh
contrast sign --out transition.sig
contrast set -c "${coordinator}:1313" -s transition.sig resources/
```
