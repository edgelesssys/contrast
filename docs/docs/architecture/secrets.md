# Secret Management

When the Coordinator is configured with the initial manifest, it generates a random secret seed.
From this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.
This derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state.

The secret seed is returned to the user on the first call to `contrast set`, encrypted with the user's public seedshare owner key.
If no seedshare owner key is provided, a key is generated and stored in the working directory.

## Persistence

The Coordinator runs as a `StatefulSet` with a dynamically provisioned persistent volume.
This volume stores the manifest history and the associated runtime policies.
The manifest is not considered sensitive information, because it needs to be passed to the untrusted infrastructure in order to start workloads.
However, the Coordinator must ensure its integrity and that the persisted data corresponds to the manifests set by authorized users.
Thus, the manifest is stored in plain text, but is signed with a private key derived from the Coordinator's secret seed.

## Recovery

When a Coordinator starts up, it does not have access to the signing secret and can thus not verify the persisted manifests' integrity.
It needs to be provided with the secret seed, from which it can derive the signing key that verifies the signatures.
This procedure is called recovery and is initiated by the workload owner.
The CLI decrypts the secret seed using the private seedshare owner key, verifies the Coordinator and sends the seed through the `Recover` method.
The Coordinator recovers its key material and verifies the manifest history signature.
