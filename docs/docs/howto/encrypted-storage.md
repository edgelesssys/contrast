# Set up encrypted persistent storage

This section guides you through the process of configuring encrypted persistent storage for your application.

## Applicability

This step is recommended for any Contrast deployment that stores sensitive data persistently.

## Prerequisites

1. [Set up your cluster](./cluster-setup/bare-metal.md)
2. [Install CLI](./install-cli.md)
3. [Deploy the Contrast runtime](./workload-deployment/runtime-deployment.md)
4. [Add Coordinator to resources](./workload-deployment/set-manifest.md)
5. [Prepare deployment files](./workload-deployment/deployment-file-preparation.md)
6. [Generate annotations and manifest](./workload-deployment/generate-annotations.md)
7. [Deploy application](./workload-deployment/deploy-application.md)

## How-to

The following demonstrates how to set up an encrypted LUKS mount for an arbitrary `/mount/path` directory to easily deploy an application with encrypted persistent storage using Contrast.

### Configuration

Add a `volume` to your pod or stateful set (called `mount-name` in the example configuration below)
which references your encrypted, persistent storage device either through a [persistent volume claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/), or an [iSCSI device](https://kubernetes.io/docs/concepts/storage/volumes/#iscsi).
To your application container, add a `volumeMount` with the `mount-name` used for the `volume`.
This `volumeMount` **must** be of type `emptyDir`.

Additionally, make your persistent volume claim or iSCSI device available to the deployment; the example below uses a PVC `1Gi` in size.
Give the device a name (the example uses `device-name`).

Finally, add the [Contrast annotation](../architecture/k8s-yaml-elements.md) `contrast.edgeless.systems/secure-pv` with value `device-name:mount-name`.

```yaml
spec: # v1.StatefulSetSpec
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/secure-pv: device-name:mount-name
    spec:
      containers:
      - volumeMounts:
        - mountPath: /mount/path
          mountPropagation: HostToContainer
          name: mount-name
      volumes:
      - emptyDir: {}
        name: mount-name
  volumeClaimTemplates:
  - apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: device-name
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
      volumeMode: Block
```

The presence of this annotation instructs an init container running `cryptsetup`
to use the workload secret at `/contrast/secrets/workload-secret-seed` to generate
a key and setup the block device as a LUKS partition. Before starting the application container,
the init container uses the generated key to open the LUKS device, which is then mounted
by the application container. For the application container, this process is completely
transparent and works like mounting any other volume. The `cryptsetup` container
will remain running to provide the necessary decryption context for the workload
container.

:::note[Persistent workload secrets]

During the initialization process of the workload pod, the Contrast Initializer
sends an attestation report to the Coordinator and receives a [workload secret](../architecture/secrets.md#workload-secrets)
derived from the Coordinator's secret seed and the workload secret ID specified in the
manifest, and writes it to a secure in-memory `volumeMount`.

:::

### Applying the configuration

To make the changes take effect, reapply your deployment:

```sh
contrast generate resources/
kubectl apply -f resources/
```

## Verifying the deployment

Users of an app may want to verify its security and identity before sharing data by writing it to the persistent volume.
Please see [this section](../getting-started/deployment.md#7-verify-deployment) of the documentation for more information on this topic.

## Example application: MySQL

The above guide can be tried out with a [MySQL](https://mysql.com) sample deployment which is part of the contrast release.
You can add the deployment file to your resources by running:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/mysql-demo.yml --output-dir resources
```


MySQL is an open-source database used to organize data into
tables and quickly retrieve information about its content. All of the data in a
MySQL database is stored in the `/var/lib/mysql` directory. In this example, we
use the workload secret to setup an encrypted LUKS mount for the
`/var/lib/mysql` directory to easily deploy an application with encrypted
persistent storage using Contrast.

The resources provided in this demo are designed for educational purposes and
shouldn't be used in a production environment without proper evaluation. When
working with persistent storage, regular backups are recommended in order to
prevent data loss. For confidential applications, please also refer to the
[security considerations](./hardening.md). Also be
aware of the differences in security implications of the workload secrets for
the data owner and the workload owner. For more details, see the [Workload
Secrets](../architecture/secrets.md#workload-secrets) documentation.

After downloading the MySQL demo file, follow [the steps given above](#applying-the-configuration) to reapply your configuration.

:::note[Runtime class and Initializer]

The deployment YAML shipped for this demo is already configured to be used with Contrast.
A [runtime class](../architecture/components/runtime) `contrast-cc`
was added to the pods to signal they should be run as Confidential Containers. During the generation process,
the Contrast [Initializer](../architecture/components/initializer.md) will be added as an init container to these
workloads. It will attest the pod to the Coordinator and fetch the workload certificates and the workload secret.

Further, the deployment YAML is also configured with the Contrast [service mesh](../architecture/components/service-mesh.md).
The configured service mesh proxy provides transparent protection for the communication between
the MySQL server and client.

:::
