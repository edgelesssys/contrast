# Contrast

Contrast runs confidential container deployments on untrusted Kubernetes at scale.

Contrast is based on the [Kata Containers](https://github.com/kata-containers/kata-containers) and
[Confidential Containers](https://github.com/confidential-containers) projects. Confidential Containers are
Kubernetes pods that are executed inside a confidential micro-VM and provide strong hardware-based isolation
from the surrounding environment. This works with unmodified containers in a lift-and-shift approach.
It currently targets the [CoCo preview on AKS](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview).

## The Contrast Coordinator

The Contrast Coordinator is the central remote attestation component of a Contrast deployment. It's a certificate
authority and issues certificates for workload pods running inside confidential containers. The Coordinator
is configured with a *manifest*, a configuration file that holds the reference values of all other parts of
a deployment. The Coordinator ensures that your app's topology adheres to your specified manifest. It verifies
the identity and integrity of all your services and establishes secure, encrypted communication channels between
the different parts of your deployment. As your app needs to scale, the Coordinator transparently verifies new
instances and then provides them with mesh credentials.

To verify your deployment, the remote attestation of the Coordinator and its manifest offers a single remote
attestation statement for your entire deployment. Anyone can use this to verify the integrity of your distributed
app, making it easier to assure stakeholders of your app's security.

## The Contrast Initializer

Contrast provides an Initializer that handles the remote attestation on the workload side transparently and
fetches the workload certificate. The Initializer runs as init container before your workload is started.

## Installation

Download the latest CLI from our release and put it into your PATH:

```sh
curl -fLo contrast https://github.com/edgelesssys/contrast/releases/download/latest/contrast
mv contrast /usr/local/bin/contrast
```

## Generic Workflow

### Prerequisite

A CoCo enabled cluster is required to run Contrast. Create it using the `az` CLI:

```sh
az extension add \
  --name aks-preview

az aks create \
  --resource-group myResourceGroup \
  --name myAKSCluster \
  --kubernetes-version 1.29 \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --node-count 1 \
  --generate-ssh-keys

az aks nodepool add \
  --resource-group myResourceGroup \
  --name nodepool2 \
  --cluster-name myAKSCluster \
  --mode System \
  --node-count 1 \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --workload-runtime KataCcIsolation

az aks get-credentials \
  --resource-group myResourceGroup \
  --name myAKSCluster
```

Check [Azure's deployment guide](https://learn.microsoft.com/en-us/azure/aks/deploy-confidential-containers-default-policy) for more detailed instructions.

### Deploy the Contrast Coordinator

Install the latest Contrast Coordinator release, comprising a single replica deployment and a
LoadBalancer service, into your cluster.

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/latest/coordinator.yml
```

### Preprare your Kubernetes resources

Contrast will add annotations to your Kubernetes YAML files. If you want to keep the original files
unchanged, you can copy the files into a separate local directory.
You can also generate files from a Helm chart or from a Kustomization.

```sh
mkdir resources
kustomize build $MY_RESOURCE_DIR > resources/all.yml
```

or

```sh
mkdir resources
helm template release-name chart-name > resources/all.yml
```

To specify that a workload (pod, deployment, etc.) should be deployed as confidential containers,
add `runtimeClassName: kata-cc-isolation` to the pod spec (pod definition or template).
In addition, add the Contrast Initializer as `initContainers` to these workloads and configure the
workload to use the certificates written to the `tls-certs` volumeMount.

```yaml
spec: # v1.PodSpec
  runtimeClassName: kata-cc-isolation
  initContainers:
  - name: initializer
    image: "ghcr.io/edgelesssys/contrast/initializer:latest"
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

### Generate policy annotations and manifest

Run the `generate` command generate the execution policies and add them as annotations to your
deployment files. A `manifest.json` with the reference values of your deployment will be created.

```sh
./contrast generate resources/*.yml
```

### Apply Resources

Apply the resources to the cluster. Your workloads will block in the initialization phase until a
manifest is set at the Coordinator.

```sh
kubectl apply -f resources/
```

### Connect to the Contrast Coordinator

For the next steps, we will need to connect to the Coordinator. The released Coordinator resource
includes a LoadBalancer definition we can use.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

> [!NOTE]
> `kubectl port-forward` uses a CRI method that is not supported by the Kata shim. If you
> cannot use a public load balancer, you can deploy a [deployments/simple/portforwarder.yml] and
> expose that with `kubectl port-forward` instead.
>
> Tracking issue: <https://github.com/kata-containers/kata-containers/issues/1693>.

### Set Manifest

Attest the Coordinator and set the manifest:

```sh
./contrast set -c "${coordinator}:1313" -m manifest.json resources/*.yml
```

After this step, the Coordinator will start issuing TLS certs to the workloads. The init container
will fetch a certificate for the workload and the workload is started.

### Verify the Coordinator

An end user (data owner) can verify the Contrast deployment using the `verify` command.

```sh
./contrast verify -c "${coordinator}:1313" -o ./verify
```

The CLI will attest the Coordinator using embedded reference values. The CLI will write the service mesh
root certificate and the history of manifests into the `verify/` directory. In addition, the policies referenced
in the manifest are also written to the directory.

### Communicate with Workloads

Connect to the workloads using the Coordinator's mesh root as a trusted CA certificate.
For example, with `curl`:

```sh
lbip=$(kubectl get svc ${MY_SERVICE} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
curl --cacert ./verify/mesh-root.pem "https://${lbip}:8443"
```

## Contributing

See the [contributing guide](CONTRIBUTING.md).
