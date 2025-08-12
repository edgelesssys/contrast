# Secrets & recovery

When the Coordinator is configured with the initial manifest, it generates a random secret seed.
From this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.
This derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state.

The secret seed is returned to the user on the first call to `contrast set`, encrypted with the user's public seed share owner key.
If no seed share owner key is provided, a key is generated and stored in the working directory.

:::danger

The secret seed and the seed share owner key are highly sensitive.

- If either of them leak, the Contrast deployment should be considered compromised.
- If the secret seed is lost, data encrypted with Contrast secrets can't be recovered.
- If the seed share owner key is lost, the Coordinator can't be recovered and needs to be redeployed with a new manifest.

:::

## Workload Secrets

The Coordinator provides each workload a secret seed during attestation.
This secret can be used by the workload to derive additional secrets for example to encrypt persistent data.
Like the workload certificates, it's written to the `secrets/workload-secret-seed` path under the shared Kubernetes volume `contrast-secrets`.

The workload secret is deterministically derived from the secret seed and a workload secret ID from the manifest.
This implies that workload secrets are stable across manifest updates and Coordinator recovery.
By default, each workload is assigned an ID based on its qualified Kubernetes resource name.
This ID can be changed by adding an annotation to the pod (or pod template) metadata:

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    contrast.edgeless.systems/workload-secret-id: my-workload-secret
```

:::warning

The seed share owner can decrypt data encrypted with secrets derived from the workload secret, because they can themselves derive the workload secret.
If the data owner fully trusts the seed share owner (when they're the same entity, for example), they can securely use the workload secrets.

:::

### Secure persistence

<!-- TODO(burgerdev): this should be a how-to. -->

Remember that persistent volumes from the cloud provider are untrusted.
Applications can set up trusted storage on top of an untrusted block device using the `contrast.edgeless.systems/secure-pv` annotation.
This annotation enables `contrast generate` to configure the Initializer to set up a LUKS-encrypted volume at the specified device and mount it to a specified volume.
The LUKS encryption utilizes the workload secret introduced above.
Configure any workload resource with the following annotation:

```yaml
metadata:
  annotations:
    contrast.edgeless.systems/secure-pv: "device-name:mount-name"
```

This requires an existing block device named `device-name` which is configured as a volume on the resource.
The volume `mount-name` has to be of type `EmptyDir` and will be created if not present.
The resulting Initializer will mount both the device and configured volume and set up the encrypted storage.
Workload containers can then use the volume as a secure storage location:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  annotations:
    contrast.edgeless.systems/secure-pv: "device:secure"
  name: my-statefulset
spec:
  template:
    spec:
      containers:
        - name: my-container
          image: "my-image@sha256:..."
          volumeMounts:
            - mountPath: /secure
              mountPropagation: HostToContainer
              name: secure
      volumes:
        - name: device
          persistentVolumeClaim:
            claimName: my-pvc
      runtimeClassName: contrast-cc
```

#### Usage `cryptsetup` subcommand

Alternatively, the `cryptsetup` subcommand of the Initializer can be used to manually set up encrypted storage.
The `cryptsetup` subcommand takes two arguments `cryptsetup -d [device-path] -m [mount-point]`, to set up a LUKS-encrypted volume at `device-path` and mount that volume at `mount-point`.

The following, slightly abbreviated resource outlines how this could be realized:

:::warning

This configuration snippet is intended to be educational and needs to be refined and adapted to your production environment.
Using it as-is may result in data corruption or data loss.

:::

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: volume-tester
spec:
  template:
    spec:
      containers:
        - name: main
          image: my.registry/my-image@sha256:0123... # <-- Original application requiring encrypted disk.
          volumeMounts:
            - mountPath: /state
              mountPropagation: HostToContainer
              name: share
      initContainers:
        - args:
            - cryptsetup # <-- cryptsetup subcommand provided as args to the initializer binary.
            - "--device-path"
            - /dev/csi0
            - "--mount-point"
            - /state
          image: "ghcr.io/edgelesssys/contrast/initializer:latest"
          name: encrypted-volume-initializer
          resources:
            limits:
              memory: 100Mi
            requests:
              memory: 100Mi
          restartPolicy: Always
          securityContext:
            privileged: true # <-- This is necessary for mounting devices.
          startupProbe:
            exec:
              command:
                - /bin/test
                - "-f"
                - /done
            failureThreshold: 20
            periodSeconds: 5
          volumeDevices:
            - devicePath: /dev/csi0
              name: state
          volumeMounts:
            - mountPath: /state
              mountPropagation: Bidirectional
              name: share
      volumes:
        - name: share
          emptyDir: {}
      runtimeClassName: contrast-cc
  volumeClaimTemplates:
    - apiVersion: v1
      kind: PersistentVolumeClaim
      metadata:
        name: state
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
        volumeMode: Block # <-- The requested volume needs to be a raw block device.
```

### Transit secrets engine

In addition to the workload secrets provisioned by the initializer, Contrast workloads can ask the Coordinator to encrypt and decrypt secrets on their behalf.
The corresponding HTTP API is compatible with a subset of the [transit secrets API](https://openbao.org/api-docs/secret/transit/) used by [HashiCorp Vault](https://www.hashicorp.com/en/products/vault), and is served on Coordinator port 8200.
Its primary use case is [auto-unsealing of Vault deployments](../howto/vault.md), which can in turn provide fine-grained secrets management to Contrast workloads.

Workloads can only access the encryption key with the same name as their `workloadSecretID`.
For example, if the workload secret ID in the manifest is `my-secret-id`, they can use the endpoints `/v1/transit/encrypt/my-secret-id` and `/v1/transit/decrypt/my-secret-id`.
Like the workload secret, the encryption key is stable across manifest updates and subject to the same limitations.

If key rotation without changing the workload secret ID is desired, clients can pass a non-zero `key_version` parameter to the encryption request.
The version is passed as an input to the key derivation mechanism, which means that the encryption key changes with the `key_version` parameter.
Explicit key import, export or rotation operations aren't supported.

:::warning

The transit secret engine uses AES-256-GCM with random nonces.
In this mode, the risk of nonce reuse increases with the number of encrypted messages (see for example [NIST SP 800-38D, section 8.3](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf)).
Vault unsealing operates within the recommended limits, but other cryptographic use cases might not, so we explicitly recommend using a Vault workload (or similar KMS) for those.

:::
