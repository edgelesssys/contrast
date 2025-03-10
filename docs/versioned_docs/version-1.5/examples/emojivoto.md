# Confidential emoji voting

<!-- TODO(katexochen): create a screenshot with fixed format -->

![screenshot of the emojivoto UI](../_media/emoijvoto.png)

**This tutorial guides you through deploying [emojivoto](https://github.com/BuoyantIO/emojivoto) as a
confidential Contrast deployment and validating the deployment from a voter's perspective.**

Emojivoto is an example app allowing users to vote for different emojis and view votes
on a leader board. It has a microservice architecture consisting of a
web frontend (`web`), a gRPC backend for listing available emojis (`emoji`), and a backend for
the voting and leader board logic (`voting`). The `vote-bot` simulates user traffic by submitting
votes to the frontend.

Emojivoto can be seen as a lighthearted example of an app dealing with sensitive data.
Contrast protects emojivoto in two ways. First, it shields emojivoto as a whole from the infrastructure, for example, Azure.
Second, it can be configured to also prevent data access even from the administrator of the app. In the case of emojivoto, this gives assurance to users that their votes remain secret.

<!-- TODO(katexochen): recreate in our design -->

![emojivoto components topology](https://raw.githubusercontent.com/BuoyantIO/emojivoto/e490d5789086e75933a474b22f9723fbfa0b29ba/assets/emojivoto-topology.png)

## Prerequisites

- Installed Contrast CLI
- A running Kubernetes cluster with support for confidential containers, either on [AKS](../getting-started/cluster-setup.md) or on [bare metal](../getting-started/bare-metal.md).

## Steps to deploy emojivoto with Contrast

### Download the deployment files

The emojivoto deployment files are part of the Contrast release. You can download them by running:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/download/v1.5.2/emojivoto-demo.yml --create-dirs --output-dir deployment
```

### Deploy the Contrast runtime

Contrast depends on a [custom Kubernetes RuntimeClass](../components/runtime.md),
which needs to be installed to the cluster initially.
This consists of a `RuntimeClass` resource and a `DaemonSet` that performs installation on worker nodes.
This step is only required once for each version of the runtime.
It can be shared between Contrast deployments.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.2/runtime-aks-clh-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.2/runtime-k3s-qemu-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.2/runtime-k3s-qemu-tdx.yml
```
</TabItem>
</Tabs>

### Deploy the Contrast Coordinator

Deploy the Contrast Coordinator, comprising a single replica deployment and a
LoadBalancer service, into your cluster:

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.2/coordinator-aks-clh-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.2/coordinator-k3s-qemu-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.2/coordinator-k3s-qemu-tdx.yml
```
</TabItem>
</Tabs>

### Generate policy annotations and manifest

Run the `generate` command to generate the execution policies and add them as
annotations to your deployment files. A `manifest.json` file with the reference values
of your deployment will be created:

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
contrast generate --reference-values aks-clh-snp deployment/
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
contrast generate --reference-values k3s-qemu-snp deployment/
```
:::note[Missing TCB values]
On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}` and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models.
:::
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
contrast generate --reference-values k3s-qemu-tdx deployment/
```
:::note[Missing TCB values]
On bare-metal TDX, `contrast generate` is unable to fill in the `MinimumTeeTcbSvn` and `MrSeam` TCB values as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `ffffffffffffffffffffffffffffffff` and `000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000` respectively and observe the real values in the error messages in the following steps. This should only be done in a secure environment.
:::
</TabItem>
</Tabs>

:::note[Runtime class and Initializer]

The deployment YAML shipped for this demo is already configured to be used with Contrast.
A [runtime class](../components/runtime) `contrast-cc`
was added to the pods to signal they should be run as Confidential Containers. During the generation process,
the Contrast [Initializer](../components/overview.md#the-initializer) will be added as an init container to these
workloads to facilitate the attestation and certificate pulling before the actual workload is started.

Further, the deployment YAML is also configured with the Contrast [service mesh](../components/service-mesh.md).
The configured service mesh proxy provides transparent protection for the communication between
the different components of emojivoto.
:::

### Set the manifest

Configure the coordinator with a manifest. It might take up to a few minutes
for the load balancer to be created and the Coordinator being available.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "The user API of your Contrast Coordinator is available at $coordinator:1313"
contrast set -c "${coordinator}:1313" deployment/
```

The CLI will use the reference values from the manifest to attest the Coordinator deployment
during the TLS handshake. If the connection succeeds, it's ensured that the Coordinator
deployment hasn't been tampered with.

:::warning
On bare metal, the [coordinator policy hash](components/policies.md#platform-differences) must be overwritten using `--coordinator-policy-hash`.
:::

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
and writes them to a `volumeMount`. The service mesh sidecar is configured to use the credentials
from the `volumeMount` when communicating with other parts of the deployment over mTLS.
The public facing frontend for voting uses the mesh certificate without client authentication.

:::

## Verifying the deployment as a user

In different scenarios, users of an app may want to verify its security and identity before sharing data, for example, before casting a vote.
With Contrast, a user only needs a single remote-attestation step to verify the deployment - regardless of the size or scale of the deployment.
Contrast is designed such that, by verifying the Coordinator, the user transitively verifies those systems the Coordinator has already verified or will verify in the future.
Successful verification of the Coordinator means that the user can be sure that the given manifest will be enforced.

### Verifying the Coordinator

A user can verify the Contrast deployment using the verify
command:

```sh
contrast verify -c "${coordinator}:1313" -m manifest.json
```

The CLI will verify the Coordinator via remote attestation using the reference values from a given manifest. This manifest needs
to be communicated out of band to everyone wanting to verify the deployment, as the `verify` command checks
if the currently active manifest at the Coordinator matches the manifest given to the CLI. If the command succeeds,
the Coordinator deployment was successfully verified to be running in the expected Confidential
Computing environment with the expected code version. The Coordinator will then return its
configuration over the established TLS channel. The CLI will store this information, namely the root
certificate of the mesh (`mesh-ca.pem`) and the history of manifests, into the `verify/` directory.
In addition, the policies referenced in the manifest history are also written into the same directory.

:::warning
On bare metal, the [coordinator policy hash](components/policies.md#platform-differences) must be overwritten using `--coordinator-policy-hash`.
:::

### Auditing the manifest history and artifacts

In the next step, the Coordinator configuration that was written by the `verify` command needs to be audited.
A potential voter should inspect the manifest and the referenced policies. They could delegate
this task to an entity they trust.

### Connecting securely to the application

After ensuring the configuration of the Coordinator fits the expectation, the user can securely connect
to the application using the Coordinator's `mesh-ca.pem` as a trusted CA certificate.

To access the web frontend, expose the service on a public IP address via a LoadBalancer service:

```sh
frontendIP=$(kubectl get svc web-svc -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Frontend is available at  https://$frontendIP, you can visit it in your browser."
```

Using `openssl`, the certificate of the service can be validated with the `mesh-ca.pem`:

```sh
openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null
```

## Updating the certificate SAN and the manifest (optional)

By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed
via load balancer IP in this demo. Tools like curl check the certificate for IP entries in the subject alternative name (SAN) field.
Validation fails since the certificate contains no IP entries as a SAN.
For example, a connection attempt using the curl and the mesh CA certificate with throw the following error:

```sh
$ curl --cacert ./verify/mesh-ca.pem "https://${frontendIP}:443"
curl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'
```

### Configuring the service SAN in the manifest

The `Policies` section of the manifest maps policy hashes to a list of SANs. To enable certificate verification
of the web frontend with tools like curl, edit the policy with your favorite editor and add the `frontendIP` to
the list that already contains the `"web"` DNS entry:

```diff
   "Policies": {
     ...
     "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": {
       "SANs": [
         "web",
-        "*"
+        "*",
+        "203.0.113.34"
       ],
       "WorkloadSecretID": "web"
      },
```

### Updating the manifest

Next, set the changed manifest at the coordinator with:

```sh
contrast set -c "${coordinator}:1313" deployment/
```

The Contrast Coordinator will rotate the mesh ca certificate on the manifest update. Workload certificates issued
after the manifest update are thus issued by another certificate authority and services receiving the new CA certificate chain
won't trust parts of the deployment that got their certificate issued before the update. This way, Contrast ensures
that parts of the deployment that received a security update won't be infected by parts of the deployment at an older
patch level that may have been compromised. The `mesh-ca.pem` is updated with the new CA certificate chain.

:::warning
On bare metal, the [coordinator policy hash](components/policies.md#platform-differences) must be overwritten using `--coordinator-policy-hash`.
:::

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
curl --cacert ./mesh-ca.pem "https://${frontendIP}:443"
```
