# Coordinator Recovery

## Problem

Restart currently causes loss of

- CA root & mesh key
- Active manifest
- Manifest & policy history

## Requirements

After a full restart of the coordinator, the workload owner needs to be able to recover the coordinator such that:

- A data owner calling `contrast verify` receives the same information they did before the restart.
- A workload certificate issued by the coordinator after the restart verifies with the bundles issued before the restart, and vice-versa.

If workload owner and data owner are not mutually trusting, it should not be possible for the workload owner to recover the active mesh CA private key outside a verified coordinator.

## Design

A couple of necessary conditions arise immediately from the requirements:

- There needs to be persistent state for storing
  - the root CA key and certificate,
  - the active mesh CA key and certificate,
  - the active manifest and
  - the history of manifests.
- This state is (partially) secret and needs to be authenticated, thus requiring a persistent secret.
- As we assume a full loss of the coordinator's internal state, the recovery secret must be kept elsewhere.

### State

There are basically two options for persistent state in a Kubernetes cluster: persistent volumes and Kubernetes objects.

Use of persistent volumes for CoCo is not homogenous: AKS supports a few choices, while the upstream project did not settle for an approach.
It's unlikely that we can come up with a `PersistentVolumeClaim` or `VolumeClaimTemplate` that works everywhere without workload owner interaction.
On the other hand, managing this persistency would be almost trivial from the Coordinator's point of view.

Storing state in Kubernetes objects is convenient because it does not require additional cloud resources.
However, there is a limit to the amounts of data that a single object can hold, usually on the order of 1MiB.
Given the average size of a policy being 50kiB, it would be necessary to split the state to support Contrast deployments of modest size.
A natural way to split the state might look like this:

- A content-addressable `Policy` resource, where the name is the SHA256 sum of the content.
- A content-addressable `Manifest` resource, which refers to a set of policies (among other manifest content).
- A `Contrast` resource, which refers to an ordered list of manifest digests and holds certificates and keys.
  This resource would need to be encrypted with authentication.

These resources would need to be managed consistently by the Coordinator.

### Secret Management

- At the first call to `contrast set`, the coordinator creates a recovery secret.
- The secret is split into a number of key-shares equal to the number of workload owner keys in `SetManifestRequest`.
- The `SetManifestResponse` includes the key-shares, each encrypted with one of the workload owner public keys.
- The caller is responsible to distribute these shares to the key owners.

Possible extensions:

- Configuring a threshold to require `m` out of `n` key shares.
- Adding or removing shareholders should rekey.

### Recovery

- Add `Recover` method to the user API.
- At startup, the coordinator checks the persistence layer for existing resources.
- If resources present, it waits for `m` calls to `Recover`.

## Future Considerations

### KMS Recovery

### Distributed Coordinator Updates

- Must be agreed upon in secret sharing mode
