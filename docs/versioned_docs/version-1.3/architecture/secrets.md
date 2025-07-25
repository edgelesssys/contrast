# Secrets & recovery

When the Coordinator is configured with the initial manifest, it generates a
random secret seed. From this seed, it uses an HKDF to derive the CA root key
and a signing key for the manifest history. This derivation is deterministic, so
the seed can be used to bring any Coordinator to this Coordinator's state.

The secret seed is returned to the user on the first call to `contrast set`,
encrypted with the user's public seed share owner key. If no seed share owner
key is provided, a key is generated and stored in the working directory.

## Persistence

The Coordinator runs as a `StatefulSet` with a dynamically provisioned
persistent volume. This volume stores the manifest history and the associated
runtime policies. The manifest isn't considered sensitive information, because
it needs to be passed to the untrusted infrastructure in order to start
workloads. However, the Coordinator must ensure its integrity and that the
persisted data corresponds to the manifests set by authorized users. Thus, the
manifest is stored in plain text, but is signed with a private key derived from
the Coordinator's secret seed.

## Recovery

When a Coordinator starts up, it doesn't have access to the signing secret and
can thus not verify the integrity of the persisted manifests. It needs to be
provided with the secret seed, from which it can derive the signing key that
verifies the signatures. This procedure is called recovery and is initiated by
the workload owner. The CLI decrypts the secret seed using the private seed
share owner key, verifies the Coordinator and sends the seed through the
`Recover` method. The Coordinator recovers its key material and verifies the
manifest history signature.

## Workload Secrets

The Coordinator provides each workload a secret seed during attestation. This
secret can be used by the workload to derive additional secrets for example to
encrypt persistent data. Like the workload certificates, it's written to the
`secrets/workload-secret-seed` path under the shared Kubernetes volume
`contrast-secrets`.

:::warning

The workload owner can decrypt data encrypted with secrets derived from the
workload secret. The workload owner can derive the workload secret themselves,
since it's derived from the secret seed known to the workload owner. If the data
owner and the workload owner is the same entity, then they can safely use the
workload secrets.

:::

### Secure persistence

Remember that persistent volumes from the cloud provider are untrusted. Using
the workload secret, applications can set up trusted storage on top of untrusted
block devices. The following, slightly abbreviated resource outlines how this
could be realized:

:::warning

This configuration snippet is intended to be educational and needs to be refined
and adapted to your production environment. Using it as-is may result in data
corruption or data loss.

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
          image: my.registry/my-image@sha256:0123...
          command:
            - /bin/sh
            - -ec
            - | # <-- Custom script that mounts the encrypted disk and then calls the original application.
                device=/dev/csi0
                if ! cryptsetup isLuks $device; then
                  cryptsetup luksFormat $device /contrast/secrets/workload-secret-seed
                  cryptsetup open $device state -d /contrast/secrets/workload-secret-seed
                  mkfs.ext4 /dev/mapper/state
                  cryptsetup close state
                fi
                cryptsetup open $device state -d /contrast/secrets/workload-secret-seed
                /path/to/original/app
          name: volume-tester
          volumeDevices:
            - name: state
              devicePath: /dev/csi0
          securityContext:
            privileged: true # <-- This is necessary for mounting devices.
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

:::note

This example assumes that you can modify the container image to include a shell
and the `cryptsetup` utility. Alternatively, you can set up a secure mount from
a sidecar container inside an `emptyDir` mount shared with the main container.
The Contrast end-to-end tests include [an example] of this type of mount.

[an example]: https://github.com/edgelesssys/contrast/blob/0662a2e/internal/kuberesource/sets.go#L504

:::
