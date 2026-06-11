# Secret Sharing

## Background

During an initial `contrast set` the secret seed is encrypted with each seed share owner's public key and returned to the user.
This means that each seed share owner has access to the complete seed, and thus the ability to recover the Coordinator in case of a restart.
The original idea was to use a secret sharing scheme to split the seed into multiple shares, and require a threshold of them to recover the seed.

This would have the advantage of not giving any single seed share owner access to the complete seed.
With access to the complete seed, the seed share owner could compute the workload secrets and decrypt any persistent encrypted mounts.
In a scenario with multiple seed share owners, this means secure persistency isn't possible if the data owner doesn't trust all seed share owners.
With only parts of the seed accessible to each seed share owner, this is no longer the case.

It also comes with the drawback that the seed share owners would need to coordinate to recover the seed, which adds complexity to the recovery process.

## Proposal

We implement a secret sharing scheme for the seed, and each seed share owner only has an encrypted seed share.
To recover the Coordinator, a threshold of seed shares need to be combined to recover the seed.
This is done by each seed share owner calling `contrast recover` with their encrypted seed share, and the Coordinator combines the shares and recovers the seed if the threshold is met.

This approach only works if there is only one Coordinator instance, since `contrast recover` only connects to one Coordinator instance.
If there are multiple instances, the seed shares would need to be distributed to all of them.
In the current peer recovery process, Coordinators that enter a stale state call the `Recover` RPC of the mesh API of other Coordinators to recover.
In order to distribute the seed shares to all instances however, we need an RPC in the opposite direction, where the Coordinator in recovery can send the received seed shares to the other Coordinators.

When implementing this, we need to additionally consider concurrent recovery attempts on different Coordinator instances.
If 2 seed shares remain to meet the threshold and 2 different seed share owners call `contrast recover` at the same time, both Coordinators have to update each other's state without running into an error.

For now, we can say the solution is to scale down the Coordinator to a single instance.

### Manifest Changes

In the manifest, we add an additional field `SeedShareOwnerThreshold` which specifies the number of seed shares that need to be combined to recover the seed:

```json
{
    ...
    "SeedShareOwnerThreshold": 3,
    "SeedShareOwnerPubKeys": [ ... ]
}
```

This threshold defaults to the number of provided seed share owner public keys, so by default every seed share owner would need to be involved in the recovery process.
The lowest value would be 1, in which case the behavior is the same as the current implementation, allowing every seed share owner to recover the Coordinator on their own.
Like the seed share owner public keys, this field can't be updated after the initial `contrast set`.

### UserAPI Changes

For the implementation we can use a secret sharing scheme based on [HashiCorp's](https://github.com/hashicorp/vault/tree/8d07273d14ae7f5a48cc96f66cc86615dea83390/shamir) or [OpenBao's](https://pkg.go.dev/github.com/openbao/openbao/sdk/v2/helper/shamir) Shamir go library.
During the initial `contrast set`, the seed is split into shares using the specified threshold and the provided seed share owner public keys, and each share is encrypted with the corresponding public key and returned to the user.

For recovery, the Coordinator needs to track the received seed shares and combine them when the threshold is met.
For this, we add a new field to the `State` struct:

```go
type State struct {
    seedEngine    *seedengine.SeedEngine
    ...

    partialRecovery *PartialRecovery // new field to track the received seed shares
    ...

    stale atomic.Bool
}

type PartialRecovery struct {
    Salt       []byte                        // captured from first submitter
    SeedShares map[manifest.HexString][]byte // keyed by submitter public key
}
```

If the Coordinator is in recovery mode, the new `PartialRecovery` struct is used to track the received salt and seed shares via the `Recover` RPC of the user API.
Peer recovery still works as before, since there are no seed shares involved.
If a `Recover` call from the user is made where the newly received seed share combined with the already received shares meets the threshold, the seed is reconstructed and the Coordinator can be recovered normally.
If the list contains some number of seed shares below the threshold and peer recovery triggers, the Coordinator is recovered via peer recovery as expected.

When the `Recover` RPC is called and the state is stale, the `ResetState` method of the guard is invoked, which takes a `SecretSourceAuthorizer` as an argument.
This interface currently only has the method `AuthorizeByManifest`, which validates the peer given the manifest from the store and returns the new seed engine and mesh CA key.
This is used both by the user API for `contrast recover` and by the mesh API during peer recovery.
The `AuthorizeByManifest` function now additionally returns the reconstructed seed if the threshold is met or the received seed share otherwise:

```go
type SecretSourceAuthorizer interface {
    AuthorizeByManifest(ctx context.Context, manifest *manifest.Manifest) (*seedengine.SeedEngine, *crypto.PublicKey, *PartialRecovery, error)
}
```

In case of peer recovery, the third output is always `nil` and the process stays the same as before.
For user recovery, the `SecretSourceAuthorizer` is initialized with the state's current seed shares and, if there are enough shares, reconstructs the seed and returns the seed engine with the recovered seed as before.
If there aren't enough shares yet, we only return the received seed share from the function without the seed engine and mesh key.
The function should already verify that the received seed share isn't present in the list of received seed shares.
It should also verify that the salt provided with the seed share matches the salt from the first received seed share, if there is one.

If the returned seed engine is `nil` and we get a seed share that isn't already in the list of received seed shares, we return a new state.
This state has the `stale` boolean still set to `true` but now includes the new `*PartialRecovery` object with the updated seed shares.
After updating the state, we return an error indicating that the threshold isn't met yet.
If we get a valid seed engine, we proceed as before.

If the Coordinator has enough seed shares to recover the seed but one of the seed shares is invalid, through, for example, a malicious seed share owner, the following signature verification of the latest transition will fail.
In that case, we don't know which seed shares are valid or invalid, so the only option is to reset the state and start the recovery process from scratch with no seed shares.

The proto messages for the `Recover` API remain basically unchanged, now including the seed share instead of the full seed:

```proto
message RecoverRequest {
    bytes SeedShare = 1; // <-- now a seed share instead of the full seed
    bytes Salt = 2;
    bool Force = 3;
}
```

In the `SeedShareDocument` returned by the `SetManifest` API, we now return the encrypted seed shares instead of the encrypted seed:

```proto
message SeedShareDocument {
  repeated SeedShare SeedShares = 1;
  bytes salt = 2;
}

message SeedShare {
  string PublicKey = 1;
  bytes EncryptedSeedShare = 2; // <-- now an encrypted seed share instead of the full encrypted seed
}
```

## Security Considerations

This proposal introduces the possibility of malicious or only partially trusted seed share owners, which wasn't documented in the previous threat model.
During recovery, such a participant can submit invalid shares and delay or prevent Coordinator recovery, but can't compromise confidentiality.
In a 1-n threshold scheme, which was the default behavior until now, a malicious seed share owner can calculate workload secrets and decrypt any persistent encrypted mounts.
With this proposal, secure persistency is possible in multi-party Contrast deployments without requiring seed share owners to fully trust each other.
