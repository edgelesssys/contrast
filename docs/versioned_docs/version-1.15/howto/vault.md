# Vault

This how-to guides you through deploying Vault as a confidential deployment, using Contrast's built-in Vault support.

## Applicability

Confidential applications often need access to cryptographic keys and other secrets.
Contrast has built-in support for operating Hashicorp Vault / OpenBao, which can be used to setup a key management service for your applications.
While this how-to uses an OpenBao server, the setup for Hashicorp Vault is very similar.

## Prerequisites

1. [Set up cluster](cluster-setup/bare-metal.md)
2. [Install CLI](install-cli.md)
3. [Deploy the Contrast runtime](workload-deployment/runtime-deployment.md)
4. Install the `bao` CLI (see [OpenBao installation instructions](https://openbao.org/docs/install/))
5. A domain name that resolves to the Vault service IP.
   For testing purposes, you can use an entry in `/etc/hosts` instead.

## How-to

The following sections explain how to add a Vault to your Contrast deployment, how to configure automatic unsealing and how to use Contrast certificates for authentication.
Refer to the [secrets page](../architecture/secrets.md#transit-secrets-engine) for more information on Contrast's transit engine API.

:::note[Example deployment]
The vault configuration is of considerable size.
To help you get started, the Contrast release includes a demo resource definition for a Vault server.
You can download the demo app by running:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/vault-demo.yml --create-dirs --output-dir resources
```

Consider this demo resource as a starting place and adapt it to fit your deployment.
:::

### Background information

The Vault deployment is defined as a `StatefulSet` using an OpenBao image, with a mounted block device for persistent storage.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: vault
spec:
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/secure-pv: state:share
```

The Contrast Initializer, running as an init container, uses the workload secret located at `/contrast/secrets/workload-secret-seed` to generate an encryption key and initialize the block device `state` as a LUKS-encrypted partition.
Before the Vault container starts, the Initializer opens the LUKS device using the generated key.
This unlocked device is then mounted by the Vault container and used as the backend storage volume `share`.
For the Vault application, this process is entirely transparent, and the device behaves like a standard volume mount.

The LUKS encryption of the block device is primarily a convenience feature, enabling persistent storage at the filesystem level on confidential virtual machines.
The primary and security-relevant encryption mechanism remains Vault's own sealing process, which provides cryptographic protection of secrets even if the underlying storage is compromised.

Because the `workload-secret-seed` is derived from the associated `workloadSecretID`, any change to the `workloadSecretID` after the block device has been initialized will result in a different key, making the mounted block device undecryptable.
Therefore, it's critical to ensure that the `workloadSecretID` is correctly aligned with the intended endpoint specified in Vault's unsealing configuration before the first `contrast set` is executed.
In this example, the `workloadSecretID` is set to `vault_unsealing` with an annotation:

```yaml
spec:
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/secure-pv: state:share
        contrast.edgeless.systems/workload-secret-id: vault_unsealing
```

Vault's TCP listener is configured to use the certificate provisioned by the initializer.
Below is the relevant excerpt from the example configuration file, denoted there as `VAULT_CONFIG`.

```hcl
listener "tcp" {
  address            = "0.0.0.0:8200"
  tls_cert_file      = "/contrast/tls-config/certChain.pem"
  tls_key_file       = "/contrast/tls-config/key.pem"
}
```

The Vault configuration file would ideally just be mounted from the `ConfigMap` into the server pod.
However, this would be insecure, as the Vault configuration is sensitive but mounts aren't integrity-checked (see [Hardening](hardening.md#limitations-inherent-to-policy-checking)).
Therefore, the demo pod adds the configuration as an environment variable and extracts it to a memory-backed file before starting the server.

### Deploy Vault

Follow the steps of the generic workload deployment instructions:

- [Add the Coordinator.](workload-deployment/add-coordinator.md)
- [Generate the annotations.](workload-deployment/generate-annotations.md)
  - After running `contrast generate`, add the desired Vault domain name to the vault `SANs` array in `manifest.json`.
- [Apply the resources.](workload-deployment/deploy-application.md)
- [Set the manifest.](workload-deployment/set-manifest.md)

### Configure DNS

The `bao` CLI, as well as the `vault` CLI, need to be configured with an HTTPS endpoint, and that endpoint needs to match the certificate that Vault is presenting.
By default, the manifest only contains the Subject Alternative Name (SAN) `vault`.

First, you need to know the public IP of the Vault service, which you can print with the following command:

```sh
kubectl get service vault -o=jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

If you added SANs in the previous step, these will also be included in the certificate.
In that case, you can configure your DNS service to resolve the additional SAN to the load balancer IP.

Alternatively, configure the system from which you access Vault to resolve `vault` to the load balancer IP.
On Linux, you can modify `/etc/hosts` for this.

### Initialize Vault

A Vault instance needs to be initialized once before it can be used.
Initialization is a sensitive operation that transfers key material, and should thus be executed on a secure system, by a trusted operator and over a secure channel.
For this demo, the machine hosting your Contrast workspace is sufficient.
Initialize the Vault with the `bao` CLI from within your workspace, using the Coordinator CA as a root of trust:

```sh
export VAULT_ADDR=https://${YOUR_VAULT_DOMAIN}:8200
export VAULT_CACERT=./coordinator-root-ca.pem
bao operator init -format json | tee vault_secrets.json
export VAULT_TOKEN=$(jq -r '.root_token' < vault_secrets.json)
```

Upon successful initialization, this prints a root token and some recovery key shares and also writes them to the file `vault_secrets.json`.
These are highly sensitive secrets and need to be guarded carefully!
The last command makes the Vault root token available to the `bao` CLI, allowing you to configure Vault in the next section.

### Configure Vault

Vault can be configured to use Contrast certificates for authorization.
The following commands enable [certificate authentication](https://developer.hashicorp.com/vault/docs/auth/cert) and assign a policy `contrast` to workloads with Contrast certificates.

```sh
bao auth enable cert
bao write auth/cert/certs/coordinator display_name=coordinator policies=contrast certificate=@./mesh-ca.pem
```

For this demo, we're going to activate the KV secrets engine, write a demo secret and add the `contrast` policy that allows Contrast workloads to access it.

```sh
bao secrets enable -version=1 kv
bao kv put kv/my-secret my-value=s3cr3t
bao policy write contrast - <<EOF
path "kv/*"
{
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF
```

The demo includes a simple deployment - `vault-client` - that attempts to authenticate to Vault with its Contrast certificate and then tries to read a secret.
After running the above commands, the pod should become ready and show the secret content in its log output.

### Restart Vault with auto-unsealing

When a Vault pod is restarted, it must unseal its encrypted state. The sealing key isn't present in the restarted pod, but can be retrieved from the Coordinator's transit engine API after the Coordinator has attested the restarted pod.

To configure Vault to use the Coordinator's transit engine API for automatic unsealing, the following configuration is used in the demo deployment, which is part of the "vault-config" config map.
Note that the `key_name` needs to be equal to the `workloadSecretID` of Vault.

```hcl
seal "transit" {
  address         = "https://coordinator:8200"
  disable_renewal = "true"
  key_name        = "vault_unsealing"
  mount_path      = "transit/"
  tls_ca_cert     = "/contrast/tls-config/mesh-ca.pem"
  tls_client_cert = "/contrast/tls-config/certChain.pem"
  tls_client_key  = "/contrast/tls-config/key.pem"
}
```

This configuration instructs Vault to unseal itself with key material obtained from the Coordinator.
To see this process in action, you can trigger a restart of your Vault pod:

```sh
kubectl rollout restart statefulset/vault
```

When a new Vault pod starts, it runs the Contrast Initializer as part of its startup sequence.
The Initializer receives the same workload secret as before, allowing it to derive the correct encryption key and unlock the existing LUKS-encrypted block device.
This process ensures that the Vault backend can reattach the previously encrypted volume and access all stored data transparently.
However, while this step enables access to the filesystem-level storage, it doesn't unlock access to the actual secrets.
When the main Vault container starts, it finds the sealed data on the volume and begins the unsealing process.

You can verify that the auto-unsealing process completed successful by inspecting the logs of the Vault pod, or by running

```sh
bao status
```
