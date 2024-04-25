# Confidential emoji voting

<!-- TODO(katexochen): create a screenshot with fixed format -->
![screenshot of the emojivoto UI](../_media/emoijvoto.png)

**This tutorial guides you through deploying [emojivoto](https://github.com/BuoyantIO/emojivoto) as a
confidential Contrast deployment and validating the deployment from a voters perspective.**


Emojivoto is an example app allowing users to vote for different emojis and view votes
on a leader board. It has a microservice architecture consisting of a
web frontend (`web`), a gRPC backend for listing available emojis (`emoji`), and a backend for
the voting and leader board logic (`voting`). The `vote-bot` simulates user traffic by submitting
votes to the frontend.

<!-- TODO(katexochen): recreate in our design -->
![emojivoto components topology](https://raw.githubusercontent.com/BuoyantIO/emojivoto/e490d5789086e75933a474b22f9723fbfa0b29ba/assets/emojivoto-topology.png)

### Motivation

Using a voting service, users' votes are considered highly sensitive data, as we require
a secret ballot. Also, users are likely interested in the fairness of the ballot. For
both requirements, we can use Confidential Computing and, specifically, workload attestation
to prove to those interested in voting that the app is running in a protected environment
where their votes are processed without leaking to the platform provider or workload owner.

## Prerequisites

- **Installed Contrast CLI.**
  See the [installation instructions](./../getting-started/install.md) on how to get it.
- **Running cluster with Confidential Containers support.**
  Please follow the [cluster setup instructions](./../getting-started/cluster-setup.md)
  to create a cluster.
- **Get the deployment.** This is currently available as part of the preview bundle.

## Steps to deploy emojivoto with Contrast

### Deploy the Contrast Coordinator

Deploy the Contrast Coordinator, comprising a single replica deployment and a
LoadBalancer service, into your cluster:

```sh
kubectl apply -f coordinator.yml
```

### Generate policy annotations and manifest

Run the `generate` command to generate the execution policies and add them as
annotations to your deployment files. A `manifest.json` file with the reference values
of your deployment will be created:

```sh
contrast generate deployment/
```

:::note[Runtime class and Initializer]

The deployment YAML shipped for this demo is already configured to be used with Contrast.
A runtime class `contrast-cc-<VERSIONHASH>` was added to the pods to signal they should be run
as Confidential Containers. In addition, the Contrast Initializer was added
as an init container to these workloads to facilitate the attestation and certificate pulling
before the actual workload is started.

:::

### Set the manifest

Configure the coordinator with a manifest. It might take up to a few minutes
for the load balancer to be created and the Coordinator being available.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "The user API of your Contrast Coordinator is available at $coordinator:1313"
contrast set -c "${coordinator}:1313" deployment/
```

The CLI will use the embedded reference values to attest the Coordinator deployment
during the TLS handshake. If the connection succeeds, we're ensured that the Coordinator
deployment hasn't been tampered with.

### Deploy emojivoto

Now that the coordinator has a manifest set, which defines the emojivoto deployment as an allowed workload,
we can deploy the application:

```sh
kubectl apply -f deployment/
```

:::note[Inter-deployment communication]

The Contrast Coordinator issues mesh certificates after successfully validating workloads.
These certificates can be used for secure inter-deployment communication. The Initializer
sends an attestation report to the Coordinator, retrieves certificates and a private key in return
and writes them to a `volumeMount`. The emojivoto version we're using is patched to only communicate
via mTLS (the original app talks plain HTTP). The different parts of the workload are configured
to use the credentials from the `volumeMount` when communicating with each other.

:::

## Voter's perspective: Verifying the ballot

As voters, we want to verify the fairness and confidentiality of the deployment before
deciding to vote. Regardless of the scale of our distributed deployment, Contrast only
needs a single remote attestation step to verify the deployment. By doing remote attestation
of the Coordinator, we transitively verify those systems the Coordinator has already attested
or will attest in the future. Successful verification of the Coordinator means that
we can be sure it will enforce the configured manifest.

### Attest the Coordinator

A potential voter can verify the Contrast deployment using the verify
command:

```sh
contrast verify -c "${coordinator}:1313"
```

The CLI will attest the Coordinator using embedded reference values. If the command succeeds,
the Coordinator deployment was successfully verified to be running in the expected Confidential
Computing environment with the expected code version. The Coordinator will then return its
configuration over the established TLS channel. The CLI will store this information, namely the root
certificate of the mesh (`mesh-root.pem`) and the history of manifests, into the `verify/` directory.
In addition, the policies referenced in the manifest history are also written into the same directory.

### Manifest history and artifact audit

In the next step, the Coordinator configuration that was written by the `verify` command needs to be audited.
A potential voter should inspect the manifest and the referenced policies. They could delegate
this task to an entity they trust.

### Confidential connection to the attested workload

After ensuring the configuration of the Coordinator fits the expectation, you can securely connect
to the workloads using the Coordinator's `mesh-root.pem` as a trusted CA certificate.

To access the web frontend, expose the service on a public IP address via a LoadBalancer service:

```sh
frontendIP=$(kubectl get svc web-svc -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Frontend is available at  https://$frontendIP, you can visit it in your browser."
```

Using `openssl`, the certificate of the service can be validated with the `mesh-root.pem`:

```sh
openssl s_client -CAfile verify/mesh-root.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null
```

## Certificate SAN and manifest update (optional)

By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed
via load balancer IP in this demo. Tools like curl check the certificate for IP entries in the SAN field.
Validation fails since the certificate contains no IP entries as a subject alternative name (SAN).
For example, a connection attempt using the curl and the mesh root certificate with throw the following error:

```sh
$ curl --cacert ./verify/mesh-root.pem "https://${frontendIP}:443"
curl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'
```

### Configure the service SAN in the manifest

The `Policies` section of the manifest maps policy hashes to a list of SANs. To enable certificate verification
of the web frontend with tools like curl, edit the policy with your favorite editor and add the `frontendIP` to
the list that already contains the `"web"` DNS entry:

```diff
   "Policies": {
     ...
     "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": [
       "web",
-      "*"
+      "*",
+      "203.0.113.34"
     ],
```

### Update the manifest

Next, set the changed manifest at the coordinator with:

```sh
contrast set -c "${coordinator}:1313" deployment/
```

The Contrast Coordinator will rotate the mesh root certificate on the manifest update. Workload certificates issued
after the manifest are thus issued by another certificate authority and services receiving the new CA certificate chain
won't trust parts of the deployment that got their certificate issued before the update. This way, Contrast ensures
that parts of the deployment that received a security update won't be infected by parts of the deployment at an older
patch level that may have been compromised. The `mesh-root.pem` is updated with the new CA certificate chain.

### Rolling out the update

The Coordinator has the new manifest set, but the different containers of the app are still
using the older certificate authority. The Contrast Initializer terminates after the initial attestation
flow and won't pull new certificates on manifest updates.

To roll out the update, use:

```sh
kubectl rollout restart deployment/emoji
kubectl rollout restart deployment/vote-bot
kubectl rollout restart deployment/voting
kubectl rollout restart deployment/web
```

After the update has been rolled out, connecting to the frontend using curl will successfully validate
the service certificate and return the HTML document of the voting site:

```sh
curl --cacert ./mesh-root.pem "https://${frontendIP}:443"
```
