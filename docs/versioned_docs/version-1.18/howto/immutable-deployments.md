# Immutable deployments with Contrast

This section explains how to setup a Contrast deployment that can't be changed after the initial setup.

## Applicability

In multi-party setups with (mutually) distrusting parties, ensuring that no party can alter the deployment or decrypt state can be desirable.

## Prerequisites

1. [Set up cluster](./cluster-setup/bare-metal.md)
2. [Install CLI](./install-cli.md)
3. [Deploy the Contrast runtime](./workload-deployment/runtime-deployment.md)
4. [Add Coordinator to resources](./workload-deployment/add-coordinator.md)
5. [Prepare deployment files](./workload-deployment/deployment-file-preparation.md)
6. [Generate annotations and manifest](./workload-deployment/generate-annotations.md)

## How-To

After running the [`contrast generate`](./workload-deployment/generate-annotations.md) command, the manifest contains fields `WorkloadOwnerKeyDigest` and `SeedshareOwnerPubKeys`.
Deleting these fields before calling [`contrast set`](./workload-deployment/set-manifest.md) for the first time will render the deployment immutable and make Coordinator recovery impossible.

### Preventing manifest updates

To prevent manifest updates, delete the `WorkloadOwnerKeyDigest` before calling `contrast set` for the first time.
While the `WorkloadOwnerKeyDigest` can be removed at a later point, it should ideally never be `set` for immutable deployments.
After removal of the key, [set the manifest](./workload-deployment/set-manifest.md) through `contrast set`.
Once completed, the Contrast Coordinator will no longer accept the workload owner key for future manifest changes.
The seed share owner can still force manifest changes through the [recovery mechanism](./workload-deployment/recover-coordinator.md), if they're able to access to the Coordinator's `ConfigMap`s.

Alternatively, specify the `--disable-updates` flag in the `generate` step for the same result.

```sh
contrast generate --reference-values <platform> --disable-updates resources/
```

:::warning

This action is irreversible.
You won't be able to make changes to your deployment after running `contrast set` with an empty `WorkloadOwnerKeyDigest`.

:::

### Preventing decryption of state and recovery

Deleting the `SeedshareOwnerPubKeys` field from the manifest before calling `contrast set` for the first time will make it impossible to recover both the Contrast Coordinator and any state associated with a deployment.
Note that deletion of the `SeedshareOwnerPubKeys` needs to occur before the first use of `contrast set`, since setting the manifest will fail if this key has been changed.
Normally, the Coordinator [can be recovered after a restart](./workload-deployment/recover-coordinator.md).
With an empty list of `SeedshareOwnerPubKeys`, this becomes impossible because no keys will be accepted.
This also preempts [workload secret recovery](../architecture/secrets.md#workload-secrets), and thus makes it impossible to re-open [encrypted persistent storage](./encrypted-storage.md) after a Coordinator restart.

:::warning

This action is irreversible.
You won't be able to recover the Contrast Coordinator after a restart.
Further, any associated state will also be irrecoverable.
Removing the `SeedshareOwnerPubKeys` effectively turns a (persistent) deployment into an ephemeral one.

**It's highly unlikely that you want to use this feature in conjunction with the encrypted persistent storage feature.**

:::

### Verifying immutability

[Deployment verification](./workload-deployment/deployment-verification.md) works as usual.
Verifying that no manifest in the manifest history contained workload owner keys or seed share owner keys guarantees that no party can update and/or recover the deployment and its state.
