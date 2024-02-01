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

## Generic Workflow

### Prerequisites

* An empty workspace directory.
* The `nunki` binary.
<!-- TODO: from where? -->

```sh
WORKSPACE=$(mktemp -d)
cd "$WORKSPACE"
curl -Lo nunki  https://...
```

### Kubernetes Resources

All Kubernetes resources that should be running in confidential containers must be present in the
resources directory. You can generate them from a Helm chart or from a Kustomization, or just copy
them over from your repository.

```sh
mkdir resources
kustomize build $MY_RESOURCE_DIR >resources/all.yaml
# or
helm template release-name chart-name >resources/all.yaml
# or
cp $MY_RESOURCE_DIR/*.yaml resources/
```

All pod definitions and templates in the resources need an additional init container that talks to
the coordinator. Furthermore, the runtime class needs to be set to `kata-cc-isolation` so that the
workloads are started as confidential containers.

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

Finally, you will need to deploy the Nunki coordinator, too. Start with the following definition
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
          image: "ghcr.io/edgelesssys/nunki/coordinator:latest"
---
apiVersion: v1
kind: Service
metadata:
  name: coordinator
spec:
  ports:
  - name: intercom
    port: 7777
    protocol: TCP
  - name: coordapi
    port: 1313
    protocol: TCP
  selector:
    app.kubernetes.io/name: coordinator
```

### Generate Policies and Manifest

Generate the runtime policy from the resource definitions, attach it to the resources as
annotation and write the coordinator manifest.

```sh
./nunki generate -m manifest.json resources/*.yaml
```

### Apply Resources

Apply the resources to the cluster. Your workloads will block in the initialization phase until a
manifest is set at the coordinator.

```sh
kubectl apply -f resources/
```

### Connect to the Coordinator

For the next steps, we will need to connect to the coordinator. If you did not expose the
coordinator with an external Kubernetes LoadBalancer, you can expose it locally:

```sh
kubectl port-forward service/coordinator 1313:coordapi &
```

### Set Manifest

Attest the coordinator and set the manifest. After this step, the coordinator should start issuing
TLS certs to the workloads, which should now leave the initialization phase.

```sh
./nunki set -c localhost:1313 -m manifest.json
```

### Verify the Coordinator

An end user can verify the Nunki deployment without an explicit list of resources by calling the
coordinator's verification RPC over attested TLS. After successful validation, the output directory
`verify/` will be populated with the TLS root certificates for the configured manifest.

```sh
./nunki verify -c localhost:1313 -o ./verify
```

### Communicate with Workloads

Connect to the workloads using the coordinator's mesh root as a trusted CA certificate.
For example, with `curl`:

```sh
kubectl port-forward service/$MY_SERVICE 8443:$MY_PORT &
curl --cacert ./verify/mesh-root.pem https://localhost:8443
```

## Contributing

See the [contributing guide](CONTRIBUTING.md).
