# RFC 002: Secure SetManifest endpoint

The SetManifest endpoint should only be accessible by the entity that first
called SetManifest.

## The problem

The manifest enforced by the coordinator can be updated. This will become
increasingly relevant once stateful applications are supported by CoCo upstream.

The proposed solution follows a trust on first use (TOFU) model, where the
initial call to the SetManifest endpoint isn't protected with the same
mechanism. Protecting the initial access isn't in scope for this document,
although it may still be useful in practice to protect this endpoint using other
methods, such as firewall rules, password authentication or other means. The
secure SetManifest endpoint should thus allow any entity to set the initial
manifest and only allow the same entity to update the manifest.

## State machine

The coordinator can be in one of three states:

- It starts in the initial state, where no manifest is set.
- The updatable state is reached after SetManifest was called at least once and
  the manifest contains at least one workload owner key.
- The final state is reached after SetManifest was called with an empty list of
  public keys.

## Key exchange

The coordinator trusts a list of x509 key pairs that belong to the entity that
first called the SetManifest endpoint. The key pairs are called workload owner
keys. A workload owner key must not be signed by the root CA / mesh CA so there
is no potential for impersonation between the workload owner and mesh
certificates.

The client sends a list of trusted public keys as part of manifest in the
initial call. In the default case, a single public key is used, allowing the
owner of the secret key to update the manifest. If the list is empty, nobody can
update the manifest after the initial call and the client doesn't need to store
a secret key on disk. Multiple keys can be specified to allow multiple parties
to perform updates. This can later be extended as described below.

The list of trusted public keys must always be included in the manifest. This
way, the data owner can validate who is allowed to update the manifest and the
trusted public keys can be rotated (or any further updates prevented) as part of
the manifest.

Another consideration is whether the same grpc endpoint should be used in both
states (initial, updating) or if a split's beneficial. This is merely an
implementation detail and not important for the overall design. We will attempt
to reuse the same endpoint.

## Enforcement of mutual TLS with client authentication on manifest updates

In the initial state, TLS client authentication isn't enforced. This allows any
client to establish a TLS connection and set a manifest. In the updatable state,
the coordinator enforces TLS client authentication on the SetManifest endpoint.
This architecture could be reused for future authenticated grpc endpoints.

The mechanism for performing cert validation has to account for the fact that
some grpc endpoints require client authentication, while others don't. In
particular, this requires any certificate validation to be performed after the
grpc endpoint is known (after the TLS handshake). If this is infeasible,
multiple listeners may be used to perform validation during the handshake
instead.

## Possible future extensions

In the future, it may be desirable to implement a fully fledged user and role
system with role-based access control (RBAC). X509 certificates could be used to
authenticate users. This can be combined with a required quota for manifest
updates for each role. Users, role assignments, user certificates and quotas
should all be part of the manifest. With this extension, each user can call
SetManifest independent of each other. For each user that calls the endpoint
with the same manifest, the coordinator remembers that the user accepted the
update. Once a quota has been reached (the specified number of users in each
role have accepted the update), the coordinator actually performs the update.
This extension is required to enable safe transitions that migrate encrypted
state between manifest versions. A similar mechanism exists today in
MarbleRun[1].

[1] <https://docs.edgeless.systems/marblerun/workflows/define-manifest#roles>

## Alternatives considered: Signed manifest updates

With this solution, the manifest data used when calling the SetManifest endpoint
for updates must be signed by a trusted key and the signature must be sent
together with the manifest data. TLS client authentication isn't used. The same
ways to establish trusted signing keys apply. As a positive side effect, the
signed manifest could be handed to the data owner to proof that the workload
owner performed the SetManifest call. As a downside, the signed manifest could
be replayed by an attacker if the signature doesn't incorporate replay
protection. Such a protection could be implemented using a challenge/response
mechanism, by rejecting already seen manifest signatures (and using a nonce), or
by adding a revision counter.
