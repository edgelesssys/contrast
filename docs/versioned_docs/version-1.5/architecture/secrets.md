# Secrets & recovery

When the Coordinator is configured with the initial manifest, it generates a random secret seed.
From this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.
This derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state.

The secret seed is returned to the user on the first call to `contrast set`, encrypted with the user's public seed share owner key.
If no seed share owner key is provided, a key is generated and stored in the working directory.

:::danger

The secret seed and the seed share owner key are highly sensitive.

* If either of them leak, the Contrast deployment should be considered compromised.
* If the secret seed is lost, data encrypted with Contrast secrets can't be recovered.
* If the seed share owner key is lost, the Coordinator can't be recovered and needs to be redeployed with a new manifest.

:::

## Persistence

The Coordinator runs as a `StatefulSet` with a dynamically provisioned persistent volume.
This volume stores the manifest history and the associated runtime policies.
The manifest isn't considered sensitive information, because it needs to be passed to the untrusted infrastructure in order to start workloads.
However, the Coordinator must ensure its integrity and that the persisted data corresponds to the manifests set by authorized users.
Thus, the manifest is stored in plain text, but is signed with a private key derived from the Coordinator's secret seed.

## Recovery

When a Coordinator starts up, it doesn't have access to the signing secret and can thus not verify the integrity of the persisted manifests.
It needs to be provided with the secret seed, from which it can derive the signing key that verifies the signatures.
This procedure is called recovery and is initiated by the seed share owner.
The CLI decrypts the secret seed using the private seed share owner key, verifies the Coordinator and sends the seed through the `Recover` method.
The Coordinator authenticates the seed share owner, recovers its key material, and verifies the manifest history signature.

## Workload Secrets

The Coordinator provides each workload a secret seed during attestation.
This secret can be used by the workload to derive additional secrets for example to encrypt persistent data.
Like the workload certificates, it's written to the `secrets/workload-secret-seed` path under the shared Kubernetes volume `contrast-secrets`.

:::warning

The seed share owner can decrypt data encrypted with secrets derived from the workload secret, because they can themselves derive the workload secret.
If the data owner fully trusts the seed share owner (when they're the same entity, for example), they can securely use the workload secrets.

:::

### Secure persistence

Remember that persistent volumes from the cloud provider are untrusted.
Using the built-in `cryptsetup` subcommand of the initializer, applications can set up trusted storage on top of untrusted block devices based on the workload secret.
Functionally the initializer will act as a sidecar container which serves to set up a secure mount inside an `emptyDir` mount shared with the main container.

#### Usage `cryptsetup` subcommand

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
          image: "ghcr.io/edgelesssys/contrast/initializer:v1.5.1@sha256:6663c11ee05b77870572279d433fe24dc5ef6490392ee29a923243cfc40f2f35"
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
