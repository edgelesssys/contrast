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

For now, we can say the solution is to scale down the Coordinator to a single instance and document this.

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
If the field is absent, this default value is used.
The lowest value would be 1, in which case the behavior is the same as the current implementation, allowing every seed share owner to recover the Coordinator on their own.
Only with a threshold of 2 or more, we calculate the seed shares from a secret sharing scheme.
If present, this field must be validated by the CLI when first reading in the manifest.
Like the seed share owner public keys, this field can't be updated after the initial `contrast set`.

### UserAPI Changes

For the implementation we can use a secret sharing scheme based on [HashiCorp's](https://github.com/hashicorp/vault/tree/8d07273d14ae7f5a48cc96f66cc86615dea83390/shamir) or [OpenBao's](https://pkg.go.dev/github.com/openbao/openbao/sdk/v2/helper/shamir) Shamir go library.
During the initial `contrast set`, the seed is split into shares using the specified threshold and the provided seed share owner public keys, and each share is encrypted with the corresponding public key and returned to the user.
If the threshold is 1, the entire seed is encrypted with each seed share owner's public key and returned to the user, as before.

For recovery, the Coordinator needs to track the received seed shares and combine them when the threshold is met.
For this, we add a new field to the UserAPI server:

```go
type Server struct {
    logger          *slog.Logger
    guard           guard
    discovery       discovery

    partialMu       sync.Mutex
    partialRecovery *partialRecovery // new field to track received seed shares during recovery

    userapi.UnimplementedUserAPIServer
}

type partialRecovery struct {
    transitionHash [history.HashSize]byte        // latest transition at first submission
    salt           []byte                        // captured from first submitter
    seedShares     map[manifest.HexString][]byte // keyed by submitter public key
}
```

If the Coordinator is in recovery mode, the new `partialRecovery` struct is used to track the received salt and seed shares via the `Recover` RPC of the UserAPI.
The `transitionHash` is captured from the first submission and used to verify that the latest transition didn't change during the recovery process.
Peer recovery still works as before, since there are no seed shares involved.
If a `Recover` call from the user is made where the newly received seed share combined with the already received shares meets the threshold, the seed is reconstructed and the Coordinator can be recovered normally.
If the list contains some number of seed shares below the threshold and peer recovery triggers, the Coordinator is recovered via peer recovery as expected.
Note that peer recovery is still preferred over user recovery if there is more than one Coordinator instance, and `--force` must be specified to submit shares in that case.
Since the `partialRecovery` struct is tied to the Coordinator pod's lifetime, the recovery process must be completed without the pod being restarted, otherwise the process has to be restarted as well.

When the `Recover` RPC is called and the state is stale, the server adds the seed to the `partialRecovery` struct.
This should already verify that the request contains the same salt as in `partialRecovery`.
The `SecretSourceAuthorizer` is then initialized with the current seed shares and passed to `ResetState`, with the struct now looking like this:

```go
type seedAuthorizer struct {
    seedShares map[manifest.HexString][]byte
}
```

The `AuthorizeByManifest` method verifies the origin of the seed share, checks if the threshold is met, and if so, reconstructs the seed engine.
Otherwise, return an error indicating that the threshold isn't met yet and propagate this to the user.
If the seed share owner's public key isn't authorized by the manifest, also return a specific error.
The UserAPI can then delete the invalid seed share from the `partialRecovery` struct and return an error to the user.

If we construct a seed engine but the signature verification of the latest transition fails, this means one or more seed shares are invalid, sent, for example, by a malicious seed share owner.
Since we don't know which seed shares are valid or invalid, we return an error and reset the `partialRecovery` struct in the UserAPI.
If we get a valid seed engine, we proceed as before.
Upon success, the `partialRecovery` struct is reset to `nil` for future recovery attempts.

For concurrent recovery attempts, the easiest option may be to lock the whole recovery process, so that only one recovery attempt can be processed at a time.
With a relatively low number of seed share owners that manually call `contrast recover`, this shouldn't cause too many problems.
If we want concurrency, we must consider cases where:
- multiple seed share owners call `contrast recover` with the threshold being reached/not reached,
- multiple seed share owners call `contrast recover` with at least one invalid share,
- the same seed share owner calls `contrast recover` with different seed shares, overwriting the map entry in `partialRecovery`.

We could also consider overwriting the seed shares in `partialRecovery` only after `ResetState` returns with an error, indicating that the threshold isn't met.
This would prevent the map being populated with invalid shares, but would mean that concurrent recovery attempts with the threshold being reached would fail, since each attempt would only see the initial seed shares from the UserAPI without the new shares from the concurrent attempts.

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
