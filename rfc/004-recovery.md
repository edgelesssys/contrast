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

The overall idea of this proposal is to derive state deterministically from non-sensitive information and a secret seed.

![recovery flow](assets/004-recovery.drawio.svg)

### State Transitions

The immutable state of a Contrast deployment is the root CA certificate and key, which is derived from the secret seed.

The mutable state consists of the active manifest, a reference to the last state and the mesh CA certificate and keys.
There's a special state $S_0$ that represents an uninitialized Coordinator, which is the same for all Coordinators.

A state transition is initialized by calling the `SetManifest` endpoint with a new manifest and policies.
On update, a new mesh CA key is derived from the manifest history (including the new manifest) and the transition event is persisted.
States thus form a backwards linked list, which can be reconstructed from a list of state transitions, and the last state transition uniquely identifies the current state.
Therefore, we can replace the _last state_ reference with a reference to the _last state transition_.

The list of state transitions needs to be checked for integrity.
Otherwise, an attacker that can manipulate the transition objects can set arbitrary manifests.
Therefore, we sign each state transition with a key derived from the secret seed.

Thus, the final transition object is assembled like this:

```go
manifest := {
  content: manifest_bytes,
  ref: hash(manifest_bytes),
}
transition := {
  manifest: manifest.ref,
  prev: prev.ref, // or empty, which means S_0
  ref: hash(manifest.ref || prev.ref),
  sig: sign(hash(manifest.ref || prev.ref)),
}
```

### Security

We need to provide security for the two different trust models supported by Contrast: the one where the workload owner is trusted
by the data owner and allowed to update the manifest. In this scenario, a data owner will use the Coordinator root CA certificate
to verify the service certificate. In the second scenario, the data owner will verify any update to establish trust in the workload
and thus only trust the mesh CA certificate.

**Scenario 1:** The transaction signing key is derived from the secret seed of the workload owner. Transactions are chained together.
Through the signing, the transactions are authenticated and integrity protected with the signing key and thus with the seed secret
only know to the workload owner. Through the chaining, any manipulation or reordering of the manifest history is prevented.

**Scenario 2:** The mesh CA key is deterministically derived from the seed, the history of transactions and the active manifest.
After recovery, this key can be derived again given the secret seed and the history. In this scenario, the integrity of the history
isn't relevant to the data owner. The data owner still has the mesh CA certificate from before the restart. Given a correct history,
the key of the Coordinator will match the users cert. If history was tampered with, it won't. Any alternation of the manifest history
by either the workload owner or an attacker won't lead to the same public key and thus won't be trusted by the data owner.

### Cryptography

This proposal relies heavily on the idea of deterministic key generation.
Since the generated keys will be used for TLS, the choice of algorithms is limited to what's usually supported by TLS implementers.
The most restrictive applicable standard is the [Baseline Requirements](https://cabforum.org/uploads/CA-Browser-Forum-BR-v2.0.0.pdf) for browsers, which require TLS certificates to use ECDSA or RSA keys.
Although the Go standard library doesn't support deterministic generation of these key types, standards-based alternatives are available: <https://pkg.go.dev/filippo.io/keygen#ECDSA>.

The transition signatures don't need to be deterministic, so we can derive an ECDSA key from the seed to sign transitions.

### Persistent State

There are basically two options for persistent state in a Kubernetes cluster: persistent volumes and Kubernetes objects.
We propose using persistent volumes because they require less business logic in the Coordinator.
The [appendix contains an alternative proposal](#kubernetes-objects) and suggestions for supporting both flavours.

As of 2024-05-07, persistent volumes on AKS CoCo are only supported in [`volumeMode: block`](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#raw-block-volume-support).
This is sufficient for our use case - we can set up a filesystem when we need one.
The upside of using a block device is that we can easily set it up as a LUKS volume after generating the seed.
This allows us to keep the content of the manifests secret, which isn't critical but still desirable.

### Recovery

On startup, the Coordinator checks whether the block device contains a LUKS header.
If a header is found, it enters into recovery mode, otherwise into normal mode.

#### Recovery Mode

While in _recovery mode_, all calls to `SetManifest` are rejected.
This prevents accidental or malicious overwrites of the Coordinator's storage.

Instead, the Coordinator waits for calls to a newly added `Recover` method.
This method accepts the seed and tries to mount the encrypted volume.
If this is successful, it starts the recovery process.

The Coordinator builds an in-memory tree of transactions loaded from persistence, with a common root at $S_0$ (the empty predecessor).
It checks the leaves for matching signatures and, on the first match, applies the transitions from $S_0$ to that leaf.
Note that there should only be one chain, but the storage API can't enforce that.
Finally, it enters _normal mode_.

#### Normal Mode

While in _normal mode_, all calls to `Recover` are rejected.
The Coordinator waits for the first call to set, which generates the secret seed.
During that call, it sets up the volume, filesystem and directory structure.

## Future Considerations

### Secret Sharing

The manifest could be modified to include a recovery threshold parameter.
If the threshold `n` is greater than 1, the response includes secret shares instead of the full secret.
Recovery would need to be called `n` times with different shares.
The threshold would be stored in plain text on the `Contrast` object.
This is fully backwards compatible with the main proposal.

### KMS Recovery

Any KMS built for Contrast would require a bootstrap secret, similar to a [Vault sealing key](https://developer.hashicorp.com/vault/docs/concepts/seal).
Such a key could be derived from the secret seed, bound to the policy of the KMS.
Recovery of the KMS could then simply be a step in the initialization process.
This scheme could be directly applied to user workloads, too (e.g. encrypted persistent storage).

### Distributed Coordinator & Updates

If there is more than one Coordinator instance, the recovery process could be automatic.
After entering _recovery mode_, the Coordinator could request the secret seed from a Coordinator in _normal mode_, subject to successful attestation.
State transitions would need to be announced to all Coordinators in order to update their persistency.

An updated Coordinator instance would start in _recovery mode_ like a restarted instance, and manual recovery would work exactly the same.
Automatic recovery of updated Coordinators is undesirable due to the change in attestation evidence.

## Appendix

### Persistent Volume Layout

```txt
.
├── manifests
│   └── 98e5da0c56eedb63ed9be454c6398c4c209be84adb7e0abfe2d1ca2a4f95b73d
│       └── manifest.json
├── policies
│   └── 0515b8248a3d44e38e959e2b1fb2b213a2cd35b5186bba84562bc4e51298712f
│       └── policy.rego
└── transitions
    └── 8bb693aaa143ee0cf97f41d98a22b4d999a46f8eb8103f4fbbb79cb52a0b28ba
        ├── manifest.sha256
        ├── previous.sha256
        └── transition.sig
...
```

### Kubernetes Objects

Storing state in Kubernetes objects is convenient because it doesn't require additional cloud resources.
However, there is a limit to the amounts of data that a single object can hold, usually on the order of 1MiB.
Given the average size of a policy being 50kiB, it would be necessary to split the state to support Contrast deployments of modest size.
A natural way to split the state might look like this:

- A content-addressable `Policy` resource, where the name is the SHA256 sum of the content.
- A content-addressable `Manifest` resource, which refers to a set of policies (among other manifest content).
- A content-addressable `Transition` resource.

Although the initial focus should be on persistent volumes, we can design a persistency abstraction that works with both backends.
The common denominator would be a key-value store interface with multi-part keys.
The first part of the key corresponds to a Kubernetes resource or a top-level directory, respectively.
The second part is the object name, under which we store the relevant content.

Example CRDs:

```yaml
apiVersion: contrast.edgeless.systems/v1
kind: Policy
metadata:
  name: 0515b8248a3d44e38e959e2b1fb2b213a2cd35b5186bba84562bc4e51298712f
spec:
  policy.rego: |
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
  manifest.json: |
    {
      "policies": { "0515b8248a3d44e38e959e2b1fb2b213a2cd35b5186bba84562bc4e51298712f": ["my-deployment"] },
      "referenceValues": ...,
      "workloadOwnerKeyDigests": ...
    }
---
apiVersion: contrast.edgeless.systems/v1
kind: Transition
metadata:
  name: 8bb693aaa143ee0cf97f41d98a22b4d999a46f8eb8103f4fbbb79cb52a0b28ba
spec:
  prevRef: 1f2606ecd68d6405e0e94f4ee5834a33e6b3696c29637cab5832dd23f5ec424a
  manifestRef: 98e5da0c56eedb63ed9be454c6398c4c209be84adb7e0abfe2d1ca2a4f95b73d
  signature: ...
```
