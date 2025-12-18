# Set up encrypted persistent storage

This section guides you through the process of configuring encrypted persistent storage for your application.

## Applicability

This step is recommended for any Contrast deployment that stores sensitive data persistently.

## Prerequisites

1. [Set up your cluster](./cluster-setup/bare-metal.md)
2. [Install CLI](./install-cli.md)
3. [Deploy the Contrast runtime](./workload-deployment/runtime-deployment.md)

## How-to

The following demonstrates how to set up an encrypted LUKS mount for an arbitrary `/mount/path` directory to easily deploy an application with encrypted persistent storage using Contrast.

### Configuration

Add a `volume` of type `emptyDir` to your pod.
Mount it through a `volumeMount` with `HostToContainer` propagation.
The specified `mountPath` is where the Contrast Initializer will mount the encrypted volume.
Use a `volumeClaimTemplate` to make a persistent storage device available to your deployment.
The Initializer creates an encrypted volume and a filesystem on this device.
Supported volume types can be found in the [k8s documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#raw-block-volume-support).

Finally, add the [Contrast annotation](../architecture/k8s-yaml-elements.md) `contrast.edgeless.systems/secure-pv` with value `device-name:mount-name`.
The presence of this annotation instructs an init container running `cryptsetup`
to use the workload secret at `/contrast/secrets/workload-secret-seed` to generate
a key and setup the block device as a LUKS partition. Before starting the application container,
the init container uses the generated key to open the LUKS device, which is then mounted
by the application container. For the application container, this process is completely
transparent and works like mounting any other volume. The `cryptsetup` container
will remain running to provide the necessary decryption context for the workload
container.

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
      - emptyDir:
          medium: Memory
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

:::note[Persistent workload secrets]

Secure persistent volumes make use of workload secrets to be able to re-open devices after container restarts.
Please read about the implications in the [workload secrets](../architecture/secrets.md#workload-secrets) section.

:::

### Deployment

Follow the steps of the generic workload deployment instructions:

- [Add the Coordinator.](./workload-deployment/add-coordinator.md)
- [Generate the annotations.](./workload-deployment/generate-annotations.md)
- [Apply the resources.](./workload-deployment/deploy-application.md)
- [Set the manifest.](./workload-deployment/set-manifest.md)

## Example application: MySQL

The above guide can be tried out with a [MySQL](https://mysql.com) sample deployment which is part of the contrast release.
You can add the deployment file to your resources by running:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/download/v1.15.1/mysql-demo.yml --output-dir resources
```

:::note[MySQL]

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

:::

The MySQL server is defined as a `StatefulSet`, with a storage setup analogous to what has been described above.
A `1Gi` block device is provided to the `mysql-backend` container through a `volumeClaimTemplate` and mounted to `/var/lib/mysql`, MySQL's default state directory.
An example MySQL client is defined through a separate deployment. The `mysql-client` container connects to the backend and calls on it to perform demo database operations.
Since the backend's MySQL state directory is located on the prepared storage device, the results of these operations are persisted.

Follow [the steps given above](#deployment) to reapply your configuration.

:::note[Runtime class and Initializer]

The deployment YAML shipped for this demo is already configured to be used with Contrast.
A [runtime class](../architecture/components/runtime) `contrast-cc`
was added to the pods to signal they should be run as confidential containers. During the generation process,
the Contrast [Initializer](../architecture/components/initializer.md) will be added as an init container to these
workloads. It will attest the pod to the Coordinator and fetch the workload certificates and the workload secret.

Further, the deployment YAML is also configured with the Contrast [service mesh](../architecture/components/service-mesh.md).
The configured service mesh proxy provides transparent protection for the communication between
the MySQL server and client.

:::
