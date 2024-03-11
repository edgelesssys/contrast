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

## How to deploy emojivoto in Contrast

### Prerequisites

Install the latest version of the [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/).
[Login to your account](https://docs.microsoft.com/en-us/cli/azure/authenticate-azure-cli) which has permissions to create an AKS cluster, by
executing:

```sh
az login
```

Create an AKS cluster with Confidential Container support:

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

### Download the latest Contrast release

Since Contrast has not yet been publicly released, a download URL will be
provided by Edgeless.

```sh
wget <URL>
unzip contrast.zip
```

### Deploy the Contrast Coordinator

Install the latest Contrast Coordinator release, comprising a single replica
deployment and a LoadBalancer service, into your cluster:

```sh
kubectl apply -f coordinator.yml
```

### Annotate the emojivoto deployment

Run the generate command generate the execution policies and add them as
annotations to your deployment files. A manifest.json with the reference values
of your deployment will be created:

```sh
./contrast generate deployment/*
```

### Set the manifest

Attest the Coordinator and set the manifest. It might take up to a few minutes
for the load balancer to be created.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo $coordinator
./contrast set -c "${coordinator}:1313" -m manifest.json deployment/
```

### Deploy emojivoto

Since the coordinator has a manifest set, which defines the amojivoto deployment
as an allowed workload, we can deploy the application:

```sh
kubectl apply -f deployment/
```

### Verify the Coordinator

An end user (data owner) can verify the Contrast deployment using the verify
command:

```sh
./contrast verify -c "${coordinator}:1313"
```

The CLI will attest the Coordinator using embedded reference values. The CLI
will write the service mesh root certificate and the history of manifests into
the `verify/` directory. In addition, the policies referenced in the manifest are
also written to the directory.

### Connect and verify the workload

Connect to the workloads using the Coordinator's mesh root as a trusted CA certificate. For example, with curl:

```sh
kubectl patch svc web-svc -p '{"spec": {"type": "LoadBalancer"}}'
lbip=$(kubectl get svc web-svc -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo $lbip
curl --cacert ./verify/mesh-root.pem -k "https://${lbip}"
```

The workload certificate is a DNS wildcard certificate, `curl`, but SAN
verification fails when accessing the workload via an IP address.
On Azure all load balancers automatically get ephemeral DNS entries, so either
use that or configure DNS yourself.

To validate the certificate locally, use `openssl`:

```sh
openssl s_client -showcerts -connect ${lbip}:443 </dev/null | sed -n -e '/-.BEGIN/,/-.END/ p' > certChain.pem
awk 'BEGIN {c=0;} /BEGIN CERT/{c++} { print > "cert." c ".pem"}' < certChain.pem
openssl verify -verbose -trusted verify/mesh-root.pem -- cert.1.pem
```

## Current limitations

Contrast is in an early development stage and most underlying projects are under development, too.
As a result there are currently certain limitations, from which we try to document the most significant
ones here:

- Only availabile on AKS with CoCo preview (AMD SEV-SNP)
- Persistent volumes currently not supported in CoCo
- While workload policies are functional in general, but [not covering all edge cases](https://github.com/microsoft/kata-containers/releases/tag/genpolicy-0.6.2-5)
- Port-forwarding isn't supported by Kata Containers yet
- CLI only available for Linux (mostly because upstream dependencies are not availabile for other platforms)

## Upcoming Contrast features

- Transparent service mesh (apps can currently use mTLS with Coordinator certs for secure communication)
- Plugin key management service (KMS) for attestation/coordinator certificate based key release
- High availability (distributed Contrast Coordinator)
