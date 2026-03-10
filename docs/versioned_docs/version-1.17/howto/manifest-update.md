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
