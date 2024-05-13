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

The design should accommodate a future extension allowing for multi-party recovery.

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

#### Persistent Volume

Use of persistent volumes for CoCo isn't homogeneous: AKS supports a few choices, while the upstream project didn't settle for an approach.
It's unlikely that we can come up with a `PersistentVolumeClaim` or `VolumeClaimTemplate` that works everywhere without workload owner interaction.
On the other hand, managing this persistency would be almost trivial from the Coordinator's point of view.

**Note**: As of 2024-05-07, persistent volumes aren't supported on AKS CoCo.
Recent changes on `msft-main` indicate that they intend to add support, but it's not clear when.

#### Kubernetes Objects

Storing state in Kubernetes objects is convenient because it doesn't require additional cloud resources.
However, there is a limit to the amounts of data that a single object can hold, usually on the order of 1MiB.
Given the average size of a policy being 50kiB, it would be necessary to split the state to support Contrast deployments of modest size.
A natural way to split the state might look like this:

- A content-addressable `Policy` resource, where the name is the SHA256 sum of the content.
- A content-addressable `Manifest` resource, which refers to a set of policies (among other manifest content).
- A `Contrast` resource, which refers to an ordered list of manifest digests and holds certificates and keys.
  This resource would need to be encrypted with authentication.

These resources would need to be managed consistently by the Coordinator.
See the [Appendix](#kubernetes-object-example) for how these objects might look like.

### Secret Management

- At the first call to `contrast set`, the coordinator creates a recovery secret.
- The `SetManifestResponse` includes the recovery secret, encrypted with the workload owner public keys.

### Recovery

- Add `Recover` method to the user API with the recovery secret in the request.
- At startup, the coordinator checks the persistence layer for an existing resource matching its name.
- If resources present, it waits for a call to `Recover`.
-
![recovery flow](assets/004-recovery.drawio.svg)

## Future Considerations

### Secret Sharing

The `SetManifestRequest` could be modified to include a recovery threshold parameter.
If the threshold `n` is greater than 1, the response includes secret shares instead of the full secret.
Recovery would need to be called `n` times with different shares.
The threshold would be stored in plain text on the `Contrast` object.
This is fully backwards compatible with the main proposal.

### KMS Recovery

TODO

### Distributed Coordinator Updates

TODO

## Appendix

### Kubernetes Object Example

```yaml
apiVersion: contrast.edgeless.systems/v1
kind: Policy
metadata:
  name: 0515b8248a3d44e38e959e2b1fb2b213a2cd35b5186bba84562bc4e51298712f
spec:
  content: |
    package agent_policy
    default AllowRequestsFailingPolicy := false
    CreateContainerRequest {
    ... more rego ...
---
apiVersion: contrast.edgeless.systems/v1
kind: Manifest
metadata:
  name: 98e5da0c56eedb63ed9be454c6398c4c209be84adb7e0abfe2d1ca2a4f95b73d
spec:
  content: |
    {
      "policies": { "0515b8248a3d44e38e959e2b1fb2b213a2cd35b5186bba84562bc4e51298712f": ["my-deployment"] },
      "referenceValues": ...,
      "workloadOwnerKeyDigests": ...
    }
---
apiVersion: contrast.edgeless.systems/v1
kind: Contrast
metadata:
  name: coordinator-deployment-name
spec:
  nonce: b4231840b79a4adecb81719d
  state: |
    aes_gcm(
      {
        "manifests": [ "98e5da0c56eedb63ed9be454c6398c4c209be84adb7e0abfe2d1ca2a4f95b73d" ],
        "root-ca": {"cert": "...", "key": ...},
        "mesh-ca": {"cert": "...", "key": ...}
      },
      nonce="b4231840b79a4adecb81719d",
      ad="coordinator-deployment-name"
    )
```
