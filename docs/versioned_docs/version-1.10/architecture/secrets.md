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
          image: "ghcr.io/edgelesssys/contrast/initializer:v1.10.0@sha256:490a8fe9ad18a39ab614915df251dbdcbffd4726b38cd4b06a034017c9d2bc26"
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
