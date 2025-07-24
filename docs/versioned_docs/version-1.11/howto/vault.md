# Vault

**This tutorial guides you through deploying a Vault as a confidential deployment by using Contrast's built-in Vault support.**

Vaults are an identity-based secrets and encryption management systems, which provide encryption services that are gated by authentication and authorization methods to ensure secure, auditable and restricted access to secrets, such as API keys, passwords, encryption keys or certificates.

There are several implementations of Vault systems, the most prominent being [HashiCorp Vault](https://www.hashicorp.com/en/products/vault) and its open-source derivative, [OpenBao](https://openbao.org/). This example utilizes the Vault image provided by OpenBao.
The associated APIs exposed by the Contrast coordinator are fully compatible with both HashiCorp Vault and OpenBao, ensuring interoperability across implementations.

Contrast can lift a Vault deployment into a confidential computing environment, further shielding the secrets from the workload operator.

## Sealing and Unsealing of Vaults

[Sealing](https://openbao.org/docs/concepts/seal/) ensures that all sensitive data within the Vault remains inaccessible and protected when the system isn't in active use.
It provides a security boundary that prevents unauthorized access during restarts or shutdowns.

Unsealing is a manual process required to transition Vault into an operational state, allowing authorized access to stored secrets.
Vault implementations by default use a set of unseal keys derived from a master key, building up on `Shamir's Secret Sharing` scheme.
Further to auto-unseal Vaults, the process can be delegated to another already initialized Vault by using an exposed [transit secrets engine API](https://openbao.org/api-docs/secret/transit/) as the unsealing mechanism.

## Transit secrets engine API of Contrast Coordinator

To automate the unsealing process in confidential deployments of Vault instances, the coordinator exposes a compatible transit secrets engine API on port 8200.
Vault deployments can be configured to integrate this transit engine to enable auto-unsealing, ensuring immediate operational readiness and seamless integration within the secure Contrast environment.

### Secure endpoints with mutual TLS

All communication between the transit secrets engine API and Vault is secured through mutual TLS (mTLS).
Only entities presenting a [mesh certificate](../architecture/components/service-mesh.md#public-key-infrastructure) signed by the current mesh CA key are trusted.
The Coordinator issues itself a valid certificate at the time of the transit secrets engine API call, while the Vault deployment obtains its certificate in the initialization phase, after attesting to the Coordinator.

### Role of `workloadSecretID`

To support persistence in the auto-unsealing process, the `workloadSecretID` is used to derive the encryption key utilized by the transit secrets engine.
Beyond key derivation, the `workloadSecretID` also plays a critical role in authorization.

Access to a specific encryption key via the transit secrets engine API is permitted only if the requested key name matches the `workloadSecretID` embedded in the corresponding certificate extension of the Contrast mesh certificate.
This ensures that each entity can only access their own set of encryption keys within the transit secrets engine.

For more details on how the workload secret is used, see [Workload Secrets](../architecture/secrets.md#workload-secrets).

## Prerequisites

- Installed Contrast CLI
- A running Kubernetes cluster with support for confidential containers, either on [AKS](../howto/cluster-setup/aks.md) or on [bare metal](../howto/cluster-setup/bare-metal.md)

## Steps to deploy Vault with Contrast

### Download the deployment files

The Vault deployment files are part of the Contrast release. You can download them by running:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/download/v1.11.0/vault-demo.yml --create-dirs --output-dir deployment
```

### Deploy the Contrast runtime

Contrast depends on a [custom Kubernetes `RuntimeClass`](../architecture/components/runtime.md), which needs to be installed to the cluster initially.
This consists of a `RuntimeClass` resource and a `DaemonSet` that performs installation on worker nodes.
This step is only required once for each version of the runtime.
It can be shared between Contrast deployments.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.11.0/runtime-aks-clh-snp.yml
```

</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.11.0/runtime-k3s-qemu-snp.yml
```

</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.11.0/runtime-k3s-qemu-tdx.yml
```

</TabItem>
</Tabs>

### Download the Contrast Coordinator resource

Download the Kubernetes resource of the Contrast Coordinator, comprising a single replica deployment and a LoadBalancer service.
Put it next to your resources:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/download/v1.11.0/coordinator.yml --output-dir deployment
```

### Generate policy annotations and manifest

Run the `generate` command to generate the execution policies and add them as annotations to your deployment files.
A `manifest.json` file with the reference values of your deployment will be created:

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
If you don't know the correct values use `{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}` and observe the real values in the error messages in the following steps.
This should only be done in a secure environment.
Note that the values will differ between CPU models.

:::

</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">

```sh
contrast generate --reference-values k3s-qemu-tdx deployment/
```
:::note[Missing TCB values]

On bare-metal TDX, `contrast generate` is unable to fill in the `MinimumTeeTcbSvn` and `MrSeam` TCB values as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `ffffffffffffffffffffffffffffffff` and `000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000` respectively and observe the real values in the error messages in the following steps.
This should only be done in a secure environment.

:::

</TabItem>
</Tabs>

:::note[Runtime class and Initializer]

The deployment YAML shipped for this demo is already configured to be used with Contrast.
A [runtime class](../architecture/components/runtime.md) `contrast-cc` was added to the pods to signal they should be run as Confidential Containers.
During the generation process, the Contrast [Initializer](../architecture/components/initializer.md) will be added as an init container to these workloads.
It will attest the pod to the Coordinator and fetch the workload certificates and the workload secret.

:::

### Deploy the Coordinator

Deploy the Coordinator resource first by applying its resource definition:

```sh
kubectl apply -f deployment/coordinator.yml
```

### Set the manifest

Configure the Coordinator with a manifest.
It might take up to a few minutes for the load balancer to be created and the Coordinator being available.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "The user API of your Contrast Coordinator is available at $coordinator:1313"
contrast set -c "${coordinator}:1313" deployment/
```

The CLI will use the reference values from the manifest to attest the Coordinator deployment during the TLS handshake.
If the connection succeeds, it's ensured that the Coordinator deployment hasn't been tampered with.

### Deploy Vault

Now that the Coordinator has a manifest set, which defines the Vault deployment as an allowed workload, we can deploy the application:

```sh
kubectl apply -f deployment/
```

<!-- TODO(jmxnzo): revise this paragraph as soon as we support filesystem volumes everywhere -->
The Vault deployment is defined as a StatefulSet using the OpenBao Vault image, with a mounted block device for persistent storage.
A Contrast Initializer, running as an init container, uses the workload secret located at `/contrast/secrets/workload-secret-seed` to generate an encryption key and initialize the block device as a LUKS-encrypted partition.
Before the Vault container starts, the Initializer opens the LUKS device using the generated key.
This unlocked device is then mounted by the Vault container and used as the backend storage volume.
For the Vault application, this process is entirely transparent, and the device behaves like a standard volume mount.
It’s important to clarify that the LUKS encryption of the block device is primarily a convenience feature, enabling persistent storage at the filesystem level on confidential virtual machines.
The primary and security-relevant encryption mechanism remains Vault’s own sealing process, which provides cryptographic protection of secrets even if the underlying storage is compromised.

Because the `workload-secret-seed` is derived from the associated `workloadSecretID`, any change to the `workloadSecretID` after the block device has been initialized will result in deriving an invalid encryption key, making the mounted block device undecryptable.

:::note[Inter-deployment communication]

The Contrast Coordinator issues mesh certificates after successfully validating workloads.
These certificates can be used for secure inter-deployment communication.
The Initializer sends an attestation report to the Coordinator, retrieves a service mesh certificate bound to it's provided public key, containing the certificate chain, as well as the current mesh CA cert.
The Initializer then writes them to a `volumeMount`, allowing to build up the secure mTLS connections based on the service mesh.
The received service mesh certificate also holds the certificate extension of the `workloadSecretID`, which is used to allow the authorization to a certain encryption key on the transit engine API.

:::

The Vault's TCP listener is configured to accept connections only from mesh certificates issued under the same CA state used to sign the Vault’s own certificate, effectively restricting communication to attested Contrast deployments.
Because updating the `workloadSecretID` after initializing the LUKS device will make it inaccessible, it's critical to ensure that the `workloadSecretID` is correctly aligned with the intended endpoint specified in Vault’s sealing configuration before the first `contrast set` is executed.
This can be achieved by using the corresponding `workload-secret-id` annotation to directly overwrite the manifest with the required `workloadSecretID`.

### Connecting to the application

Other confidential containers can securely connect to the Vault server via the [Service Mesh](../architecture/components/service-mesh.md).
As previously noted, access to the Vault endpoint is restricted to peers that present a service mesh certificate valid under the currently set manifest.
While such a certificate enables mTLS-based communication with the Vault server, it doesn't, on its own, grant authorization to perform Vault-related operations.
Permissions for accessing secrets within Vault must be explicitly configured using the root token obtained during Vault initialization.
The configured `openbao-client` deployment is responsible for executing Vault-related operations, including initialization, secret creation, and sealing instructions.

For more information on the Vault management and administration, please follow the official [OpenBao documentation](https://openbao.org/docs/).

### Restarting the Vault deployment with auto-unsealing

If you want to make changes to your resources, ensure that the `workloadSecretID` of the Vault remains unchanged.
You can restart the Vault using:

```sh
kubectl rollout restart statefulset/vault
```

When a new Vault pod starts, it runs the Contrast Initializer as part of its startup sequence.
The Initializer receives the same workload secret as before, allowing it to derive the correct encryption key and unlock the existing LUKS-encrypted block device.
This process ensures that the Vault backend can reattach the previously encrypted volume and access all stored data transparently.
However, while this step enables access to the filesystem-level storage, it doesn't unlock access to the actual secrets.
Once Vault has been initialized, subsequent restarts rely on the auto-unsealing process, which is triggered via the transit secrets engine API provided by the Coordinator.

You can verify that the auto-unsealing process completed successful by inspecting the logs of the `openbao-server` container of the Vault pod:

```sh
kubectl logs vault-0 -c openbao-server
```

The log entries will indicate that the Vault has transitioned into unsealed state.
