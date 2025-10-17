# Immutable deployments with Contrast

This section provides guidance on how to setup an immutable deployment with Contrast, that is, a deployment which can't be changed after the initial setup.

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
Deleting these fields before calling [`contrast set`](./workload-deployment/set-manifest.md) will render the deployment immutable.

```json
{
  "WorkloadOwnerKeyDigests": [
    "<remove this digest to prevent manifest updates>" 
  ],
  "SeedshareOwnerPubKeys": [
    "<remove this key to prevent state decryption and coordinator recovery>"
  ]
}
```

### Preventing manifest updates

To prevent manifest updates, delete the `WorkloadOwnerKeyDigest`.
Now [set the manifest](./workload-deployment/set-manifest.md) through `contrast set`.
Once completed, the Contrast Coordinator will no longer accept the workload owner key for future manifest changes.

Alternatively, specify the `--disable-updates` flag in the `generate` step for the same result.

```sh
contrast generate --reference-values <platform> --disable-updates resources/
```

:::warning

This action is irreversible.
You won't be able to make changes to your deployment after running `contrast set` with an empty `WorkloadOwnerKeyDigest`.

:::

### Preventing decryption of state and recovery

Deleting the `SeedshareOwnerPubKeys` field from the manifest before calling `contrast set` will make it impossible to recover both the Contrast Coordinator and any state associated with a deployment.
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
The `contrast verify` step for a manifest with removed `WorkloadOwnerKeyDigest` and/or `SeedshareOwnerPubKeys` will only succeed if the manifest set on the Coordinator is also missing the corresponding keys,
thereby guaranteeing no party can update and/or recover the deployment and its state.
