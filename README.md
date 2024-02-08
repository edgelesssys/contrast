# Nunki

Nunki ([/ˈnʌŋki/](https://en.wikipedia.org/wiki/Sigma_Sagittarii)) runs confidential container deployments
on untrusted Kubernetes at scale.

Nunki is based on the [Kata Containers](https://github.com/kata-containers/kata-containers) and
[Confidential Containers](https://github.com/confidential-containers) projects. Confidential Containers are
Kubernetes pods that are executed inside a confidential micro-VM and provide strong hardware-based isolation
from the surrounding environment. This works with unmodified containers in a lift-and-shift approach.

## The Nunki Coordinator

The Nunki Coordinator is the central remote attestation component of a Nunki deployment. It's a certificate
authority and issues certificates for workload pods running inside confidential containers. The Coordinator
is configured with a *manifest*, a configuration file that holds the reference values of all other parts of
a deployment. The Coordinator ensures that your app's topology adheres to your specified manifest. It verifies
the identity and integrity of all your services and establishes secure, encrypted communication channels between
the different parts of your deployment. As your app needs to scale, the Coordinator transparently verifies new
instances and then provides them with mesh credentials.

To verify your deployment, the remote attestation of the Coordinator and its manifest offers a single remote
attestation statement for your entire deployment. Anyone can use this to verify the integrity of your distributed
app, making it easier to assure stakeholders of your app's security.

## The Nunki Initializer

Nunki provides an Initializer that handles the remote attestation on the workload side transparently and
fetches the workload certificate. The Initializer runs as init container before your workload is started.

## Installation

Download the latest CLI from our release and put it into your PATH:

```sh
curl -fLo nunki https://github.com/edgelesssys/nunki/releases/download/latest/nunki
mv nunki /usr/local/bin/nunki
```

## Generic Workflow

### Preprare your Kubernetes resources

Nunki will add annotations to your Kubernetes YAML files. If you want to keep the original files
unchanged, you can copy the files into a separate local directory.
You can also generate files from a Helm chart or from a Kustomization.

```sh
mkdir resources
kustomize build $MY_RESOURCE_DIR > resources/all.yaml
```

or

```sh
mkdir resources
helm template release-name chart-name > resources/all.yaml
```

To specify that a workload (pod, deployment, etc.) should be deployed as confidential containers,
add `runtimeClassName: kata-cc-isolation` to the pod spec (pod definition or template).
In addition, add the Nunki Initializer as `initContainers` to these workloads and configure the
workload to use the certificates written to the `tls-certs` volumeMount.

```yaml
spec: # v1.PodSpec
  runtimeClassName: kata-cc-isolation
  initContainers:
  - name: initializer
    image: "ghcr.io/edgelesssys/nunki/initializer:latest"
    env:
    - name: COORDINATOR_HOST
      value: coordinator
    volumeMounts:
    - name: tls-certs
      mountPath: /tls-config
  volumes:
  - name: tls-certs
    emptyDir: {}
```

Finally, you will need to deploy the Nunki Coordinator, too. Start with the following definition
in `resources/coordinator.yaml` and adjust it as you see fit (e.g. labels, namespace, service attributes).

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coordinator
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: coordinator
  replicas: 1
  template:
    metadata:
      labels:
        app.kubernetes.io/name: coordinator
      annotations:
        nunki.edgeless.systems/pod-role: coordinator
    spec:
      runtimeClassName: kata-cc-isolation
      containers:
        - name: coordinator
          image: "ghcr.io/edgelesssys/nunki/coordinator:v0.1.0"
---
apiVersion: v1
kind: Service
metadata:
  name: coordinator
spec:
  ports:
  - port: 7777
  - port: 1313
  selector:
    app.kubernetes.io/name: coordinator
```

### Generate policy annotations and manifest

Run the `generate` command generate the execution policies and add them as annotations to your
deployment files. A `manifest.json` with the reference values of your deployment will be created.

```sh
./nunki generate resources/*.yaml
```

### Apply Resources

Apply the resources to the cluster. Your workloads will block in the initialization phase until a
manifest is set at the Coordinator.

```sh
kubectl apply -f resources/
```

### Set Manifest

For the next steps, we will need to connect to the Coordinator. If you did not expose the
coordinator with an external Kubernetes LoadBalancer, you can expose it locally:

```sh
kubectl port-forward service/coordinator 1313:coordapi &
```

### Set Manifest

Attest the Coordinator and set the manifest:

```sh
./nunki set -c localhost:1313 -m manifest.json
```

After this step, the Coordinator will start issuing TLS certs to the workloads. The init container
will fetch a certificate for the workload and the workload is started.

### Verify the Coordinator

An end user (data owner) can verify the Nunki deployment using the `verify` command.

```sh
./nunki verify -c localhost:1313 -o ./verify
```

The CLI will attest the Coordinator using embedded reference values. The CLI will write the service mesh
root certificate and the history of manifests into the `verify/` directory. In addition, the policies referenced
in the manifest are also written to the directory.

### Communicate with Workloads

Connect to the workloads using the Coordinator's mesh root as a trusted CA certificate.
For example, with `curl`:

```sh
kubectl port-forward service/$MY_SERVICE 8443:$MY_PORT &
curl --cacert ./verify/mesh-root.pem https://localhost:8443
```

## Contributing

See the [contributing guide](CONTRIBUTING.md).
