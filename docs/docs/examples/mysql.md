# Encrypted volume mount

**This tutorial guides you through deploying a simple application with an
encrypted MySQL database using the Contrast [workload
secret](../architecture/secrets.md#workload-secrets).**

[MySQL](https://mysql.com) is an open-source database used to organize data into
tables and quickly retrieve information about its content. All of the data in a
MySQL database is stored in the `/var/lib/mysql` directory. In this example, we
use the workload secret to setup an encrypted LUKS mount for the
`/var/lib/mysql` directory to easily deploy an application with encrypted
persistent storage using Contrast.

The resources provided in this demo are designed for educational purposes and
shouldn't be used in a production environment without proper evaluation. When
working with persistent storage, regular backups are recommended in order to
prevent data loss. For confidential applications, please also refer to the
[security considerations](../architecture/security-considerations.md). Also be
aware of the differences in security implications of the workload secrets for
the data owner and the workload owner. For more details, see the [Workload
Secrets](../architecture/secrets.md#workload-secrets) documentation.

## Prerequisites

- Installed Contrast CLI
- A running Kubernetes cluster with support for confidential containers, either on [AKS](../getting-started/cluster-setup.md) or on [bare metal](../getting-started/bare-metal.md)

## Steps to deploy MySQL with Contrast

### Download the deployment files

The MySQL deployment files are part of the Contrast release. You can download them by running:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/mysql-demo.yml --create-dirs --output-dir deployment
```

### Deploy the Contrast runtime

Contrast depends on a [custom Kubernetes `RuntimeClass`](../components/runtime.md),
which needs to be installed to the cluster initially.
This consists of a `RuntimeClass` resource and a `DaemonSet` that performs installation on worker nodes.
This step is only required once for each version of the runtime.
It can be shared between Contrast deployments.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-aks-clh-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-tdx.yml
```
</TabItem>
</Tabs>

### Deploy the Contrast Coordinator

Deploy the Contrast Coordinator, comprising a single replica deployment and a
`LoadBalancer` service, into your cluster:

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator-aks-clh-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator-k3s-qemu-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator-k3s-qemu-tdx.yml
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
workloads. It will attest the pod to the Coordinator and fetch the workload certificates and the workload secret.

Further, the deployment YAML is also configured with the Contrast [service mesh](../components/service-mesh.md).
The configured service mesh proxy provides transparent protection for the communication between
the MySQL server and client.
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

### Deploy MySQL

Now that the coordinator has a manifest set, which defines the MySQL deployment as an allowed workload,
we can deploy the application:

```sh
kubectl apply -f deployment/
```

:::note[Persistent workload secrets]

During the initialization process of the workload pod, the Contrast Initializer
sends an attestation report to the Coordinator and receives a workload secret
derived from the Coordinator's secret seed and the workload secret ID specified in the
manifest, and writes it to a secure in-memory `volumeMount`.

:::

The MySQL deployment is declared as a StatefulSet with a mounted block device.
An init container running `cryptsetup` uses the workload secret at
`/contrast/secrets/workload-secret-seed` to generate a key and setup the block
device as a LUKS partition. Before starting the MySQL container, the init
container uses the generated key to open the LUKS device, which is then mounted
by the MySQL container. For the MySQL container, this process is completely
transparent and works like mounting any other volume. The `cryptsetup` container
will remain running to provide the necessary decryption context for the workload
container.

## Verifying the deployment as a user

In different scenarios, users of an app may want to verify its security and identity before sharing data, for example, before connecting to the database.
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
A user of the application should inspect the manifest and the referenced policies. They could delegate
this task to an entity they trust.

### Connecting to the application

Other confidential containers can securely connect to the MySQL server via the
[Service Mesh](../components/service-mesh.md). The configured `mysql-client`
deployment connects to the MySQL server and inserts test data into a table. To
view the logs of the `mysql-client` deployment, use the following commands:

```sh
kubectl logs -l app.kubernetes.io/name=mysql-client -c mysql-client
```

The Service Mesh ensures an mTLS connection between the MySQL client and server
using the mesh certificates. As a result, no other workload can connect to the
MySQL server unless explicitly allowed in the manifest.

## Updating the deployment

Because the workload secret is derived from the `WorkloadSecredID` specified in
the manifest and not to an individual pod, once the pod restarts, the
`cryptsetup` init container can deterministically generate the same key again
and open the already partitioned LUKS device.
For more information on using the workload secret, see [Workload
Secrets](../architecture/secrets.md#workload-secrets).

For example, after making changes to the deployment files, the runtime policies
need to be regenerated with `contrast generate` and the new manifest needs to be
set using `contrast set`.

```sh
contrast generate deployment/
contrast set -c "${coordinator}:1313" deployment/
```

The new deployment can then be applied by running:

```sh
kubectl rollout restart statefulset/mysql-backend
kubectl rollout restart deployment/mysql-client
```

The new MySQL backend pod will then start up the `cryptsetup` init container which
receives the same workload secret as before and can therefore generate the
correct key to open the LUKS device. All previously stored data in the MySQL
database is available in the newly created pod in an encrypted volume mount.
