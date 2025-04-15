# Hardening

This section provides guidance on writing secure applications and explains the trust boundaries developers need to be aware of when working with Contrast.

## Applicability

Recommended for all Contrast deployments.

## Prerequisites

A running Contrast deployment.

## How-to

Contrast ensures application integrity and provides secure means of communication and bootstrapping (see [Security](../security.md) section).
However, care must be taken when interacting with the outside of Contrast's confidential environment.

## General recommendations

### Authentication

The application receives credentials from the Contrast Coordinator during initialization.
This allows to authenticate towards peers and to verify credentials received from peers.
The application should use the certificate bundle to authenticate incoming requests and be wary of unauthenticated requests or requests with a different root of trust (for example the internet PKI).

The recommendation to authenticate not only applies to network traffic, but also to volumes, GPUs and other devices.
Generally speaking, all information provided by the world outside the confidential VM should be treated with due scepticism, especially if it's not authenticated.
Common cases where Kubernetes apps interact with external services include DNS, Kubernetes API clients and cloud storage endpoints.

### Encryption

Any external persistence should be encrypted with an authenticated cipher.
This recommendation applies to block devices or filesystems mounted into the container, but also to cloud blob storage or external databases.

## Contrast security guarantees

If an application authenticates with a certificate signed by the Contrast Mesh CA of a given manifest, Contrast provides the following guarantees:

1. The container images used by the app are the images specified in the resource definitions.
2. The command line arguments of containers are exactly the arguments specified in the resource definitions.
3. All environment variables are either specified in resource definitions, in the container image manifest or in a settings file for the Contrast CLI.
4. The containers run in a confidential VM that matches the reference values in the manifest.
5. The containers' root filesystems are mounted in encrypted memory.

### Limitations inherent to policy checking

Workload policies serve as workload identities.
From the perspective of the Contrast Coordinator, all workloads that authenticate with the same policy are equal.
Thus, it's not possible to disambiguate, for example, pods spawned from a deployment or to limit the amount of certificates issued per policy.

Container image references from Kubernetes resource definitions are taken into account when generating the policy.
A mutable reference may lead to policy failures or unverified image content, depending on the Contrast runtime.
Reliability and security can only be ensured with a full image reference, including digest.
The [`docker pull` documentation] explains pinned image references in detail.

Policies can only verify what can be inferred at generation time.
Some attributes of Kubernetes pods can't be predicted and thus can't be verified.
Particularly the [downward API] contains many fields that are dynamic or depend on the host environment, rendering it unsafe for process environment or arguments.
The same goes for `ConfigMap` and `Secret` resources, which can also be used to populate container fields.
If the application requires such external information, it should be injected as a mount point and carefully inspected before use.

Another type of dynamic content are persistent volumes.
Any volumes mounted to the pod need to be scrutinized, and sensitive data must not be written to unprotected volumes.
Ideally, a volume is mounted as a raw block device and authenticated encryption is added within the confidential container.

[`docker pull` documentation]: https://docs.docker.com/reference/cli/docker/image/pull/#pull-an-image-by-digest-immutable-identifier
[downward API]: https://kubernetes.io/docs/concepts/workloads/pods/downward-api/

### Logs

By default, container logs are visible to the host to enable normal Kubernetes operations, for example debugging using `kubectl logs`.
The application needs to ensure that sensitive information isn't logged.

If logs access isn't required, it can be denied with a manual change to the policy settings.
After the initial run of `contrast generate`, there will be a `settings.json` file in the working directory.
Modify the default for `ReadStreamRequest` like shown in the diff below and run `contrast generate` again.

<!-- TODO(burgerdev): this should reference a man page for advanced config -->

```diff
diff --git a/settings.json b/settings-no-logs.json
index fd998a4..6760000 100644
--- a/settings.json
+++ b/settings-no-logs.json
@@ -330,7 +330,7 @@
             "regex": []
         },
         "CloseStdinRequest": false,
-        "ReadStreamRequest": true,
+        "ReadStreamRequest": false,
         "UpdateEphemeralMountsRequest": false,
         "WriteStreamRequest": false
     }

```
