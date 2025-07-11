# The Contrast Coordinator

The Contrast Coordinator is the central remote attestation service of a Contrast deployment.
It runs inside a confidential container inside your cluster.
The Coordinator can be verified via remote attestation, and a Contrast deployment is self-contained.
The Coordinator is configured with a _manifest_, a configuration file containing the reference attestation values of your deployment.
It ensures that your deployment's topology adheres to your specified manifest by verifying the identity and integrity of all confidential pods inside the deployment.
The Coordinator is also a certificate authority and issues certificates for your workload pods during the attestation procedure.
Your workload pods can establish secure, encrypted communication channels between themselves based on these certificates using the Coordinator as the root CA.
As your app needs to scale, the Coordinator transparently verifies new instances and then provides them with their certificates to join the deployment.

To verify your deployment, the Coordinator's remote attestation statement combined with the manifest offers a concise single remote attestation statement for your entire deployment.
A third party can use this to verify the integrity of your distributed app, making it easy to assure stakeholders of your app's identity and integrity.

## The Manifest

The manifest is the configuration file for the Coordinator, defining your confidential deployment.
It's automatically generated from your deployment by the Contrast CLI.
It currently consists of the following parts:

- _Policies_: The identities of your Pods, represented by the hashes of their respective runtime policies.
- _Reference Values_: The remote attestation reference values for the Kata confidential micro-VM that's the runtime environment of your Pods.
- _WorkloadOwnerKeyDigest_: The workload owner's public key digest. Used for authenticating subsequent manifest updates.
- _SeedshareOwnerKeys_: public keys of seed share owners. Used to authenticate user recovery and permission to handle the secret seed.

<!-- TODO(burgerdev): document manifest storage. -->

## State

A Contrast Coordinator can be in one of three states:

- After a fresh installation, there is no manifest history and the Coordinator waits for its initialization by `contrast set`.
- When the Coordinator starts up and finds an existing manifest history, it enters _recovery mode_.
  It periodically tries to recover from its peers, or waits for the user to run `contrast recover` if there are none.
  All other API requests fail as long as the Coordinator is in recovery mode.
- If the Coordinator is synchronized to the latest manifest in history, it transitions to the `Ready` state and starts accepting requests from workload initializers.

## Services

The Contrast Coordinator comes with two services: `coordinator` and `coordinator-ready`.
The `coordinator` service is backed by all Coordinators, ready or not ready, and is intended to serve user API (that is, `contrast` CLI commands).
The `coordinator-ready` service only selects ready Coordinators which can serve the mesh API, and is intended to be used by initializers.
This endpoint is also suitable for verifying clients, since they will only get a successful response from a ready Coordinator.

## Automatic recovery and high availability {#peer-recovery}

The Contrast Coordinator is deployed as a single replica in its default configuration.
When this replica is restarted, for example for node maintenance, it needs to be recovered manually.
This can be avoided by running multiple replicas of the Coordinator, allowing the Coordinators to recover their peers automatically.

Newly started (or restarted) Coordinator instances discover ready peers using the Kubernetes API server.
If a manifest history exists in the cluster and the Coordinator isn't updated yet to the latest manifest, it stays in a recovery loop.
For each ready Coordinator peer, it tries to reach that peer's mesh API endpoint directly (using the pod IP), attest to that Coordinator and receive the secret seed and the mesh CA credentials.

As long as a single Coordinator is initialized, the other instances will eventually recover from it.
`StatefulSet` semantics guarantee that Coordinator pods are started predictably, and only after all existing Coordinators are ready.
For automatic peer recovery and high-availability, the Coordinator should be [scaled to at least 3 replicas](../../howto/coordinator-ha.md).
