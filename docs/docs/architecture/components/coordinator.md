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

## Automatic recovery and high availability

The Contrast Coordinator is deployed as a single replica in its default configuration.
When this replica is restarted, for example for node maintenance, it needs to be recovered manually.
For automatic peer recovery and high-availability, the Coordinator should be [scaled to at least 3 replicas](../../howto/coordinator-ha.md).
