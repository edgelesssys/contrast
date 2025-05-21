# Manifest update

This section guides you through the process of updating a manifest at the Contrast Coordinator.

## Applicability

If the manifest changes—for example, due to modifications in the service mesh configuration—it must be updated at the Coordinator.

## Prerequisites

1. A running Contrast deployment
2. [Connect to the Coordinator](./workload-deployment/deploy-coordinator#connect-to-the-contrast-coordinator)

## How-to


Set the changed manifest at the Coordinator with:

```sh
contrast set -c "${coordinator}:1313" deployment/
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
