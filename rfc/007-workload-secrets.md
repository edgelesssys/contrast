# RFC 007: Workload secrets

## Objective

Provide Contrast workloads with secret key material, such that:

1. Workloads can use key material to derive additional secrets autonomously.
2. The key material is stable across workload upgrades.
3. The key material is stable across manifest generations.
4. The key material is stable across Contrast upgrades.

## Problem statement

Due to the verification mechanism, Contrast workloads are naturally stateless.
Unless the container images are encrypted, there is no secure way to embed a
persistent secret.

However, many applications need persistence of some sort. Since Contrast
workloads usually process sensitive data, the data they persist is often going
to be sensitive, too. As of today, the only option for persistence is a trusted
external service that validates mesh certificates, presenting a bootstrapping
problem that Contrast is designed to avoid. If workloads somehow had access to a
stable secret, they could encrypt their sensitive data before sending it to an
untrusted storage provider.

A concrete use case would be to host a [Hashicorp Vault] (or [OpenBao])
instance. Vault has a concept of [storage sealing] to protect its secret store.
In a nutshell, Vault encrypts its secrets with a sealing key. The Vault process
starts in sealed mode, unable to access its own persistence. After being
provisioned with the unsealing key it can resume normal operation. When running
Vault as a Contrast workload, access to a persistent secret would allow
automatic unsealing without operator intervention.

[Hashicorp Vault]: https://www.hashicorp.com/products/vault
[OpenBao]: https://openbao.org/
[storage sealing]: https://developer.hashicorp.com/vault/docs/concepts/seal

## Proposal

### Secret source

Workload secrets would need to be either persisted or deterministically derived.
The only secure persistence option for the Coordinator is encryption with a key
derived from the secret seed (see [RFC 004](004-recovery.md)). Since the
security and access guarantees of such a scheme are directly tied to the seed,
deriving the workload secret directly from the seed is the easier option, and
more reliable.

The derived key should be large enough to allow secure derivation of common
asymmetric key material. Recommended[^1] for P-512 would be 48 bytes, and a
secret key for X448 also needs 48 bytes. Since the maximum entropy is limited by
the Coordinator's secret seed size, derive a key of the same size. Increasing
the entropy of the Coordinator seed should then be considered separately.

[^1]: https://pkg.go.dev/filippo.io/keygen#ECDSA

### Workload identity

Access to workload secrets should be tied to a _workload identity_. This
identity needs to be stable for at least all pods sharing a runtime policy,
because that's the finest-grained identity Contrast can verify. However, runtime
policies aren't stable enough to satisfy the outlined conditions. For example,
policy rules can change between Contrast versions, or when workload parameters
change. Therefore, the identity can't be the policy hash, and similar arguments
hold for other Contrast-managed identity schemes.

Instead of deriving the identity from workload characteristics, users are going
to label workloads with an identity. To allow that, we're' going to change the
`Manifest` schema. The policies field of the manifest is currently a plain map
of policy hashes to SANs.

```json
{
  "Policies": {
    "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": [
      "web",
      "*",
      "203.0.113.34"
    ]
  }
}
```

Changing this to an object with named fields allows adding new metadata, for
example a `WorkloadSecretID`.

```json
{
  "Policies": {
    "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": {
      "SAN": ["web", "*", "203.0.113.34"],
      "WorkloadSecretID": "openbao-prod"
      // ...
    }
  }
}
```

### Implementation

After successful workload verification, the Coordinator derives a key with HKDF,
setting the info argument to `workload-key:$WorkloadSecretID`. This key is
returned as a new field `WorkloadKey` in the `NewMeshCertResponse`. The
initializer writes the key to a file in the shared tmpfs, from where it can be
used by the workload.

## Considerations

### Extension to encrypted storage

A persistent workload key could be used to set up transparently encrypted block
storage. Due to the required permissions involved, and Kubernetes' volume
sharing limitations, this would best be handled by a CVM-level service (a guest
component in CoCo terms).

### Manifest updates and workload owner trust

Due to being derived from the secret seed, the workload key is stable across
manifest updates. This implies that no migration is needed, as long as the
`WorkloadSecretID` doesn't change. On the other hand, this means that the
workload secret can be obtained by the workload owner, and should be treated
accordingly.

### Contrast upgrades

The stability of the workload secret is directly tied to the stability of the
Coordinator secret. Since we intend to keep the Coordinator's key derivation
stable (for recovery), workload secrets will remain stable, too.
