# Secret Sharing

## Background

During an initial `contrast set` the secret seed is encrypted with each seed share owner's public key and returned to the user.
This means that each seed share owner has access to the complete seed, and thus the ability to recover the Coordinator in case of a restart.
The original idea was to use a secret sharing scheme to split the seed into multiple shares, and require a threshold of them to recover the seed.
This would have the advantage of not giving any single seed share owner access to the complete seed.
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

For the implementation we can use a secret sharing scheme based on OpenBao's [Shamir](https://pkg.go.dev/github.com/openbao/openbao/sdk/v2/helper/shamir) go library.
During the initial `contrast set`, the seed is split into shares using the specified threshold and the provided seed share owner public keys, and each share is encrypted with the corresponding public key and returned to the user.

For recovery, the Coordinator needs to track the received seed shares and combine them when the threshold is met.
For this, we add a new field to the `State` struct:

```go
type State struct {
    seedEngine    *seedengine.SeedEngine
    ...

    seedShares [][]byte // new field to track the received seed shares
    ...

    stale atomic.Bool
}
```

If the Coordinator is in recovery mode, this list is used to track the received seed shares via the `Recover` RPC of the user API.
Peer recovery still works as before, since there are no seed shares involved.
If a `Recover` call from the user is made where the newly received seed share combined with the already received shares meets the threshold, the seed is reconstructed and the Coordinator can be recovered normally.
If the list contains some number of seed shares below the threshold and peer recovery triggers, the Coordinator is recovered via peer recovery as expected.

When the `Recover` RPC is called and the state is stale, the `ResetState` method of the guard is invoked, which takes a `SecretSourceAuthorizer` as an argument.
This interface currently only has the method `AuthorizeByManifest`, which validates the peer given the manifest from the store and returns the new seed engine and mesh CA key.
This is used both by the user API for `contrast recover` and by the mesh API during peer recovery.
The `AuthorizeByManifest` function can now additionally take the seed shares as an argument to reconstruct the seed if the threshold is met and return the received seed share otherwise:

```go
type SecretSourceAuthorizer interface {
    AuthorizeByManifest(ctx context.Context, manifest *manifest.Manifest, seedShares [][]byte) (seedengine.SeedEngine, crypto.PublicKey, []byte, error)
}
```

In case of peer recovery, the seed shares aren't relevant, so the mesh API can just ignore the additional input to `AuthorizeByManifest`.
For user recovery, if there are enough shares, we reconstruct the seed and return the seed engine with the recovered seed as before.
If there aren't enough shares yet, we only return the received seed share from the function without the seed engine and mesh key.
The function should already verify that the received seed share isn't present in the list of received seed shares.

If the returned seed engine is `nil` and we get a seed share that isn't already in the list of received seed shares, we return a new state.
This state has the `stale` bool still set to `true` but now includes the list of received seed shares from before plus the new received seed share.
After updating the state, we return an error indicating that the threshold isn't met yet.
If we get a valid seed engine, we proceed as before.

The proto messages for the `Recover` API remain basically unchanged, now including the encrypted seed share instead of the encrypted seed:

```proto
message SeedShare {
  string PublicKey = 1;
  bytes EncryptedSeedShare = 2;
}
```
